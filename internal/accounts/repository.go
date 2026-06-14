package accounts

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/strelov1/freehire/internal/db"
)

// QueriesRepository is the production Repository backed by sqlc-generated
// *db.Queries and a *pgxpool.Pool for transaction management.
type QueriesRepository struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

// NewQueriesRepository constructs a QueriesRepository.
func NewQueriesRepository(q *db.Queries, pool *pgxpool.Pool) *QueriesRepository {
	return &QueriesRepository{q: q, pool: pool}
}

// Compile-time assertion that QueriesRepository satisfies Repository.
var _ Repository = (*QueriesRepository)(nil)

// UserIDByIdentity returns the local user id for an external identity, or
// ErrIdentityNotFound when no identity row matches.
func (r *QueriesRepository) UserIDByIdentity(ctx context.Context, provider, providerUserID string) (int64, error) {
	row, err := r.q.GetUserByIdentity(ctx, db.GetUserByIdentityParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrIdentityNotFound
	}
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}

// LinkOrCreateByEmail links the given identity to the account with the
// supplied (already-lowercased) email, creating a passwordless account when
// none exists. The operation runs in a single transaction. It returns
// ErrIdentityConflict on a unique-violation (concurrent callback race).
func (r *QueriesRepository) LinkOrCreateByEmail(ctx context.Context, provider, providerUserID, email string) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.q.WithTx(tx)

	var userID int64

	existing, err := q.GetUserByEmail(ctx, email)
	switch {
	case err == nil:
		userID = existing.ID
	case errors.Is(err, pgx.ErrNoRows):
		created, err := q.CreateUser(ctx, db.CreateUserParams{
			Email:        email,
			PasswordHash: pgtype.Text{}, // passwordless OAuth account
		})
		if err != nil {
			if isUniqueViolation(err) {
				return 0, ErrIdentityConflict
			}
			return 0, err
		}
		userID = created.ID
	default:
		return 0, err
	}

	if err := q.CreateUserIdentity(ctx, db.CreateUserIdentityParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
		UserID:         userID,
	}); err != nil {
		if isUniqueViolation(err) {
			return 0, ErrIdentityConflict
		}
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		if isUniqueViolation(err) {
			return 0, ErrIdentityConflict
		}
		return 0, err
	}

	return userID, nil
}

// isUniqueViolation reports whether err is a PostgreSQL unique-constraint
// violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
