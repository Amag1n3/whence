package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

// Anchoring: does this record still attach to the code it was written about?
//
// A record that loses its anchor is a diary entry; one that keeps it is a
// control. So the answer is never assumed. The span is re-read and re-hashed on
// every lookup, and when it no longer matches, the record says so out loud
// rather than pointing confidently at whatever now sits on that line. A tool
// that silently points at the wrong line teaches you to distrust everything it
// says — see DECISIONS §6.
//
// Tree-sitter AST paths are NOT here. A content hash plus a window scan already
// survives insertion, deletion, reindentation and whole-block moves, which is
// most real drift. AST paths buy the remainder — a block moved into a different
// function, a signature rewritten around unchanged statements — for the price
// of a CGo dependency.
//
// ponytail: per-line content hashes + a window scan, no AST. Add tree-sitter
// when a real repo produces orphans this cannot explain.

type AnchorState string

// The state strings are the display strings. There is one vocabulary for this
// in the CLI, the injected context and the site, and a second mapping layer
// would only be somewhere for them to drift apart.
const (
	StateLineOnly AnchorState = "line range only, unverified"
	StateExact    AnchorState = "intact, exact range"
	StateDrifted  AnchorState = "intact, moved"
	StateWeak     AnchorState = "altered"
	StateOrphaned AnchorState = "ORPHANED — anchor lost, needs a human"
)

// Content and position are separate readings. Collapsing them into one number
// is what this file used to get wrong.
//
// A block can be byte-identical and 400 lines away, or sitting exactly where it
// was recorded and half rewritten. Those are opposite situations and one score
// cannot say which is which — so position is reported as positions (see locate)
// and the number reports content alone.
//
// What was here before charged a flat 0.90 for having moved. That figure
// measured nothing: it was a constant, so it did not track distance, and it
// triggered on any insertion anywhere above the block, which says nothing about
// the block. Every record in a file people work in converged on it within days,
// and a reading they all share cannot tell any of them apart.
const (
	// weakFloor is how much of a recorded block has to survive before this is
	// still recognisably that block rather than a lookalike sharing a few lines.
	//
	// Picked by eye, not measured — the knob to calibrate once real repos
	// produce real orphans. There is deliberately no ceiling to go with it: a
	// cap would throw away the ratio the scan just finished computing, which is
	// the one number in this file that is actually a measurement.
	weakFloor = 0.60

	// rareAt is how many times a line may appear in a file and still count as
	// identifying a place in it. `store.Set("CHECKOUT_role", s.Role)` appears
	// once and pins a location exactly; `}` appears forty times and pins
	// nothing. Two rather than one because a line legitimately appearing twice
	// (an early return and a final one) is still narrow enough to be evidence.
	rareAt = 2
)

// Anchor is where a record points in the file as it is now.
type Anchor struct {
	State AnchorState

	// Start and End are the record's CURRENT span, which is not necessarily
	// its recorded one. Both are zero when the anchor is lost: an orphan has
	// no line to point at, and inventing one is the exact failure this whole
	// file exists to prevent.
	//
	// These two carry the position reading. How far a record travelled is
	// Start minus the record's own Start, and locate() shows both numbers
	// rather than the difference, because "now at 187, recorded at 142" is
	// read faster than "moved 45".
	Start, End int

	// Integrity is how much of the recorded content survives at Start..End,
	// weighted so a rare line counts for more than a common one. 1.0 means
	// every recorded line is still there.
	//
	// Set rather than computed for exact and drifted: both are proven
	// byte-identical matches, so there is nothing left to measure.
	Integrity float64
}

// hline is one significant line of a file: its hash, and the 1-based line
// number it came from. Anchoring works over these rather than over raw text, so
// blank lines and indentation cannot move an anchor.
type hline struct {
	n int
	h string
}

// hashLine hashes one line's significant content, truncated to 8 hex chars —
// long enough that two different lines colliding inside one file is not a
// practical concern, short enough that a record stays readable.
//
// Deliberately NOT salted with the file path. Salting would stop these hashes
// being comparable across files, and following a block that moved to another
// file is on the roadmap (§6 lists `git blame -C -M` as prior art). What
// salting would buy is small either way: the store is committed next to the
// code it describes (§14), so the plaintext is already sitting right there, and
// Phase 3 sync is E2E encrypted (§7.5). This is also why storing per-line
// hashes does not violate §7.2's "never raw file contents" — a hash of a line
// is not the line, and it is what makes graded confidence possible at all.
func hashLine(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:4])
}

// hashSpan returns a hash per significant line of the given lines, in order.
// This is what a record stores as its anchor.
func hashSpan(lines []string) []string {
	var out []string
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			out = append(out, hashLine(t))
		}
	}
	return out
}

// significant is hashSpan over a whole file, keeping line numbers so a match
// can be reported as a line range.
func significant(lines []string) []hline {
	var out []hline
	for i, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			out = append(out, hline{n: i + 1, h: hashLine(t)})
		}
	}
	return out
}

