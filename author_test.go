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
// their own; `NOTE:` and friends describe current code only when the note says
// why. TODO and FIXME are proposals, even when they carry a reason.
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
		{"todo with an owner and a reason is still a task", []string{
			"// TODO(amogh): drop the shim since v2 ships in March.",
			"func f() {}"}, false, ""},
		{"the reason may be on a continuation line", []string{
			"// NOTE: three retries, not one.",
			"// A single attempt fails the batch because the upstream 502s under load.",
			"func f() {}"}, true, "NOTE comment"},
		{"an ordinary comment is left alone", []string{
			"// transfer moves money between accounts.", "func f() {}"}, false, ""},
		// CPython uses XXX as its task marker, so it is gated like NOTE: —
		// these are verbatim from the 2026-08-16 corpus test.
		{"xxx as an open question is a task, not a decision", []string{
			"# XXX: is this test needed?", "def f(): pass"}, false, ""},
		{"xxx as an instruction is a task, not a decision", []string{
			"# XXX: implement", "def f(): pass"}, false, ""},
		{"xxx with a reason is still a decision", []string{
			"# XXX: seek() is bypassed because the buffered reader owns the position.",
			"def f(): pass"}, true, "XXX comment"},
		// Round two of the corpus test: a question is not a decision, even when
		// a reason word admits it — "reason" is doing the admitting here.
		{"xxx as a question is rejected despite the reason word", []string{
			"# XXX: is there any reason to assume differently?",
			"def f(): pass"}, false, ""},
		// But a question after the decision sentence is rhetoric, not a task:
		// the headline is the first sentence, and it does not end on the "?".
		{"a decision followed by a question is still a decision", []string{
			"// HACK: we cannot use the fast path here. Why would the lock be free?",
			"func f() {}"}, true, "HACK comment"},
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

func TestBackfillRejectsFlagsAfterTheDirectory(t *testing.T) {
	if err := validateBackfillArgs([]string{".", "--yes"}); err == nil {
		t.Fatal("a flag after the directory must be rejected rather than ignored")
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
		// Two words of decision is a fragment, not a headline — since the
		// 2026-08-16 corpus test a cut this thin is refused and the note stays
		// whole.
		{
			"Retry twice, because the upstream 502s",
			"Retry twice, because the upstream 502s",
			"",
		},
		// Trailing punctuation belongs to neither half. The thin-split refusal
		// took this assertion off the case above, so it lives here on a
		// decision long enough to survive the cut.
		{
			"Retry the upstream call at most twice, because the gateway 502s intermittently",
			"Retry the upstream call at most twice",
			"because the gateway 502s intermittently",
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
		// From k8s pkg/apis/scheduling/types.go: cutting at "to avoid" stored
		// the decision as "In order" with the whole rule in why.
		{
			"In order to avoid conflict of names, all the names must start with SystemPriorityClassPrefix.",
			"In order to avoid conflict of names, all the names must start with SystemPriorityClassPrefix.",
			"",
		},
		// From vscode hrtime.ts: five words of decision is still a fragment.
		{
			"This check is added probably because it's missed without strictFunctionTypes on",
			"This check is added probably because it's missed without strictFunctionTypes on",
			"",
		},
		// From k8s kubemanager.go: cutting at "otherwise" left a one-word why.
		{
			"Return true when the flag is set and false otherwise.",
			"Return true when the flag is set and false otherwise.",
			"",
		},
		// From rust zerocopy/src/macros.rs, corpus round two: "rather than"
		// sits inside the parenthetical, so it is skipped and the cut lands at
		// "because" — the decision keeps the whole macro statement instead of
		// being cut to "This must be a macro (".
		{
			"This must be a macro (rather than a function with trait bounds) because there's no way, in a generic context, to enforce that two types have the same size",
			"This must be a macro (rather than a function with trait bounds)",
			"because there's no way, in a generic context, to enforce that two types have the same size",
		},
		// The round-two word-count fix, proven on its own: "()" carries no
		// letter, so five real words plus a stray bracket is still under the
		// floor — strings.Fields alone would have scored it six and split.
		{
			"This must be a macro () because there's no way to enforce it",
			"This must be a macro () because there's no way to enforce it",
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

// A generated file's comments belong to its generator, not to the repo —
// 20 of k8s's 92 corpus-test candidates were one identical protoc line.
func TestBackfillSkipsGeneratedFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api_grpc.pb.go", []string{
		"// Code generated by protoc-gen-go. DO NOT EDIT.",
		"",
		"package api",
		"",
		"// NOTE: this should be embedded by value because the generator says so.",
		"type T struct{}",
	})
	if lines := readSource(filepath.Join(dir, "api_grpc.pb.go")); lines != nil {
		t.Fatalf("a generated file must yield nothing to harvest, got %d lines", len(lines))
	}
}

// An expected-output fixture is the same defect as generated Go in an
// extension nobody would guess: Rust's lint tests keep a machine-written
// .fixed twin beside each .rs fixture, and 22 round-two corpus candidates
// arrived as identical pairs. The test framework rewrites the file on every
// run, so the anchor is rewritten with it.
func TestBackfillSkipsExpectedOutputFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "lint.fixed", []string{
		"// NOTE: this should be derived because the compiler says so.",
		"fn main() {}",
	})
	if lines := readSource(filepath.Join(dir, "lint.fixed")); lines != nil {
		t.Fatalf("an expected-output fixture must yield nothing to harvest, got %d lines", len(lines))
	}
}

func TestBackfillIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "bank.go", noted)

	// The default is a dry run: it shows what it would store and writes
	// nothing, because the store is committed and shared (§7.2).
	backfillCmd([]string{"."})
	store := filepath.Join(dir, storeDirName, recordsFileName)
	if rs, err := Load(store); err != nil || len(rs) != 0 {
		t.Fatalf("a dry run must write nothing, got %d records (%v)", len(rs), err)
	}

	// --yes is the explicit opt-in that actually writes.
	backfillCmd([]string{"--yes", "."})
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

	backfillCmd([]string{"--yes", "."})
	second, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 {
		t.Errorf("a second run must add nothing, got %d records", len(second))
	}
}

// Round two of the corpus test found 72 repeated texts in linux and 36 in
// rust — "MUST NOT be called from interrupt context" above sixteen separate
// drivers, each one true. The dry run groups identical decision texts so a
// reviewer approves one sentence once; a text found once prints exactly as it
// always has.
func TestGroupDryRuns(t *testing.T) {
	cases := []struct {
		name         string
		finds        []dryFind
		want         []string
		wantDistinct int
	}{
		{"a single text prints exactly as it always has", []dryFind{
			{rel: "drivers/net/eth.c", start: 40, end: 42, decision: "MUST NOT be called from interrupt context"},
		}, []string{
			"  would add  drivers/net/eth.c:40-42  MUST NOT be called from interrupt context",
		}, 1},
		{"two locations of one text collapse into a group of two", []dryFind{
			{rel: "drivers/ata/ahci.c", start: 100, end: 102, decision: "MUST NOT be called from interrupt context"},
			{rel: "drivers/scsi/sd.c", start: 55, end: 57, decision: "MUST NOT be called from interrupt context"},
		}, []string{
			"  would add  ×2  MUST NOT be called from interrupt context",
			"      drivers/ata/ahci.c:100-102",
			"      drivers/scsi/sd.c:55-57",
		}, 1},
		{"three texts, only the middle one repeats", []dryFind{
			{rel: "a.go", start: 1, end: 2, decision: "first decision"},
			{rel: "b.go", start: 3, end: 4, decision: "second decision"},
			{rel: "c.go", start: 5, end: 6, decision: "second decision"},
			{rel: "d.go", start: 7, end: 8, decision: "third decision"},
		}, []string{
			"  would add  a.go:1-2  first decision",
			"  would add  ×2  second decision",
			"      b.go:3-4",
			"      c.go:5-6",
			"  would add  d.go:7-8  third decision",
		}, 3},
		{"groups keep first-encounter order", []dryFind{
			{rel: "z.go", start: 1, end: 2, decision: "met first"},
			{rel: "y.go", start: 3, end: 4, decision: "met second"},
			{rel: "x.go", start: 5, end: 6, decision: "met first"},
			{rel: "w.go", start: 7, end: 8, decision: "met second"},
		}, []string{
			"  would add  ×2  met first",
			"      z.go:1-2",
			"      x.go:5-6",
			"  would add  ×2  met second",
			"      y.go:3-4",
			"      w.go:7-8",
		}, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, distinct := groupDryRuns(c.finds)
			if !sameSeq(got, c.want) {
				t.Errorf("lines:\n got %q\nwant %q", got, c.want)
			}
			if distinct != c.wantDistinct {
				t.Errorf("distinct = %d, want %d", distinct, c.wantDistinct)
			}
		})
	}
}

