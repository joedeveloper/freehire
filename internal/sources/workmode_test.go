package sources

import "testing"

func TestWorkModeFromRemote(t *testing.T) {
	if got := workModeFromRemote(true); got != "remote" {
		t.Errorf("workModeFromRemote(true) = %q, want remote", got)
	}
	// A false flag does not imply onsite vs hybrid — leave it unknown.
	if got := workModeFromRemote(false); got != "" {
		t.Errorf("workModeFromRemote(false) = %q, want empty", got)
	}
}

func TestWorkplaceTypeMode(t *testing.T) {
	cases := map[string]string{
		"remote":      "remote",
		"hybrid":      "hybrid",
		"on-site":     "onsite",
		"onsite":      "onsite",
		"On-Site":     "onsite",
		"unspecified": "",
		"":            "",
		"weird":       "",
	}
	for in, want := range cases {
		if got := workplaceTypeMode(in); got != want {
			t.Errorf("workplaceTypeMode(%q) = %q, want %q", in, got, want)
		}
	}
}
