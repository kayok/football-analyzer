package usecase

import (
	"context"
	"encoding/json"
	"football/internal/application/ports"
	member "football/internal/member/domain"
	"strings"
	"testing"
	"time"
)

type memberMemory struct {
	users    map[string]member.Member
	sessions map[string]string
	expiry   map[string]time.Time
}

func (m *memberMemory) CreateMember(_ context.Context, u member.Member) error {
	if _, ok := m.users[u.Email]; ok {
		return ports.ErrEmailExists
	}
	m.users[u.Email] = u
	return nil
}
func (m *memberMemory) MemberByEmail(_ context.Context, email string) (member.Member, error) {
	u, ok := m.users[email]
	if !ok {
		return u, ports.ErrMemberNotFound
	}
	return u, nil
}
func (m *memberMemory) CreateSession(_ context.Context, hash, id string, now, expiry time.Time) error {
	m.sessions[hash] = id
	m.expiry[hash] = expiry
	return nil
}
func (m *memberMemory) SessionMember(_ context.Context, hash string, now time.Time) (member.Member, error) {
	if m.expiry[hash].After(now) {
		for _, u := range m.users {
			if m.sessions[hash] == u.ID {
				return u, nil
			}
		}
	}
	return member.Member{}, ports.ErrMemberNotFound
}
func (m *memberMemory) DeleteSession(_ context.Context, hash string) error {
	delete(m.sessions, hash)
	return nil
}

type fakePasswords struct{}

func (fakePasswords) Hash(s string) (string, error) { return "hashed:" + s, nil }
func (fakePasswords) Verify(hash, s string) bool    { return hash == "hashed:"+s }

type fakeTokens struct{}

func (fakeTokens) New() (string, error) { return strings.Repeat("x", 43), nil }
func TestMembershipValidationAndSessions(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	store := &memberMemory{map[string]member.Member{}, map[string]string{}, map[string]time.Time{}}
	svc, err := NewMembership(store, fakePasswords{}, fakeTokens{}, fixedClock{now}, ids{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, email, password string }{
		{"", "a@example.test", "long-password"}, {"Name", "bad-email", "long-password"}, {"Name", "a@example.test", "short"}, {"Name", "a@example.test", strings.Repeat("ก", 25)},
		{"Name", "a@example.test", strings.Repeat("ก", 10)},
		{"Name", "a@example.test", "english password"},
	} {
		if _, _, err := svc.Register(ctx, tc.name, tc.email, tc.password); err == nil {
			t.Fatal("invalid registration accepted", tc.email)
		}
	}
	u, token, err := svc.Register(ctx, " Name ", " A@EXAMPLE.test ", "long-password")
	if err != nil || u.Name != "Name" || u.Email != "a@example.test" {
		t.Fatal(u, err)
	}
	if _, ok := store.sessions[token]; ok {
		t.Fatal("raw session token stored")
	}
	raw, _ := json.Marshal(u)
	if strings.Contains(string(raw), "hashed:") {
		t.Fatal("password exposed")
	}
	if _, _, err = svc.Register(ctx, "Name", u.Email, "long-password"); err == nil {
		t.Fatal("duplicate accepted")
	}
	if _, _, err = svc.Login(ctx, u.Email, "wrong-password"); err == nil {
		t.Fatal("wrong password accepted")
	}
	if _, _, err = svc.Login(ctx, "unknown@example.test", "long-password"); err == nil {
		t.Fatal("unknown accepted")
	}
	if _, _, err = svc.Login(ctx, u.Email, "long-password"); err != nil {
		t.Fatal(err)
	}
	if current, err := svc.Current(ctx, token); err != nil || current.ID != u.ID {
		t.Fatal(current, err)
	}
	svc.clock = fixedClock{now.Add(SessionLifetime)}
	if _, err = svc.Current(ctx, token); err == nil {
		t.Fatal("expired accepted")
	}
	svc.clock = fixedClock{now}
	if err = svc.Logout(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Current(ctx, token); err == nil {
		t.Fatal("logged out accepted")
	}
	// Tightening registration must not lock out legacy non-ASCII passwords.
	store.users["legacy@example.test"] = member.Member{ID: "legacy-id", Email: "legacy@example.test", PasswordHash: "hashed:ภาษาไทยเดิม"}
	if _, _, err = svc.Login(ctx, "legacy@example.test", "ภาษาไทยเดิม"); err != nil {
		t.Fatal("legacy login rejected", err)
	}

}

func TestOwnerOnlyMembership(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	store := &memberMemory{map[string]member.Member{
		"owner@example.test": {ID: "owner", Email: "owner@example.test", PasswordHash: "hashed:owner-password"},
		"other@example.test": {ID: "other", Email: "other@example.test", PasswordHash: "hashed:other-password"},
	}, map[string]string{}, map[string]time.Time{}}
	for _, email := range []string{"", "invalid"} {
		if _, err := NewPrivateMembership(store, fakePasswords{}, fakeTokens{}, fixedClock{now}, ids{}, email); err == nil {
			t.Fatal("invalid owner configuration accepted")
		}
	}
	svc, err := NewPrivateMembership(store, fakePasswords{}, fakeTokens{}, fixedClock{now}, ids{}, " OWNER@example.test ")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Register(ctx, "New", "new@example.test", "new-password"); err == nil {
		t.Fatal("public registration accepted")
	}
	if len(store.users) != 2 {
		t.Fatal("registration changed accounts")
	}
	if _, _, err := svc.Login(ctx, "other@example.test", "other-password"); err == nil {
		t.Fatal("other account accepted")
	}
	if len(store.sessions) != 0 {
		t.Fatal("unauthorized login created session")
	}
	if _, _, err := svc.Login(ctx, "owner@example.test", "wrong-password"); err == nil {
		t.Fatal("wrong owner password accepted")
	}
	_, token, err := svc.Login(ctx, " OWNER@example.test ", "owner-password")
	if err != nil {
		t.Fatal(err)
	}
	if u, err := svc.Current(ctx, token); err != nil || u.ID != "owner" {
		t.Fatal("owner session rejected", err)
	}
	legacy := strings.Repeat("y", 43)
	store.CreateSession(ctx, sessionHash(legacy), "other", now, now.Add(SessionLifetime))
	if _, err := svc.Current(ctx, legacy); err == nil {
		t.Fatal("old non-owner session accepted")
	}
	if len(store.users) != 2 {
		t.Fatal("owner restriction deleted accounts")
	}
}
