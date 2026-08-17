package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Anchoring is the moat and it is the part that fails silently. A wrong anchor
// does not error — it points at real code with a real line number and is simply
// about something else. So every state gets a case, and the orphan cases assert
// that no line number is claimed at all.

// The block a record is written about. Line 1 of the file is `func persist(`.
var block = []string{
	"func persist(s Session) {",
	`	store.Set("CHECKOUT_userToken", s.Token)`,
	`	store.Set("CHECKOUT_userID", s.ID)`,
	`	store.Set("CHECKOUT_role", s.Role)`,
	"}",
}

// rec anchors lines 2-4 of the given file content: the three Set calls.
func rec(lines []string) Record {
	return Record{ID: "b5", Date: "2026-07-27", File: "session.go", Start: 2, End: 4,
		Decision: "namespace session keys", Lines: hashSpan(lines[1:4])}
}

func TestAnchorExact(t *testing.T) {
	a := resolveAnchor(block, rec(block))
	if a.State != StateExact {
		t.Fatalf("unchanged file should anchor exactly, got %q", a.State)
	}
	if a.Start != 2 || a.End != 4 {
		t.Errorf("span should be unchanged, got %d-%d", a.Start, a.End)
	}
	if a.Integrity != 1 {
		t.Errorf("an exact match is fully intact, got %.2f", a.Integrity)
	}
}

func TestAnchorSurvivesReindentAndBlankLines(t *testing.T) {
	// Reformatting is not drift. Indentation changes and an inserted blank line
	// inside the block must still read as the same content.
	got := []string{
		block[0],
		"        " + strings.TrimSpace(block[1]),
		"",
		"  " + strings.TrimSpace(block[2]),
		strings.TrimSpace(block[3]),
		block[4],
	}
	if a := resolveAnchor(got, rec(block)); a.State != StateExact && a.State != StateDrifted {
		t.Errorf("reformatting must not lose the anchor, got %q at %.2f", a.State, a.Integrity)
	}
}

func TestAnchorFollowsDrift(t *testing.T) {
	// Two lines inserted above: every recorded line number is now wrong, but
	// the content hashes the same, so the anchor follows it down.
	moved := append([]string{"store := sessionStore(ctx)", ""}, block...)
	a := resolveAnchor(moved, rec(block))
	if a.State != StateDrifted {
		t.Fatalf("moved code should anchor by content hash, got %q", a.State)
	}
	if a.Start != 4 || a.End != 6 {
		t.Errorf("anchor should have followed to 4-6, got %d-%d", a.Start, a.End)
	}
	// Byte-identical content that merely moved. Charging it would contradict
	// the reason content drives the score at all.
	if a.Integrity != 1 {
		t.Errorf("a clean move is fully intact, got %.2f", a.Integrity)
	}
}

func TestAnchorDriftPrefersTheNearerCandidate(t *testing.T) {
	// The same block appears twice. The anchor must not jump to the far copy.
	dup := append(append([]string{}, block...), block...)
	r := rec(block)
	r.Start, r.End = 7, 9 // recorded against the second copy
	if a := resolveAnchor(dup, r); a.Start != 7 {
		t.Errorf("should hold the nearer copy at 7, got %d", a.Start)
	}
}

func TestAnchorDecaysWhenContentChanges(t *testing.T) {
	// One of the three lines is rewritten. Two thirds survive, which is above
	// the floor: recognisably the same block, no longer trustworthy.
	weakened := []string{
		block[0],
		block[1],
		`	store.Set("role", s.Role) // regression: dropped the namespace`,
		block[3],
		block[4],
	}
	a := resolveAnchor(weakened, rec(block))
	if a.State != StateWeak {
		t.Fatalf("a partly rewritten block should be weak, got %q at %.2f", a.State, a.Integrity)
	}
	if a.Integrity >= 1 {
		t.Errorf("a rewritten line must cost integrity, got %.2f", a.Integrity)
	}
	if a.Integrity < weakFloor {
		t.Errorf("2 of 3 lines surviving should clear the floor, got %.2f", a.Integrity)
	}
}

