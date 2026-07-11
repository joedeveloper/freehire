//go:build integration

// Integration test for the on-demand LLM fit endpoints:
// GET /api/v1/jobs/:slug/fit serves the cached analysis (or a null one), and
// POST computes the three-stage chain, caches it per (user, job), and returns it
// fresh. The job/company lookup and the cache hit a real Postgres; the LLM is a
// canned model and the CV a fake blob store. Run with:
//
//	go test -tags=integration ./internal/handler/
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tmc/langchaingo/llms"

	"github.com/strelov1/freehire/internal/auth"
	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/jobfit"
	"github.com/strelov1/freehire/internal/llm"
	"github.com/strelov1/freehire/internal/resume"
	"github.com/strelov1/freehire/internal/userprofile"
)

// fit stage responses for the canned three-stage model.
const (
	fitStage1 = `{"requirements":[{"text":"Go","priority":"required","status":"covered","evidence":"5y"},{"text":"Kafka","priority":"preferred","status":"missing-gap"}]}`
	fitStage2 = `{"title_alignment":{"score":80},"experience_relevance":{"score":70},"seniority_fit":{"score":60},"skills_coverage":{"score":50},"company_context":{"score":40},"strengths":["Strong Go"],"gaps":["No Kafka"],"recommendation":"Apply."}`
	fitStage3 = `{"title_alignment":{"score":80},"experience_relevance":{"score":60},"seniority_fit":{"score":60},"skills_coverage":{"score":50},"company_context":{"score":40},"strengths":["Strong Go"],"gaps":["No Kafka"],"recommendation":"Apply, address Kafka."}`
)

// fitModel returns canned responses in order — one per stage of the chain.
type fitModel struct {
	resp []string
	n    int
}

func (m *fitModel) GenerateContent(context.Context, []llms.MessageContent, ...llms.CallOption) (*llms.ContentResponse, error) {
	r := m.resp[m.n]
	m.n++
	return &llms.ContentResponse{Choices: []*llms.ContentChoice{{Content: r}}}, nil
}
func (*fitModel) Call(context.Context, string, ...llms.CallOption) (string, error) { return "", nil }

type fitBody struct {
	Data struct {
		HasCV    bool             `json:"has_cv"`
		Stale    bool             `json:"stale"`
		Analysis *jobfit.Analysis `json:"analysis"`
	} `json:"data"`
}

func fitAPI(pool *pgxpool.Pool, queries *db.Queries, iss *auth.Issuer, store *resume.Store, an *jobfit.Analyzer) *API {
	return &API{
		pool: pool, queries: queries, issuer: iss,
		userProfile: userprofile.New(ownedProfile()),
		resume:      store, jobFit: an, jobFitCache: queries,
	}
}

