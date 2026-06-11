package sources

import (
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	data := []byte(`
sources:
  - company: Cohere
    provider: greenhouse
    board: cohere
  - company: Vercel
    provider: ashby
    board: vercel
`)

	cfg, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if len(cfg.Sources) != 2 {
		t.Fatalf("len(Sources) = %d, want 2", len(cfg.Sources))
	}
	want := CompanyEntry{Company: "Cohere", Provider: "greenhouse", Board: "cohere"}
	if cfg.Sources[0] != want {
		t.Errorf("Sources[0] = %+v, want %+v", cfg.Sources[0], want)
	}
}

func TestConfigValidateRejectsUnknownProvider(t *testing.T) {
	cfg := Config{Sources: []CompanyEntry{
		{Company: "Acme", Provider: "myspace", Board: "acme"},
	}}

	err := cfg.Validate(reg(fakeSource{"greenhouse"}))
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
	if !strings.Contains(err.Error(), "myspace") {
		t.Errorf("error %q should name the unknown provider", err.Error())
	}
}

func TestConfigValidateRejectsEmptyBoard(t *testing.T) {
	cfg := Config{Sources: []CompanyEntry{
		{Company: "Cohere", Provider: "greenhouse", Board: ""},
	}}

	err := cfg.Validate(reg(fakeSource{"greenhouse"}))
	if err == nil {
		t.Fatal("expected error for empty board, got nil")
	}
	if !strings.Contains(err.Error(), "Cohere") {
		t.Errorf("error %q should name the offending company", err.Error())
	}
}

func TestConfigValidateRejectsEmptyCompany(t *testing.T) {
	cfg := Config{Sources: []CompanyEntry{
		{Company: "", Provider: "greenhouse", Board: "cohere"},
	}}

	if err := cfg.Validate(reg(fakeSource{"greenhouse"})); err == nil {
		t.Fatal("expected error for empty company, got nil")
	}
}

func TestConfigValidateAcceptsKnownProviders(t *testing.T) {
	cfg := Config{Sources: []CompanyEntry{
		{Company: "Cohere", Provider: "greenhouse", Board: "cohere"},
	}}

	if err := cfg.Validate(reg(fakeSource{"greenhouse"})); err != nil {
		t.Errorf("Validate: unexpected error %v", err)
	}
}