// The case that exposed the cap. A forty-line span picks up one extra argument
// on one line — the other thirty-nine are untouched — and the old code clamped
// the score to a ceiling, so it read identically to a block half rewritten.
// That is the distinction the number exists to make, so there is no ceiling now
// and this test is what fails if one comes back.
func TestAnchorScoresASmallEditFarAboveARewrite(t *testing.T) {
	var file []string
	for i := 0; i < 40; i++ {
		file = append(file, "step"+strconv.Itoa(i)+" := compute"+strconv.Itoa(i)+"(ctx)")
	}
	r := Record{ID: "wide", File: "a.go", Start: 1, End: 40, Lines: hashSpan(file)}

	// One line gains an argument. Thirty-nine of forty survive untouched.
	nudged := append([]string{}, file...)
	nudged[17] = "step17 := compute17(ctx, authorHuman)"
	small := resolveAnchor(nudged, r)
	if small.State != StateWeak {
		t.Fatalf("an edited line breaks the sequence, so this is altered: got %q", small.State)
	}
	if small.Integrity < 0.90 {
		t.Errorf("39 of 40 lines intact should score high, got %.2f", small.Integrity)
	}

	// Half the block rewritten, for contrast.
	gutted := append([]string{}, file...)
	for i := 0; i < 20; i++ {
		gutted[i] = "replaced" + strconv.Itoa(i) + " := somethingElse(ctx)"
	}
	big := resolveAnchor(gutted, r)
	if big.Integrity >= small.Integrity {
		t.Errorf("half a block gone (%.2f) must score below one edited line (%.2f)",
			big.Integrity, small.Integrity)
	}
}

func TestAnchorOrphansRatherThanGuess(t *testing.T) {
	// The block is refactored away entirely. The one thing that must not happen
	// is a confident line number.
	gone := []string{block[0], "	persistNamespaced(s)", block[4]}
	a := resolveAnchor(gone, rec(block))
	if a.State != StateOrphaned {
		t.Fatalf("a rewritten block should orphan, got %q at %.2f", a.State, a.Integrity)
	}
	if a.Start != 0 || a.End != 0 {
		t.Errorf("an orphan must claim no line, got %d-%d", a.Start, a.End)
	}
}

// --- rare lines vs boilerplate ------------------------------------------

// The dangerous case. A record's span is nothing but lines that occur all over
// the file. Delete the original site and a naive scan finds the same sequence
// somewhere else, then reports a confident 0.90 move onto unrelated code.
func TestAnchorRefusesToChaseABoilerplateSpan(t *testing.T) {
	// `os.Exit(0)` and `}` five times over. The record covers the first pair;
	// then that pair is removed.
	var file []string
	for i := 0; i < 5; i++ {
		file = append(file, "if err != nil {", "os.Exit(0)", "}")
	}
	r := Record{ID: "boiler", File: "a.go", Start: 2, End: 3,
		Lines: hashSpan([]string{"os.Exit(0)", "}"})}

	// Still there, exactly where recorded: fine, no searching involved.
	if a := resolveAnchor(file, r); a.State != StateExact {
		t.Fatalf("an unmoved span should still anchor exactly, got %q", a.State)
	}

	// Now remove the recorded pair. The identical pair still exists four times
	// over, and the anchor must NOT claim any of them.
	cut := append([]string{"if err != nil {"}, file[3:]...)
	a := resolveAnchor(cut, r)
	if a.State == StateDrifted {
		t.Fatalf("claimed a confident move to %d-%d on lines that appear all over the file",
			a.Start, a.End)
	}
	if a.Integrity >= weakFloor {
		t.Errorf("boilerplate must not earn integrity, got %q at %.2f", a.State, a.Integrity)
	}
}