func TestJobFitEndpoints(t *testing.T) {
	pool := startPostgres(t)
	ctx := context.Background()
	queries := db.New(pool)

	var userID, jobID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email) VALUES ('fit@example.test') RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO jobs (source, external_id, url, title, description, company_slug, public_slug, skills, content_hash)
		 VALUES ('test','f1','http://e.test','Senior Go Engineer','Build backends in Go.',
		         'acme','fit-job', ARRAY['go'], 'hash-1') RETURNING id`).Scan(&jobID); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO companies (slug, name, company_info) VALUES ('acme','Acme', '{"tagline":"We ship"}')`); err != nil {
		t.Fatalf("seed company: %v", err)
	}

	iss := auth.NewIssuer("test-secret", time.Hour)
	token, err := iss.Issue(userID)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	storeWithCVFor := func(t *testing.T) *resume.Store {
		t.Helper()
		s := resume.New(newFakeResumeBlobs(), &fakeResumeRepo{})
		if _, err := s.Put(ctx, userID, "text/plain", []byte("Backend engineer, 5y Go at Acme.")); err != nil {
			t.Fatalf("seed CV: %v", err)
		}
		return s
	}

	do := func(t *testing.T, app *fiber.App, method, slug, tok string) (int, fitBody) {
		t.Helper()
		req := httptest.NewRequest(method, "/api/v1/jobs/"+slug+"/fit", nil)
		if tok != "" {
			req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("%s fit: %v", method, err)
		}
		defer resp.Body.Close()
		var body fitBody
		_ = json.NewDecoder(resp.Body).Decode(&body)
		return resp.StatusCode, body
	}

	appFor := func(store *resume.Store, an *jobfit.Analyzer) *fiber.App {
		h := fitAPI(pool, queries, iss, store, an)
		app := fiber.New(fiber.Config{ErrorHandler: RenderError})
		g := auth.RequireAuth(iss)
		app.Get("/api/v1/jobs/:slug/fit", g, h.GetJobFit)
		app.Post("/api/v1/jobs/:slug/fit", g, h.PostJobFit)
		return app
	}

	t.Run("unauthenticated is 401", func(t *testing.T) {
		app := appFor(storeWithCVFor(t), jobfit.NewAnalyzer(nil))
		if status, _ := do(t, app, fiber.MethodGet, "fit-job", ""); status != fiber.StatusUnauthorized {
			t.Errorf("status = %d, want 401", status)
		}
	})

	t.Run("unknown slug is 404", func(t *testing.T) {
		app := appFor(storeWithCVFor(t), jobfit.NewAnalyzer(nil))
		if status, _ := do(t, app, fiber.MethodGet, "no-such", token); status != fiber.StatusNotFound {
			t.Errorf("status = %d, want 404", status)
		}
	})

	t.Run("GET without a stored CV → has_cv false, no LLM", func(t *testing.T) {
		app := appFor(resume.New(newFakeResumeBlobs(), &fakeResumeRepo{}), jobfit.NewAnalyzer(nil))
		status, body := do(t, app, fiber.MethodGet, "fit-job", token)
		if status != fiber.StatusOK || body.Data.HasCV {
			t.Errorf("got status=%d has_cv=%v, want 200/false", status, body.Data.HasCV)
		}
	})

	t.Run("GET never-analyzed → has_cv true, null analysis", func(t *testing.T) {
		app := appFor(storeWithCVFor(t), jobfit.NewAnalyzer(nil))
		status, body := do(t, app, fiber.MethodGet, "fit-job", token)
		if status != fiber.StatusOK || !body.Data.HasCV || body.Data.Analysis != nil {
			t.Errorf("got status=%d has_cv=%v analysis=%v, want 200/true/nil", status, body.Data.HasCV, body.Data.Analysis)
		}
	})

	t.Run("POST LLM off → has_cv true, null analysis, nothing cached", func(t *testing.T) {
		app := appFor(storeWithCVFor(t), jobfit.NewAnalyzer(nil))
		status, body := do(t, app, fiber.MethodPost, "fit-job", token)
		if status != fiber.StatusOK || !body.Data.HasCV || body.Data.Analysis != nil {
			t.Errorf("got status=%d has_cv=%v analysis=%v, want 200/true/nil", status, body.Data.HasCV, body.Data.Analysis)
		}
		var n int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM user_job_analysis WHERE user_id=$1`, userID).Scan(&n)
		if n != 0 {
			t.Errorf("cache rows = %d, want 0 with LLM off", n)
		}
	})

	t.Run("POST computes, caches, GET returns fresh", func(t *testing.T) {
		model := &fitModel{resp: []string{fitStage1, fitStage2, fitStage3}}
		app := appFor(storeWithCVFor(t), jobfit.NewAnalyzer(llm.NewWithModel(model)))

		status, body := do(t, app, fiber.MethodPost, "fit-job", token)
		if status != fiber.StatusOK || body.Data.Analysis == nil {
			t.Fatalf("POST got status=%d analysis=%v, want 200 + analysis", status, body.Data.Analysis)
		}
		if body.Data.Analysis.Verdict == "" || len(body.Data.Analysis.Dimensions) != 6 {
			t.Errorf("analysis = %+v, want verdict + 6 dimensions", body.Data.Analysis)
		}
		if len(body.Data.Analysis.RequirementMatch) != 2 {
			t.Errorf("requirement_match = %d, want 2", len(body.Data.Analysis.RequirementMatch))
		}

		// The row was cached and a fresh GET (no CV/job change) serves it non-stale, no LLM.
		var n int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM user_job_analysis WHERE user_id=$1 AND job_id=$2`, userID, jobID).Scan(&n)
		if n != 1 {
			t.Fatalf("cache rows = %d, want 1", n)
		}
		gstatus, gbody := do(t, appFor(storeWithCVFor(t), jobfit.NewAnalyzer(nil)), fiber.MethodGet, "fit-job", token)
		if gstatus != fiber.StatusOK || gbody.Data.Analysis == nil || gbody.Data.Stale {
			t.Errorf("GET after compute = status %d stale %v analysis %v, want 200/false/present",
				gstatus, gbody.Data.Stale, gbody.Data.Analysis)
		}
	})
}

