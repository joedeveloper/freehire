package main

import "testing"

func TestEmitVocab(t *testing.T) {
	got := emitVocab("Seniority", "SENIORITY_VALUES", []string{"junior", "senior"})
	want := "export const SENIORITY_VALUES = ['junior', 'senior'] as const;\n" +
		"export type Seniority = (typeof SENIORITY_VALUES)[number];\n"
	if got != want {
		t.Errorf("emitVocab mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestEmitVocabEmpty(t *testing.T) {
	got := emitVocab("X", "X_VALUES", nil)
	want := "export const X_VALUES = [] as const;\n" +
		"export type X = (typeof X_VALUES)[number];\n"
	if got != want {
		t.Errorf("emitVocab(empty) = %q, want %q", got, want)
	}
}
