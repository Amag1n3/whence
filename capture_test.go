package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// A transcript in the shape Claude Code actually writes, reduced to the fields
// capture reads. The details that matter and are easy to get wrong:
//
//   - a user turn is a bare string; a tool RESULT is also role "user" but its
//     content is an array, and treating one as the other attributes an edit to
//     machinery instead of to the person who asked for it
//   - thinking blocks persist with empty text, which is the finding that shaped
//     this command — so one appears here, and must not be mistaken for prose
//   - isMeta entries are session bookkeeping, not anything anyone said
const trail = `{"type":"mode","sessionId":"x"}
{"type":"user","isMeta":true,"message":{"role":"user","content":"<system-reminder>ignore me</system-reminder>"}}
{"type":"user","message":{"role":"user","content":"the retry loop looks redundant, clean it up"}}
{"type":"assistant","timestamp":"2026-08-02T10:00:00Z","message":{"role":"assistant","content":[{"type":"thinking","thinking":"","signature":"CAIS"}]}}
{"type":"assistant","timestamp":"2026-08-02T10:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"Keeping the backoff. The provider rate-limits per account, not per key, so a bare retry stampedes."}]}}
{"type":"assistant","timestamp":"2026-08-02T10:00:02Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"SUB/api.go","new_string":"\tfor i := 0; i < 3; i++ {\n\t\tbackoff(i)\n\t}"}}]}}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"ok"}]}}
{"type":"assistant","timestamp":"2026-08-02T10:00:03Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"SUB/api.go","new_string":"gone from the file now"}}]}}
{"type":"assistant","timestamp":"2026-08-02T10:00:04Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]}}
`

func TestReadTrailPairsAnEditWithWhatWasSaidBeforeIt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(path, []byte(trail), 0o600); err != nil {
		t.Fatal(err)
	}

	ms, err := readTrail(path)
	if err != nil {
		t.Fatal(err)
	}
	// Two edits. The Bash call is not one, and neither is the tool result.
	if len(ms) != 2 {
		t.Fatalf("got %d edits, want 2", len(ms))
	}

	m := ms[0]
	if m.Asked != "the retry loop looks redundant, clean it up" {
		t.Errorf("asked = %q — a tool result or a meta entry was read as a user turn", m.Asked)
	}
	if m.Said == "" || m.Said[:14] != "Keeping the ba" {
		t.Errorf("said = %q, want the assistant prose before the edit", m.Said)
	}
	if m.Tool != "Edit" || m.File != "SUB/api.go" {
		t.Errorf("tool/file = %s %s", m.Tool, m.File)
	}

	// The second edit inherits the same user turn — one request, several edits —
	// but must not inherit prose from before the first edit it did not precede.
	if ms[1].Asked != m.Asked {
		t.Errorf("second edit lost the user turn: %q", ms[1].Asked)
	}
}

func TestCaptureRequiresAnExplicitSession(t *testing.T) {
	for _, args := range [][]string{nil, {}, {"one.jsonl", "two.jsonl"}} {
		if _, err := capturePath(args); err == nil {
			t.Fatalf("capture args %v must be rejected without exactly one session", args)
		}
	}
	if got, err := capturePath([]string{"session.jsonl"}); err != nil || got != "session.jsonl" {
		t.Fatalf("explicit session path = %q, %v", got, err)
	}
}

// The footer is the command's whole output, and its first version reported
// "stated reasoning" for every edit in three real sessions because it counted
// any non-empty string. These are verbatim from those runs: the announcements
// outnumbered the reasons roughly one to one, and no metric that scores them
// alike can show that.
func TestAnnouncementsAreNotReasons(t *testing.T) {
	announced := []string{
		"Now the code.",
		"Now the edits:",
		"Implementing both now.",
		"Continuing — wiring it into dispatch and usage.",
		"Now the CI workflow and the site.",
	}
	for _, s := range announced {
		if hasReason(s) {
			t.Errorf("counted as a reason: %q", s)
		}
	}

	reasoned := []string{
		"It's a real inconsistency: the note was admitted because it gave a reason, " +
			"then stored with no reason.",
		"The guard was a proxy for the real problem, since `reason` and `caused` " +
			"are the only entries that appear mid-clause.",
		"Left narrow to avoid a garbage record in a committed store.",
	}
	for _, s := range reasoned {
		if !hasReason(s) {
			t.Errorf("missed a reason: %q", s)
		}
	}

	// A KNOWN MISS, kept here as the evidence rather than as a passing assertion
	// dressed up as coverage. This is one of the strongest decision statements in
	// the corpus and the word list does not see it: the reason is carried by
	// "which is the one thing X exists to prevent", not by a conjunction.
	//
	// reasonWords already carries a ponytail comment saying to widen it only
	// against real misses from a real repo, never by imagining phrasings. This is
	// that miss, and widening is not capture's call to make alone — the list is
	// backfill's admission gate for a committed, shared store.
	missed := "it promoted a guess to a certainty, which is the one thing " +
		"anchor.go exists to prevent."
	if hasReason(missed) {
		t.Errorf("word list widened without revisiting this: %q", missed)
	}
}