// TestJobFitQuota covers the per-user monthly cap: a new job over the limit is a 429 (no
// LLM call), a recompute of an already-analyzed job is always allowed, and GET reports
// usage. It seeds analysis rows directly to place the user at a chosen usage level.
func TestJobFitQuota(t *testing.T) {
	pool := startPostgres(t)
	ctx := context.Background()
	queries := db.New(pool)
	iss := auth.NewIssuer("test-secret", time.Hour)

	seedUser := func(t *testing.T, email string) (int64, string) {
		t.Helper()
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ($1) RETURNING id`, email).Scan(&id); err != nil {
			t.Fatalf("seed user: %v", err)
		}
		tok, err := iss.Issue(id)
		if err != nil {
			t.Fatalf("issue token: %v", err)
		}
		return id, tok
	}
	seedJob := func(t *testing.T, ext, slug string) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(ctx,
			`INSERT INTO jobs (source, external_id, url, title, description, public_slug, skills, content_hash)
			 VALUES ('test',$1,'http://e.test','Go Engineer','Build backends in Go.',$2, ARRAY['go'], 'h')
			 RETURNING id`, ext, slug).Scan(&id); err != nil {
			t.Fatalf("seed job %s: %v", slug, err)
		}
		return id
	}
	// seedAnalysis places a prior analysis row for (user, job) at the given age, so it
	// counts toward (recent) or outside (old) the rolling window.
	seedAnalysis := func(t *testing.T, userID, jobID int64, age time.Duration) {
		t.Helper()
		if _, err := pool.Exec(ctx,
			`INSERT INTO user_job_analysis (user_id, job_id, analysis, model, created_at)
			 VALUES ($1,$2,'{}','seed-model', now() - $3::interval)`,
			userID, jobID, age.String()); err != nil {
			t.Fatalf("seed analysis: %v", err)
		}
	}
	storeWithCVFor := func(t *testing.T, userID int64) *resume.Store {
		t.Helper()
		s := resume.New(newFakeResumeBlobs(), &fakeResumeRepo{})
		if _, err := s.Put(ctx, userID, "text/plain", []byte("5y Go.")); err != nil {
			t.Fatalf("seed CV: %v", err)
		}
		return s
	}
	appFor := func(store *resume.Store, an *jobfit.Analyzer) *fiber.App {
		h := &API{
			pool: pool, queries: queries, issuer: iss,
			userProfile: userprofile.New(ownedProfile()),
			resume:      store, jobFit: an, jobFitCache: queries,
		}
		app := fiber.New(fiber.Config{ErrorHandler: RenderError})
		g := auth.RequireAuth(iss)
		app.Get("/api/v1/jobs/:slug/fit", g, h.GetJobFit)
		app.Post("/api/v1/jobs/:slug/fit", g, h.PostJobFit)
		app.Get("/api/v1/jobs/:slug/fit/stream", g, h.StreamJobFit)
		return app
	}
	postFit := func(t *testing.T, app *fiber.App, slug, tok string) (int, fitBody) {
		t.Helper()
		req := httptest.NewRequest(fiber.MethodPost, "/api/v1/jobs/"+slug+"/fit", nil)
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("POST fit: %v", err)
		}
		defer resp.Body.Close()
		var body fitBody
		_ = json.NewDecoder(resp.Body).Decode(&body)
		return resp.StatusCode, body
	}
	// fillQuota seeds `n` distinct recently-analyzed jobs for a user (used = n).
	fillQuota := func(t *testing.T, userID int64, tag string, n int) {
		t.Helper()
		for i := 0; i < n; i++ {
			jid := seedJob(t, tag+"-"+strconv.Itoa(i), tag+"-job-"+strconv.Itoa(i))
			seedAnalysis(t, userID, jid, time.Hour)
		}
	}

	t.Run("new job over the limit is 429 and never calls the LLM", func(t *testing.T) {
		userID, token := seedUser(t, "over@example.test")
		fillQuota(t, userID, "over", int(fitAnalysisLimit))
		seedJob(t, "over-new", "over-new")
		model := &fitModel{resp: []string{fitStage1, fitStage2, fitStage3}}
		app := appFor(storeWithCVFor(t, userID), jobfit.NewAnalyzer(llm.NewWithModel(model)))

		status, _ := postFit(t, app, "over-new", token)
		if status != fiber.StatusTooManyRequests {
			t.Errorf("status = %d, want 429", status)
		}
		if model.n != 0 {
			t.Errorf("LLM was called %d times, want 0 when over quota", model.n)
		}
		var n int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM user_job_analysis WHERE user_id=$1`, userID).Scan(&n)
		if n != int(fitAnalysisLimit) {
			t.Errorf("cache rows = %d, want %d (nothing persisted on 429)", n, fitAnalysisLimit)
		}
	})

	t.Run("recompute of an analyzed job is allowed even at the limit", func(t *testing.T) {
		userID, token := seedUser(t, "recompute@example.test")
		fillQuota(t, userID, "rc", int(fitAnalysisLimit))
		// The last-seeded job is one the user has already analyzed → recompute must run.
		var createdBefore time.Time
		if err := pool.QueryRow(ctx,
			`SELECT created_at FROM user_job_analysis WHERE user_id=$1 ORDER BY created_at LIMIT 1`,
			userID).Scan(&createdBefore); err != nil {
			t.Fatalf("read created_at: %v", err)
		}
		model := &fitModel{resp: []string{fitStage1, fitStage2, fitStage3}}
		app := appFor(storeWithCVFor(t, userID), jobfit.NewAnalyzer(llm.NewWithModel(model)))

		status, body := postFit(t, app, "rc-job-0", token)
		if status != fiber.StatusOK || body.Data.Analysis == nil {
			t.Fatalf("recompute got status=%d analysis=%v, want 200 + analysis", status, body.Data.Analysis)
		}
		// A recompute must NOT re-bump created_at (else it would re-age the row into the
		// window and mis-count the quota).
		var createdAfter time.Time
		_ = pool.QueryRow(ctx,
			`SELECT created_at FROM user_job_analysis WHERE user_id=$1 AND job_id=(SELECT id FROM jobs WHERE public_slug='rc-job-0')`,
			userID).Scan(&createdAfter)
		if !createdAfter.Equal(createdBefore) {
			t.Errorf("recompute changed created_at (%s → %s); want preserved", createdBefore, createdAfter)
		}
	})

	t.Run("under the limit a new job runs and GET reports usage", func(t *testing.T) {
		userID, token := seedUser(t, "under@example.test")
		fillQuota(t, userID, "under", int(fitAnalysisLimit)-1) // used = 9
		seedJob(t, "under-new", "under-new")
		model := &fitModel{resp: []string{fitStage1, fitStage2, fitStage3}}
		app := appFor(storeWithCVFor(t, userID), jobfit.NewAnalyzer(llm.NewWithModel(model)))

		// GET before compute: used=9, remaining=1.
		greq := httptest.NewRequest(fiber.MethodGet, "/api/v1/jobs/under-new/fit", nil)
		greq.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
		gresp, _ := app.Test(greq)
		var gbody struct {
			Data struct {
				Quota *fitQuota `json:"quota"`
			} `json:"data"`
		}
		_ = json.NewDecoder(gresp.Body).Decode(&gbody)
		gresp.Body.Close()
		if gbody.Data.Quota == nil || gbody.Data.Quota.Used != fitAnalysisLimit-1 ||
			gbody.Data.Quota.Limit != fitAnalysisLimit || gbody.Data.Quota.Remaining != 1 {
			t.Fatalf("GET quota = %+v, want used=%d limit=%d remaining=1", gbody.Data.Quota, fitAnalysisLimit-1, fitAnalysisLimit)
		}

		// POST the 10th distinct job succeeds; the next new job is then over the limit.
		if status, body := postFit(t, app, "under-new", token); status != fiber.StatusOK || body.Data.Analysis == nil {
			t.Fatalf("10th analysis got status=%d analysis=%v, want 200 + analysis", status, body.Data.Analysis)
		}
		seedJob(t, "under-new2", "under-new2")
		if status, _ := postFit(t, app, "under-new2", token); status != fiber.StatusTooManyRequests {
			t.Errorf("11th distinct job status = %d, want 429", status)
		}
	})

	t.Run("stream over the limit is 429 before opening the stream", func(t *testing.T) {
		userID, token := seedUser(t, "stream-over@example.test")
		fillQuota(t, userID, "so", int(fitAnalysisLimit))
		seedJob(t, "so-new", "so-new")
		model := &fitModel{resp: []string{fitStage1, fitStage2, fitStage3}}
		app := appFor(storeWithCVFor(t, userID), jobfit.NewAnalyzer(llm.NewWithModel(model)))

		req := httptest.NewRequest(fiber.MethodGet, "/api/v1/jobs/so-new/fit/stream", nil)
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("stream: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != fiber.StatusTooManyRequests {
			t.Errorf("stream status = %d, want 429", resp.StatusCode)
		}
		if model.n != 0 {
			t.Errorf("LLM called %d times, want 0 for an over-quota stream", model.n)
		}
	})
}
