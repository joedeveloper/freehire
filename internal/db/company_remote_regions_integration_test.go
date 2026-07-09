//go:build integration

// Integration tests for the remote-hiring-regions backfill query semantics:
// SetCompanyRemoteRegions updates an existing company's remote_regions and merges
// remote_regions_raw into company_info while leaving every other column (name,
// job_count, is_reference, and the job-derived facets) untouched; an unmatched slug
// affects zero rows and inserts nothing; and re-running is idempotent. Verified
// against a real Postgres. Run with: go test -tags=integration ./internal/db/
package db

import (
	"context"
	"encoding/json"
	"testing"
)

// companyInfoString reads a string value by key out of the stored company_info
// JSONB (Postgres re-serializes JSONB, so parse rather than string-compare).
func companyInfoString(t *testing.T, raw []byte, key string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("company_info not valid JSON (%s): %v", raw, err)
	}
	s, _ := m[key].(string)
	return s
}

func TestSetCompanyRemoteRegions(t *testing.T) {
	pool := startPostgres(t)
	q := New(pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "TRUNCATE companies RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	t.Run("existing company is annotated without disturbing other columns", func(t *testing.T) {
		// Seed a job-backed company carrying job-derived facets and a pre-existing
		// company_info key, so we can prove the update leaves them intact.
		if _, err := pool.Exec(ctx,
			`INSERT INTO companies (slug, name, job_count, regions, countries, company_types, company_sizes, company_info)
			 VALUES ('globex', 'Globex', 7, ARRAY['north_america'], ARRAY['us'], ARRAY['product'], ARRAY['201-500'], '{"homepage":"globex.com"}')`); err != nil {
			t.Fatalf("seed globex: %v", err)
		}
		n, err := q.SetCompanyRemoteRegions(ctx, SetCompanyRemoteRegionsParams{
			Slug:             "globex",
			RemoteRegions:    []string{"eu", "uk"},
			RemoteRegionsRaw: "UK, Europe",
		})
		if err != nil {
			t.Fatalf("set: %v", err)
		}
		if n != 1 {
			t.Fatalf("affected rows = %d, want 1", n)
		}
		c, err := q.GetCompany(ctx, "globex")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if len(c.RemoteRegions) != 2 || c.RemoteRegions[0] != "eu" || c.RemoteRegions[1] != "uk" {
			t.Errorf("remote_regions = %v, want [eu uk]", c.RemoteRegions)
		}
		if got := companyInfoString(t, c.CompanyInfo, "remote_regions_raw"); got != "UK, Europe" {
			t.Errorf("remote_regions_raw = %q, want %q", got, "UK, Europe")
		}
		if got := companyInfoString(t, c.CompanyInfo, "homepage"); got != "globex.com" {
			t.Errorf("merge clobbered existing company_info: homepage = %q", got)
		}
		// Nothing else moves.
		if c.Name != "Globex" || c.JobCount != 7 || c.IsReference {
			t.Errorf("name/job_count/is_reference disturbed: %q %d %v", c.Name, c.JobCount, c.IsReference)
		}
		if len(c.Regions) != 1 || c.Regions[0] != "north_america" {
			t.Errorf("job-derived regions disturbed: %v", c.Regions)
		}
		if len(c.CompanyTypes) != 1 || c.CompanyTypes[0] != "product" ||
			len(c.CompanySizes) != 1 || c.CompanySizes[0] != "201-500" {
			t.Errorf("job-derived enrichment facets disturbed: %v %v", c.CompanyTypes, c.CompanySizes)
		}
	})

	t.Run("unmatched slug affects zero rows and inserts nothing", func(t *testing.T) {
		n, err := q.SetCompanyRemoteRegions(ctx, SetCompanyRemoteRegionsParams{
			Slug:             "ghost",
			RemoteRegions:    []string{"eu"},
			RemoteRegionsRaw: "Europe",
		})
		if err != nil {
			t.Fatalf("set: %v", err)
		}
		if n != 0 {
			t.Errorf("affected rows = %d, want 0 for unmatched slug", n)
		}
		exists, err := q.CompanyExists(ctx, "ghost")
		if err != nil {
			t.Fatalf("exists: %v", err)
		}
		if exists {
			t.Error("unmatched slug created a company row")
		}
	})

	t.Run("empty region set still records the raw source string", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `INSERT INTO companies (slug, name) VALUES ('acme', 'Acme')`); err != nil {
			t.Fatalf("seed acme: %v", err)
		}
		n, err := q.SetCompanyRemoteRegions(ctx, SetCompanyRemoteRegionsParams{
			Slug:             "acme",
			RemoteRegions:    []string{},
			RemoteRegionsRaw: "Atlantis",
		})
		if err != nil {
			t.Fatalf("set: %v", err)
		}
		if n != 1 {
			t.Fatalf("affected rows = %d, want 1", n)
		}
		c, err := q.GetCompany(ctx, "acme")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if len(c.RemoteRegions) != 0 {
			t.Errorf("remote_regions = %v, want empty", c.RemoteRegions)
		}
		if got := companyInfoString(t, c.CompanyInfo, "remote_regions_raw"); got != "Atlantis" {
			t.Errorf("remote_regions_raw = %q, want %q", got, "Atlantis")
		}
	})

	t.Run("re-running is idempotent", func(t *testing.T) {
		p := SetCompanyRemoteRegionsParams{Slug: "globex", RemoteRegions: []string{"eu", "uk"}, RemoteRegionsRaw: "UK, Europe"}
		if _, err := q.SetCompanyRemoteRegions(ctx, p); err != nil {
			t.Fatalf("set 2: %v", err)
		}
		c, err := q.GetCompany(ctx, "globex")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if len(c.RemoteRegions) != 2 || c.RemoteRegions[0] != "eu" || c.RemoteRegions[1] != "uk" {
			t.Errorf("idempotent re-run changed remote_regions: %v", c.RemoteRegions)
		}
	})
}

// TestRefreshCompanyFacetsLeavesRemoteRegions is the recompute guard: the periodic
// facet recompute owns the job-derived arrays but must never read or write the
// curated remote_regions column, so a backfilled value survives every recompute.
func TestRefreshCompanyFacetsLeavesRemoteRegions(t *testing.T) {
	pool := startPostgres(t)
	q := New(pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "TRUNCATE companies, jobs RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	// A company with one open job in north_america, and a curated remote_regions of
	// {eu} that has nothing to do with that job's geography.
	insertCompany(t, pool, "globex", "Globex")
	insertJobWithFacets(t, pool, "globex:1", "globex", []string{"north_america"}, []string{"us"}, "{}")
	if _, err := q.SetCompanyRemoteRegions(ctx, SetCompanyRemoteRegionsParams{
		Slug:             "globex",
		RemoteRegions:    []string{"eu"},
		RemoteRegionsRaw: "Europe",
	}); err != nil {
		t.Fatalf("set remote regions: %v", err)
	}

	if _, err := q.RefreshCompanyFacets(ctx); err != nil {
		t.Fatalf("refresh facets: %v", err)
	}

	c, err := q.GetCompany(ctx, "globex")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	// The recompute populated the job-derived regions from the open job...
	if len(c.Regions) != 1 || c.Regions[0] != "north_america" {
		t.Errorf("job-derived regions = %v, want [north_america]", c.Regions)
	}
	// ...but left the curated remote_regions untouched.
	if len(c.RemoteRegions) != 1 || c.RemoteRegions[0] != "eu" {
		t.Errorf("remote_regions = %v, want [eu] (recompute must not touch it)", c.RemoteRegions)
	}
}
