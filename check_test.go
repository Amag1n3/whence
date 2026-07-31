package main

import "testing"

// Diff parsing is the fragile part of check: get a hunk header wrong and the gate
// either misses every change or flags the whole file. `--unified=0` is what keeps
// it simple — the new-side range of each hunk IS the changed lines.
func TestChangedRanges(t *testing.T) {
	diff := `diff --git a/auth.go b/auth.go
index 111..222 100644
--- a/auth.go
+++ b/auth.go
@@ -12,3 +12,4 @@ func persist(s Session) {
-old
+new
@@ -40,0 +41,2 @@
+added
+added
@@ -70,2 +72,0 @@
-deleted
-deleted
diff --git a/gone.go b/gone.go
--- a/gone.go
+++ /dev/null
@@ -1,5 +0,0 @@
diff --git a/single.go b/single.go
--- a/single.go
+++ b/single.go
@@ -9 +9 @@
-x
+y
`
	got := changedRanges(diff)

	want := map[string][]lineSpan{
		"auth.go": {
			{12, 15}, // +12,4
			{41, 42}, // +41,2
			{72, 72}, // +72,0 — a pure deletion, flagged at the line it happened
		},
		"single.go": {{9, 9}}, // "+9" with no count means one line
	}
	if len(got) != len(want) {
		t.Fatalf("files = %v, want %v", keysOf(got), keysOf(want))
	}
	for f, spans := range want {
		if len(got[f]) != len(spans) {
			t.Fatalf("%s: got %v, want %v", f, got[f], spans)
		}
		for i := range spans {
			if got[f][i] != spans[i] {
				t.Errorf("%s hunk %d: got %v, want %v", f, i, got[f][i], spans[i])
			}
		}
	}
	// A deleted file has no new side, so nothing can be anchored in it.
	if _, ok := got["gone.go"]; ok {
		t.Error("a file deleted in the diff should not appear")
	}
}

func keysOf(m map[string][]lineSpan) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// --- what gets reported -------------------------------------------------

func TestInspectFlagsARecordTheDiffTouches(t *testing.T) {
	now := block // unchanged, so the record anchors exactly at 2-4
	got := inspect([]Record{rec(block)}, map[string]bool{"b5": true}, "session.go", now, now, []lineSpan{{3, 3}})
	if len(got) != 1 {
		t.Fatalf("a change on line 3 is inside the record's span, want 1 finding, got %d", len(got))
	}
	if got[0].lost {
		t.Error("the anchor holds; this is a touch, not a loss")
	}
}

func TestInspectIgnoresAChangeOutsideEverySpan(t *testing.T) {
	now := block
	if got := inspect([]Record{rec(block)}, map[string]bool{"b5": true}, "session.go", now, now, []lineSpan{{5, 5}}); len(got) != 0 {
		t.Errorf("line 5 is outside the record's 2-4 span, got %d findings", len(got))
	}
}

// The record drifted, and the diff touches where it lives NOW. Checking against
// the recorded range would miss it — the same reason anchoring exists at all.
func TestInspectFollowsDriftBeforeComparing(t *testing.T) {
	moved := append([]string{"store := sessionStore(ctx)", ""}, block...)
	got := inspect([]Record{rec(block)}, map[string]bool{"b5": true}, "session.go", moved, block, []lineSpan{{5, 5}})
	if len(got) != 1 {
		t.Fatalf("the record now lives at 4-6, so line 5 touches it: got %d", len(got))
	}
	if got[0].anchor.State != StateDrifted {
		t.Errorf("expected the drifted anchor, got %q", got[0].anchor.State)
	}
}

// The highest-value output: this change destroyed the link between a decision and
// its code, which nobody reading the diff would notice.
func TestInspectReportsAnAnchorThisDiffDestroyed(t *testing.T) {
	gone := []string{block[0], "	persistNamespaced(s)", block[4]}
	got := inspect([]Record{rec(block)}, map[string]bool{"b5": true}, "session.go", gone, block, []lineSpan{{2, 2}})
	if len(got) != 1 {
		t.Fatalf("want 1 finding for a destroyed anchor, got %d", len(got))
	}
	if !got[0].lost {
		t.Error("should be reported as a lost anchor, not a touch")
	}
}

// A record that was already broken before this change is somebody else's problem.
// Failing CI for it teaches people to skip the gate.
func TestInspectIgnoresAnOrphanItDidNotCause(t *testing.T) {
	gone := []string{block[0], "	persistNamespaced(s)", block[4]}
	if got := inspect([]Record{rec(block)}, map[string]bool{"b5": true}, "session.go", gone, gone, []lineSpan{{2, 2}}); len(got) != 0 {
		t.Errorf("the anchor was already lost at base, got %d findings", len(got))
	}
}

// A record the diff itself introduces is not a prior decision. Without this, any
// pull request that records a decision fails its own gate, and a gate that cries
// wolf on correct behaviour gets switched off.
func TestInspectSkipsRecordsAddedByThisDiff(t *testing.T) {
	now := block
	notYetInBase := map[string]bool{} // the base store has no record b5
	if got := inspect([]Record{rec(block)}, notYetInBase, "session.go", now, now, []lineSpan{{3, 3}}); len(got) != 0 {
		t.Errorf("a record absent from the base revision must not be reported, got %d", len(got))
	}
}

func TestInspectSkipsOtherFiles(t *testing.T) {
	if got := inspect([]Record{rec(block)}, map[string]bool{"b5": true}, "other.go", block, block, []lineSpan{{3, 3}}); len(got) != 0 {
		t.Errorf("a record on session.go must not answer for other.go, got %d", len(got))
	}
}

// A diff can leave a record perfectly anchored and still knock the ground out
// from under it — the evidence usually lives in a different file, so nobody
// reviewing the change would ever open the record's own file.
func TestGroundsLostReportsEvidenceThisDiffRemoved(t *testing.T) {
	was := []string{
		"func render(u User) {",
		`	read("CHECKOUT_userToken")`,
		`	read("CHECKOUT_role")`,
		"}",
	}
	now := []string{"func render(u User) {", "	renderV2(u)", "}"}

	r := rec(block) // anchored to session.go, untouched by this diff
	r.Evidence = []Evidence{{
		Ref: "dashboard.go:2-3", File: "dashboard.go", Start: 2, End: 3,
		Lines: hashSpan(was[1:3]),
	}}
	prior := map[string]bool{"b5": true}

	got := groundsLostIn([]Record{r}, prior, "dashboard.go", now, was)
	if len(got) != 1 {
		t.Fatalf("want 1 finding for removed evidence, got %d", len(got))
	}
	if got[0].ground == nil {
		t.Fatal("the finding should carry the evidence that was lost")
	}
	if got[0].ground.Ref != "dashboard.go:2-3" {
		t.Errorf("wrong evidence reported: %q", got[0].ground.Ref)
	}

	// Unchanged evidence is not a finding.
	if got := groundsLostIn([]Record{r}, prior, "dashboard.go", was, was); len(got) != 0 {
		t.Errorf("evidence that still holds must not be reported, got %d", len(got))
	}
	// Already broken before this diff is not this diff's problem.
	if got := groundsLostIn([]Record{r}, prior, "dashboard.go", now, now); len(got) != 0 {
		t.Errorf("evidence already gone at base must not be reported, got %d", len(got))
	}
	// And a record the diff itself introduced is not a prior decision.
	if got := groundsLostIn([]Record{r}, map[string]bool{}, "dashboard.go", now, was); len(got) != 0 {
		t.Errorf("a record absent from base must not be reported, got %d", len(got))
	}
}
