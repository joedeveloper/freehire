package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/strelov1/freehire/internal/auth"
)

// meJobsApp mounts the my-jobs listing behind RequireAuth on a handler with no
// DB. The cases below (auth gate, filter validation) reject before any query
// runs, so the nil queries is never dereferenced; the DB-backed listing contract
// is covered by the integration test.
func meJobsApp(t *testing.T) (*fiber.App, string) {
	t.Helper()
	iss := auth.NewIssuer("test-secret", time.Hour)
	token, err := iss.Issue(1)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	h := &Handler{issuer: iss}
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	app.Get("/me/jobs", auth.RequireAuth(iss), h.ListMyJobs)
	return app, token
}

func getMeJobs(t *testing.T, app *fiber.App, path, token string) int {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, path, nil)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestListMyJobs_RequiresAuth(t *testing.T) {
	app, _ := meJobsApp(t)
	if got := getMeJobs(t, app, "/me/jobs", ""); got != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want 401", got)
	}
}

func TestListMyJobs_UnknownFilter(t *testing.T) {
	app, token := meJobsApp(t)
	if got := getMeJobs(t, app, "/me/jobs?filter=bogus", token); got != fiber.StatusBadRequest {
		t.Errorf("status = %d, want 400", got)
	}
}