// An edit under a different root can only ever become a record in a different
// store, so it is not this store's business — but it is counted, because a tool
// showing fewer edits than the session made must say so.
func TestEditsInOtherReposAreCountedNotDropped(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "api.go", []string{"package api"})
	if _, _, err := add(filepath.Join(dir, "api.go"), 1, 1, "d", "w", "s", authorHuman, nil); err != nil {
		t.Fatal(err)
	}

	here, elsewhere := hereOnly([]moment{
		{File: filepath.Join(dir, "api.go")},
		{File: filepath.Join(t.TempDir(), "resume.tex")}, // no store above it
	})
	if len(here) != 1 || elsewhere != 1 {
		t.Fatalf("here = %d, elsewhere = %d; want 1 and 1", len(here), elsewhere)
	}
}

// The surprise signal (§22.5): a reason-bearing statement follows something
// unexpected, and the cheap reliable marker of that is the tool result that
// preceded the edit — machine output, where "FAIL" means FAIL. The reason-word
// floor lies about prose; this is not vocabulary about prose at all.
func TestFollowsSurprise(t *testing.T) {
	cases := []struct {
		res  string
		want bool
	}{
		{"", false},
		{"ok", false},
		{"--- FAIL: TestRetry", true},
		{"panic: runtime error", true},
		{"exit status 1", true},
		{"cannot find package", false}, // a miss, not a garbage record
	}
	for _, c := range cases {
		if got := followsSurprise(c.res); got != c.want {
			t.Errorf("followsSurprise(%q) = %v, want %v", c.res, got, c.want)
		}
	}
}

// toolResultText must read both shapes a tool result takes: a bare string and
// an array of blocks.
func TestToolResultText(t *testing.T) {
	raw := json.RawMessage(`[{"type":"tool_result","content":"--- FAIL: TestRetry"}]`)
	if got := toolResultText(raw); got != "--- FAIL: TestRetry" {
		t.Errorf("string content = %q", got)
	}
	raw = json.RawMessage(`[{"type":"tool_result","content":[{"type":"text","text":"--- FAIL"}]}]`)
	if got := toolResultText(raw); got != "--- FAIL" {
		t.Errorf("block content = %q", got)
	}
	if got := toolResultText(json.RawMessage(`[{"type":"thinking","thinking":""}]`)); got != "" {
		t.Errorf("non-result content = %q, want empty", got)
	}
}

func TestLocateSpanRefusesToGuess(t *testing.T) {
	lines := []string{
		"package api",
		"",
		"func send() {",
		"\tfor i := 0; i < 3; i++ {",
		"\t\tbackoff(i)",
		"\t}",
		"}",
	}

	start, end := locateSpan(lines, "\tfor i := 0; i < 3; i++ {\n\t\tbackoff(i)\n\t}")
	if start != 4 || end != 6 {
		t.Errorf("span = %d-%d, want 4-6", start, end)
	}

	// Surrounding blank lines are noise from the edit payload, not content.
	start, end = locateSpan(lines, "\n\nfunc send() {\n\n")
	if start != 3 || end != 3 {
		t.Errorf("padded span = %d-%d, want 3-3", start, end)
	}

	// Changed since: no span at all, rather than the nearest thing that fits.
	if start, _ := locateSpan(lines, "\tfor i := 0; i < 5; i++ {\n\t\tbackoff(i)\n\t}"); start != 0 {
		t.Errorf("start = %d, want 0 for a block that no longer matches", start)
	}
	if start, _ := locateSpan(lines, ""); start != 0 {
		t.Errorf("start = %d, want 0 for empty text", start)
	}
}
