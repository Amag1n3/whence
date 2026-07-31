package main

import (
	"os"
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
	if a.Confidence != 1 {
		t.Errorf("confidence should be 1 on an exact match, got %.2f", a.Confidence)
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
		t.Errorf("reformatting must not lose the anchor, got %q at %.2f", a.State, a.Confidence)
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
	if a.Confidence != driftedConfidence {
		t.Errorf("confidence = %.2f, want %.2f", a.Confidence, driftedConfidence)
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
		t.Fatalf("a partly rewritten block should be weak, got %q at %.2f", a.State, a.Confidence)
	}
	if a.Confidence >= driftedConfidence {
		t.Errorf("weak must score below a clean drift, got %.2f", a.Confidence)
	}
	if a.Confidence < weakFloor {
		t.Errorf("2 of 3 lines surviving should clear the floor, got %.2f", a.Confidence)
	}
}

func TestAnchorOrphansRatherThanGuess(t *testing.T) {
	// The block is refactored away entirely. The one thing that must not happen
	// is a confident line number.
	gone := []string{block[0], "	persistNamespaced(s)", block[4]}
	a := resolveAnchor(gone, rec(block))
	if a.State != StateOrphaned {
		t.Fatalf("a rewritten block should orphan, got %q at %.2f", a.State, a.Confidence)
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
	if a.Confidence >= weakFloor {
		t.Errorf("boilerplate must not earn confidence, got %q at %.2f", a.State, a.Confidence)
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
			a.State, a.Confidence)
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

// The payoff: `why file:187` has to find a record that was written about line
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
