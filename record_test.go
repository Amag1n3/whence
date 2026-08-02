package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The store is committed, so it gets merged, and git merges lines — which is the
// whole reason a record has to be exactly one line. The legacy array shape still
// has to load, or upgrading whence would quietly empty somebody's decisions.
func TestStoreReadsBothShapesAndWritesOneRecordPerLine(t *testing.T) {
	dir := t.TempDir()

	legacy := filepath.Join(dir, legacyRecordsName)
	if err := os.WriteFile(legacy, []byte(`[
  {"id":"a1","file":"x.go","line_start":1,"line_end":2,"decision":"one"},
  {"id":"b2","file":"y.go","line_start":3,"line_end":4,"decision":"two"}
]`), 0o644); err != nil {
		t.Fatal(err)
	}
	rs, err := Load(legacy)
	if err != nil || len(rs) != 2 {
		t.Fatalf("a legacy array store must still load, got %d (%v)", len(rs), err)
	}

	out := filepath.Join(dir, recordsFileName)
	if err := save(out, rs); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n"); len(lines) != 2 {
		t.Errorf("one record per line is the point; got %d lines:\n%s", len(lines), b)
	}

	back, err := Load(out)
	if err != nil || len(back) != 2 || back[0].ID != "a1" || back[1].Decision != "two" {
		t.Fatalf("the round trip lost records: %+v (%v)", back, err)
	}

	// A malformed line is reported, never skipped. Silently dropping a decision
	// is the failure this whole tool exists to prevent.
	bad := filepath.Join(dir, "bad.jsonl")
	if err := os.WriteFile(bad, []byte("{\"id\":\"a1\"}\nnot json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(bad); err == nil {
		t.Error("a malformed line must be an error, not a silent skip")
	}
}

// The non-trivial logic in Phase 0 is store resolution, line-range overlap and
// path normalisation. If any of them breaks, why silently shows the wrong
// records or none at all — which is worse than showing nothing, because it
// trains you to distrust the tool. So all three get checks.

var fixture = []Record{
	{ID: "b5", Date: "2026-07-27", File: "src/auth/session.go", Start: 140, End: 150,
		Decision: "namespace session keys", Source: "code review"},
	{ID: "old", Date: "2026-07-01", File: "src/auth/session.go", Start: 145, End: 145,
		Decision: "earlier note on the same line", Source: "code review"},
	{ID: "other", Date: "2026-07-28", File: "src/http/router.go", Start: 10, End: 20,
		Decision: "unrelated file", Source: "commit"},
}

// unanchored wraps hand-built records the way Match would, for the display
// tests. These carry no line hashes, so they resolve to the line-only state
// without touching the filesystem.
func unanchored(rs []Record) []Resolved {
	out := make([]Resolved, len(rs))
	for i, r := range rs {
		out[i] = Resolved{Record: r, Anchor: resolveAnchor(nil, r)}
	}
	return out
}

// --- store resolution ---------------------------------------------------

// Regression test for the real bug: a session rooted in one repo editing a file
// in a sibling repo resolved the store from the session's cwd, found the wrong
// repo (or none), then compared an absolute path against repo-relative records
// and matched nothing — silently.
func TestFindStoreResolvesByFileNotSession(t *testing.T) {
	tmp := t.TempDir()
	back := filepath.Join(tmp, "backend")
	front := filepath.Join(tmp, "frontend")
	for _, r := range []string{back, front} {
		mkStore(t, r)
	}

	// A frontend file must resolve to the FRONTEND store, no matter which repo
	// the session happens to be rooted in.
	store, root, ok := FindStore(filepath.Join(front, "src", "a.js"))
	if !ok {
		t.Fatal("should have found the frontend store")
	}
	if root != front {
		t.Errorf("root = %q, want %q", root, front)
	}
	if want := filepath.Join(front, storeDirName, recordsFileName); store != want {
		t.Errorf("store = %q, want %q", store, want)
	}

	// And the backend file resolves to its own, not the frontend's.
	if _, root, ok := FindStore(filepath.Join(back, "src", "a.js")); !ok || root != back {
		t.Errorf("backend file should resolve to backend store, got root=%q ok=%v", root, ok)
	}
}

