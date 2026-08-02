package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// `why check` — the CI gate.
//
// It compares a diff against the records covering the lines that diff touches,
// and exits non-zero when any are found. That exit code is the point: it is the
// one place whence stops a change rather than merely informing it.
//
// WHAT IT DOES NOT DO: decide whether the change is wrong. DECISIONS §3 is
// explicit that the failure mode of this project is drifting into code review —
// "if whence starts judging diffs, it has lost" — and deciding that an edit
// contradicts a recorded decision is a judgement about code. So check reports
// *coverage*, never verdicts: these lines are governed by these decisions, go
// and confirm. The README's older mock said "contradicts record #4f2a", which
// claimed more than this can honestly do.
//
// Two things get reported, and the second is the one worth having:
//
//  1. touched — the diff modifies lines a record currently anchors to.
//  2. anchor lost — the record anchored cleanly in the base revision and does
//     not anchor at all now. That means this change destroyed the link between
//     a decision and the code it was about, which no human reviewing the diff
//     would otherwise notice.
func checkCmd(args []string) {
	fl := flag.NewFlagSet("check", flag.ExitOnError)
	base := fl.String("base", "origin/main", "revision to compare against")
	if err := fl.Parse(args); err != nil {
		os.Exit(2)
	}

	// check reads git, and only git's diff. That is not a reversal of §14's
	// "nothing in the tool reads git state" — that decision is about refusing to
	// let git decide anything about the store (whether it is tracked, whether it
	// may be written). A diff is this command's input; there is nothing else it
	// could be.
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		fmt.Fprintln(os.Stderr, "whence: not a git repository")
		os.Exit(2)
	}
	root = strings.TrimSpace(root)

	// base → working tree, rather than base...HEAD. In CI the tree is clean so
	// they are the same thing, and locally this also catches what you have not
	// committed yet, which is when it is most useful.
	diff, err := git("-C", root, "diff", "--unified=0", "--no-color", *base)
	if err != nil {
		fmt.Fprintf(os.Stderr, "whence: cannot diff against %s (fetch it first?)\n", *base)
		os.Exit(2)
	}

	byFile := changedRanges(diff)
	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files) // stable output: CI logs get compared by humans

	total := 0
	priorByStore := map[string]map[string]bool{}

	for _, file := range files {
		changed := byFile[file]
		abs := filepath.Join(root, file)

		store, storeRoot, ok := FindStore(abs)
		if !ok {
			continue
		}
		rs, err := Load(store)
		if err != nil || len(rs) == 0 {
			continue
		}
		rel := Rel(storeRoot, abs)

		prior, seen := priorByStore[store]
		if !seen {
			prior = priorIDs(root, *base, store)
			priorByStore[store] = prior
		}

		// The base revision's copy of the file, to tell "this diff broke the
		// anchor" apart from "the anchor was already broken".
		wasLines := gitLines(root, *base, file)

		nowLines := fileLines(abs)
		for _, f := range inspect(rs, prior, rel, nowLines, wasLines, changed) {
			report(f)
			total++
		}
		for _, f := range groundsLostIn(rs, prior, rel, nowLines, wasLines) {
			report(f)
			total++
		}
	}

	if total == 0 {
		fmt.Println("no records cover this diff.")
		return
	}
	fmt.Printf("\n%d record(s) to confirm. whence does not judge the change — it reports that\n"+
		"recorded decisions cover these lines. Confirm each, then re-run.\n", total)
	os.Exit(1)
}

// lineSpan is an inclusive range of line numbers in a file's current revision.
type lineSpan struct{ start, end int }

type finding struct {
	r      Record
	anchor Anchor
	lost   bool // anchored in the base revision, anchors nowhere now
	at     []lineSpan

	// eroded is the record's integrity BEFORE this diff, set only when the change
	// reduced it without destroying it outright. Zero means not eroded: integrity
	// cannot fall to something lower than zero, so a real erosion always leaves a
	// positive number here.
	eroded float64

	// Set when the thing that made this record TRUE is what the diff destroyed,
	// rather than the code the record was about. The record itself can be
	// perfectly anchored and still be left standing on nothing.
	ground *Grounded
}

