package usecase

import (
	"context"
	"errors"
	"football/internal/application/ports"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type syncMemory struct {
	mu     sync.Mutex
	status ports.SyncStatus
	locked bool
}

func (s *syncMemory) SyncStatus(context.Context) (ports.SyncStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status, nil
}
func (s *syncMemory) TrySyncLock(context.Context) (ports.SyncLock, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return nil, false, nil
	}
	s.locked = true
	return &memorySyncLock{s}, true, nil
}

type memorySyncLock struct{ s *syncMemory }

func (l *memorySyncLock) Save(_ context.Context, st ports.SyncStatus) error {
	l.s.mu.Lock()
	defer l.s.mu.Unlock()
	l.s.status = st
	return nil
}
func (l *memorySyncLock) Release() { l.s.mu.Lock(); defer l.s.mu.Unlock(); l.s.locked = false }

type syncRunFunc func(context.Context, bool) error

func (f syncRunFunc) Run(ctx context.Context, seed bool) error { return f(ctx, seed) }
func syncLog() *slog.Logger                                    { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
func TestSyncSharedLockAndRequestCancellation(t *testing.T) {
	store := &syncMemory{}
	entered, finish := make(chan struct{}), make(chan struct{})
	runner := syncRunFunc(func(ctx context.Context, _ bool) error {
		close(entered)
		select {
		case <-finish:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	manager := NewSyncManager(context.Background(), store, runner, fixedClock{time.Now()}, time.Minute, 0, syncLog())
	req, cancel := context.WithCancel(context.Background())
	if _, err := manager.Start(req, "owner"); err != nil {
		t.Fatal(err)
	}
	cancel()
	<-entered
	other := NewSyncManager(context.Background(), store, runner, fixedClock{time.Now()}, time.Minute, 0, syncLog())
	if err := other.Run(context.Background()); err == nil {
		t.Fatal("CLI overlapped web sync")
	}
	close(finish)
	manager.Wait()
	status, _ := manager.Status(context.Background())
	if status.State != "succeeded" || status.LastSuccessAt == nil {
		t.Fatal(status)
	}
	// A new manager represents a process restart and must honor persisted cooldown.
	if err := other.Run(context.Background()); err == nil {
		t.Fatal("restart bypassed cooldown")
	}
}
func TestSyncFailureRetainsSuccessAndSanitizesMessage(t *testing.T) {
	old := time.Now().Add(-time.Hour)
	store := &syncMemory{status: ports.SyncStatus{LastSuccessAt: &old}}
	manager := NewSyncManager(context.Background(), store, syncRunFunc(func(context.Context, bool) error { return errors.New("secret-provider-key") }), fixedClock{time.Now()}, time.Minute, 0, syncLog())
	if err := manager.Run(context.Background()); err == nil {
		t.Fatal("missing failure")
	}
	st, _ := manager.Status(context.Background())
	if st.State != "failed" || st.LastSuccessAt == nil || !st.LastSuccessAt.Equal(old) || st.Message == "secret-provider-key" {
		t.Fatal(st)
	}
	if store.locked {
		t.Fatal("lock leaked")
	}
}
func TestScheduledSyncDueAndDisabled(t *testing.T) {
	now := time.Now()
	store := &syncMemory{}
	calls := 0
	runner := syncRunFunc(func(context.Context, bool) error { calls++; return nil })
	m := NewSyncManager(context.Background(), store, runner, fixedClock{now}, time.Minute, time.Hour, syncLog())
	if err := m.Scheduled(context.Background()); err != nil || calls != 0 {
		t.Fatal("started before due", err)
	}
	m.first = now
	if err := m.Scheduled(context.Background()); err != nil {
		t.Fatal(err)
	}
	m.Wait()
	if calls != 1 {
		t.Fatal(calls)
	}
	st, _ := m.Status(context.Background())
	if st.Source != "schedule" || !st.Automatic {
		t.Fatal(st)
	}
	disabled := NewSyncManager(context.Background(), store, runner, fixedClock{now.Add(2 * time.Hour)}, time.Minute, 0, syncLog())
	if err := disabled.Scheduled(context.Background()); err != nil || calls != 1 {
		t.Fatal("disabled scheduler ran")
	}
}
