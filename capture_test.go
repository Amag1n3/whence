package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// lastSaid is read backwards from the end of a live transcript, which is the one
// place a mistake is invisible: pick the wrong entry and the record still gets
// written, just attributing the wrong reason to the edit.
func TestLastSaidTakesTheNearestProseNotTheFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "live.jsonl")
	if err := os.WriteFile(path, []byte(trail), 0o600); err != nil {
		t.Fatal(err)
	}
	// trail ends with a Bash tool_use, so the nearest prose is the backoff line
	// two assistant turns earlier — not the thinking block between them, which
	// persists with empty text, and not the user turn.
	if got := lastSaid(path); got[:14] != "Keeping the ba" {
		t.Errorf("lastSaid = %q, want the assistant prose nearest the end", got)
	}

	// A transcript with no assistant prose at all states no reason. Empty, not a
	// fallback to the user's request — the user asking for something is not the
	// agent explaining it.
	quiet := filepath.Join(t.TempDir(), "quiet.jsonl")
	if err := os.WriteFile(quiet, []byte(`{"type":"user","message":{"role":"user","content":"do the thing"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := lastSaid(quiet); got != "" {
		t.Errorf("lastSaid = %q, want empty when nothing was said", got)
	}
}

// The tail window starts mid-entry on any real session, so the first line in it
// is a fragment. Parsing that fragment is harmless, but keeping it in the scan
// while the window is 1MB into a file is how an off-by-one here would hide.
func TestLastSaidSurvivesAWindowThatStartsMidEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.jsonl")
	// One entry larger than the window, then the prose, then the edit — the shape
	// a session takes right after an agent writes a large file.
	filler := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"` +
		strings.Repeat("x", 1<<20) + `"}]}}`
	body := filler + "\n" +
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Real bug: the guard only gated the exact scan."}]}}` + "\n" +
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"a.go","new_string":"x"}}]}}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := lastSaid(path); got != "Real bug: the guard only gated the exact scan." {
		t.Errorf("lastSaid = %q — the window trim dropped a whole entry", got)
	}
}

