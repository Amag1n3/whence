package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	storeDirName    = ".whence"
	recordsFileName = "records.json"
	surfacedLogName = "surfaced.jsonl"
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
}

// Resolved is a record plus where it actually points in the code as it stands.
// Anchor is a named field rather than embedded because both halves have a Start
// and an End, and that ambiguity is the one bug this type exists to prevent.
type Resolved struct {
	Record
	Anchor Anchor
}

// FindStore walks up from a file looking for the nearest .whence/records.json,
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
		cand := filepath.Join(dir, storeDirName, recordsFileName)
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return cand, dir, true
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
	var rs []Record
	if err := json.Unmarshal(b, &rs); err != nil {
		return nil, err
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
// `why file.go:187`, not to the line it was born on.
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
		out = append(out, Resolved{Record: r, Anchor: a})
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