// resolveEvidence reports what has happened to each of a record's grounds.
//
// A pointer at a place in the code goes through resolveAnchor, because the
// question is identical: is the thing this refers to still there? That is the
// payoff of storing grounds as pointers rather than as copied snippets — when the
// code somebody cited as evidence is deleted, whence can say the grounds are gone
// without a human checking anything.
func resolveEvidence(root string, r Record) []Grounded {
	if len(r.Evidence) == 0 {
		return nil
	}
	out := make([]Grounded, 0, len(r.Evidence))
	for _, e := range r.Evidence {
		g := Grounded{Evidence: e}
		if e.anchored() {
			g.Anchor = resolveAnchor(fileLinesWithin(filepath.Join(root, e.File), root), e.asRecord())
		}
		out = append(out, g)
	}
	return out
}

// fileLines reads a file into 1-indexed-by-convention lines. An unreadable file
// yields nil, which resolveAnchor reads as "orphaned" — the file being gone is
// a legitimate answer, not an error to propagate.
//
// A record's File and its evidence File come from the store, and the store is
// pulled: a record claiming "../../etc/hosts" must not be READ, let alone
// resolved. Contents are never printed, so it is not exfiltration — but the
// intact/orphaned verdict is a one-bit hash oracle on any file on the machine,
// asked silently on every edit (§7 threat model: pulled records are untrusted).
// The guard lives in the one place every anchor read passes through, so no
// caller can route around it.
//
// ponytail: lexical guard only — a symlink inside the repo pointing out is not
// resolved here. That path needs an attacker who can commit a symlink into the
// repo, a much higher bar than a record line; upgrade to filepath.EvalSymlinks
// if a pulled store ever carries one.
func fileLines(path string) []string {
	return fileLinesWithin(path, "")
}

