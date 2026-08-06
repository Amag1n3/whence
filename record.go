package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	storeDirName    = ".whence"
	recordsFileName = "records.jsonl"

	// legacyRecordsName is the single-JSON-array store written before records
	// became line-delimited. Still read, never written: a store that silently
	// stops being found is the worst way for this change to reach anyone.
	legacyRecordsName = "records.json"

	attributesName   = ".gitattributes"
	surfacedLogName  = "surfaced.jsonl"
	retractedLogName = "retracted.jsonl"
)

// Record is one decision, anchored to a span of one file.
//
// Start and End are where the decision was made, forever. They are not where it
// points now — code moves, and Lines is what follows it. Never overwrite the
// recorded range with a resolved one: the fact that a record has travelled 40
// lines is information, and losing it makes drift invisible.
//
// Still absent, on purpose: the tree-sitter AST path from §6, and record
// signing (§7.3), which lands with capture because it only matters once records
// stop being written by a human.
type Record struct {
	ID       string `json:"id"`
	Date     string `json:"date"` // ISO YYYY-MM-DD; sorted lexically, so the format matters
	Source   string `json:"source"`
	File     string `json:"file"`
	Start    int    `json:"line_start"`
	End      int    `json:"line_end"`
	Decision string `json:"decision"`
	Why      string `json:"why"`

	// Lines holds one hash per significant line of the recorded span — the
	// anchor. Empty means a record written before anchoring existed, or by
	// hand; those fall back to exact line ranges and say so when displayed.
	Lines []string `json:"line_hashes,omitempty"`

	// Evidence is what somebody else could check to see whether this record is
	// true. Optional, and most records will not have any — see the Evidence type.
	Evidence []Evidence `json:"evidence,omitempty"`

	// Author is who wrote this — "human" or "agent". Empty means human, which is
	// every record written before capture existed.
	//
	// Not about identity or blame; §7.3's signing covers that. This is about how
	// much human attention the claim has had. A human writing a record is one
	// deliberate act of attention. An agent writing one is zero.
	Author string `json:"author,omitempty"`

	// Verified is the date a human confirmed an agent-written record, empty until
	// then. Human-written records need no entry: writing one IS the confirmation.
	//
	// DECISIONS §17.4 — Wikipedia requires content to be *verifiable* but not
	// verified when first added, and that gap is what circular citation exploits.
	// This closes it for the only records that have the gap.
	Verified string `json:"verified,omitempty"`
}

const (
	authorHuman = "human"
	authorAgent = "agent"
)

// unchecked reports whether a record is asserting something no human has ever
// looked at. Only agent-written records can be in that state.
func (r Record) unchecked() bool {
	return r.Author == authorAgent && r.Verified == ""
}

// Evidence is a pointer to something that can be checked WITHOUT consulting
// another record.
//
// That exclusion is the whole design. A record justified by another record is
// how one wrong record makes the next one look credible, which is the loop
// DECISIONS §17 is about — Wikipedia calls its version citogenesis and its rule
// is the same: link to another article freely, never use one as your source.
// Cross-referencing records for reading is fine; citing one as grounds is not.
//
// Four useful shapes, in descending order of how much they buy:
//
//	a place in the code   auth.go:88-94   — anchored, so its rot is detectable
//	a command             go test ./...   — anyone can re-run it
//	a commit              9f2a1c3         — immutable
//	something external    a ticket, an incident, a review
//
// A code location is the strongest because it gets its own anchor. When the code
// cited as grounds is deleted, whence can say *the evidence for this decision is
// gone* on its own, with no human checking anything.
//
// Deliberately NOT a snippet of the code. A copy goes stale silently, which is
// the disease this is meant to treat.
//
// Records with no evidence are normal and not second-class. Most decisions are
// judgement under constraints ("a per-account lock is a cache-invalidation
// problem nobody asked for") and have no artifact behind them. The point of the
// field is not to force evidence onto everything — it is to stop unfalsifiable
// judgement from reading as established fact because it sits in the same
// paragraph as something checkable.
type Evidence struct {
	Ref string `json:"ref"` // as written, and what gets displayed

	// Set only when Ref named a place in the code, so it can be re-anchored.
	File  string   `json:"file,omitempty"`
	Start int      `json:"line_start,omitempty"`
	End   int      `json:"line_end,omitempty"`
	Lines []string `json:"line_hashes,omitempty"`
}

// anchored reports whether this pointer is at a place in the code, and so has a
// state that can rot.
func (e Evidence) anchored() bool { return len(e.Lines) > 0 }

// asRecord adapts a code-location pointer so it can go through resolveAnchor —
// the same machinery, because the question is the same question.
func (e Evidence) asRecord() Record {
	return Record{File: e.File, Start: e.Start, End: e.End, Lines: e.Lines}
}

// Resolved is a record plus where it actually points in the code as it stands.
// Anchor is a named field rather than embedded because both halves have a Start
// and an End, and that ambiguity is the one bug this type exists to prevent.
type Resolved struct {
	Record
	Anchor  Anchor
	Grounds []Grounded
}

