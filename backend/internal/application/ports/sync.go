package ports

import (
	"context"
	"time"
)

type SyncStatus struct {
	State           string     `json:"state"`
	Source          string     `json:"source"`
	StartedAt       *time.Time `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at"`
	LastSuccessAt   *time.Time `json:"last_success_at"`
	RetryAt         *time.Time `json:"retry_at"`
	Deadline        *time.Time `json:"deadline"`
	Message         string     `json:"message"`
	Automatic       bool       `json:"automatic"`
	NextScheduledAt *time.Time `json:"next_scheduled_at"`
}

// A dedicated lock is held across fetching and committing, including CLI runs.
type SyncLock interface {
	Save(context.Context, SyncStatus) error
	Release()
}
type SyncStore interface {
	SyncStatus(context.Context) (SyncStatus, error)
	TrySyncLock(context.Context) (SyncLock, bool, error)
}
