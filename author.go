package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Writing records.
//
// Records were hand-authored JSON until anchoring landed. That stops working the
// moment a record carries per-line hashes: nobody computes those by hand, and a
// record with a wrong anchor is worse than one with no anchor, because it points
// somewhere with confidence. So authoring moves into the tool — not as a
// convenience, but because the anchor has to be computed from the file to be
// true at all.

// --- one record ---------------------------------------------------------

func addCmd(args []string) {
	// The target comes first — `why add auth.go:142-148 -d "..."` reads the way
	// every other tool works. Go's flag package stops at the first positional
	// argument, so it has to be lifted out before parsing rather than left for
	// fl.Arg(0), which would silently drop every flag after it.
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		addUsage()
	}
	target := args[0]

	fl := flag.NewFlagSet("add", flag.ExitOnError)
	decision := fl.String("d", "", "the decision, one line")
	why := fl.String("w", "", "why it was made")
	source := fl.String("s", "manual", "where the decision came from")
	if err := fl.Parse(args[1:]); err != nil {
		os.Exit(2)
	}
	if *decision == "" {
		addUsage()
	}

	file, start, end := splitSpan(target)
	if start == 0 {
		fmt.Fprintln(os.Stderr, "why: add needs a line or a range, e.g. auth.go:142-148")
		os.Exit(2)
	}

	rec, store, err := add(file, start, end, *decision, *why, *source)
	if err != nil {
		fmt.Fprintln(os.Stderr, "why:", err)
		os.Exit(1)
	}
	fmt.Println(store)
	// Print it back through the normal display path, resolved. If the anchor
	// this just computed does not read as "exact", the hash is wrong and the
	// record was born orphaned — better to see that now than in three months.
	print1(rec)
}

func addUsage() {
	fmt.Fprintln(os.Stderr, `usage: why add <file>:<start>-<end> -d "decision" [-w "why"] [-s source]`)
	os.Exit(2)
}

// add anchors a decision to a span of a file and appends it to the nearest
// store. It returns the record as it resolves, so the caller can show the
// anchor it just computed rather than assert it worked.
func add(file string, start, end int, decision, why, source string) (Resolved, string, error) {
	abs, err := filepath.Abs(file)
	if err != nil {
		return Resolved{}, "", err
	}
	lines := fileLines(abs)
	if lines == nil {
		return Resolved{}, "", fmt.Errorf("cannot read %s — a record has to be anchored to real lines", file)
	}
	if start < 1 || end < start || end > len(lines) {
		return Resolved{}, "", fmt.Errorf("%s has %d lines; %d-%d is not in it", file, len(lines), start, end)
	}
	hashes := hashSpan(lines[start-1 : end])
	if len(hashes) == 0 {
		return Resolved{}, "", fmt.Errorf("%s:%d-%d is blank — nothing there to anchor to", file, start, end)
	}

	store, root, ok := FindStore(abs)
	if !ok {
		if store, root, err = createStore(); err != nil {
			return Resolved{}, "", err
		}
		fmt.Println("created", filepath.Join(root, storeDirName))
	}
	rs, err := Load(store)
	if err != nil {
		return Resolved{}, "", err
	}

	r := Record{
		// Local date, not UTC. A record's date is read by humans and sorted on;
		// someone east of Greenwich recording a decision after midnight means
		// today, and stamping it yesterday makes the store disagree with their
		// own notes and commits. The surfacing log stays UTC — that one is
		// machine-facing and compared across machines.
		Date:     time.Now().Format("2006-01-02"),
		Source:   source,
		File:     filepath.ToSlash(Rel(root, abs)),
		Start:    start,
		End:      end,
		Decision: decision,
		Why:      why,
		Lines:    hashes,
	}
	r.ID = newID(r)

	if err := save(store, append(rs, r)); err != nil {
		return Resolved{}, "", err
	}
	return Resolved{Record: r, Anchor: resolveAnchor(lines, r)}, store, nil
}

