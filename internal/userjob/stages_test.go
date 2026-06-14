package userjob

import "testing"

func TestStagesOrder(t *testing.T) {
	want := []string{"applied", "screening", "responded", "interview", "offer", "accepted", "rejected", "withdrawn"}
	if len(Stages) != len(want) {
		t.Fatalf("len(Stages) = %d, want %d", len(Stages), len(want))
	}
	for i, s := range want {
		if Stages[i] != s {
			t.Errorf("Stages[%d] = %q, want %q", i, Stages[i], s)
		}
	}
}

func TestValidStage(t *testing.T) {
	for _, s := range Stages {
		if !ValidStage(s) {
			t.Errorf("ValidStage(%q) = false, want true", s)
		}
	}
	if ValidStage("bogus") {
		t.Error("ValidStage(\"bogus\") = true, want false")
	}
	if ValidStage("") {
		t.Error("ValidStage(\"\") = true, want false")
	}
}
