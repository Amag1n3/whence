package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdir moves into dir for the duration of a test. Authoring resolves the store
// from the working directory when none exists yet, so these tests have to be
// somewhere disposable.
func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func writeFile(t *testing.T, dir, name string, lines []string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name),
		[]byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A record whose hash is computed wrong is born orphaned, and nothing in the
// tool would report that as an error — it would just quietly never match again.
// So the round trip is the check that matters: add it, read it back, and it must
// resolve as exact against the file it came from.
func TestAddRoundTripsToAnExactAnchor(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)

	got, store, err := add("session.go", 2, 4, "namespace session keys", "the staff dashboard reads them", "manual", authorHuman, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Anchor.State != StateExact {
		t.Errorf("a record must anchor exactly to the file it was just read from, got %q", got.Anchor.State)
	}
	if len(got.Lines) != 3 {
		t.Errorf("3 significant lines in 2-4, got %d hashes", len(got.Lines))
	}
	if got.ID == "" || got.Date == "" {
		t.Error("add should fill in the id and the date")
	}

	rs, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 1 {
		t.Fatalf("store should hold 1 record, got %d", len(rs))
	}
	// And it survives the JSON round trip, which is where a hash gets mangled.
	if a := resolveAnchor(fileLines(filepath.Join(dir, "session.go")), rs[0]); a.State != StateExact {
		t.Errorf("record reloaded from disk should still anchor exactly, got %q", a.State)
	}
}

func TestAddRefusesASpanItCannotAnchor(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)

	if _, _, err := add("session.go", 2, 900, "past the end", "", "manual", authorHuman, nil); err == nil {
		t.Error("a range past the end of the file must be refused, not anchored to nothing")
	}
	if _, _, err := add("nope.go", 1, 2, "no such file", "", "manual", authorHuman, nil); err == nil {
		t.Error("an unreadable file must be refused")
	}
}

// --- backfill -----------------------------------------------------------

var noted = []string{
	"package x",
	"",
	"// ponytail: global lock, not per-account. Throughput is 20 writes a day and",
	"// a per-account map is a cache-invalidation problem nobody asked for.",
	"func transfer(a, b Account) {",
	"	mu.Lock()",
	"}",
}

func TestHarvestSpansTheNoteAndTheCodeBelowIt(t *testing.T) {
	got := harvest(noted)
	if len(got) != 1 {
		t.Fatalf("one note in this file, got %d", len(got))
	}
	// Lines 3-4 are the comment; 5 is the declaration it is about. Anchoring to
	// the comment alone would be circular — the record would live exactly as
	// long as the comment and say nothing about the code.
	if got[0].start != 3 || got[0].end != 5 {
		t.Errorf("span should be 3-5, got %d-%d", got[0].start, got[0].end)
	}
	if !strings.Contains(got[0].text, "cache-invalidation") {
		t.Errorf("continuation lines should be folded in, got %q", got[0].text)
	}
	if strings.Contains(got[0].text, "//") {
		t.Errorf("comment markers should be stripped, got %q", got[0].text)
	}

	decision, why := firstSentence(got[0].text)
	if decision != "global lock, not per-account." {
		t.Errorf("decision should be the first sentence, got %q", decision)
	}
	if !strings.HasPrefix(why, "Throughput") {
		t.Errorf("the rest is the why, got %q", why)
	}
}

// The marker is a literal string, so the code that defines it and any fixture
// that quotes it are both candidates for harvesting themselves. Records are
// committed and shared; a garbage one is not recoverable the way a missed one is.
func TestHarvestIgnoresTheMarkerOutsideAComment(t *testing.T) {
	decoys := []string{
		`	marker = "ponytail:"`,          // the constant defining it
		`		"// ponytail: fixture data",`, // a quoted note in a test
		`	if strings.Contains(s, "ponytail:") {`,
		"// a `ponytail:` comment IS a decision record — prose discussing the marker",
	}
	if got := harvest(decoys); len(got) != 0 {
		t.Errorf("the marker outside a comment is not a note, harvested %d: %+v", len(got), got)
	}
}

