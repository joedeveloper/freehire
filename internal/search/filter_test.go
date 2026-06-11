package search

import (
	"reflect"
	"testing"
)

func TestEq_QuotesAndEscapes(t *testing.T) {
	if got := Eq("seniority", "senior"); got != `seniority = "senior"` {
		t.Errorf("Eq = %q", got)
	}
	// A value carrying a quote must be escaped so it cannot break out of the
	// string literal and inject filter logic.
	if got := Eq("title", `a"b`); got != `title = "a\"b"` {
		t.Errorf("Eq escape = %q", got)
	}
}

func TestEqBool(t *testing.T) {
	if got := EqBool("remote", true); got != "remote = true" {
		t.Errorf("EqBool true = %q", got)
	}
	if got := EqBool("visa_sponsorship", false); got != "visa_sponsorship = false" {
		t.Errorf("EqBool false = %q", got)
	}
}

func TestGteLte(t *testing.T) {
	if got := Gte("salary_min", 100000); got != "salary_min >= 100000" {
		t.Errorf("Gte = %q", got)
	}
	if got := Lte("salary_max", 200000); got != "salary_max <= 200000" {
		t.Errorf("Lte = %q", got)
	}
}

func TestFilter_NilWhenEmpty(t *testing.T) {
	if got := Filter(); got != nil {
		t.Errorf("Filter() = %v, want nil", got)
	}
	if got := Filter([]string{}, nil); got != nil {
		t.Errorf("Filter(empty groups) = %v, want nil", got)
	}
}

func TestFilter_NestsAndOfOr(t *testing.T) {
	got := Filter(
		[]string{Eq("category", "backend"), Eq("category", "frontend")}, // OR within
		[]string{EqBool("remote", true)},
	)
	want := [][]string{
		{`category = "backend"`, `category = "frontend"`},
		{"remote = true"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Filter = %#v, want %#v", got, want)
	}
}

func TestFilter_SkipsEmptyGroups(t *testing.T) {
	got := Filter(
		[]string{Eq("seniority", "senior")},
		[]string{}, // dropped
		nil,        // dropped
	)
	want := [][]string{{`seniority = "senior"`}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Filter = %#v, want %#v", got, want)
	}
}
