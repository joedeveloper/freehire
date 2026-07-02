package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// errorApp mounts a route that returns errFn's error through RenderError, so the
// status mapping can be asserted end to end.
func errorApp(errFn func(*fiber.Ctx) error) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: RenderError})
	app.Get("/x", errFn)
	return app
}

func errorStatus(t *testing.T, app *fiber.App) int {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/x", nil))
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	return resp.StatusCode
}

// A write that references a missing parent row (e.g. applying to a non-existent
// job id) raises a foreign-key violation; the caller deserves 404, not 500.
func TestErrorHandler_MapsForeignKeyViolationTo404(t *testing.T) {
	app := errorApp(func(*fiber.Ctx) error { return &pgconn.PgError{Code: "23503"} })
	if got := errorStatus(t, app); got != fiber.StatusNotFound {
		t.Errorf("status = %d, want 404", got)
	}
}

func TestErrorHandler_MapsNoRowsTo404(t *testing.T) {
	app := errorApp(func(*fiber.Ctx) error { return pgx.ErrNoRows })
	if got := errorStatus(t, app); got != fiber.StatusNotFound {
		t.Errorf("status = %d, want 404", got)
	}
}

func TestErrorHandler_DefaultsTo500(t *testing.T) {
	app := errorApp(func(*fiber.Ctx) error { return errString("boom") })
	if got := errorStatus(t, app); got != fiber.StatusInternalServerError {
		t.Errorf("status = %d, want 500", got)
	}
}

// classify decides both the HTTP mapping and whether an error is an unexpected
// fault worth reporting to Sentry. Only the fall-through 500 must be reported;
// routine 4xx and mapped 404s must not, so the error inbox stays signal, not noise.
func TestClassify_ReportsOnlyUnexpected500(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantReport bool
	}{
		{"generic error is an unexpected 500", errString("boom"), fiber.StatusInternalServerError, true},
		{"fiber 4xx is routine", fiber.NewError(fiber.StatusBadRequest, "bad"), fiber.StatusBadRequest, false},
		{"no rows maps to 404", pgx.ErrNoRows, fiber.StatusNotFound, false},
		{"foreign-key violation maps to 404", &pgconn.PgError{Code: "23503"}, fiber.StatusNotFound, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _, report := classify(tt.err)
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}
			if report != tt.wantReport {
				t.Errorf("report = %v, want %v", report, tt.wantReport)
			}
		})
	}
}

type errString string

func (e errString) Error() string { return string(e) }