// The marker set, and the line it draws. `HACK:` and friends are admissions on
// their own; `NOTE:` and `TODO:` are mostly descriptive and only count when the
// note says why — which is what separates a decision from a task.
func TestHarvestMarkerSet(t *testing.T) {
	cases := []struct {
		name    string
		lines   []string
		want    bool
		wantSrc string
	}{
		{"hack needs no reason", []string{
			"// HACK: single global lock here.", "func f() {}"}, true, "HACK comment"},
		{"workaround needs no reason", []string{
			"// WORKAROUND: pin to v1.2.", "func f() {}"}, true, "WORKAROUND comment"},
		{"note with a reason is a decision", []string{
			"// NOTE: retry five times because the upstream 502s under load.",
			"func f() {}"}, true, "NOTE comment"},
		{"note without a reason is not", []string{
			"// NOTE: see the design doc.", "func f() {}"}, false, ""},
		{"bare todo is a task, not a decision", []string{
			"// TODO: fix this properly.", "func f() {}"}, false, ""},
		{"todo with an owner and a reason", []string{
			"// TODO(amogh): drop the shim since v2 ships in March.",
			"func f() {}"}, true, "TODO comment"},
		{"the reason may be on a continuation line", []string{
			"// NOTE: three retries, not one.",
			"// A single attempt fails the batch because the upstream 502s under load.",
			"func f() {}"}, true, "NOTE comment"},
		{"an ordinary comment is left alone", []string{
			"// transfer moves money between accounts.", "func f() {}"}, false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := harvest(c.lines)
			if c.want && len(got) != 1 {
				t.Fatalf("want it harvested, got %d", len(got))
			}
			if !c.want {
				if len(got) != 0 {
					t.Fatalf("want it skipped, harvested %q", got[0].text)
				}
				return
			}
			if got[0].src != c.wantSrc {
				t.Errorf("source = %q, want %q", got[0].src, c.wantSrc)
			}
		})
	}
}

// Re-running backfill must not duplicate. It is the kind of command someone
// runs twice because they are not sure whether the first one worked.
// A note is admitted for giving a reason, so the reason has to reach the field
// that means it. Found by running backfill on prometheus/prometheus, where a
// one-sentence TODO qualified on "to avoid" and then stored an empty why.
func TestBackfillSplitsAOneSentenceNoteAtItsReason(t *testing.T) {
	for _, c := range []struct{ in, decision, why string }{
		// The case from prometheus. Both halves, one sentence, no boundary.
		{
			"Change to 0 in the interface for set check to avoid pointer mangling",
			"Change to 0 in the interface for set check",
			"to avoid pointer mangling",
		},
		// A sentence boundary is the stronger signal and still wins.
		{
			"Limit concurrency here. Parallel tests exhaust mmaps.",
			"Limit concurrency here.",
			"Parallel tests exhaust mmaps.",
		},
		// Trailing punctuation belongs to neither half.
		{
			"Retry twice, because the upstream 502s",
			"Retry twice",
			"because the upstream 502s",
		},
		// Opening on the reason would leave no decision at all.
		{
			"since the upstream 502s we retry twice",
			"since the upstream 502s we retry twice",
			"",
		},
		// "reason" as an ordinary noun must not cut a note into fragments — this
		// is why a decision half has to be more than a word or two.
		{
			"the reason code is duplicated",
			"the reason code is duplicated",
			"",
		},
	} {
		d, w := firstSentence(c.in)
		if d != c.decision || w != c.why {
			t.Errorf("firstSentence(%q)\n got decision %q why %q\nwant decision %q why %q",
				c.in, d, w, c.decision, c.why)
		}
	}
}

func TestBackfillIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "bank.go", noted)

	backfillCmd([]string{"."})
	store := filepath.Join(dir, storeDirName, recordsFileName)
	first, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 {
		t.Fatalf("should have harvested 1 note, got %d", len(first))
	}
	if first[0].Source != "ponytail comment" {
		t.Errorf("source should mark where it came from, got %q", first[0].Source)
	}

	backfillCmd([]string{"."})
	second, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 {
		t.Errorf("a second run must add nothing, got %d records", len(second))
	}
}

