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
// Phase 0 anchors are hand-written and therefore exact: a repo-relative path
// plus a line range. Phase 1 adds a normalised content hash and a tree-sitter
// AST path so a record survives the code moving, plus a confidence score that
// decays as the anchor drifts and an explicit orphaned state when it no longer
// attaches to anything. None of that exists yet, on purpose — see
// "01 - Phase 0 Plan" in the vault.
type Record struct {
	ID       string `json:"id"`
	Date     string `json:"date"` // ISO YYYY-MM-DD; sorted lexically, so the format matters
	Source   string `json:"source"`
	File     string `json:"file"`
	Start    int    `json:"line_start"`
	End      int    `json:"line_end"`
	Decision string `json:"decision"`
	Why      string `json:"why"`
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

// Match returns the records governing file, newest first.
//
// line == 0 means "anywhere in this file", which is what the hook asks for: an
// agent about to edit a file has not told us which line it intends to touch, so
// every record on the file is relevant. A specific line narrows to records
// whose span contains it.
func Match(rs []Record, file string, line int) []Record {
	var out []Record
	for _, r := range rs {
		if !samePath(r.File, file) {
			continue
		}
		if line != 0 && (line < r.Start || line > r.End) {
			continue
		}
		out = append(out, r)
	}
	// Newest first. Dates are ISO, so a string compare is a date compare.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date > out[j].Date })
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
