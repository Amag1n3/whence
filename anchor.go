package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
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
	StateExact    AnchorState = "anchored, exact range"
	StateDrifted  AnchorState = "anchored, content hash"
	StateWeak     AnchorState = "weak — content changed"
	StateOrphaned AnchorState = "ORPHANED — anchor lost, needs a human"
)

// Confidence is driven by content, not by distance. A block that moved 400
// lines but still hashes identically is not less certainly the same block —
// so drift costs a small fixed amount for the surrounding context having
// changed, and nothing more.
//
// Both numbers are picked by eye, not measured. They are the knobs to
// calibrate once real repos produce real orphans; weakFloor especially, since
// it is the line between "the code changed but this is recognisably the same
// block" and "a different block that happens to share a few lines".
const (
	driftedConfidence = 0.90
	weakCeiling       = 0.85
	weakFloor         = 0.60
)

// Anchor is where a record points in the file as it is now.
type Anchor struct {
	State AnchorState

	// Start and End are the record's CURRENT span, which is not necessarily
	// its recorded one. Both are zero when the anchor is lost: an orphan has
	// no line to point at, and inventing one is the exact failure this whole
	// file exists to prevent.
	Start, End int

	Confidence float64
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

// fileLines reads a file into 1-indexed-by-convention lines. An unreadable file
// yields nil, which resolveAnchor reads as "orphaned" — the file being gone is
// a legitimate answer, not an error to propagate.
func fileLines(path string) []string {
	b, err := os.ReadFile(path)
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
			return Anchor{State: StateExact, Start: r.Start, End: r.End, Confidence: 1}
		}
	}

	sig := significant(lines)
	h := len(r.Lines)
	if len(sig) < h {
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
		return Anchor{State: StateDrifted, Start: sig[best].n, End: sig[best+h-1].n,
			Confidence: driftedConfidence}
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
		if s := overlap(r.Lines, sig[i:i+h]); s > bestSim {
			bestSim, bestAt = s, i
		}
	}
	if bestSim >= weakFloor {
		// Capped: the sequence check above already failed, so this window
		// holds the same lines in a different order or with substitutions.
		// That is not the same block, whatever the ratio says.
		if bestSim > weakCeiling {
			bestSim = weakCeiling
		}
		return Anchor{State: StateWeak, Start: sig[bestAt].n, End: sig[bestAt+h-1].n,
			Confidence: bestSim}
	}
	return Anchor{State: StateOrphaned, Confidence: bestSim}
}

// overlap is how much of the recorded block survives in this window, as a
// fraction of the record's own length. Asymmetric on purpose: the question is
// what happened to the recorded lines, not how similar two blocks are to each
// other. Multiset, so three identical lines are not matched by one.
func overlap(want []string, got []hline) float64 {
	if len(want) == 0 {
		return 0
	}
	pool := make(map[string]int, len(got))
	for _, g := range got {
		pool[g.h]++
	}
	hit := 0
	for _, w := range want {
		if pool[w] > 0 {
			pool[w]--
			hit++
		}
	}
	return float64(hit) / float64(len(want))
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