func TestSplitSpan(t *testing.T) {
	cases := []struct {
		in         string
		file       string
		start, end int
	}{
		{"src/a.go:142-148", "src/a.go", 142, 148},
		{"src/a.go:142", "src/a.go", 142, 142},
		{"src/a.go", "src/a.go", 0, 0},
		{"src/a.go:x-y", "src/a.go:x-y", 0, 0},
	}
	for _, c := range cases {
		f, s, e := splitSpan(c.in)
		if f != c.file || s != c.start || e != c.end {
			t.Errorf("splitSpan(%q) = (%q, %d, %d), want (%q, %d, %d)",
				c.in, f, s, e, c.file, c.start, c.end)
		}
	}
}

// --- evidence -----------------------------------------------------------

// A pointer at code gets its own anchor, which is the entire reason evidence is
// a pointer and not a copied snippet: when the cited code goes, whence can say
// the grounds are gone without anyone checking.
func TestEvidencePointingAtCodeGetsItsOwnAnchor(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)
	writeFile(t, dir, "dashboard.go", []string{
		"func render(u User) {",
		"\tread(\"CHECKOUT_userToken\")",
		"\tread(\"CHECKOUT_role\")",
		"}",
	})

	got, store, err := add("session.go", 2, 4, "namespace session keys",
		"the dashboard reads them", "code review", authorHuman, []string{"dashboard.go:2-3"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Evidence) != 1 {
		t.Fatalf("want 1 piece of evidence, got %d", len(got.Evidence))
	}
	if !got.Evidence[0].anchored() {
		t.Fatal("a file:line pointer should have been anchored")
	}
	if len(got.Grounds) != 1 || got.Grounds[0].Anchor.State != StateExact {
		t.Fatalf("grounds should resolve exactly against the file just read, got %+v", got.Grounds)
	}

	// Now delete the code the record leaned on. The record itself is untouched
	// and still anchors — but its grounds are gone, and that has to show.
	writeFile(t, dir, "dashboard.go", []string{"func render(u User) {", "\trenderV2(u)", "}"})
	rs, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	g := resolveEvidence(dir, rs[0])
	if len(g) != 1 || g[0].Anchor.State != StateOrphaned {
		t.Fatalf("grounds should have gone orphaned, got %+v", g)
	}
	if !strings.Contains(ground(g[0]), "GONE") {
		t.Errorf("a reader must be told the grounds are gone, got %q", ground(g[0]))
	}
	// And the record's own anchor is unaffected — the two rot independently.
	if a := resolveAnchor(fileLines(filepath.Join(dir, "session.go")), rs[0]); a.State != StateExact {
		t.Errorf("the record's own anchor should be untouched, got %q", a.State)
	}
}

// Re-pointing rotted evidence must not go through rm, because rm writes to the
// retraction log — the one number that measures how often this store is wrong.
// Fixing a stale pointer is bookkeeping, and bookkeeping in that log destroys it.
func TestRegroundRepointsEvidenceWithoutRetracting(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)
	writeFile(t, dir, "dashboard.go", []string{
		"func render(u User) {",
		"\tread(\"CHECKOUT_userToken\")",
		"}",
	})

	if _, _, err := add("session.go", 2, 4, "namespace session keys",
		"the dashboard reads them", "code review", authorHuman,
		[]string{"dashboard.go:2"}); err != nil {
		t.Fatal(err)
	}

	// The cited line moves into a new helper. The grounds are still real, they
	// are just somewhere else now — exactly the case that must not read as a
	// record having been wrong.
	writeFile(t, dir, "dashboard.go", []string{
		"func render(u User) {",
		"\treadKeys(u)",
		"}",
		"func readKeys(u User) {",
		"\tread(\"CHECKOUT_userToken\")",
		"}",
	})

	rs, err := Load(filepath.Join(dir, storeDirName, recordsFileName))
	if err != nil || len(rs) != 1 {
		t.Fatalf("expected one record, got %d (%v)", len(rs), err)
	}
	id := rs[0].ID

	got, _, err := reground(id, []string{"dashboard.go:5"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Grounds) != 1 || got.Grounds[0].Anchor.State != StateExact {
		t.Fatalf("re-pointed grounds should anchor exactly, got %+v", got.Grounds)
	}
	if got.ID != id {
		t.Errorf("the record keeps its identity, got %q want %q", got.ID, id)
	}
	if _, err := os.Stat(filepath.Join(dir, storeDirName, retractedLogName)); err == nil {
		t.Error("regrounding wrote to the retraction log; that log counts records that were WRONG")
	}

	// The claim itself is untouched — only what backs it up moved.
	after, err := Load(filepath.Join(dir, storeDirName, recordsFileName))
	if err != nil || len(after) != 1 {
		t.Fatalf("the record must still be there, got %d (%v)", len(after), err)
	}
	if after[0].Decision != "namespace session keys" || after[0].Start != 2 {
		t.Errorf("reground touched the claim, not just the grounds: %+v", after[0])
	}
}

