package ports

import (
	"context"
	"errors"
	member "football/internal/member/domain"
	"time"
)

var ErrEmailExists = errors.New("email already registered")
var ErrMemberNotFound = errors.New("member not found")

type MemberStore interface {
	CreateMember(context.Context, member.Member) error
	MemberByEmail(context.Context, string) (member.Member, error)
	CreateSession(context.Context, string, string, time.Time, time.Time) error
	SessionMember(context.Context, string, time.Time) (member.Member, error)
	DeleteSession(context.Context, string) error
}
type Passwords interface {
	Hash(string) (string, error)
	Verify(string, string) bool
}
type Tokens interface{ New() (string, error) }
