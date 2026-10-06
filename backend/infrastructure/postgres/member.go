package postgres

import (
	"context"
	"errors"
	"football/internal/application/ports"
	member "football/internal/member/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"time"
)

func (s *Store) CreateMember(ctx context.Context, m member.Member) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Same lock as worker/selection; only the first account claims legacy local picks.
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(7102401)"); err != nil {
		return err
	}
	var first bool
	if err = tx.QueryRow(ctx, "SELECT NOT EXISTS(SELECT 1 FROM members)").Scan(&first); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO members(id,name,email,password_hash,created_at) VALUES($1,$2,$3,$4,$5)", m.ID, m.Name, m.Email, m.PasswordHash, m.CreatedAt); err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return ports.ErrEmailExists
		}
		return err
	}
	if first {
		if _, err = tx.Exec(ctx, "UPDATE user_picks SET user_id=$1 WHERE user_id IS NULL", m.ID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (s *Store) MemberByEmail(ctx context.Context, email string) (member.Member, error) {
	var m member.Member
	err := s.pool.QueryRow(ctx, "SELECT id,name,email,password_hash,created_at FROM members WHERE email=$1", email).Scan(&m.ID, &m.Name, &m.Email, &m.PasswordHash, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ports.ErrMemberNotFound
	}
	return m, err
}
func (s *Store) CreateSession(ctx context.Context, hash, userID string, now, expiry time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "DELETE FROM member_sessions WHERE expires_at <= $1", now); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO member_sessions(token_hash,member_id,created_at,expires_at) VALUES($1,$2,$3,$4)", hash, userID, now, expiry); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Store) SessionMember(ctx context.Context, hash string, now time.Time) (member.Member, error) {
	var m member.Member
	err := s.pool.QueryRow(ctx, `SELECT m.id,m.name,m.email,m.created_at FROM member_sessions s JOIN members m ON m.id=s.member_id WHERE s.token_hash=$1 AND s.expires_at>$2`, hash, now).Scan(&m.ID, &m.Name, &m.Email, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ports.ErrMemberNotFound
	}
	return m, err
}
func (s *Store) DeleteSession(ctx context.Context, hash string) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM member_sessions WHERE token_hash=$1", hash)
	return err
}