// The same argument as reground, for the other half of a record. A block gets
// rewritten in place, the record about it is still exactly right, and the only
// fix used to be rm plus add — which files a live decision in the log that
// counts records that turned out to be WRONG.
func TestReanchorRepointsAClaimThatSurvivedARewrite(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)

	if _, _, err := add("session.go", 2, 4, "namespace session keys",
		"the staff dashboard reads them", "code review", authorHuman, nil); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(dir, storeDirName, recordsFileName)
	rs, err := Load(store)
	if err != nil || len(rs) != 1 {
		t.Fatalf("expected one record, got %d (%v)", len(rs), err)
	}
	id, was := rs[0].ID, rs[0].Lines

	// One line of the block is rewritten. The decision — namespace all three —
	// is untouched by that, and still true about the code that replaced it.
	rewritten := append([]string(nil), block...)
	rewritten[3] = `	store.Set("CHECKOUT_role", roleOf(s))`
	writeFile(t, dir, "session.go", rewritten)

	if a := resolveAnchor(fileLines(filepath.Join(dir, "session.go")), rs[0]); a.State != StateWeak {
		t.Fatalf("the rewrite should leave the record weak, got %q — this test is not testing what it thinks", a.State)
	}

	got, _, err := reanchor(id, "session.go:2-4")
	if err != nil {
		t.Fatal(err)
	}
	if got.Anchor.State != StateExact {
		t.Errorf("a re-pointed record must anchor exactly to the code it was just read from, got %q", got.Anchor.State)
	}
	if got.ID != id {
		t.Errorf("the record keeps its identity, got %q want %q", got.ID, id)
	}
	if _, err := os.Stat(filepath.Join(dir, storeDirName, retractedLogName)); err == nil {
		t.Error("reanchoring wrote to the retraction log; that log counts records that were WRONG")
	}

	after, err := Load(store)
	if err != nil || len(after) != 1 {
		t.Fatalf("the record must still be there, got %d (%v)", len(after), err)
	}
	// Only the anchor moves. The recorded range is where the decision was made,
	// and how far the record has travelled since is information.
	if after[0].Start != 2 || after[0].End != 4 || after[0].Decision != "namespace session keys" {
		t.Errorf("reanchor touched more than the anchor: %+v", after[0])
	}
	if sameSeq(after[0].Lines, was) {
		t.Error("the hashes did not change, so nothing was re-pointed")
	}

	// The span is the human's, never inherited: a degraded record's window is a
	// guess, and re-hashing a guess stores it as a certainty.
	if _, _, err := reanchor(id, "session.go"); err == nil {
		t.Error("reanchor must ask which lines the decision is about rather than infer them")
	}
	// A decision whose code moved to another file is a different record, not this
	// one pointed sideways — its recorded range names lines in the old file.
	writeFile(t, dir, "elsewhere.go", block)
	if _, _, err := reanchor(id, "elsewhere.go:2-4"); err == nil {
		t.Error("reanchor must refuse a span in another file")
	}
}

// The citogenesis rule, enforced rather than documented: a record may never be
// grounded in another record. That link is what lets one wrong record make the
// next one look credible (DECISIONS §17).
func TestEvidenceCannotCiteTheRecordStore(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)

	for _, ref := range []string{".whence/records.json", ".whence/records.json:4-9"} {
		if _, _, err := add("session.go", 2, 4, "circular", "", "manual", authorHuman, []string{ref}); err == nil {
			t.Errorf("evidence %q cites the store and must be refused", ref)
		}
	}
}

