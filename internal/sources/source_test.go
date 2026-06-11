package sources

import (
	"context"
	"testing"
)

type fakeSource struct{ provider string }

func (f fakeSource) Provider() string { return f.provider }

func (f fakeSource) Fetch(context.Context, CompanyEntry) ([]Job, error) { return nil, nil }

func TestRegIndexesByProvider(t *testing.T) {
	r := reg(fakeSource{"greenhouse"}, fakeSource{"lever"})

	if len(r) != 2 {
		t.Fatalf("len(reg) = %d, want 2", len(r))
	}
	if _, ok := r["greenhouse"]; !ok {
		t.Errorf("reg missing provider %q", "greenhouse")
	}
	if got := r["lever"].Provider(); got != "lever" {
		t.Errorf("reg[lever].Provider() = %q, want %q", got, "lever")
	}
}

func TestRegPanicsOnDuplicateProvider(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("reg with duplicate provider should panic")
		}
	}()

	reg(fakeSource{"greenhouse"}, fakeSource{"greenhouse"})
}