// fileLinesWithin is fileLines plus the repo root the path must stay inside.
// Records are written root-relative (Rel), so a stored path that resolves
// outside root was crafted, not authored.
//
// The path is symlink-resolved before reading, so a symlink inside the repo
// pointing out cannot turn a record into a one-bit hash oracle on a file the
// record never concerned (the guard outsideRoot already resolves, so resolving
// here too keeps the two reads consistent — see record.go's outsideRoot).
func fileLinesWithin(path, root string) []string {
	if outsideRoot(path, root) {
		return nil
	}
	resolved := path
	if r, err := filepath.EvalSymlinks(path); err == nil {
		resolved = r
	}
	b, err := os.ReadFile(resolved)
	if err != nil {
		return nil
	}
	s := strings.TrimSuffix(string(b), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// resolveAnchor answers where a record points in the file as it is now.
func resolveAnchor(lines []string, r Record) Anchor {
	// A record with no hashes predates anchoring, or was written by hand. Its
	// line range is all there is, and claiming a confidence for it would be
	// inventing one.
	if len(r.Lines) == 0 {
		return Anchor{State: StateLineOnly, Start: r.Start, End: r.End}
	}
	if len(lines) == 0 {
		return Anchor{State: StateOrphaned}
	}

	// Still exactly where it says it is: the common case, and the cheap one.
	if r.Start >= 1 && r.Start <= r.End && r.End <= len(lines) {
		if sameSeq(hashSpan(lines[r.Start-1:r.End]), r.Lines) {
			return Anchor{State: StateExact, Start: r.Start, End: r.End, Integrity: 1}
		}
	}

	sig := significant(lines)
	h := len(r.Lines)
	if len(sig) < h {
		return Anchor{State: StateOrphaned}
	}
	freq := counts(sig)

	// Searching only makes sense if the span contains something distinctive. A
	// span of `os.Exit(0)` and `}` matches in a dozen places, so whichever match
	// is nearest would be reported as a confident move onto unrelated code —
	// the one failure this file exists to prevent.
	//
	// This gate has to come before BOTH scans, not just the exact one. The
	// weighted score below is a ratio, so a span where every line is common
	// scores 1.0 when it survives: common lines over common lines. Weighting
	// catches a span that loses its rare lines and keeps its scaffolding; it
	// cannot catch a span that never had a rare line to begin with. Only this
	// can.
	//
	// The span was already checked at its recorded lines above, so reaching here
	// means it is not where it says it is. Unidentifiable and not where it was
	// recorded is exactly what orphaned means.
	if !identifiable(r.Lines, freq) {
		return Anchor{State: StateOrphaned}
	}

	// The same content somewhere else: the code moved. Prefer the candidate
	// nearest the recorded span, so a block that appears twice does not send
	// the anchor to the far end of the file.
	best := -1
	for i := 0; i+h <= len(sig); i++ {
		if !sameWindow(sig[i:i+h], r.Lines) {
			continue
		}
		if best < 0 || abs(sig[i].n-r.Start) < abs(sig[best].n-r.Start) {
			best = i
		}
	}
	if best >= 0 {
		// Integrity 1, not a discount for having moved. sameWindow demands the
		// whole recorded hash sequence, in order, so reaching here proves the
		// content is byte-identical — it is only somewhere else. Charging that
		// would contradict the reason content drives the score in the first
		// place.
		return Anchor{State: StateDrifted, Start: sig[best].n, End: sig[best+h-1].n,
			Integrity: 1}
	}

	// Nothing matches outright. Find whatever survived best, and be honest
	// about how little that is.
	//
	// ponytail: O(file × span) scan, no index. A 3k-line file against a 10-line
	// span is ~30k hash comparisons — microseconds, and it only runs on records
	// that already failed both cheap checks. Revisit if capture ever starts
	// anchoring records to spans hundreds of lines long.
	bestSim, bestAt := 0.0, -1
	for i := 0; i+h <= len(sig); i++ {
		if s := overlap(r.Lines, sig[i:i+h], freq); s > bestSim {
			bestSim, bestAt = s, i
		}
	}
	// The floor is span-aware: a recorded span of h lines may lose at most one
	// line and still count as altered rather than lost, so the effective floor
	// is min(weakFloor, (h-1)/h). harvest anchors the comment plus the
	// declaration that follows it (author.go:1372-1379), which makes two lines
	// the commonest span the tool produces — and against a flat 0.60 a
	// two-line record that loses its declaration scores at most 0.5, printing
	// ORPHANED for a comment still sitting in the file untouched. Six of the
	// 2026-08-17 run's 40 orphans were exactly this
	// (ANCHOR-SURVIVAL-2026-08-17.md, defect 3): decay the span exists to
	// report was landing as total loss. The floor only ever loosens, never
	// tightens — for h=10, (h-1)/h is 0.9 and weakFloor stands.
	//
	// Guarded at h >= 2: (h-1)/h at h=1 is 0, which would read any nonzero
	// overlap as altered — a one-line record either matched exactly above or
	// is genuinely gone.
	floor := weakFloor
	if h >= 2 {
		if hf := float64(h-1) / float64(h); hf < floor {
			floor = hf
		}
	}
	if bestSim >= floor {
		// Reported as measured. A cap used to sit here, flattening every
		// altered block to 0.85 — so one argument added to one line of a forty
		// line span read exactly like a block half rewritten, which is the
		// distinction the number exists to make.
		//
		// The cap was guarding a real case the wrong way. Reaching here means
		// the recorded sequence is nowhere in the file, so a span whose lines
		// all survive in a different order scores 1.0. State and integrity are
		// answering different questions — "is the sequence intact" and "is the
		// content still there" — and a reader given both can tell a resequenced
		// block from an eroded one. Given only a capped number, they cannot.
		return Anchor{State: StateWeak, Start: sig[bestAt].n, End: sig[bestAt+h-1].n,
			Integrity: bestSim}
	}
	return Anchor{State: StateOrphaned, Integrity: bestSim}
}

// overlap is how much of the recorded block survives in this window, weighted
// so that rare lines count for more than common ones.
//
// Asymmetric on purpose: the question is what happened to the recorded lines,
// not how similar two blocks are to each other. Counted as a multiset, so three
// identical lines are not matched by one.
//
// The weighting is what stops scaffolding from carrying a dead record. A block
// rewritten down to its `func` line and its closing brace keeps two of four
// lines — an unweighted 0.50, comfortably "weak, still here" — while every line
// that meant anything is gone. Dividing each line's contribution by how often it
// occurs in the file puts that case near zero, where it belongs.
func overlap(want []string, got []hline, freq map[string]int) float64 {
	if len(want) == 0 {
		return 0
	}
	pool := make(map[string]int, len(got))
	for _, g := range got {
		pool[g.h]++
	}
	var total, hit float64
	for _, w := range want {
		// A line absent from the file entirely is treated as maximally rare: it
		// is unique to the record, and its absence is the strongest evidence
		// there is that the block is gone.
		wt := 1 / float64(max(1, freq[w]))
		total += wt
		if pool[w] > 0 {
			pool[w]--
			hit += wt
		}
	}
	if total == 0 {
		return 0
	}
	return hit / total
}

// counts is how many times each line appears in the file.
func counts(sig []hline) map[string]int {
	freq := make(map[string]int, len(sig))
	for _, s := range sig {
		freq[s.h]++
	}
	return freq
}

// identifiable reports whether a span contains at least one line rare enough to
// pin a location. A span made entirely of lines that occur all over the file
// cannot be found by content — searching for it anyway produces a confident
// answer about whichever lookalike happens to be nearest.
func identifiable(want []string, freq map[string]int) bool {
	for _, w := range want {
		if n := freq[w]; n > 0 && n <= rareAt {
			return true
		}
	}
	return false
}

func sameSeq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameWindow(w []hline, want []string) bool {
	for i := range want {
		if w[i].h != want[i] {
			return false
		}
	}
	return true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
