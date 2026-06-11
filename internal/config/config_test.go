package config

import (
	"testing"
	"time"
)

func TestLoad_JWTSecretFromEnv(t *testing.T) {
	t.Setenv("JWT_SECRET", "s3cret")

	if got := Load().JWTSecret; got != "s3cret" {
		t.Errorf("JWTSecret = %q, want %q", got, "s3cret")
	}
}

func TestLoad_JWTTTLDefaultsWhenUnset(t *testing.T) {
	t.Setenv("JWT_TTL", "")

	if got := Load().JWTTTL; got != 24*time.Hour {
		t.Errorf("JWTTTL = %v, want 24h", got)
	}
}

func TestLoad_JWTTTLParsesDuration(t *testing.T) {
	t.Setenv("JWT_TTL", "1h30m")

	if got := Load().JWTTTL; got != 90*time.Minute {
		t.Errorf("JWTTTL = %v, want 1h30m", got)
	}
}

func TestLoad_JWTTTLFallsBackOnGarbage(t *testing.T) {
	t.Setenv("JWT_TTL", "not-a-duration")

	if got := Load().JWTTTL; got != 24*time.Hour {
		t.Errorf("JWTTTL = %v, want 24h fallback", got)
	}
}

func TestLoad_MeiliURLDefaultsWhenUnset(t *testing.T) {
	t.Setenv("MEILI_URL", "")

	if got := Load().MeiliURL; got != "http://localhost:7700" {
		t.Errorf("MeiliURL = %q, want default", got)
	}
}

func TestLoad_MeiliURLFromEnv(t *testing.T) {
	t.Setenv("MEILI_URL", "http://meili:7700")

	if got := Load().MeiliURL; got != "http://meili:7700" {
		t.Errorf("MeiliURL = %q, want env value", got)
	}
}

func TestLoad_MeiliKeyFromEnv(t *testing.T) {
	t.Setenv("MEILI_MASTER_KEY", "master-key")

	if got := Load().MeiliKey; got != "master-key" {
		t.Errorf("MeiliKey = %q, want %q", got, "master-key")
	}
}