func TestFindStoreWalksUpFromNested(t *testing.T) {
	tmp := t.TempDir()
	mkStore(t, tmp)
	deep := filepath.Join(tmp, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, root, ok := FindStore(filepath.Join(deep, "f.go")); !ok || root != tmp {
		t.Errorf("should walk up to %q, got root=%q ok=%v", tmp, root, ok)
	}
}

func TestFindStoreAbsentTerminates(t *testing.T) {
	// No store anywhere: must report false rather than loop to the root forever.
	if _, _, ok := FindStore(filepath.Join(t.TempDir(), "f.go")); ok {
		t.Error("expected ok=false when no store exists")
	}
}

func mkStore(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, storeDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, storeDirName, recordsFileName)
	if err := os.WriteFile(p, []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// --- matching -----------------------------------------------------------

func TestMatchWholeFile(t *testing.T) {
	got := Match("", fixture, "src/auth/session.go", 0)
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
	if got := Match("", fixture, "src/auth/session.go", 145); len(got) != 2 {
		t.Fatalf("line 145 should match both overlapping records: got %d, want 2", len(got))
	}
}

func TestMatchLineOutsideSpan(t *testing.T) {
	if got := Match("", fixture, "src/auth/session.go", 200); len(got) != 0 {
		t.Errorf("line 200 is outside every span, got %d records", len(got))
	}
	// Boundaries are inclusive.
	if got := Match("", fixture, "src/auth/session.go", 140); len(got) != 1 {
		t.Errorf("span start should be inclusive, got %d", len(got))
	}
	if got := Match("", fixture, "src/auth/session.go", 150); len(got) != 1 {
		t.Errorf("span end should be inclusive, got %d", len(got))
	}
}

func TestMatchWrongFile(t *testing.T) {
	if got := Match("", fixture, "src/http/router.go", 15); len(got) != 1 || got[0].ID != "other" {
		t.Errorf("should match only the record on that file, got %d", len(got))
	}
	if got := Match("", fixture, "nope.go", 0); len(got) != 0 {
		t.Errorf("unknown file should match nothing, got %d", len(got))
	}
}

func TestSamePathTolerance(t *testing.T) {
	// A hand-written record may carry a "./" prefix; it must still match.
	rs := []Record{{ID: "x", Date: "2026-01-01", File: "./src/a.go", Start: 1, End: 9}}
	if got := Match("", rs, "src/a.go", 5); len(got) != 1 {
		t.Errorf(`"./src/a.go" should match "src/a.go", got %d`, len(got))
	}
}

func TestRel(t *testing.T) {
	root := filepath.FromSlash("/repo")
	if got := Rel(root, filepath.FromSlash("/repo/src/a.go")); filepath.ToSlash(got) != "src/a.go" {
		t.Errorf("Rel should relativise inside root, got %q", got)
	}
	// Outside root: returned unchanged, never as "../..", so it matches nothing.
	outside := filepath.FromSlash("/elsewhere/b.go")
	if got := Rel(root, outside); got != outside {
		t.Errorf("Rel should leave outside paths alone, got %q", got)
	}
	if got := Rel(root, "src/a.go"); got != "src/a.go" {
		t.Errorf("Rel should leave relative paths alone, got %q", got)
	}
}

// --- rendering ----------------------------------------------------------

func TestRenderContextIsFramedAsData(t *testing.T) {
	// The preamble is the prompt-injection mitigation. If it ever goes missing,
	// injected records read as instructions to an agent. Guard it.
	out := renderContext(unanchored(fixture[:1]))
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
	out := renderContext(unanchored(many))
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
