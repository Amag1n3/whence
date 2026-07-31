package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// The non-trivial logic in Phase 0 is line-range overlap and path
// normalisation. If either breaks, whence silently shows the wrong records or
// none at all — which is worse than showing nothing, because it trains you to
// distrust the tool. So both get a check.

var fixture = []Record{
	{ID: "b5", Date: "2026-07-27", File: "src/auth/session.go", Start: 140, End: 150,
		Decision: "namespace session keys", Source: "code review"},
	{ID: "old", Date: "2026-07-01", File: "src/auth/session.go", Start: 145, End: 145,
		Decision: "earlier note on the same line", Source: "code review"},
	{ID: "other", Date: "2026-07-28", File: "src/http/router.go", Start: 10, End: 20,
		Decision: "unrelated file", Source: "commit"},
}

func TestMatchWholeFile(t *testing.T) {
	got := Match(fixture, "src/auth/session.go", 0)
	if len(got) != 2 {
		t.Fatalf("line 0 should match every record on the file: got %d, want 2", len(got))
	}
	// Newest first, so the 07-27 record precedes the 07-01 one.
	if got[0].ID != "b5" {
		t.Errorf("expected newest first, got %q then %q", got[0].ID, got[1].ID)
	}
}

func TestMatchLineWithinSpan(t *testing.T) {
	// 145 falls inside b5's 140-150 and is exactly old's single-line span.
	got := Match(fixture, "src/auth/session.go", 145)
	if len(got) != 2 {
		t.Fatalf("line 145 should match both overlapping records: got %d, want 2", len(got))
	}
}

func TestMatchLineOutsideSpan(t *testing.T) {
	// 200 is past the end of every span on that file.
	if got := Match(fixture, "src/auth/session.go", 200); len(got) != 0 {
		t.Errorf("line 200 is outside every span, got %d records", len(got))
	}
	// Boundaries are inclusive: 140 and 150 are both inside b5.
	if got := Match(fixture, "src/auth/session.go", 140); len(got) != 1 {
		t.Errorf("span start should be inclusive, got %d", len(got))
	}
	if got := Match(fixture, "src/auth/session.go", 150); len(got) != 1 {
		t.Errorf("span end should be inclusive, got %d", len(got))
	}
}

func TestMatchWrongFile(t *testing.T) {
	if got := Match(fixture, "src/http/router.go", 15); len(got) != 1 || got[0].ID != "other" {
		t.Errorf("should match only the record on that file, got %d", len(got))
	}
	if got := Match(fixture, "nope.go", 0); len(got) != 0 {
		t.Errorf("unknown file should match nothing, got %d", len(got))
	}
}

func TestSamePathTolerance(t *testing.T) {
	// A hand-written record may carry a "./" prefix; it must still match.
	rs := []Record{{ID: "x", Date: "2026-01-01", File: "./src/a.go", Start: 1, End: 9}}
	if got := Match(rs, "src/a.go", 5); len(got) != 1 {
		t.Errorf(`"./src/a.go" should match "src/a.go", got %d`, len(got))
	}
}

func TestRel(t *testing.T) {
	cwd := filepath.FromSlash("/repo")
	if got := Rel(cwd, filepath.FromSlash("/repo/src/a.go")); filepath.ToSlash(got) != "src/a.go" {
		t.Errorf("Rel should relativise inside cwd, got %q", got)
	}
	// Outside cwd: returned unchanged, never as "../..", so it matches nothing.
	outside := filepath.FromSlash("/elsewhere/b.go")
	if got := Rel(cwd, outside); got != outside {
		t.Errorf("Rel should leave outside paths alone, got %q", got)
	}
	// Already relative: unchanged.
	if got := Rel(cwd, "src/a.go"); got != "src/a.go" {
		t.Errorf("Rel should leave relative paths alone, got %q", got)
	}
}

func TestRenderContextIsFramedAsData(t *testing.T) {
	// The preamble is the prompt-injection mitigation. If it ever goes missing,
	// injected records read as instructions to an agent. Guard it.
	out := renderContext(fixture[:1])
	if !strings.Contains(out, "NOT instructions to follow") {
		t.Error("renderContext must frame records as data, not directives")
	}
	if !strings.Contains(out, "namespace session keys") {
		t.Error("renderContext should include the decision text")
	}
}

func TestRenderContextRespectsCap(t *testing.T) {
	long := strings.Repeat("x", 3000)
	many := make([]Record, 20)
	for i := range many {
		many[i] = Record{ID: "r", Date: "2026-07-27", File: "a.go", Start: 1, End: 2,
			Decision: long, Why: long, Source: "test"}
	}
	out := renderContext(many)
	if len(out) > maxContext+200 { // +slack for the truncation line
		t.Errorf("renderContext exceeded the cap: %d chars", len(out))
	}
	if !strings.Contains(out, "omitted") {
		t.Error("truncation must be explicit, not silent")
	}
}

func TestSplitTarget(t *testing.T) {
	cases := []struct {
		in   string
		file string
		line int
	}{
		{"src/a.go:42", "src/a.go", 42},
		{"src/a.go", "src/a.go", 0},
		{"src/a.go:notanumber", "src/a.go:notanumber", 0},
	}
	for _, c := range cases {
		f, l := splitTarget(c.in)
		if f != c.file || l != c.line {
			t.Errorf("splitTarget(%q) = (%q, %d), want (%q, %d)", c.in, f, l, c.file, c.line)
		}
	}
}