// The mild case. A block is gutted but its scaffolding survives, so an unweighted
// count reads "most of it is still here" when everything meaningful is gone.
func TestAnchorDoesNotSurviveOnScaffoldingAlone(t *testing.T) {
	var file []string
	for i := 0; i < 12; i++ { // plenty of braces elsewhere, so `}` is common
		file = append(file, "func f"+strconv.Itoa(i)+"() {", "\tdoWork()", "}")
	}
	// The record covers a unique middle line plus a common brace.
	file[19] = "\tcritical := loadTheThing()"
	r := Record{ID: "gutted", File: "a.go", Start: 20, End: 21,
		Lines: hashSpan([]string{"critical := loadTheThing()", "}"})}
	if a := resolveAnchor(file, r); a.State != StateExact {
		t.Fatalf("setup wrong: expected exact, got %q at %d-%d", a.State, a.Start, a.End)
	}

	// Rewrite the unique line. The brace still matches — half the span by count.
	file[19] = "\tcritical := loadSomethingElse()"
	a := resolveAnchor(file, r)
	if a.State != StateOrphaned {
		t.Errorf("only the brace survives, so this should orphan, got %q at %.2f",
			a.State, a.Integrity)
	}
}

func TestAnchorOrphansWhenFileIsGone(t *testing.T) {
	if a := resolveAnchor(nil, rec(block)); a.State != StateOrphaned || a.Start != 0 {
		t.Errorf("a missing file should orphan with no line, got %q at %d", a.State, a.Start)
	}
}

func TestAnchorLineOnlyWithoutHashes(t *testing.T) {
	// Records written before anchoring existed keep working, and must not claim
	// a confidence they have no basis for.
	r := Record{ID: "old", File: "a.go", Start: 10, End: 20}
	a := resolveAnchor(block, r)
	if a.State != StateLineOnly {
		t.Fatalf("a record with no hashes is line-only, got %q", a.State)
	}
	if a.Start != 10 || a.End != 20 {
		t.Errorf("line-only should keep the recorded span, got %d-%d", a.Start, a.End)
	}
}

// --- through Match ------------------------------------------------------

