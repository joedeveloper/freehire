// Package accounts resolves external sign-in identities into local user accounts.
package accounts

import (
	"context"
	"errors"
	"strings"
)

var (
	// ErrIdentityNotFound is returned by the repository when no identity row
	// matches the given (provider, providerUserID) pair.
	ErrIdentityNotFound = errors.New("accounts: identity not found")

	// ErrIdentityConflict is returned by the repository when a concurrent
	// callback already inserted the same identity row (unique violation).
	ErrIdentityConflict = errors.New("accounts: identity already exists")

	// ErrNoVerifiedEmail is returned by the service when the caller does not
	// supply a verified, non-empty email — a hard requirement before linking or
	// creating an account (unverified email is an account-takeover vector).
	ErrNoVerifiedEmail = errors.New("accounts: no verified email")
)

// Repository is the persistence boundary for the accounts service.
// Implementations must be safe for concurrent use.
type Repository interface {
	// UserIDByIdentity returns the local user id for an external identity, or
	// ErrIdentityNotFound when none is linked yet.
	UserIDByIdentity(ctx context.Context, provider, providerUserID string) (int64, error)

	// LinkOrCreateByEmail links the identity to the account with this email
	// (creating a passwordless account when none exists), atomically. It returns
	// ErrIdentityConflict if the identity was inserted concurrently.
	LinkOrCreateByEmail(ctx context.Context, provider, providerUserID, email string) (int64, error)
}

// Service resolves an external OAuth identity to a local user id.
type Service struct{ repo Repository }

// New returns a Service backed by the given Repository.
func New(repo Repository) *Service { return &Service{repo: repo} }

// ResolveOAuthAccount maps a provider identity to a local user id, following
// the identity-first, verified-email-gate, link-or-create, race-retry policy.
func (s *Service) ResolveOAuthAccount(ctx context.Context, provider, providerUserID, email string, emailVerified bool) (int64, error) {
	// 1. identity-first: cheapest and safest path.
	id, err := s.repo.UserIDByIdentity(ctx, provider, providerUserID)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, ErrIdentityNotFound) {
		return 0, err
	}

	// 2. verified-email gate (anti-takeover): never link or create without one.
	if !emailVerified || strings.TrimSpace(email) == "" {
		return 0, ErrNoVerifiedEmail
	}
	normalized := strings.ToLower(strings.TrimSpace(email))

	// 3. link existing account by email or create a new passwordless account.
	id, err = s.repo.LinkOrCreateByEmail(ctx, provider, providerUserID, normalized)
	if errors.Is(err, ErrIdentityConflict) {
		// 4. lost a race with a concurrent callback — the identity now exists;
		// return whichever goroutine won.
		return s.repo.UserIDByIdentity(ctx, provider, providerUserID)
	}
	return id, err
}