// A secret-shaped string in a harvested comment must never reach a committed
// store. The refusal is in add, so it covers backfill and `whence add` alike.
func TestSecretShapeRefused(t *testing.T) {
	for _, text := range []string{
		"hardcode key sk-abc123 because vault is down",
		"token ghp_XYZ for the deploy",
		"AWS AKIAIOSFODNN7EXAMPLE left in",
		"-----BEGIN PRIVATE KEY----- oops",
		"bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.sig",
	} {
		if !secretShape(text) {
			t.Errorf("should refuse a secret shape: %q", text)
		}
	}
	for _, text := range []string{
		"cache the lookup because the upstream 502s",
		"norm() here MUST stay identical to the worker",
	} {
		if secretShape(text) {
			t.Errorf("a legitimate decision must not be refused: %q", text)
		}
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

// A record's File comes from a pulled store, so a path escaping the repo root
// must never be read — the intact/orphaned verdict would be a one-bit hash
// oracle on any file on the machine. reground resolves the record for display,
// which makes it the probe: the outside file is placed exactly where the crafted
// path points, so if the guard is ever removed the record reads exact again.
func TestRegroundNeverReadsARecordFileOutsideTheRoot(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	site := filepath.Join(dir, "site")
	for _, d := range []string{repo, site} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	chdir(t, repo)
	writeFile(t, repo, "session.go", block)
	if _, _, err := add("session.go", 2, 4, "namespace session keys",
		"the dashboard reads them", "manual", authorHuman, nil); err != nil {
		t.Fatal(err)
	}

	// The oracle: a file outside the repo whose contents exactly match the
	// record's anchor, at the exact path the crafted record names.
	writeFile(t, site, "oracle.go", block)
	store := filepath.Join(repo, storeDirName, recordsFileName)
	rs, err := Load(store)
	if err != nil || len(rs) != 1 {
		t.Fatalf("expected one record, got %d (%v)", len(rs), err)
	}
	rs[0].File = "../site/oracle.go"
	if err := save(store, rs); err != nil {
		t.Fatal(err)
	}

	got, _, err := reground(rs[0].ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Anchor.State != StateOrphaned {
		t.Fatalf("a crafted record path must stay unread, got %q at %d-%d",
			got.Anchor.State, got.Anchor.Start, got.Anchor.End)
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

	for _, ref := range []string{".whence/records.jsonl", ".whence/records.jsonl:4-9"} {
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

func TestSourceAgentImpliesAuthor(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)
	addCmd([]string{"session.go:2-4", "-d", "an agent wrote this", "-s", "agent"})
	rs, err := Load(filepath.Join(dir, storeDirName, recordsFileName))
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 1 {
		t.Fatalf("want 1 record, got %d", len(rs))
	}
	if rs[0].Author != authorAgent {
		t.Errorf("Author=%q, want %q", rs[0].Author, authorAgent)
	}
	if rs[0].Verified != "" {
		t.Errorf("must not self-verify, Verified=%q", rs[0].Verified)
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

func TestRemovingARecordKeepsItWhenRetractionLogCannotBeWritten(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)

	rec, store, err := add("session.go", 2, 4, "must stay for the dashboard", "the dashboard reads it", "manual", authorHuman, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, storeDirName, retractedLogName), 0o755); err != nil {
		t.Fatal(err)
	}

	rs, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := removeRecord(store, dir, rs, rec.ID, "the claim was wrong"); err == nil {
		t.Fatal("removal must fail when its retraction cannot be logged")
	}

	after, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 || after[0].ID != rec.ID {
		t.Fatalf("a failed retraction must keep the record, got %+v", after)
	}
}

// The first ". " in a note is not always a sentence break. A real note read
// "...might belong to other services (e.g. <names>), but since they are used
// directly..." and the cut landed inside the abbreviation, so the decision — the
// field an agent is shown first — ended mid-parenthesis and stated nothing.
// Fixtures here are paraphrases: whence's store is public and committed, and the
// notes that found this bug live in an employer repository.
func TestFirstSentenceDoesNotCutInsideAnAbbreviation(t *testing.T) {
	// A real sentence break exists later in the note, so refusing the `e.g.`
	// cut must not mean giving up on the note — it means scanning on.
	d, w := firstSentence("Skip the shared helpers (e.g. sessionStore.js, tokenCache.js) here. They assume a request context this worker has no access to.")
	if strings.HasSuffix(d, "e.g.") {
		t.Errorf("the decision was cut inside the abbreviation: %q", d)
	}
	if d != "Skip the shared helpers (e.g. sessionStore.js, tokenCache.js) here." {
		t.Errorf("decision should be the real first sentence, got %q", d)
	}
	if !strings.HasPrefix(w, "They assume") {
		t.Errorf("the why should be the second sentence, got %q", w)
	}

	// When the only ". " is the abbreviation's, there is no sentence boundary at
	// all — so this has to fall through to splitAtReason rather than return a
	// decision that says nothing. A note admitted for giving a reason must not
	// be stored with an empty why.
	d, w = firstSentence("Read the cap from the request (i.e. not the global default) because a tenant may raise it mid-session")
	if d == "" || w == "" {
		t.Errorf("should fall through to the reason split, got decision %q why %q", d, w)
	}
	if strings.HasSuffix(d, "i.e.") {
		t.Errorf("the decision was cut inside the abbreviation: %q", d)
	}

	// An initial is the same shape as an abbreviation and the same failure: a
	// decision of "J." is worse than no split, because it is confidently empty.
	d, w = firstSentence("J. Smith owns this mapping. Do not change it without asking him.")
	if d == "J." {
		t.Error(`the decision must not be an initial ("J.")`)
	}
	if d != "J. Smith owns this mapping." || !strings.HasPrefix(w, "Do not change") {
		t.Errorf("decision/why should split at the real break, got %q / %q", d, w)
	}

	// Ordinary two-sentence and inline-reason notes are unchanged by this: that
	// regression is already covered by TestHarvestSpansTheNoteAndTheCodeBelowIt
	// and TestBackfillSplitsAOneSentenceNoteAtItsReason, so it is not duplicated
	// here.
}

// reasonWords' own ponytail: comment names its trigger — widen only against real
// misses from a real repo. The trigger fired on 4 Aug: three notes, all genuine
// decisions, all rejected because they state no cause. They commit to something
// instead. Paraphrased fixtures, for the reason given above.
func TestHarvestAdmitsANoteThatCommitsWithoutGivingACause(t *testing.T) {
	commitments := [][]string{
		// The most valuable shape: a cross-file invariant. Editing one side and
		// not the other is exactly the regression whence exists to prevent, and
		// the gate was dropping it for having no causal connective.
		{"// NOTE: tagFor() here MUST stay identical to tagFor() in the ingest worker.", "func tagFor(s string) string {}"},
		// A deliberate omission. "for" is not a reason word, so this was dropped.
		{"// NOTE: signature verification intentionally omitted for the QA harness.", "func verify() bool {}"},
		// Commitment stated as a guarantee rather than a cause.
		{"// NOTE: keying on the immutable id ensures the path stays stable and maintains a readable change history.", "func pathFor(id string) string {}"},
	}
	for _, lines := range commitments {
		got := harvest(lines)
		if len(got) != 1 {
			t.Errorf("a note that commits is a decision, not harvested: %q", lines[0])
			continue
		}
		// Admitting it is only half the job — it has to reach the store with a
		// decision field that says something.
		if d, _ := firstSentence(got[0].text); d == "" {
			t.Errorf("admitted with an empty decision: %q", lines[0])
		}
	}
}

// The regression guard for the whole widening. These three were rejected on
// 4 Aug and the rejections were verified CORRECT — tasks wearing a decision's
// clothes. If one flips to admitted, the change has made a committed, shared
// store worse rather than better, which is the trade the marker set exists to
// refuse.
func TestHarvestStillRejectsTasksAfterTheCommitmentWidening(t *testing.T) {
	tasks := [][]string{
		// "required" reads as commitment and is not: this is why bare `must`,
		// `required`, `never`, `always` and `keep` stay out of the list.
		{"// TODO: Need to check if this is required", "func f() {}"},
		{"// TODO: someone to check if we are still using this", "func f() {}"},
		{"// TODO: SAVE users data to another file", "func f() {}"},
	}
	for _, lines := range tasks {
		if got := harvest(lines); len(got) != 0 {
			t.Errorf("a task is not a decision, but it was harvested: %q -> %q", lines[0], got[0].text)
		}
	}
}

// commitmentWords answer whether a note commits, never where its explanation
// starts — "MUST stay identical" marks no split point. A commitment-only note is
// therefore stored whole, and that is the right outcome: a note left whole is
// recoverable by hand, a note cut into fragments is a garbage record in a
// committed store.
func TestACommitmentOnlyNoteIsStoredWholeRatherThanSplit(t *testing.T) {
	got := harvest([]string{
		"// NOTE: tagFor() here MUST stay identical to tagFor() in the ingest worker",
		"func tagFor(s string) string {}",
	})
	if len(got) != 1 {
		t.Fatalf("should be admitted, harvested %d", len(got))
	}
	d, w := firstSentence(got[0].text)
	if w != "" {
		t.Errorf("a commitment word must never reach the splitter, got why %q", w)
	}
	if d != got[0].text {
		t.Errorf("the note should be stored whole, got decision %q", d)
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

func seedPending(t *testing.T, file string, start, end int, decision, why string) (Record, string, string) {
	t.Helper()
	abs, err := filepath.Abs(file)
	if err != nil {
		t.Fatal(err)
	}
	store, root, ok := FindStore(abs)
	if !ok {
		t.Fatal("no store")
	}
	r, _, err := makeRecord(abs, root, file, start, end, decision, why, "capture", authorAgent, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := save(pendingFile(root), []Record{r}); err != nil {
		t.Fatal(err)
	}
	return r, store, root
}

func TestConfirmPromotesPending(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)
	if _, _, err := add("session.go", 2, 4, "human", "why", "manual", authorHuman, nil); err != nil {
		t.Fatal(err)
	}

	r, store, root := seedPending(t, "session.go", 2, 4, "agent decision", "because the span is still there")
	rs, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := promotePending(root, store, rs, r.ID)
	if err != nil || !ok {
		t.Fatalf("promote: ok=%v err=%v", ok, err)
	}
	if got.Verified == "" {
		t.Fatal("confirm must stamp Verified")
	}

	prs, err := Load(pendingFile(root))
	if err != nil || len(prs) != 0 {
		t.Fatalf("pending still has %d records (%v)", len(prs), err)
	}
	after, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, x := range after {
		if x.ID == r.ID && x.Verified != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("promoted record missing from the shared store")
	}
}

func TestConfirmRefusesOrphanedPending(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)
	if _, _, err := add("session.go", 2, 4, "human", "why", "manual", authorHuman, nil); err != nil {
		t.Fatal(err)
	}

	r, store, root := seedPending(t, "session.go", 2, 4, "agent decision", "because the span is still there")
	writeFile(t, dir, "session.go", []string{"package gone", "func f() {}"})

	rs, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := promotePending(root, store, rs, r.ID); err == nil {
		t.Fatal("confirm must refuse a pending record whose anchor is gone")
	}

	prs, err := Load(pendingFile(root))
	if err != nil || len(prs) != 1 || prs[0].ID != r.ID {
		t.Fatalf("refused promote must leave pending untouched, got %+v (%v)", prs, err)
	}
	after, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range after {
		if x.ID == r.ID {
			t.Fatal("refused promote wrote the shared store")
		}
	}
}

func TestConfirmRefusesAlteredPending(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)
	if _, _, err := add("session.go", 2, 4, "human", "why", "manual", authorHuman, nil); err != nil {
		t.Fatal(err)
	}

	r, store, root := seedPending(t, "session.go", 2, 4, "agent decision", "because the span is still there")
	rewritten := append([]string(nil), block...)
	rewritten[3] = `	store.Set("CHECKOUT_role", roleOf(s))`
	writeFile(t, dir, "session.go", rewritten)

	rs, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	if a := resolveAnchor(fileLines(filepath.Join(dir, "session.go")), r); a.State != StateWeak {
		t.Fatalf("the rewrite should leave the pending record altered, got %q — this test is not testing what it thinks", a.State)
	}
	_, _, err = promotePending(root, store, rs, r.ID)
	if err == nil {
		t.Fatal("confirm must refuse a pending record whose block was altered")
	}
	if !strings.Contains(err.Error(), string(StateWeak)) {
		t.Errorf("refusal must name the resolved state, got %q", err)
	}
	if !strings.Contains(err.Error(), "whence add") || !strings.Contains(err.Error(), "whence rm "+r.ID) {
		t.Errorf("refusal must name whence add and whence rm, got %q", err)
	}
	if strings.Contains(err.Error(), "reanchor") {
		t.Errorf("refusal must not mention reanchor, got %q", err)
	}

	prs, err := Load(pendingFile(root))
	if err != nil || len(prs) != 1 || prs[0].ID != r.ID {
		t.Fatalf("refused promote must leave pending untouched, got %+v (%v)", prs, err)
	}
	after, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range after {
		if x.ID == r.ID {
			t.Fatal("refused promote wrote the shared store")
		}
	}
}

func TestConfirmPromotesMovedPending(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)
	if _, _, err := add("session.go", 2, 4, "human", "why", "manual", authorHuman, nil); err != nil {
		t.Fatal(err)
	}

	r, store, root := seedPending(t, "session.go", 2, 4, "agent decision", "because the span is still there")
	moved := append([]string{"store := sessionStore(ctx)", ""}, block...)
	writeFile(t, dir, "session.go", moved)

	rs, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := promotePending(root, store, rs, r.ID)
	if err != nil || !ok {
		t.Fatalf("promote: ok=%v err=%v", ok, err)
	}
	if got.Anchor.State != StateDrifted {
		t.Fatalf("a moved but identical block should promote as drifted, got %q", got.Anchor.State)
	}
	if got.Anchor.Start != 4 || got.Anchor.End != 6 {
		t.Fatalf("promoted record should land on the new lines 4-6, got %d-%d", got.Anchor.Start, got.Anchor.End)
	}

	prs, err := Load(pendingFile(root))
	if err != nil || len(prs) != 0 {
		t.Fatalf("pending still has %d records (%v)", len(prs), err)
	}
	after, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, x := range after {
		if x.ID == r.ID {
			found = true
			a := resolveAnchor(fileLines(filepath.Join(dir, "session.go")), x)
			if a.Start != 4 || a.End != 6 {
				t.Fatalf("store copy should still resolve to 4-6, got %d-%d", a.Start, a.End)
			}
		}
	}
	if !found {
		t.Fatal("promoted record missing from the shared store")
	}
}

func TestRmPendingDoesNotRetract(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)
	if _, _, err := add("session.go", 2, 4, "human", "why", "manual", authorHuman, nil); err != nil {
		t.Fatal(err)
	}

	r, _, root := seedPending(t, "session.go", 2, 4, "agent decision", "because the span is still there")
	gone, ok, err := dropPending(root, r.ID)
	if err != nil || !ok || gone.ID != r.ID {
		t.Fatalf("dropPending: ok=%v err=%v gone=%+v", ok, err, gone)
	}
	if _, err := os.Stat(filepath.Join(dir, storeDirName, retractedLogName)); err == nil {
		t.Fatal("rm on pending wrote retracted.jsonl")
	}
	prs, err := Load(pendingFile(root))
	if err != nil || len(prs) != 0 {
		t.Fatalf("pending still has %d (%v)", len(prs), err)
	}
}

func TestEnvValueRefused(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	writeFile(t, dir, "session.go", block)
	if err := os.WriteFile(filepath.Join(dir, ".env"),
		[]byte("API_TOKEN=whence-test-token-9f2a1c\nPORT=3000\nNODE_ENV=development\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := add("session.go", 2, 4,
		"hardcode whence-test-token-9f2a1c because vault is down", "",
		"manual", authorHuman, nil); err == nil {
		t.Fatal("a record containing a .env value must be refused")
	}

	// Ordinary values from the same file must not widen the net.
	if _, _, err := add("session.go", 2, 4,
		"namespace session keys", "the dashboard reads them",
		"manual", authorHuman, nil); err != nil {
		t.Fatalf("a record without a secret value was refused: %v", err)
	}
}
