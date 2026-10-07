package postgres

import (
	"context"
	"encoding/json"
	"football/internal/application/ports"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

func (s *Store) SyncStatus(ctx context.Context) (ports.SyncStatus, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, "SELECT data FROM sync_status WHERE id=1").Scan(&raw)
	var status ports.SyncStatus
	if err == nil {
		err = json.Unmarshal(raw, &status)
	}
	return status, err
}

type syncLock struct{ conn *pgxpool.Conn }

func (s *Store) TrySyncLock(ctx context.Context) (ports.SyncLock, bool, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, false, err
	}
	var locked bool
	err = conn.QueryRow(ctx, "SELECT pg_try_advisory_lock(7102403)").Scan(&locked)
	if err != nil || !locked {
		conn.Release()
		return nil, false, err
	}
	return &syncLock{conn}, true, nil
}
func (l *syncLock) Save(ctx context.Context, status ports.SyncStatus) error {
	raw, err := json.Marshal(status)
	if err != nil {
		return err
	}
	_, err = l.conn.Exec(ctx, "UPDATE sync_status SET data=$1 WHERE id=1", raw)
	return err
}
func (l *syncLock) Release() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := l.conn.Exec(ctx, "SELECT pg_advisory_unlock(7102403)")
	if err != nil {
		_ = l.conn.Conn().Close(ctx)
	}
	l.conn.Release()
}
