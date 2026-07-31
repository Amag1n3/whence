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

	got, store, err := add("session.go", 2, 4, "namespace session keys", "the staff dashboard reads them", "manual", nil)
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

	if _, _, err := add("session.go", 2, 900, "past the end", "", "manual", nil); err == nil {
		t.Error("a range past the end of the file must be refused, not anchored to nothing")
	}
	if _, _, err := add("nope.go", 1, 2, "no such file", "", "manual", nil); err == nil {
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

// Re-running backfill must not duplicate. It is the kind of command someone
// runs twice because they are not sure whether the first one worked.
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
	if first[0].Source != backfillSource {
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
		"the dashboard reads them", "code review", []string{"dashboard.go:2-3"})
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

// The citogenesis rule, enforced rather than documented: a record may never be
// grounded in another record. That link is what lets one wrong record make the
// next one look credible (DECISIONS §17).
func TestEvidenceCannotCiteTheRecordStore(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)

	for _, ref := range []string{".whence/records.json", ".whence/records.json:4-9"} {
		if _, _, err := add("session.go", 2, 4, "circular", "", "manual", []string{ref}); err == nil {
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
	got, _, err := add("session.go", 2, 4, "several kinds of grounds", "", "manual", refs)
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

	if _, _, err := add("session.go", 2, 4, "typo", "", "manual", []string{"dashbord.go:2-3"}); err == nil {
		t.Error("a file:line pointer at a file that does not exist must be refused")
	}
	if _, _, err := add("session.go", 2, 4, "past end", "", "manual", []string{"session.go:400-410"}); err == nil {
		t.Error("a file:line pointer past the end of the file must be refused")
	}
}