// createStore makes a store in the working directory. Deliberately not "the git
// root": nothing in this tool reads git state (§14), and guessing a root wrong
// puts records where the next clone will not look. The working directory is
// where the developer is standing, and add prints what it created so the guess
// is never silent.
func createStore() (store, root string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(cwd, storeDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	// The surfacing log holds timestamps and absolute local paths, so it stays
	// local while the records themselves are committed on purpose (§14). A
	// nested .gitignore keeps that decision inside the directory it concerns,
	// rather than editing a repo-level file this tool does not own.
	ign := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(ign); os.IsNotExist(err) {
		// *.tmp too: save writes through one, and an interrupted write should
		// not leave a committable artifact behind.
		if err := os.WriteFile(ign, []byte(surfacedLogName+"\n*.tmp\n"), 0o644); err != nil {
			return "", "", err
		}
	}
	store = filepath.Join(dir, recordsFileName)
	if _, err := os.Stat(store); os.IsNotExist(err) {
		if err := os.WriteFile(store, []byte("[]\n"), 0o644); err != nil {
			return "", "", err
		}
	}
	return store, cwd, nil
}

// save writes the store back through a temp file and a rename.
//
// The store IS the product. A half-written records.json loses decisions
// permanently, and this is the one place that rewrites the whole file, so the
// atomic version costs two lines and removes the only data-loss path in the
// tool.
func save(path string, rs []Record) error {
	b, err := json.MarshalIndent(rs, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// newID derives a short id from the record's own content.
//
// ponytail: 4 hex chars, matching the #4f2a shape the README uses. ~65k values,
// so a collision becomes likely around a few hundred records; nothing keys off
// the id yet, so a duplicate is cosmetic today. Widen when `why check` starts
// citing ids in CI output.
func newID(r Record) string {
	sum := sha256.Sum256([]byte(r.File + strconv.Itoa(r.Start) + r.Decision + r.Date))
	return hex.EncodeToString(sum[:2])
}

// splitSpan parses "src/a.go:142-148", "src/a.go:142" and "src/a.go" (0, 0).
func splitSpan(s string) (file string, start, end int) {
	file, start = splitTarget(s)
	if start != 0 {
		return file, start, start
	}
	// splitTarget gave up, which for "a.go:142-148" means the range form.
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return s, 0, 0
	}
	a, b, ok := strings.Cut(s[i+1:], "-")
	if !ok {
		return s, 0, 0
	}
	x, err1 := strconv.Atoi(a)
	y, err2 := strconv.Atoi(b)
	if err1 != nil || err2 != nil {
		return s, 0, 0
	}
	return s[:i], x, y
}

// --- backfill -----------------------------------------------------------

// backfillCmd harvests decisions that are already written down.
//
// The Phase 0 plan calls hand-seeding "the valuable part", and it is right — but
// a `ponytail:` comment IS a hand-authored decision record with a file and a
// line already attached to it. Retyping those into JSON is manual work with no
// judgement in it.
//
// This is the cheap end of Phase 1's backfill: no model, no capture, no guessing
// which reasoning is worth keeping. Somebody already made that call by writing
// the comment. Git history and ADR docs are the same trick against messier
// input, and they come next.
func backfillCmd(args []string) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "why:", err)
		os.Exit(1)
	}

	added, skipped := 0, 0
	err = filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // an unreadable directory is not a reason to stop
		}
		if d.IsDir() {
			if skipDir[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		lines := readSource(p)
		if lines == nil {
			return nil
		}
		for _, f := range harvest(lines) {
			decision, why := firstSentence(f.text)
			rel, _ := filepath.Rel(abs, p)

			// Idempotent: re-running must not duplicate. Keyed on file plus
			// decision text rather than on the line range, because the whole
			// point of anchoring is that the range moves.
			if store, root, ok := FindStore(p); ok {
				if rs, err := Load(store); err == nil && has(rs, filepath.ToSlash(Rel(root, p)), decision) {
					skipped++
					continue
				}
			}
			r, _, err := add(p, f.start, f.end, decision, why, backfillSource)
			if err != nil {
				fmt.Fprintf(os.Stderr, "why: %s:%d — %v\n", rel, f.start, err)
				continue
			}
			added++
			print1(r)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "why:", err)
		os.Exit(1)
	}
	fmt.Printf("\n%d record(s) added, %d already present\n", added, skipped)
}