// Anything that is not a file keeps working as plain text — a command, a commit,
// a link. Most evidence will be this, and it must not need a special form.
func TestEvidenceThatIsNotCodeIsKeptVerbatim(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)

	refs := []string{"go test ./...", "9f2a1c3", "https://github.com/x/y/pull/42"}
	got, _, err := add("session.go", 2, 4, "several kinds of grounds", "", "manual", authorHuman, refs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Evidence) != 3 {
		t.Fatalf("want 3, got %d", len(got.Evidence))
	}
	for i, e := range got.Evidence {
		if e.Ref != refs[i] {
			t.Errorf("evidence %d should be kept verbatim: got %q want %q", i, e.Ref, refs[i])
		}
		if e.anchored() {
			t.Errorf("evidence %q is not a place in the code and must not be anchored", e.Ref)
		}
	}
}

// A typo'd file reference must fail loudly. Stored as plain text it would look
// perfectly fine while silently buying none of the rot detection it was for.
func TestEvidenceRefusesABrokenFileReference(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)

	if _, _, err := add("session.go", 2, 4, "typo", "", "manual", authorHuman, []string{"dashbord.go:2-3"}); err == nil {
		t.Error("a file:line pointer at a file that does not exist must be refused")
	}
	if _, _, err := add("session.go", 2, 4, "past end", "", "manual", authorHuman, []string{"session.go:400-410"}); err == nil {
		t.Error("a file:line pointer past the end of the file must be refused")
	}
}

// --- who wrote it, and has anyone checked ------------------------------

// A human writing a record IS the confirmation; an agent's record waits for one.
// That asymmetry is the whole point of tracking the author, so it gets a test.
func TestAgentRecordsStartUncheckedAndHumanOnesDoNot(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)

	human, _, err := add("session.go", 2, 4, "a human wrote this", "", "manual", authorHuman, nil)
	if err != nil {
		t.Fatal(err)
	}
	if human.unchecked() {
		t.Error("writing a record by hand is itself the confirmation")
	}
	if human.Verified == "" {
		t.Error("a human record should be stamped confirmed at birth")
	}
	if trust(human.Record) != "" {
		t.Errorf("a human record needs no warning, got %q", trust(human.Record))
	}

	agent, _, err := add("session.go", 2, 4, "an agent wrote this", "", "capture", authorAgent, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !agent.unchecked() {
		t.Error("an agent's record has had no human attention and must say so")
	}
	if agent.Verified != "" {
		t.Error("an agent record must not confirm itself")
	}
	if !strings.Contains(trust(agent.Record), "UNCHECKED") {
		t.Errorf("an agent record must be flagged, got %q", trust(agent.Record))
	}
}

// Records written before the author field existed have no author. They were all
// written by a human by hand, so they must not suddenly read as unchecked.
func TestRecordsPredatingTheAuthorFieldAreNotFlagged(t *testing.T) {
	old := Record{ID: "old", File: "a.go", Start: 1, End: 2, Decision: "from before"}
	if old.unchecked() {
		t.Error("an empty author means human, and must not read as unchecked")
	}
}

// Deleting a record silently makes it impossible to ever answer "how often is
// this tool wrong?" — which is the hole §8's counter cannot see (DECISIONS §17.6).
func TestRemovingARecordLeavesATrace(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)

	rec, store, err := add("session.go", 2, 4, "wrong on purpose", "", "manual", authorHuman, nil)
	if err != nil {
		t.Fatal(err)
	}
	rmCmd([]string{rec.ID, "-w", "the dashboard never read those keys"})

	rs, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 0 {
		t.Fatalf("record should be gone from the store, %d left", len(rs))
	}

	log, err := os.ReadFile(filepath.Join(dir, storeDirName, retractedLogName))
	if err != nil {
		t.Fatalf("a retraction must be logged: %v", err)
	}
	for _, want := range []string{rec.ID, "wrong on purpose", "the dashboard never read those keys"} {
		if !strings.Contains(string(log), want) {
			t.Errorf("retraction log missing %q, got %s", want, log)
		}
	}
}

func TestConfirmMarksAnAgentRecordChecked(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)

	rec, store, err := add("session.go", 2, 4, "needs a human", "", "capture", authorAgent, nil)
	if err != nil {
		t.Fatal(err)
	}
	confirmCmd([]string{rec.ID})

	rs, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 1 || rs[0].Verified == "" {
		t.Fatalf("confirm should stamp a date, got %+v", rs)
	}
	if rs[0].unchecked() {
		t.Error("a confirmed record is no longer unchecked")
	}
}