// The payoff: `whence file:187` has to find a record that was written about line
// 142 and has since drifted there. Matching on the recorded range would miss it,
// which is the whole reason anchoring exists.
func TestMatchFindsDriftedRecordAtItsNewLine(t *testing.T) {
	root := t.TempDir()
	moved := append([]string{"store := sessionStore(ctx)", ""}, block...)
	if err := os.WriteFile(filepath.Join(root, "session.go"),
		[]byte(strings.Join(moved, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rs := []Record{rec(block)} // recorded at 2-4, now living at 4-6

	if got := Match(root, rs, "session.go", 5); len(got) != 1 {
		t.Errorf("line 5 is inside the drifted span, should match: got %d", len(got))
	}
	if got := Match(root, rs, "session.go", 2); len(got) != 0 {
		t.Errorf("line 2 is where it used to be, not where it is: got %d", len(got))
	}
}

func TestMatchSinksOrphansBelowLiveAnchors(t *testing.T) {
	// renderContext truncates at the context cap, so this ordering decides what
	// an agent sees. A lost anchor must never crowd out one that holds.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "session.go"),
		[]byte(strings.Join(block, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	orphan := Record{ID: "lost", Date: "2026-12-31", File: "session.go", Start: 1, End: 1,
		Decision: "about code that no longer exists",
		Lines:    []string{hashLine("gone(); alsoGone(); stillGone()")}}

	got := Match(root, []Record{orphan, rec(block)}, "session.go", 0)
	if len(got) != 2 {
		t.Fatalf("both records are on the file, got %d", len(got))
	}
	if got[0].ID != "b5" {
		t.Errorf("the live anchor should lead despite the older date, got %q", got[0].ID)
	}
	if got[1].Anchor.State != StateOrphaned {
		t.Errorf("the newer record should have orphaned, got %q", got[1].Anchor.State)
	}
}

// --- decay against this repo's history --------------------------------

// resolveAnchor has only ever been tested against synthetic edits written by
// someone who knew the algorithm. This walks first-parent history and asks
// what state today's records resolve to in each past snapshot they were alive
// for. A record is eligible at a commit only when its Date is not later than
// that commit's date — rows older than the record are pre-history, not decay.
// File-absent orphans are a missing path; file-present orphans are the miss.
// Existence is cat-file, not "did show error", so a read failure cannot hide
// in the absent column.
//
// Gated: the suite stays offline and hermetic unless you ask.
func TestAnchorDecay(t *testing.T) {
	if os.Getenv("WHENCE_DECAY") != "1" {
		t.Skip("set WHENCE_DECAY=1 to measure anchor decay against git history")
	}

	root, err := decayGit("rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("not a git repository: %v", err)
	}
	root = strings.TrimSpace(root)

	out, err := decayGit("-C", root, "log", "--first-parent", "--format=%H %cd", "--date=short", "HEAD")
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	type commit struct{ hash, date string }
	var commits []commit
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		hash, date, ok := strings.Cut(line, " ")
		if !ok || hash == "" || date == "" {
			t.Fatalf("git log line %q: want hash and YYYY-MM-DD", line)
		}
		commits = append(commits, commit{hash: hash, date: date})
	}
	if len(commits) == 0 {
		t.Fatal("no first-parent commits")
	}

	rs, err := Load(filepath.Join(root, storeDirName, recordsFileName))
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) == 0 {
		t.Fatal("no records to measure")
	}

	type snap struct {
		lines   []string
		present bool
	}
	cache := map[string]snap{}
	show := func(rev, path string) snap {
		key := rev + "\x00" + path
		if s, ok := cache[key]; ok {
			return s
		}
		spec := rev + ":" + path
		if _, err := decayGit("-C", root, "cat-file", "-e", spec); err != nil {
			s := snap{present: false}
			cache[key] = s
			return s
		}
		blob, err := decayGit("-C", root, "show", spec)
		if err != nil {
			t.Fatalf("%s exists at %s but could not be read: %v", path, rev, err)
		}
		s := snap{present: true}
		text := strings.TrimSuffix(blob, "\n")
		if text != "" {
			s.lines = strings.Split(text, "\n")
		}
		cache[key] = s
		return s
	}

	t.Logf("%d records × %d first-parent commits", len(rs), len(commits))
	t.Logf("%4s  %8s  %9s  %5s  %7s  %4s  %13s  %14s",
		"back", "eligible", "line-only", "exact", "drifted", "weak", "orphan-absent", "orphan-present")

	for back, c := range commits {
		var eligible, lineOnly, exact, drifted, weak, absent, present int
		for _, r := range rs {
			if r.Date > c.date {
				continue
			}
			eligible++
			s := show(c.hash, r.File)
			switch resolveAnchor(s.lines, r).State {
			case StateLineOnly:
				lineOnly++
			case StateExact:
				exact++
			case StateDrifted:
				drifted++
			case StateWeak:
				weak++
			case StateOrphaned:
				if s.present {
					present++
				} else {
					absent++
				}
			}
		}
		if got := lineOnly + exact + drifted + weak + absent + present; got != eligible {
			t.Fatalf("back %d: counted %d of %d eligible", back, got, eligible)
		}
		t.Logf("%4d  %8d  %9d  %5d  %7d  %4d  %13d  %14d",
			back, eligible, lineOnly, exact, drifted, weak, absent, present)
	}
}

// decayGit is git with stderr discarded, so a missing path at an old revision
// does not leak `fatal:` lines outside the test log.
func decayGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Stderr = io.Discard
	out, err := cmd.Output()
	return string(out), err
}