const (
	marker         = "ponytail:"
	backfillSource = "ponytail comment"
	maxSourceBytes = 1 << 20
)

var skipDir = map[string]bool{
	".git": true, storeDirName: true, "node_modules": true,
	"dist": true, "build": true, "vendor": true, ".next": true,
}

// found is one harvested note: the comment block plus the declaration it sits
// above, which is the code the note is actually about.
type found struct {
	start, end int
	text       string
}

// readSource reads a file if it plausibly holds source. Binary sniffing is a NUL
// byte in the first 512 bytes — the same heuristic git uses, and wrong only for
// files nobody writes comments in.
func readSource(p string) []string {
	st, err := os.Stat(p)
	if err != nil || st.Size() > maxSourceBytes {
		return nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	head := b
	if len(head) > 512 {
		head = head[:512]
	}
	for _, c := range head {
		if c == 0 {
			return nil
		}
	}
	if len(b) == 0 {
		return nil
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

// harvest finds every ponytail note in a file and the span it concerns.
//
// The span runs from the marker line through the first line that is not a
// comment — the declaration the note sits above. Anchoring to the comment alone
// would be circular: the record would survive exactly as long as the comment
// does, and re-surfacing a comment the agent can already see is worth nothing.
// Including the declaration means rewriting the code decays the anchor, which is
// the behaviour that matters.
//
// A note is a comment line that OPENS with the marker. Not a line containing the
// word — that harvests the constant defining the marker, every fixture quoting
// it, and every comment discussing it, all of which are in this file. The cost is
// that a note trailing after code is missed; notes are written on their own line
// by convention, and a missed note is recoverable in a way that a garbage record
// in a committed, shared store is not.
//
// ponytail: comment syntax by prefix, not by parser. Covers //, # and block-
// comment continuations, which is every language in this repo. A note inside a
// language that comments some other way is simply not harvested — silent, but
// silent-and-absent rather than silent-and-wrong.
func harvest(lines []string) []found {
	var out []found
	for i := 0; i < len(lines); i++ {
		body, isComment := commentBody(lines[i])
		if !isComment || !strings.HasPrefix(body, marker) {
			continue
		}
		text := strings.TrimSpace(strings.TrimPrefix(body, marker))
		j := i + 1
		for j < len(lines) {
			c, isComment := commentBody(lines[j])
			if !isComment || c == "" {
				break
			}
			text += " " + c
			j++
		}
		end := j + 1 // 1-based: the line after the comment block
		if end > len(lines) {
			end = len(lines)
		}
		if text != "" {
			out = append(out, found{start: i + 1, end: end, text: text})
		}
		i = j
	}
	return out
}

// commentBody returns a comment line's text and whether the line is a comment at
// all.
func commentBody(s string) (string, bool) {
	t := strings.TrimSpace(s)
	if t == "*/" {
		return "", true // ends a block comment: a comment, but no content
	}
	for _, p := range []string{"//", "#", "*"} {
		if strings.HasPrefix(t, p) {
			return strings.TrimSpace(strings.TrimPrefix(t, p)), true
		}
	}
	return "", false
}

// firstSentence splits a note into the shortcut and the reasoning. The first
// sentence of a ponytail note is always the corner that was cut; the rest is why
// it was acceptable and when to revisit.
func firstSentence(s string) (decision, why string) {
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i+1], strings.TrimSpace(s[i+2:])
	}
	return s, ""
}

func has(rs []Record, file, decision string) bool {
	for _, r := range rs {
		if samePath(r.File, file) && r.Decision == decision {
			return true
		}
	}
	return false
}
