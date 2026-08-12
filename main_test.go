package main

import (
	"strings"
	"testing"
)

func TestRenderContextOmitsEmptyWhy(t *testing.T) {
	r := Resolved{
		Record: Record{Date: "2026-08-09", File: "main.go", Decision: "keep this path", Source: "NOTE comment"},
		Anchor: Anchor{State: StateExact, Start: 1, End: 1},
	}
	got := renderContext([]Resolved{r})
	if strings.Contains(got, "  why:") {
		t.Fatalf("empty why should not produce a why row:\n%s", got)
	}

	r.Why = "the caller depends on it"
	got = renderContext([]Resolved{r})
	if !strings.Contains(got, "  why: the caller depends on it\n") {
		t.Fatalf("non-empty why should produce a why row:\n%s", got)
	}
}
