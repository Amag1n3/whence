package main

import "testing"

func TestExcerptCollapsesWhitespace(t *testing.T) {
	got := Excerpt("  the  model chose   this branch  ", 80)
	want := "the model chose this branch"
	if got != want {
		t.Fatalf("Excerpt() = %q, want %q", got, want)
	}
}

func TestExcerptTruncates(t *testing.T) {
	got := Excerpt("aaaaaaaaaaaaaaaaaaaa", 10)
	want := "aaaaaaa..."
	if got != want {
		t.Fatalf("Excerpt() = %q, want %q", got, want)
	}
}
