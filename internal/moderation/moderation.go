// Package moderation contains the moderator-authored job use cases: create a
// hand-curated vacancy and edit an existing one. It owns validation and the
// deterministic derivation (via jobderive); the Repository owns persistence (the
// transactional create + enrichment enqueue, and the source-scoped update). The HTTP
// handler stays thin: it translates the wire body into these inputs and renders the
// returned row.
package moderation

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/jobderive"
)

// Sentinel errors. ErrInvalid wraps every validation failure (the handler maps it to
// 400, surfacing the wrapped message); ErrJobNotFound is the missing-or-non-manual edit
// target (mapped to 404).
var (
	// ErrInvalid wraps validation failures. Its text is user-facing: the handler
	// surfaces the wrapped message in the 400 body, so it carries no package prefix.
	ErrInvalid     = errors.New("invalid request")
	ErrJobNotFound = errors.New("moderation: job not found")
)

// manualSource is the fixed source identity for moderator-authored jobs; the URL is the
// external id, so re-creating the same URL is an idempotent upsert.
const manualSource = "manual"

// CreateInput is the moderator-supplied content for a new vacancy. URL (the dedup key),
// Title, and Company are required; the rest is optional.
type CreateInput struct {
	URL         string
	Title       string
	Company     string
	Location    string
	Remote      bool
	Description string
	PostedAt    *time.Time
}

// UpdatePatch is a partial edit: a nil field is left unchanged. The source identity is
// not editable, so URL is absent here.
type UpdatePatch struct {
	Title       *string
	Company     *string
	Location    *string
	Remote      *bool
	Description *string
	PostedAt    *time.Time
}

// Repository is the persistence contract. Create runs the upsert and enrichment enqueue
// atomically; BySlug loads a manual job (ErrJobNotFound when missing or not manual);
// Update writes the full resulting row, scoped to the manual source.
type Repository interface {
	Create(ctx context.Context, p db.UpsertManualJobParams) (db.Job, error)
	BySlug(ctx context.Context, slug string) (db.Job, error)
	Update(ctx context.Context, p db.UpdateManualJobParams) (db.Job, error)
}

// Service implements the moderator-authored job use cases.
type Service struct {
	repo Repository
}

// New creates a Service backed by the given Repository.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create validates the input, derives the slugs and dictionary facets, and persists the
// new job (source 'manual', external_id = url). created_by and updated_by are both stamped
// with the acting moderator: created_by is written on insert, updated_by on a re-create of
// the same URL.
func (s *Service) Create(ctx context.Context, actorID int64, in CreateInput) (db.Job, error) {
	if err := in.validate(); err != nil {
		return db.Job{}, err
	}
	d := jobderive.Derive(jobderive.Input{
		Title:       in.Title,
		Company:     in.Company,
		Source:      manualSource,
		ExternalID:  in.URL,
		Location:    in.Location,
		Description: in.Description,
		WorkMode:    remoteWorkMode(in.Remote),
	})
	return s.repo.Create(ctx, db.UpsertManualJobParams{
		ExternalID:  in.URL,
		URL:         in.URL,
		Title:       in.Title,
		Company:     in.Company,
		CompanySlug: d.CompanySlug,
		Location:    in.Location,
		Remote:      in.Remote,
		Description: in.Description,
		PostedAt:    toTimestamptz(in.PostedAt),
		PublicSlug:  d.PublicSlug,
		Countries:   d.Countries,
		Regions:     d.Regions,
		WorkMode:    d.WorkMode,
		Skills:      d.Skills,
		Seniority:   d.Seniority,
		Category:    d.Category,
		CreatedBy:   actorID,
		UpdatedBy:   actorID,
	})
}

// Update loads the manual job, overlays the supplied (nil-means-unchanged) fields, and
// re-derives the deterministic facets from the merged content — so editing the location,
// description, or company keeps geography/skills/company-slug consistent. The source
// identity (url/external_id/public_slug) is never recomputed, keeping the public slug
// stable. A missing or non-manual slug surfaces ErrJobNotFound.
func (s *Service) Update(ctx context.Context, actorID int64, slug string, p UpdatePatch) (db.Job, error) {
	if err := p.validate(); err != nil {
		return db.Job{}, err
	}
	cur, err := s.repo.BySlug(ctx, slug)
	if err != nil {
		return db.Job{}, err
	}

	title := stringOr(p.Title, cur.Title)
	company := stringOr(p.Company, cur.Company)
	location := stringOr(p.Location, cur.Location)
	description := stringOr(p.Description, cur.Description)
	remote := cur.Remote
	if p.Remote != nil {
		remote = *p.Remote
	}
	postedAt := cur.PostedAt
	if p.PostedAt != nil {
		postedAt = toTimestamptz(p.PostedAt)
	}

	// External id stays the create-time identity; only the dictionary facets re-derive.
	d := jobderive.Derive(jobderive.Input{
		Title:       title,
		Company:     company,
		Source:      manualSource,
		ExternalID:  cur.ExternalID,
		Location:    location,
		Description: description,
		WorkMode:    remoteWorkMode(remote),
	})
	return s.repo.Update(ctx, db.UpdateManualJobParams{
		PublicSlug:  slug,
		Title:       title,
		Company:     company,
		CompanySlug: d.CompanySlug,
		Location:    location,
		Remote:      remote,
		Description: description,
		PostedAt:    postedAt,
		Countries:   d.Countries,
		Regions:     d.Regions,
		WorkMode:    d.WorkMode,
		Skills:      d.Skills,
		Seniority:   d.Seniority,
		Category:    d.Category,
		UpdatedBy:   actorID,
	})
}

// validate enforces the required fields and that the URL is an absolute http(s) link
// (the URL is the dedup key, so it must be well-formed and stable).
func (in CreateInput) validate() error {
	if strings.TrimSpace(in.URL) == "" {
		return fmt.Errorf("%w: url is required", ErrInvalid)
	}
	if strings.TrimSpace(in.Title) == "" {
		return fmt.Errorf("%w: title is required", ErrInvalid)
	}
	if strings.TrimSpace(in.Company) == "" {
		return fmt.Errorf("%w: company is required", ErrInvalid)
	}
	u, err := url.Parse(in.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%w: url must be an absolute http(s) URL", ErrInvalid)
	}
	return nil
}

// validate rejects an edit that would blank a required field: a supplied (non-nil)
// title or company must not be empty, mirroring Create's required-field guard. URL is
// the immutable identity and is not editable here, so it is not checked.
func (p UpdatePatch) validate() error {
	if p.Title != nil && strings.TrimSpace(*p.Title) == "" {
		return fmt.Errorf("%w: title must not be empty", ErrInvalid)
	}
	if p.Company != nil && strings.TrimSpace(*p.Company) == "" {
		return fmt.Errorf("%w: company must not be empty", ErrInvalid)
	}
	return nil
}

// remoteWorkMode maps the moderator's structured remote flag onto a work-mode signal
// (the same role the ATS adapters' workplace-type enum plays): remote=true yields the
// "remote" facet; otherwise the value is left to the location parser's hint.
func remoteWorkMode(remote bool) string {
	if remote {
		return "remote"
	}
	return ""
}

// stringOr returns *p when set, else the fallback — the nil-means-unchanged merge.
func stringOr(p *string, fallback string) string {
	if p != nil {
		return *p
	}
	return fallback
}

// toTimestamptz maps an optional time to the pgtype the params expect; nil becomes NULL.
func toTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