// Grounded is one piece of evidence and, when it names a place in the code, what
// has happened to that place since.
type Grounded struct {
	Evidence
	Anchor Anchor
}

// FindStore walks up from a file looking for the nearest .whence/records.jsonl,
// the way git finds .git. It returns the store path and the directory holding
// the .whence dir — the root that record paths are relative to.
//
// Stores are found by FILE, never by session. An agent working in one repo
// routinely edits files in a sibling repo, and a hook's cwd is the session's
// root rather than the edited file's repo. Resolving from cwd would search the
// wrong repo, then compare an absolute path against repo-relative records, and
// match nothing — silently, which is the worst way for this to fail.
func FindStore(file string) (store, root string, ok bool) {
	dir := filepath.Dir(file)
	for {
		// Line-delimited first, then the legacy array. Both are read; only the
		// first is ever written.
		for _, name := range [...]string{recordsFileName, legacyRecordsName} {
			cand := filepath.Join(dir, storeDirName, name)
			if st, err := os.Stat(cand); err == nil && !st.IsDir() {
				return cand, dir, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir { // hit the filesystem root
			return "", "", false
		}
		dir = parent
	}
}

// Load reads every record from path.
//
// A missing file is deliberately not an error. A repo with no records yet is
// the normal starting state, and the hook has to stay silent rather than fail
// in it — see the fail-open rule in main.go.
//
// ponytail: whole-file JSON read and a linear scan in Match. Fine at the ~20
// hand-written records of Phase 0 (microseconds). Move to SQLite with an index
// on (file, line_start, line_end) once Phase 1 capture makes this thousands of
// records, because PreToolUse blocks every edit and latency becomes visible.
func Load(path string) ([]Record, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// One JSON array is the legacy shape, kept readable so an existing store
	// does not quietly come back empty.
	s := string(b)
	if t := strings.TrimLeft(s, " \t\r\n"); t != "" && t[0] == '[' {
		var rs []Record
		if err := json.Unmarshal(b, &rs); err != nil {
			return nil, err
		}
		return rs, nil
	}

	// One record per line. A malformed line is an error rather than a skip:
	// silently dropping a decision is the failure this whole tool is about, and
	// the line number is what makes it fixable by hand.
	var rs []Record
	for i, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var r Record
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, i+1, err)
		}
		rs = append(rs, r)
	}
	return rs, nil
}

// Match returns the records governing file, each resolved against the file as
// it is now, newest first.
//
// line == 0 means "anywhere in this file", which is what the hook asks for: an
// agent about to edit a file has not told us which line it intends to touch, so
// every record on the file is relevant. A specific line narrows to records whose
// span contains it — the span they point at NOW, which is the entire reason
// anchoring exists. A record that drifted from 142 to 187 has to answer to
// `whence file.go:187`, not to the line it was born on.
//
// root is where record paths are relative to, as reported by FindStore. It is
// only used to read the file back for anchoring, so a caller with no root (a
// test over hand-built records) still gets line-range behaviour.
func Match(root string, rs []Record, file string, line int) []Resolved {
	var out []Resolved
	var lines []string
	read := false

	for _, r := range rs {
		if !samePath(r.File, file) {
			continue
		}
		// One read per Match call, not per record: every record here is on the
		// same file, and PreToolUse blocks the edit that is waiting on us.
		if len(r.Lines) > 0 && !read {
			lines, read = fileLines(filepath.Join(root, file)), true
		}
		a := resolveAnchor(lines, r)

		// An orphan has no current span, so it cannot claim a specific line.
		// It still surfaces in the whole-file view — an agent about to edit a
		// file with a lost decision on it is exactly who needs to know.
		if line != 0 && (a.Start == 0 || line < a.Start || line > a.End) {
			continue
		}
		out = append(out, Resolved{Record: r, Anchor: a, Grounds: resolveEvidence(root, r)})
	}

	// Live anchors first, then newest first. Dates are ISO, so a string compare
	// is a date compare. Ordering is not cosmetic here: renderContext truncates
	// at the context cap, so this decides what an agent actually sees, and a
	// record whose anchor is lost should never crowd out one that holds.
	sort.SliceStable(out, func(i, j int) bool {
		oi := out[i].Anchor.State == StateOrphaned
		oj := out[j].Anchor.State == StateOrphaned
		if oi != oj {
			return oj
		}
		return out[i].Date > out[j].Date
	})
	return out
}

// samePath compares two repo-relative paths, tolerating "./" prefixes and
// separator differences so a record written by hand still matches.
func samePath(a, b string) bool {
	return filepath.ToSlash(filepath.Clean(a)) == filepath.ToSlash(filepath.Clean(b))
}

// Rel converts an absolute path into the repo-relative form records use, given
// the root that FindStore reported.
//
// A file outside root comes back unchanged rather than as a "../.." path: a
// record can only concern a file in its own repo, so an outside path should
// simply fail to match.
func Rel(root, abs string) string {
	if root == "" || !filepath.IsAbs(abs) {
		return abs
	}
	r, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(r, "..") {
		return abs
	}
	return r
}