// inspect is the whole decision, kept free of git and the filesystem so it can
// be tested directly. now and was are the file's lines in the working tree and
// in the base revision; was may be nil for a newly added file.
func inspect(rs []Record, prior map[string]bool, file string, now, was []string, changed []lineSpan) []finding {
	var out []finding
	for _, r := range rs {
		if !samePath(r.File, file) {
			continue
		}
		// Only records that existed in the base revision. A record added by this
		// very diff is not a prior decision the change could be walking into —
		// and without this, any pull request that uses `why add` fails its own
		// gate, which is the fastest way to teach a team to ignore the gate.
		if !prior[r.ID] {
			continue
		}
		a := resolveAnchor(now, r)
		var before Anchor
		if was != nil {
			before = resolveAnchor(was, r)
		}

		if a.State == StateOrphaned {
			// Only report an orphan this diff is responsible for. One that was
			// already orphaned in the base revision is a real problem, but it is
			// not this pull request's problem, and failing CI for it would train
			// people to pass -no-verify.
			if was != nil && before.State != StateOrphaned {
				out = append(out, finding{r: r, anchor: a, lost: true})
			}
			continue
		}
		if a.Start == 0 {
			continue
		}

		// Eroded but not lost — and this has to be tested BEFORE the line overlap
		// below, because the overlap misses exactly this case.
		//
		// Once a record degrades, its span is a best-match window of h significant
		// lines: a fixed count, not the real region. Add lines inside the block and
		// the window slides off the edit that caused the damage, so the change that
		// hurt the record most is the one the overlap test cannot see. Comparing
		// integrity across the base revision asks the question directly instead of
		// inferring it from line numbers, which is the same mistake the anchor
		// scoring used to make.
		if was != nil && a.Integrity < before.Integrity {
			out = append(out, finding{r: r, anchor: a, eroded: before.Integrity})
			continue
		}
		var hits []lineSpan
		for _, c := range changed {
			if c.start <= a.End && c.end >= a.Start {
				hits = append(hits, c)
			}
		}
		if len(hits) > 0 {
			out = append(out, finding{r: r, anchor: a, at: hits})
		}
	}
	return out
}

// groundsLostIn reports records whose evidence pointed into this file and does
// not survive the change.
//
// Kept separate from inspect because it asks a different question about a
// different file. A record's own anchor concerns the code the decision was
// *about*; its grounds concern the code that made the decision *true*, which
// usually lives somewhere else entirely. So a diff can leave a record perfectly
// anchored and still knock the ground out from under it — and that is invisible
// in review, because the record's own file was never opened.
func groundsLostIn(rs []Record, prior map[string]bool, file string, now, was []string) []finding {
	var out []finding
	for _, r := range rs {
		if !prior[r.ID] {
			continue
		}
		for _, e := range r.Evidence {
			if !e.anchored() || !samePath(e.File, file) {
				continue
			}
			nowA := resolveAnchor(now, e.asRecord())
			if nowA.State != StateOrphaned {
				continue
			}
			// Only what this diff is responsible for.
			if was == nil || resolveAnchor(was, e.asRecord()).State == StateOrphaned {
				continue
			}
			g := Grounded{Evidence: e, Anchor: nowA}
			out = append(out, finding{r: r, ground: &g})
		}
	}
	return out
}

func report(f finding) {
	if f.ground != nil {
		fmt.Printf("\n  ✗ %s — this change removes the evidence for record [%s]\n",
			f.ground.Ref, f.r.ID)
		fmt.Printf("    the record still anchors to %s:%d-%d and reads as current\n",
			f.r.File, f.r.Start, f.r.End)
		fmt.Printf("    %s\n", f.r.Decision)
		fmt.Printf("    what made it true is gone, so the decision now rests on nothing.\n")
		fmt.Printf("    point it at what makes it true now, or retract it.\n")
		return
	}
	if f.eroded > 0 {
		fmt.Printf("\n  ! %s — this change erodes record [%s] (%s)\n", f.r.File, f.r.ID, f.r.Date)
		fmt.Printf("    %.0f%% of the recorded block survived before this diff, %.0f%% now\n",
			f.eroded*100, f.anchor.Integrity*100)
		fmt.Printf("    %s\n", f.r.Decision)
		if f.r.Why != "" {
			fmt.Printf("    why: %s\n", f.r.Why)
		}
		fmt.Printf("    %s · %s\n", locate(Resolved{Record: f.r, Anchor: f.anchor}), f.anchor.State)
		// Deliberately not pre-filled with the span above. That span is a
		// best-match window, and a command you can paste without reading is a
		// command that stores a guess as a certainty.
		fmt.Printf("    if the rewrite kept the decision, re-point it with `why reanchor %s %s:<start>-<end>`.\n",
			f.r.ID, f.r.File)
		return
	}
	if f.lost {
		fmt.Printf("\n  ✗ %s — record [%s] lost its anchor in this change\n", f.r.File, f.r.ID)
		fmt.Printf("    it anchored at %d-%d before this diff; nothing matches now\n", f.r.Start, f.r.End)
		fmt.Printf("    %s\n", f.r.Decision)
		if f.r.Why != "" {
			fmt.Printf("    why: %s\n", f.r.Why)
		}
		fmt.Printf("    the decision is still on record; the code it described is gone.\n")
		fmt.Printf("    re-point it with `why reanchor %s %s:<start>-<end>`, or retract it deliberately.\n",
			f.r.ID, f.r.File)
		return
	}
	fmt.Printf("\n  ! %s:%s — touches record [%s] (%s)\n",
		f.r.File, spans(f.at), f.r.ID, f.r.Date)
	fmt.Printf("    %s\n", f.r.Decision)
	if f.r.Why != "" {
		fmt.Printf("    why: %s\n", f.r.Why)
	}
	fmt.Printf("    %s · %s\n", locate(Resolved{Record: f.r, Anchor: f.anchor}), f.anchor.State)
}