// Verbatim from the three sessions this filter was read off. The accepts are
// reasons worth a permanent record; the rejects are the two ways the pairing
// produces junk — narration, and prose about something other than the edit.
func TestCaptureWorthy(t *testing.T) {
	cases := []struct {
		name string
		said string
		text string
		path string
		want bool
	}{{
		name: "a correction naming a symbol in the edit",
		said: "One real finding: the weighting fixed the mixed case but not the all-boilerplate case. Because the score is a ratio, a span where every line is common scores 1.0. The `identifiable` guard has to gate both scans, not just the exact one.",
		text: "\tif identifiable(lines) {",
		path: "/repo/anchor.go",
		want: true,
	}, {
		name: "a correction naming the file rather than a symbol",
		said: "One more false positive: `author.go:199` is a comment that mentions the marker mid-sentence. The convention is that a note begins with it, so a prefix is stricter and simpler.",
		text: "\tif !strings.HasPrefix(line, marker) {",
		path: "/repo/author.go",
		want: true,
	}, {
		// The dominant discard: 14 of 38 distinct reasons in one session.
		name: "sequencing narration",
		said: "Now the authoring side — `add`, and `backfill` reusing it.",
		text: "func add(file string) error {",
		path: "/repo/author.go",
		want: false,
	}, {
		// Real: this was paired with a file edit because it was what the
		// assistant happened to be saying at the time.
		name: "prose about something other than the code",
		said: "Claude for Open Source is the one to apply for right now — Max 20x, 6 months free, and it needs no company. Sitting on it would be wrong.",
		text: "func add(file string) error {",
		path: "/repo/author.go",
		want: false,
	}, {
		name: "a marker but nothing tying it to this edit",
		said: "Real bug: the retry loop stampedes. Fixed it.",
		text: "func add(file string) error {",
		path: "/repo/author.go",
		want: false,
	}}

	for _, c := range cases {
		if got := captureWorthy(c.said, c.text, c.path); got != c.want {
			t.Errorf("%s: captureWorthy = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestBackticked(t *testing.T) {
	got := backticked("the `identifiable` guard gates `exactScan` and a stray ` tick")
	if len(got) != 2 || got[0] != "identifiable" || got[1] != "exactScan" {
		t.Errorf("backticked = %q, want the two closed spans and nothing from the unterminated one", got)
	}
	if got := backticked("no ticks here"); got != nil {
		t.Errorf("backticked = %q, want nil", got)
	}
}

func runHookPost(t *testing.T, payload string) string {
	t.Helper()
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(inW, payload); err != nil {
		t.Fatal(err)
	}
	inW.Close()

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldIn, oldOut := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = inR, outW
	defer func() { os.Stdin, os.Stdout = oldIn, oldOut }()

	hookPost()
	outW.Close()
	b, err := io.ReadAll(outR)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// postPayload is hookJSON plus the transcript, which is the only input hookPost
// has that hookPre does not.
func postPayload(t *testing.T, abs, transcript, neu string) string {
	t.Helper()
	in := hookIn{Cwd: filepath.Dir(abs), SessionID: "s1", TranscriptPath: transcript}
	in.ToolInput.FilePath = abs
	in.ToolInput.NewString = neu
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func transcriptSaying(t *testing.T, said string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session.jsonl")
	entry := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"role":    "assistant",
			"content": []map[string]any{{"type": "text", "text": said}},
		},
	}
	b, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// The whole path, because every piece of it is individually tested and the thing
// that actually matters is that a real payload ends as one pending record and
// not two. Agent records land in pending.jsonl — local, gitignored — never the
// shared store. Properties that cannot be walked back: agent authorship,
// unconfirmed, and no duplicate on a second edit under the same prose.
func TestHookPostRecordsAStatedReasonOnce(t *testing.T) {
	dir, abs, _, _ := hookRepo(t)
	store := filepath.Join(dir, storeDirName, recordsFileName)
	pending := filepath.Join(dir, storeDirName, pendingLogName)

	before, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}

	tr := transcriptSaying(t, "Real bug: the counter was off by one, so `unique_line_20` was skipped entirely. Renumbering from the top fixes it.")
	out := runHookPost(t, postPayload(t, abs, tr, "var unique_line_20 = 20"))

	if after, err := Load(store); err != nil || len(after) != len(before) {
		t.Fatalf("hookPost wrote the shared store: %d → %d", len(before), len(after))
	}
	rs, err := Load(pending)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 1 {
		t.Fatalf("pending went to %d records, want 1", len(rs))
	}
	got := rs[0]

	if got.Source != "capture" {
		t.Errorf("source = %q, want capture", got.Source)
	}
	if got.Author != authorAgent {
		t.Errorf("author = %q — a hook-written record claiming human authorship launders it past the confirm gate", got.Author)
	}
	if !got.unchecked() {
		t.Errorf("record is not UNCHECKED (verified = %q); nothing here has had human attention", got.Verified)
	}
	if got.Start != 20 || got.End != 20 {
		t.Errorf("anchored to %d-%d, want 20-20 — the span the edit actually wrote", got.Start, got.End)
	}
	if got.Why == "" {
		t.Error("why is empty; the sentence after the first is the reason and it was dropped")
	}
	if !strings.Contains(hookContext(t, out), got.ID) {
		t.Errorf("the agent was not told what was recorded; context = %q", hookContext(t, out))
	}

	// Same prose, another edit in the same batch. One assistant message covers
	// several edits, so without the dedup this is where pending doubles.
	runHookPost(t, postPayload(t, abs, tr, "var unique_line_21 = 21"))
	if again, err := Load(pending); err != nil || len(again) != 1 {
		t.Errorf("second edit under the same prose added a pending record: %d, want 1", len(again))
	}
}

// One message, two edits, a reason for each. Both gates pass over the WHOLE
// message for both edits, so this is where a true reason used to land on the
// wrong edit — the only failure the 51 graded records showed. Scoped to the
// paragraph that names the edit, each record carries its own reason, and the
// third edit is the case where two paragraphs claim it: the message never said
// which reason applies, so nothing may be written.
func TestHookPostScopesTheReasonToTheParagraphNamingTheEdit(t *testing.T) {
	dir, abs, _, _ := hookRepo(t)
	pending := filepath.Join(dir, storeDirName, pendingLogName)

	tr := transcriptSaying(t, "Real bug: the counter was off by one, so `unique_line_20` was skipped entirely. Renumbering from the top fixes it.\n\nSeparately, `unique_line_21` was wrong for its own reason: the cap was hard-coded. Reading it from the config instead.")
	runHookPost(t, postPayload(t, abs, tr, "var unique_line_20 = 20"))
	runHookPost(t, postPayload(t, abs, tr, "var unique_line_21 = 21"))

	rs, err := Load(pending)
	if err != nil || len(rs) != 2 {
		t.Fatalf("pending = %d records, want one per edit (%v)", len(rs), err)
	}
	for _, r := range rs {
		mine, theirs := "unique_line_20", "unique_line_21"
		switch r.Start {
		case 20:
		case 21:
			mine, theirs = theirs, mine
		default:
			t.Fatalf("record anchored at %d, want 20 or 21", r.Start)
		}
		if blob := r.Decision + " " + r.Why; !strings.Contains(blob, mine) || strings.Contains(blob, theirs) {
			t.Errorf("the record at line %d reads %q / %q — that is the other edit's reason", r.Start, r.Decision, r.Why)
		}
	}

	both := transcriptSaying(t, "Real bug: `unique_line_22` was never reached. The guard above returns first.\n\nAlso wrong: `unique_line_22` had the cap hard-coded. Reading it from the config instead.")
	runHookPost(t, postPayload(t, abs, both, "var unique_line_22 = 22"))
	if again, err := Load(pending); err != nil || len(again) != 2 {
		t.Errorf("pending = %d, want 2 — two paragraphs claimed one edit and one of them was recorded anyway", len(again))
	}
}

func TestHookPostWritesPendingNotRecords(t *testing.T) {
	dir, abs, _, _ := hookRepo(t)
	store := filepath.Join(dir, storeDirName, recordsFileName)
	pending := filepath.Join(dir, storeDirName, pendingLogName)
	before, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}

	tr := transcriptSaying(t, "Real bug: the counter was off by one, so `unique_line_20` was skipped entirely. Renumbering from the top fixes it.")
	runHookPost(t, postPayload(t, abs, tr, "var unique_line_20 = 20"))

	if after, err := Load(store); err != nil || len(after) != len(before) {
		t.Fatalf("shared store grew from %d to %d", len(before), len(after))
	}
	prs, err := Load(pending)
	if err != nil || len(prs) != 1 {
		t.Fatalf("pending = %d, want 1 (%v)", len(prs), err)
	}
	if prs[0].Author != authorAgent || prs[0].Source != "capture" {
		t.Errorf("pending record = %+v", prs[0])
	}
}

func TestLookupAndCheckIgnorePending(t *testing.T) {
	dir, abs, _, _ := hookRepo(t)
	tr := transcriptSaying(t, "Real bug: the counter was off by one, so `unique_line_20` was skipped entirely. Renumbering from the top fixes it.")
	runHookPost(t, postPayload(t, abs, tr, "var unique_line_20 = 20"))

	prs, err := Load(filepath.Join(dir, storeDirName, pendingLogName))
	if err != nil || len(prs) != 1 {
		t.Fatalf("need a pending record, got %d (%v)", len(prs), err)
	}
	decision := prs[0].Decision

	store, root, ok := FindStore(abs)
	if !ok {
		t.Fatal("store vanished")
	}
	rs, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range Match(root, rs, Rel(root, abs), 0) {
		if r.ID == prs[0].ID || r.Decision == decision {
			t.Fatal("lookup surfaced a pending record")
		}
	}

	out := runHookPre(t, hookJSON(t, "s1", abs, "var unique_line_20 = 20", "var unique_line_20 = 21", false))
	if strings.Contains(out, decision) {
		t.Errorf("PreToolUse hook surfaced pending: %q", out)
	}

	findings := inspect(rs, map[string]bool{prs[0].ID: true}, Rel(root, abs),
		fileLines(abs), fileLines(abs), []lineSpan{{20, 20}})
	for _, f := range findings {
		if f.r.ID == prs[0].ID {
			t.Fatal("check reported a pending record")
		}
	}
}

func TestHookPostDedupSeesPending(t *testing.T) {
	dir, abs, _, _ := hookRepo(t)
	pending := filepath.Join(dir, storeDirName, pendingLogName)
	tr := transcriptSaying(t, "Real bug: the counter was off by one, so `unique_line_20` was skipped entirely. Renumbering from the top fixes it.")
	runHookPost(t, postPayload(t, abs, tr, "var unique_line_20 = 20"))
	runHookPost(t, postPayload(t, abs, tr, "var unique_line_20 = 20"))
	prs, err := Load(pending)
	if err != nil || len(prs) != 1 {
		t.Fatalf("dedup missed pending: %d (%v)", len(prs), err)
	}
}

// Narration is the majority of what an agent says, so the quiet path is the one
// that runs most often. Writing nothing has to mean writing nothing — no record,
// no output, no store created.
func TestHookPostWritesNothingForNarration(t *testing.T) {
	dir, abs, _, _ := hookRepo(t)
	store := filepath.Join(dir, storeDirName, "records.jsonl")
	before, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}

	tr := transcriptSaying(t, "Now the onboarding flow — `unique_line_20`, and the callers that use it.")
	if out := runHookPost(t, postPayload(t, abs, tr, "var unique_line_20 = 20")); out != "" {
		t.Errorf("hook spoke on a narration turn: %q", out)
	}
	if after, err := Load(store); err != nil || len(after) != len(before) {
		t.Errorf("store grew from %d to %d on narration", len(before), len(after))
	}
}

func TestHookPostRefusesEnvValue(t *testing.T) {
	dir, abs, _, _ := hookRepo(t)
	if err := os.WriteFile(filepath.Join(dir, ".env"),
		[]byte("API_TOKEN=whence-test-token-9f2a1c\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	pending := filepath.Join(dir, storeDirName, pendingLogName)
	store := filepath.Join(dir, storeDirName, recordsFileName)
	before, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}

	tr := transcriptSaying(t, "Real bug: leaked whence-test-token-9f2a1c into `unique_line_20`. Never write the token down.")
	if out := runHookPost(t, postPayload(t, abs, tr, "var unique_line_20 = 20")); out != "" {
		t.Errorf("hook spoke after refusing a secret: %q", out)
	}
	if after, err := Load(store); err != nil || len(after) != len(before) {
		t.Errorf("shared store grew on a secret-bearing write")
	}
	if prs, err := Load(pending); err != nil || len(prs) != 0 {
		t.Errorf("pending grew on a secret-bearing write: %d (%v)", len(prs), err)
	}
}
