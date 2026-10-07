package usecase

import (
	"context"
	"football/internal/application/ports"
	"log/slog"
	"sync"
	"time"
)

type SyncRunner interface {
	Run(context.Context, bool) error
}
type SyncManager struct {
	store              ports.SyncStore
	runner             SyncRunner
	clock              ports.Clock
	root               context.Context
	cooldown, interval time.Duration
	first              time.Time
	log                *slog.Logger
	mu                 sync.Mutex
	closed             bool
	wg                 sync.WaitGroup
}

func NewSyncManager(root context.Context, store ports.SyncStore, runner SyncRunner, clock ports.Clock, cooldown, interval time.Duration, log *slog.Logger) *SyncManager {
	return &SyncManager{store: store, runner: runner, clock: clock, root: root, cooldown: cooldown, interval: interval, first: clock.Now().Add(interval), log: log}
}
func (m *SyncManager) Status(ctx context.Context) (ports.SyncStatus, error) {
	st, err := m.store.SyncStatus(ctx)
	if err != nil {
		return st, err
	}
	now := m.clock.Now()
	if st.State == "running" && st.Deadline != nil && !now.Before(*st.Deadline) {
		st.State = "interrupted"
		st.Message = "งานก่อนหน้าหยุดทำงาน ข้อมูลเดิมยังอยู่"
	}
	st.Automatic = m.interval > 0
	if st.Automatic {
		next := m.first
		if st.StartedAt != nil {
			next = st.StartedAt.Add(m.interval)
		}
		if st.RetryAt != nil && next.Before(*st.RetryAt) {
			next = *st.RetryAt
		}
		st.NextScheduledAt = &next
	}
	return st, nil
}
func (m *SyncManager) prepare(ctx context.Context, source string) (ports.SyncLock, ports.SyncStatus, error) {
	lock, ok, err := m.store.TrySyncLock(ctx)
	if err != nil {
		return nil, ports.SyncStatus{}, err
	}
	if !ok {
		return nil, ports.SyncStatus{}, fail("SYNC_RUNNING", "กำลังซิงก์ข้อมูลอยู่แล้ว", 409)
	}
	st, err := m.store.SyncStatus(ctx)
	if err != nil {
		lock.Release()
		return nil, st, err
	}
	now := m.clock.Now()
	if st.RetryAt != nil && now.Before(*st.RetryAt) {
		lock.Release()
		return nil, st, fail("SYNC_COOLDOWN", "กรุณารอช่วงพักก่อนซิงก์อีกครั้ง", 429)
	}
	deadline := now.Add(2 * time.Minute)
	retry := now.Add(m.cooldown)
	st.State = "running"
	st.Source = source
	st.StartedAt = &now
	st.FinishedAt = nil
	st.Deadline = &deadline
	st.RetryAt = &retry
	st.Message = "กำลังซิงก์ ข้อมูลเดิมยังแสดงอยู่"
	if err = lock.Save(ctx, st); err != nil {
		lock.Release()
		return nil, st, err
	}
	return lock, st, nil
}
func (m *SyncManager) execute(lock ports.SyncLock, st ports.SyncStatus) error {
	defer lock.Release()
	ctx, cancel := context.WithTimeout(m.root, 2*time.Minute)
	defer cancel()
	err := m.runner.Run(ctx, false)
	now := m.clock.Now()
	st.FinishedAt = &now
	st.Deadline = nil
	retry := now.Add(m.cooldown)
	st.RetryAt = &retry
	if err == nil {
		st.State = "succeeded"
		st.LastSuccessAt = &now
		st.Message = "ซิงก์สำเร็จ"
	} else {
		st.State = "failed"
		st.Message = "ซิงก์ไม่สำเร็จ ข้อมูลเดิมยังอยู่ ตรวจแพ็กเกจ API โควตา หรือการเชื่อมต่อ"
		m.log.Error("sync failed", "error", err)
	}
	// Cancellation must not prevent writing the final outcome and releasing the lock.
	finish, cancelFinish := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFinish()
	if saveErr := lock.Save(finish, st); saveErr != nil {
		m.log.Error("save sync status", "error", saveErr)
		if err == nil {
			return saveErr
		}
	}
	return err
}

// Start reserves the database lock before returning; work outlives the HTTP request.
func (m *SyncManager) Start(ctx context.Context, source string) (ports.SyncStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.root.Err() != nil {
		return ports.SyncStatus{}, fail("SHUTTING_DOWN", "ระบบกำลังหยุดทำงาน", 503)
	}
	lock, st, err := m.prepare(ctx, source)
	if err != nil {
		return st, err
	}
	m.wg.Add(1)
	st.Automatic = m.interval > 0
	go func(snapshot ports.SyncStatus) { defer m.wg.Done(); _ = m.execute(lock, snapshot) }(st)
	return st, nil
}
func (m *SyncManager) Run(ctx context.Context) error {
	lock, st, err := m.prepare(ctx, "worker")
	if err != nil {
		return err
	}
	return m.execute(lock, st)
}
func (m *SyncManager) Scheduled(ctx context.Context) error {
	if m.interval <= 0 {
		return nil
	}
	st, err := m.Status(ctx)
	if err != nil {
		return err
	}
	if st.State == "running" || st.NextScheduledAt == nil || m.clock.Now().Before(*st.NextScheduledAt) {
		return nil
	}
	_, err = m.Start(ctx, "schedule")
	return err
}
func (m *SyncManager) Serve() {
	if m.interval <= 0 {
		return
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.wg.Add(1)
	m.mu.Unlock()
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-m.root.Done():
				return
			case <-ticker.C:
				if err := m.Scheduled(m.root); err != nil {
					m.log.Error("scheduled sync", "error", err)
				}
			}
		}
	}()
}
func (m *SyncManager) Wait() { m.mu.Lock(); m.closed = true; m.mu.Unlock(); m.wg.Wait() }