func spans(ss []lineSpan) string {
	parts := make([]string, len(ss))
	for i, s := range ss {
		if s.start == s.end {
			parts[i] = strconv.Itoa(s.start)
			continue
		}
		parts[i] = fmt.Sprintf("%d-%d", s.start, s.end)
	}
	return strings.Join(parts, ",")
}

// changedRanges parses `git diff --unified=0` into the lines each file gained or
// had rewritten, keyed by repo-relative path.
//
// Zero context is what makes this simple: every hunk header's new-side range IS
// the changed lines, with no surrounding unchanged lines mixed in. A hunk with a
// count of 0 is a pure deletion, recorded as the single line it happened at, so
// that removing the code a record covers still registers.
func changedRanges(diff string) map[string][]lineSpan {
	out := map[string][]lineSpan{}
	file := ""
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "):
			p := strings.TrimPrefix(line, "+++ ")
			if p == "/dev/null" { // the file was deleted
				file = ""
				continue
			}
			file = strings.TrimPrefix(p, "b/")

		case strings.HasPrefix(line, "@@ ") && file != "":
			// @@ -12,0 +13,4 @@ optional trailing context
			fields := strings.Fields(line)
			if len(fields) < 3 {
				continue
			}
			start, count, ok := parseHunk(fields[2])
			if !ok {
				continue
			}
			end := start + count - 1
			if count == 0 {
				end = start // a deletion: flag the line it happened at
			}
			out[file] = append(out[file], lineSpan{start: start, end: end})
		}
	}
	return out
}

// parseHunk reads "+13,4" or "+13" into (13, 4) / (13, 1).
func parseHunk(f string) (start, count int, ok bool) {
	f = strings.TrimPrefix(f, "+")
	a, b, hasCount := strings.Cut(f, ",")
	start, err := strconv.Atoi(a)
	if err != nil {
		return 0, 0, false
	}
	if !hasCount {
		return start, 1, true
	}
	count, err = strconv.Atoi(b)
	if err != nil {
		return 0, 0, false
	}
	return start, count, true
}

func git(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	return string(out), err
}

// priorIDs is the set of record ids that existed in the base revision's store.
//
// A store absent or unreadable at base yields an empty set, so nothing is
// reported. That is the right way round: a repository adopting whence in this
// very pull request should not have its own adoption fail the gate. Like the
// hook, check would rather say nothing than say something false.
func priorIDs(root, base, store string) map[string]bool {
	rel, err := filepath.Rel(root, store)
	if err != nil {
		return nil
	}
	out, err := git("-C", root, "show", base+":"+filepath.ToSlash(rel))
	if err != nil {
		return nil
	}
	var rs []Record
	if err := json.Unmarshal([]byte(out), &rs); err != nil {
		return nil
	}
	ids := make(map[string]bool, len(rs))
	for _, r := range rs {
		ids[r.ID] = true
	}
	return ids
}

// gitLines is a file as of a revision, or nil if it did not exist there.
func gitLines(root, rev, path string) []string {
	out, err := git("-C", root, "show", rev+":"+path)
	if err != nil {
		return nil
	}
	s := strings.TrimSuffix(out, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
