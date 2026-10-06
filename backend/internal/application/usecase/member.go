package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"football/internal/application/ports"
	member "football/internal/member/domain"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

type Membership struct {
	store      ports.MemberStore
	passwords  ports.Passwords
	tokens     ports.Tokens
	clock      ports.Clock
	ids        ports.IDs
	dummyHash  string
	ownerEmail string
}

const SessionLifetime = 7 * 24 * time.Hour

func NewMembership(store ports.MemberStore, passwords ports.Passwords, tokens ports.Tokens, clock ports.Clock, ids ports.IDs) (*Membership, error) {
	dummy, err := passwords.Hash("unused-password-for-timing")
	if err != nil {
		return nil, err
	}
	return &Membership{store: store, passwords: passwords, tokens: tokens, clock: clock, ids: ids, dummyHash: dummy}, nil
}

// NewPrivateMembership restricts HTTP access to the configured owner, including existing sessions.
func NewPrivateMembership(store ports.MemberStore, passwords ports.Passwords, tokens ports.Tokens, clock ports.Clock, ids ports.IDs, rawOwnerEmail string) (*Membership, error) {
	email, err := emailAddress(rawOwnerEmail)
	if err != nil {
		return nil, errors.New("OWNER_EMAIL must be a valid owner email")
	}
	s, err := NewMembership(store, passwords, tokens, clock, ids)
	if err != nil {
		return nil, err
	}
	s.ownerEmail = email
	return s, nil
}
func emailAddress(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > 254 {
		return "", fail("INVALID_EMAIL", "อีเมลไม่ถูกต้อง", 400)
	}
	return email, nil
}
func sessionHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
func (s *Membership) Register(ctx context.Context, name, rawEmail, password string) (member.Member, string, error) {
	if s.ownerEmail != "" {
		return member.Member{}, "", fail("REGISTRATION_CLOSED", "ระบบส่วนตัว ไม่เปิดรับสมัครสมาชิก", 403)
	}
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return member.Member{}, "", fail("INVALID_NAME", "ชื่อที่แสดงต้องมี 1–80 ตัวอักษร", 400)
	}
	email, err := emailAddress(rawEmail)
	if err != nil {
		return member.Member{}, "", err
	}
	if err := member.ValidatePassword(password); err != nil {
		return member.Member{}, "", fail("INVALID_PASSWORD", err.Error(), 400)
	}
	hash, err := s.passwords.Hash(password)
	if err != nil {
		return member.Member{}, "", err
	}
	user := member.Member{ID: s.ids.New(), Name: name, Email: email, PasswordHash: hash, CreatedAt: s.clock.Now()}
	if err = s.store.CreateMember(ctx, user); err != nil {
		if errors.Is(err, ports.ErrEmailExists) {
			err = fail("EMAIL_EXISTS", "อีเมลนี้สมัครไว้แล้ว กรุณาเข้าสู่ระบบ", 409)
		}
		return member.Member{}, "", err
	}
	token, err := s.session(ctx, user.ID)
	return user, token, err
}
func (s *Membership) Login(ctx context.Context, rawEmail, password string) (member.Member, string, error) {
	email, err := emailAddress(rawEmail)
	if err != nil {
		return member.Member{}, "", fail("INVALID_CREDENTIALS", "อีเมลหรือรหัสผ่านไม่ถูกต้อง", 401)
	}
	user, err := s.store.MemberByEmail(ctx, email)
	if err != nil && !errors.Is(err, ports.ErrMemberNotFound) {
		return member.Member{}, "", err
	}
	hash := user.PasswordHash
	if err != nil || (s.ownerEmail != "" && email != s.ownerEmail) {
		hash = s.dummyHash
	}
	valid := s.passwords.Verify(hash, password)
	if !valid || err != nil || (s.ownerEmail != "" && email != s.ownerEmail) {
		return member.Member{}, "", fail("INVALID_CREDENTIALS", "อีเมลหรือรหัสผ่านไม่ถูกต้อง", 401)
	}
	token, err := s.session(ctx, user.ID)
	return user, token, err
}
func (s *Membership) session(ctx context.Context, userID string) (string, error) {
	token, err := s.tokens.New()
	if err != nil {
		return "", err
	}
	now := s.clock.Now()
	return token, s.store.CreateSession(ctx, sessionHash(token), userID, now, now.Add(SessionLifetime))
}
func (s *Membership) Current(ctx context.Context, token string) (member.Member, error) {
	if len(token) != 43 {
		return member.Member{}, fail("UNAUTHENTICATED", "กรุณาเข้าสู่ระบบ", 401)
	}
	user, err := s.store.SessionMember(ctx, sessionHash(token), s.clock.Now())
	if err == nil && s.ownerEmail != "" && user.Email != s.ownerEmail {
		return member.Member{}, fail("UNAUTHENTICATED", "กรุณาเข้าสู่ระบบด้วยบัญชีเจ้าของ", 401)
	}
	if errors.Is(err, ports.ErrMemberNotFound) {
		err = fail("UNAUTHENTICATED", "เซสชันหมดอายุ กรุณาเข้าสู่ระบบอีกครั้ง", 401)
	}
	return user, err
}
func (s *Membership) Logout(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, sessionHash(token))
}
