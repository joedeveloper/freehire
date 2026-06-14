package accounts

import (
	"context"
	"errors"
	"testing"
)

// fakeRepo is a test double for Repository.
// It records calls and returns canned responses.
type fakeRepo struct {
	// UserIDByIdentity responses: looked up in order per call
	identityResults []idResult
	identityCallIdx int

	// LinkOrCreateByEmail responses: looked up in order per call
	linkResults []linkResult
	linkCallIdx int

	// recorded calls
	linkCalls []linkCall
}

type idResult struct {
	id  int64
	err error
}

type linkResult struct {
	id  int64
	err error
}

type linkCall struct {
	provider      string
	providerUserID string
	email         string
}

func (f *fakeRepo) UserIDByIdentity(_ context.Context, provider, providerUserID string) (int64, error) {
	if f.identityCallIdx >= len(f.identityResults) {
		return 0, ErrIdentityNotFound
	}
	r := f.identityResults[f.identityCallIdx]
	f.identityCallIdx++
	return r.id, r.err
}

func (f *fakeRepo) LinkOrCreateByEmail(_ context.Context, provider, providerUserID, email string) (int64, error) {
	f.linkCalls = append(f.linkCalls, linkCall{provider, providerUserID, email})
	if f.linkCallIdx >= len(f.linkResults) {
		return 0, errors.New("fakeRepo: unexpected LinkOrCreateByEmail call")
	}
	r := f.linkResults[f.linkCallIdx]
	f.linkCallIdx++
	return r.id, r.err
}

// (a) identity found → returns that id; LinkOrCreateByEmail NOT called.
func TestResolveOAuthAccount_IdentityFound(t *testing.T) {
	repo := &fakeRepo{
		identityResults: []idResult{{id: 7, err: nil}},
	}
	svc := New(repo)

	id, err := svc.ResolveOAuthAccount(context.Background(), "google", "gid-1", "user@example.com", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 7 {
		t.Errorf("want id=7, got %d", id)
	}
	if len(repo.linkCalls) != 0 {
		t.Errorf("LinkOrCreateByEmail should NOT be called, got %d calls", len(repo.linkCalls))
	}
}

// (b) not found + emailVerified=true + non-empty email → LinkOrCreateByEmail called
// with the LOWER/trimmed email, returns its id.
func TestResolveOAuthAccount_LinkOrCreate(t *testing.T) {
	repo := &fakeRepo{
		identityResults: []idResult{{id: 0, err: ErrIdentityNotFound}},
		linkResults:     []linkResult{{id: 99, err: nil}},
	}
	svc := New(repo)

	id, err := svc.ResolveOAuthAccount(context.Background(), "github", "ghid-2", "  USER@Example.COM  ", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 99 {
		t.Errorf("want id=99, got %d", id)
	}
	if len(repo.linkCalls) != 1 {
		t.Fatalf("want 1 LinkOrCreateByEmail call, got %d", len(repo.linkCalls))
	}
	got := repo.linkCalls[0]
	if got.email != "user@example.com" {
		t.Errorf("email not normalized: want %q, got %q", "user@example.com", got.email)
	}
	if got.provider != "github" || got.providerUserID != "ghid-2" {
		t.Errorf("provider args wrong: %+v", got)
	}
}

// (c) not found + emailVerified=false → ErrNoVerifiedEmail; LinkOrCreateByEmail NOT called.
func TestResolveOAuthAccount_UnverifiedEmail(t *testing.T) {
	repo := &fakeRepo{
		identityResults: []idResult{{id: 0, err: ErrIdentityNotFound}},
	}
	svc := New(repo)

	_, err := svc.ResolveOAuthAccount(context.Background(), "google", "gid-3", "user@example.com", false)
	if !errors.Is(err, ErrNoVerifiedEmail) {
		t.Errorf("want ErrNoVerifiedEmail, got %v", err)
	}
	if len(repo.linkCalls) != 0 {
		t.Errorf("LinkOrCreateByEmail should NOT be called, got %d calls", len(repo.linkCalls))
	}
}

// (d) not found + empty/whitespace email (even if emailVerified=true) → ErrNoVerifiedEmail; not called.
func TestResolveOAuthAccount_EmptyEmail(t *testing.T) {
	for _, email := range []string{"", "   ", "\t"} {
		repo := &fakeRepo{
			identityResults: []idResult{{id: 0, err: ErrIdentityNotFound}},
		}
		svc := New(repo)

		_, err := svc.ResolveOAuthAccount(context.Background(), "google", "gid-4", email, true)
		if !errors.Is(err, ErrNoVerifiedEmail) {
			t.Errorf("email=%q: want ErrNoVerifiedEmail, got %v", email, err)
		}
		if len(repo.linkCalls) != 0 {
			t.Errorf("email=%q: LinkOrCreateByEmail should NOT be called", email)
		}
	}
}

// (e) race: LinkOrCreateByEmail returns ErrIdentityConflict,
// follow-up UserIDByIdentity returns id 42 → service returns 42.
func TestResolveOAuthAccount_Race_ReturnsWinner(t *testing.T) {
	repo := &fakeRepo{
		identityResults: []idResult{
			{id: 0, err: ErrIdentityNotFound}, // first lookup misses
			{id: 42, err: nil},                // retry after conflict finds the winner
		},
		linkResults: []linkResult{
			{id: 0, err: ErrIdentityConflict},
		},
	}
	svc := New(repo)

	id, err := svc.ResolveOAuthAccount(context.Background(), "google", "gid-5", "winner@example.com", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("want id=42, got %d", id)
	}
}

// (f) race retry still ErrIdentityNotFound → service returns a non-nil error.
func TestResolveOAuthAccount_Race_RetryFails(t *testing.T) {
	repo := &fakeRepo{
		identityResults: []idResult{
			{id: 0, err: ErrIdentityNotFound}, // first lookup misses
			{id: 0, err: ErrIdentityNotFound}, // retry also misses
		},
		linkResults: []linkResult{
			{id: 0, err: ErrIdentityConflict},
		},
	}
	svc := New(repo)

	_, err := svc.ResolveOAuthAccount(context.Background(), "google", "gid-6", "ghost@example.com", true)
	if err == nil {
		t.Fatal("want non-nil error when race retry also returns ErrIdentityNotFound")
	}
}
