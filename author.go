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
	var evidence multiFlag
	fl.Var(&evidence, "e", "something checkable: a file:line, a command, a commit, a link (repeatable)")
	asAgent := fl.Bool("agent", false, "an agent wrote this, so it needs a human to confirm it")
	if err := fl.Parse(args[1:]); err != nil {
		os.Exit(2)
	}
	if *decision == "" {
		addUsage()
	}

	file, start, end := splitSpan(target)
	if start == 0 {
		fmt.Fprintln(os.Stderr, "whence: add needs a line or a range, e.g. auth.go:142-148")
		os.Exit(2)
	}

	rec, store, err := add(file, start, end, *decision, *why, *source, author(*asAgent), evidence)
	if err != nil {
		fmt.Fprintln(os.Stderr, "whence:", err)
		os.Exit(1)
	}
	fmt.Println(store)
	// Print it back through the normal display path, resolved. If the anchor
	// this just computed does not read as "exact", the hash is wrong and the
	// record was born orphaned — better to see that now than in three months.
	print1(rec)
}

func author(asAgent bool) string {
	if asAgent {
		return authorAgent
	}
	return authorHuman
}

// confirmCmd records that a human has read an agent-written record and stands
// behind it.
//
// This is the whole human-attention budget of the design, and it is deliberately
// one command on one record. §17.7's warning applies: a gate hit too often gets
// rubber-stamped, and a rubber-stamped gate is worse than none because it
// launders unchecked claims as checked ones. If this ever feels like a chore,
// capture is writing too much — fix that end, not this one.
func confirmCmd(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: why confirm <id>")
		os.Exit(2)
	}
	store, root, rs := openStore()

	for i := range rs {
		if rs[i].ID != args[0] {
			continue
		}
		if rs[i].Verified != "" {
			fmt.Printf("[%s] was already confirmed on %s\n", rs[i].ID, rs[i].Verified)
			return
		}
		rs[i].Verified = time.Now().Format("2006-01-02")
		if err := save(store, rs); err != nil {
			fmt.Fprintln(os.Stderr, "whence:", err)
			os.Exit(1)
		}
		fmt.Printf("confirmed [%s]\n", rs[i].ID)
		print1(Resolved{
			Record:  rs[i],
			Anchor:  resolveAnchor(fileLines(filepath.Join(root, rs[i].File)), rs[i]),
			Grounds: resolveEvidence(root, rs[i]),
		})
		return
	}
	fmt.Fprintf(os.Stderr, "whence: no record [%s] in %s\n", args[0], store)
	os.Exit(1)
}

// --- re-pointing evidence -----------------------------------------------

// regroundCmd replaces a record's evidence without retracting the record.
//
// Evidence rots independently of the claim it supports: the code cited as
// grounds gets edited, the pointer goes weak or orphaned, and the record itself
// stays perfectly anchored and perfectly true. Fixing that used to mean `rm`
// then `add` — and `rm` writes to retracted.jsonl, the log whose entire purpose
// is counting how often a record turned out to be WRONG. Routine bookkeeping
// would have inflated the one number that measures whether this store can be
// trusted, so re-pointing needs its own verb.
//
// The whole list is replaced rather than appended to. Grounds are a set of
// claims about what makes a record true, and an append-only flag would leave
// the stale pointer sitting next to the one that replaced it — two answers to
// the same question, which is the condition this command exists to end. No -e
// at all is therefore a deliberate way to say the record now rests on nothing
// checkable, which is honest and sometimes correct.
func regroundCmd(args []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, `usage: whence reground <id> -e <evidence> [-e ...]`)
		os.Exit(2)
	}
	fl := flag.NewFlagSet("reground", flag.ExitOnError)
	var evidence multiFlag
	fl.Var(&evidence, "e", "what makes this record true, re-pointed (repeatable)")
	if err := fl.Parse(args[1:]); err != nil {
		os.Exit(2)
	}

	rec, store, err := reground(args[0], evidence)
	if err != nil {
		fmt.Fprintln(os.Stderr, "whence:", err)
		os.Exit(1)
	}
	fmt.Println(store)
	// Resolved, so a pointer that was already stale when it was typed shows as
	// stale immediately rather than three months from now.
	print1(rec)
}

// reground swaps in a fresh evidence list, anchored against the files as they
// stand now, and returns the record as it resolves.
func reground(id string, refs []string) (Resolved, string, error) {
	store, root, rs := openStore()
	for i := range rs {
		if rs[i].ID != id {
			continue
		}
		ev, err := buildEvidence(root, refs)
		if err != nil {
			return Resolved{}, store, err
		}
		rs[i].Evidence = ev
		if err := save(store, rs); err != nil {
			return Resolved{}, store, err
		}
		return Resolved{
			Record:  rs[i],
			Anchor:  resolveAnchor(fileLines(filepath.Join(root, rs[i].File)), rs[i]),
			Grounds: resolveEvidence(root, rs[i]),
		}, store, nil
	}
	return Resolved{}, store, fmt.Errorf("no record [%s] in %s", id, store)
}

// --- re-pointing the record itself --------------------------------------

// reanchorCmd re-points a record's own anchor at the code as it stands now.
//
// The twin of reground, for the other half of a record. Both halves rot on
// their own: grounds get deleted while the claim holds, and a block gets
// rewritten in place while the decision about it is still exactly right. `check`
// reports that erosion, and the only answer available was rm plus add — which
// writes to retracted.jsonl, the log whose entire purpose is counting how often
// a record turned out to be WRONG. Re-pointing a live claim is not a
// retraction, and putting that bookkeeping in that log would have destroyed the
// one number measuring whether this store can be trusted. Same argument as
// reground, other half of the record.
//
// ONLY the hashes are rewritten. Start and End stay where the decision was made,
// per the invariant on Record: how far a record has travelled is information,
// and a reanchor has no reason to burn it. The record goes on reading "now at
// 245-258, recorded at 209-222" — which is true, and strictly more than it could
// say if the recorded range were overwritten with the resolved one.
//
// This does not confirm anything about the decision. A human deciding the claim
// still holds is the input to running this, not something it can attest to.
func reanchorCmd(args []string) {
	if len(args) != 2 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "usage: whence reanchor <id> <file>:<start>-<end>")
		os.Exit(2)
	}
	rec, store, err := reanchor(args[0], args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "whence:", err)
		os.Exit(1)
	}
	fmt.Println(store)
	// Resolved, so an anchor that did not come out exact is visible immediately
	// rather than three months from now — same reason add prints itself back.
	print1(rec)
}

// reanchor re-hashes a record against an explicitly named span of its own file.
//
// The span is always the human's, never inherited from where the record
// currently resolves. A degraded record's span is a best-match window of a fixed
// number of significant lines, so it routinely sits a line or two off the real
// block — and re-hashing that window would turn a best guess into a stored
// certainty, which is exactly the confident-wrong-line failure anchor.go exists
// to prevent. The window's line numbers are printed by `why` and by `check`, so
// agreeing with it costs a copy and a paste; the point is that agreeing is an
// act somebody performs.
func reanchor(id, target string) (Resolved, string, error) {
	store, root, rs := openStore()
	for i := range rs {
		if rs[i].ID != id {
			continue
		}
		f, start, end := splitSpan(target)
		if start == 0 {
			return Resolved{}, store, fmt.Errorf(
				"reanchor needs the lines the decision is about now, e.g. %s:142-148", rs[i].File)
		}
		// A decision whose code moved to another file is a different record: its
		// recorded range would name lines in a file it no longer concerns, and
		// every "recorded at" it printed from then on would be a claim about the
		// wrong file. Refused rather than quietly rewritten.
		fabs, err := filepath.Abs(f)
		if err != nil {
			return Resolved{}, store, err
		}
		if rel := filepath.ToSlash(Rel(root, fabs)); !samePath(rel, rs[i].File) {
			return Resolved{}, store, fmt.Errorf(
				"record [%s] is about %s, not %s — a decision that moved to another file is a new record. Add it there, and retract this one with a reason", id, rs[i].File, rel)
		}

		abs := filepath.Join(root, rs[i].File)
		lines := fileLines(abs)
		hashes, err := anchorSpan(lines, rs[i].File, start, end)
		if err != nil {
			return Resolved{}, store, err
		}
		rs[i].Lines = hashes
		if err := save(store, rs); err != nil {
			return Resolved{}, store, err
		}
		return Resolved{
			Record:  rs[i],
			Anchor:  resolveAnchor(lines, rs[i]),
			Grounds: resolveEvidence(root, rs[i]),
		}, store, nil
	}
	return Resolved{}, store, fmt.Errorf("no record [%s] in %s", id, store)
}

// openStore finds the store from the working directory and loads it, or exits.
func openStore() (store, root string, rs []Record) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "whence:", err)
		os.Exit(1)
	}
	store, root, ok := FindStore(filepath.Join(cwd, "x"))
	if !ok {
		fmt.Fprintf(os.Stderr, "no %s/%s found above %s\n", storeDirName, recordsFileName, cwd)
		os.Exit(1)
	}
	rs, err = Load(store)
	if err != nil {
		fmt.Fprintln(os.Stderr, "whence:", err)
		os.Exit(1)
	}
	return store, root, rs
}

// multiFlag is a flag that may be given more than once.
type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ", ") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

// buildEvidence turns what the author typed into pointers, anchoring the ones
// that name a place in the code.
//
// Two things are refused rather than accepted quietly, both because a bad
// pointer is silent forever otherwise:
//
//   - anything aimed at the record store, which is the citogenesis rule made
//     literal (DECISIONS §17);
//   - anything that reads as file:lines but cannot be read there, which is
//     almost always a typo — and stored as plain text it would look fine while
//     buying none of the rot detection that made it worth writing.
func buildEvidence(root string, refs []string) ([]Evidence, error) {
	var out []Evidence
	for _, ref := range refs {
		if strings.Contains(ref, storeDirName) {
			return nil, fmt.Errorf(
				"evidence %q points into the record store — a record may reference another for reading, but never cite one as its grounds, because that is the link that lets one wrong record make the next look credible (§17)", ref)
		}

		e := Evidence{Ref: ref}
		f, start, end := splitSpan(ref)

		// A URL has a colon and digits in it and is not a line range.
		if start == 0 || strings.Contains(ref, "://") {
			out = append(out, e)
			continue
		}

		lines := fileLines(filepath.Join(root, f))
		if lines == nil || start < 1 || end < start || end > len(lines) {
			return nil, fmt.Errorf(
				"evidence %q reads as a line range but %s has no lines %d-%d. If it is not a file, write it in a form that cannot be mistaken for one",
				ref, f, start, end)
		}
		hs := hashSpan(lines[start-1 : end])
		if len(hs) == 0 || !identifiable(hs, counts(significant(lines))) {
			return nil, fmt.Errorf(
				"evidence %q is blank or has nothing distinctive in it, so its own anchor could not be trusted. Point at lines unique to what you mean", ref)
		}
		e.File, e.Start, e.End, e.Lines = filepath.ToSlash(f), start, end, hs
		out = append(out, e)
	}
	return out, nil
}

func addUsage() {
	fmt.Fprintln(os.Stderr, `usage: why add <file>:<start>-<end> -d "decision" [-w "why"] [-s source]`)
	os.Exit(2)
}

// anchorSpan hashes file:start-end into an anchor, refusing anything that could
// not be found again.
//
// The last check is the one worth having. A span whose every line occurs all
// over the file has nothing to distinguish this block from any other, so the
// moment the code moves the anchor lands on whichever lookalike is nearest.
// Failing here is worth much more than failing at lookup: right now the author
// is looking at the file and can widen the span, which is the actual fix.
func anchorSpan(lines []string, file string, start, end int) ([]string, error) {
	if lines == nil {
		return nil, fmt.Errorf("cannot read %s — a record has to be anchored to real lines", file)
	}
	if start < 1 || end < start || end > len(lines) {
		return nil, fmt.Errorf("%s has %d lines; %d-%d is not in it", file, len(lines), start, end)
	}
	hashes := hashSpan(lines[start-1 : end])
	if len(hashes) == 0 {
		return nil, fmt.Errorf("%s:%d-%d is blank — nothing there to anchor to", file, start, end)
	}
	if !identifiable(hashes, counts(significant(lines))) {
		return nil, fmt.Errorf(
			"%s:%d-%d has nothing distinctive in it — every line appears elsewhere in the file, so this anchor could not tell the block apart from any lookalike. Widen the span to include a line unique to it",
			file, start, end)
	}
	return hashes, nil
}

// add anchors a decision to a span of a file and appends it to the nearest
// store. It returns the record as it resolves, so the caller can show the
// anchor it just computed rather than assert it worked.
func add(file string, start, end int, decision, why, source, author string, evidence []string) (Resolved, string, error) {
	abs, err := filepath.Abs(file)
	if err != nil {
		return Resolved{}, "", err
	}
	lines := fileLines(abs)
	hashes, err := anchorSpan(lines, file, start, end)
	if err != nil {
		return Resolved{}, "", err
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

	ev, err := buildEvidence(root, evidence)
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
		Evidence: ev,
		Author:   author,
	}
	// A human writing a record is the confirmation. Only an agent's record waits
	// for one, which is the whole point of tracking who wrote it.
	if author != authorAgent {
		r.Verified = r.Date
	}
	r.ID = newID(r)

	if err := save(store, append(rs, r)); err != nil {
		return Resolved{}, "", err
	}
	return Resolved{
		Record:  r,
		Anchor:  resolveAnchor(lines, r),
		Grounds: resolveEvidence(root, r),
	}, store, nil
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
// Was 4 hex chars, matching the #4f2a shape in the README, on the grounds that
// nothing keyed off an id so a collision was cosmetic. `why check` cites ids in
// CI output, where a human reads one and looks it up — a duplicate stops being
// cosmetic there. Widened to 6 on that trigger, which the record covering this
// function named as its own condition.
//
// ponytail: 6 hex chars, ~16.7M values, so collisions get likely somewhere
// around a few thousand records. Records already written keep their 4-char ids;
// nothing parses an id's length. Widen again, or key on a real hash, if the
// store ever gets that big.
func newID(r Record) string {
	sum := sha256.Sum256([]byte(r.File + strconv.Itoa(r.Start) + r.Decision + r.Date))
	return hex.EncodeToString(sum[:3])
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

// --- removing one -------------------------------------------------------

// rmCmd deletes a record by id.
//
// Exists because `why check` tells you an orphaned record must be re-anchored or
// deleted deliberately, and advice you cannot act on is worse than none. Deleting
// is the honest end of an orphan's life: the code a decision described is gone,
// so either the decision moved (re-add it, anchored to where it lives now) or it
// stopped being true.
//
// No confirmation prompt: the store is committed, so `git checkout` is the undo,
// and it is a better one than anything this could offer.
//
// Every removal is logged to .whence/retracted.jsonl, which is committed like the
// store itself. That log is the second instrument DECISIONS §17.6 argues for.
// §8's falsification number counts times whence caught a contradiction — and a
// store full of confident nonsense produces MORE catches, so that number rises
// while the tool rots. The only way to see the rot is to count how often a record
// turned out to be wrong, which means a deletion cannot be silent.
func rmCmd(args []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, `usage: why rm <id> [-w "why it was wrong"]`)
		os.Exit(2)
	}
	id := args[0]
	fl := flag.NewFlagSet("rm", flag.ExitOnError)
	reason := fl.String("w", "", "why this record is being retracted")
	if err := fl.Parse(args[1:]); err != nil {
		os.Exit(2)
	}

	store, root, rs := openStore()

	kept := make([]Record, 0, len(rs))
	var gone *Record
	for i, r := range rs {
		if r.ID == id {
			gone = &rs[i]
			continue
		}
		kept = append(kept, r)
	}
	if gone == nil {
		fmt.Fprintf(os.Stderr, "whence: no record [%s] in %s\n", id, store)
		os.Exit(1)
	}
	if err := save(store, kept); err != nil {
		fmt.Fprintln(os.Stderr, "whence:", err)
		os.Exit(1)
	}
	appendRetracted(root, *gone, *reason)
	fmt.Printf("removed [%s] %s:%d-%d — %s\n",
		gone.ID, gone.File, gone.Start, gone.End, gone.Decision)
	if *reason == "" {
		fmt.Println("no reason given. `-w \"...\"` next time — the retraction log is how you find out how often this tool is wrong.")
	}
}

// appendRetracted logs a removal. Committed, unlike the surfacing log, because
// this one is evidence about the store's own accuracy and is worthless if it only
// exists on the machine that happened to do the deleting.
func appendRetracted(root string, r Record, reason string) {
	f, err := os.OpenFile(filepath.Join(root, storeDirName, retractedLogName),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return // never fail a removal over bookkeeping
	}
	defer f.Close()
	_ = json.NewEncoder(f).Encode(map[string]any{
		"at":       time.Now().Format("2006-01-02"),
		"id":       r.ID,
		"file":     r.File,
		"decision": r.Decision,
		"author":   r.Author,
		"reason":   reason,
	})
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
		fmt.Fprintln(os.Stderr, "whence:", err)
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
			r, _, err := add(p, f.start, f.end, decision, why, f.src, authorHuman, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "whence: %s:%d — %v\n", rel, f.start, err)
				continue
			}
			added++
			print1(r)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "whence:", err)
		os.Exit(1)
	}
	fmt.Printf("\n%d record(s) added, %d already present\n", added, skipped)
}

const maxSourceBytes = 1 << 20

// The marker set: which comment prefixes count as a decision.
//
// This is the whole reach of backfill. A store that starts empty is a tool that
// does nothing on day one, and `ponytail:` alone only exists in repos already
// using that plugin — so in anyone else's repository backfill found nothing and
// whence arrived useless.
//
// The markers people actually write split cleanly in two, and the split is what
// keeps recall from costing precision.
var (
	// alwaysMarkers are never written about something obvious. The word itself
	// is the admission that a choice was made against a constraint, so the
	// comment is a decision by construction.
	alwaysMarkers = []string{"ponytail:", "HACK:", "WORKAROUND:", "XXX:", "GOTCHA:"}

	// reasonMarkers are mostly descriptive. Harvesting them wholesale fills a
	// committed, shared store with "fix this later", which is a task wearing a
	// decision's clothes — and a store full of those is one people stop reading,
	// which is worse than an empty one. They are admitted only when the note says
	// WHY, because that is the only part a record exists to carry.
	reasonMarkers = []string{"NOTE:", "TODO:", "FIXME:", "WARNING:", "CAVEAT:"}

	// reasonWords are what giving a reason sounds like. Deliberately a small,
	// boring list: every word here is one people write without thinking about it,
	// which is the point — a heuristic nobody has to learn.
	//
	// ponytail: conservative word list, so real reasons phrased around it are
	// missed — "the upstream 502s, so one attempt fails" has no word from this
	// list. Left narrow on purpose: a missed note is recoverable by hand, and a
	// garbage record in a committed shared store is not. Widen only against real
	// misses from a real repo, never by imagining phrasings.
	reasonWords = []string{
		"because", "so that", "otherwise", "since", "to avoid",
		"rather than", "instead of", "reason", "caused",
	}
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
	src        string // which marker produced it, recorded as the record's source
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
		if !isComment {
			continue
		}
		text, src, needsReason, ok := openingMarker(body)
		if !ok {
			continue
		}
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
		// The reason test runs on the WHOLE note, not the opening line. People
		// state the shortcut first and justify it underneath, which is the shape
		// worth harvesting; testing the first line alone would reject exactly the
		// well-written ones.
		if text != "" && (!needsReason || hasReason(text)) {
			out = append(out, found{start: i + 1, end: end, text: text, src: src})
		}
		i = j
	}
	return out
}

// openingMarker matches a comment body against the marker set, returning the
// note with the marker stripped, the source to record it under, and whether it
// still has to prove itself by giving a reason.
func openingMarker(body string) (text, src string, needsReason, ok bool) {
	for _, m := range alwaysMarkers {
		if t, hit := afterMarker(body, m); hit {
			return t, sourceFor(m), false, true
		}
	}
	for _, m := range reasonMarkers {
		if t, hit := afterMarker(body, m); hit {
			return t, sourceFor(m), true, true
		}
	}
	return "", "", false, false
}

// afterMarker strips a marker from the front of a comment body, tolerating the
// owner form (`TODO(amogh):`) that every codebase writes at least once.
//
// Anchored to the START of the body on purpose. A line merely containing the
// word harvests the constant defining it, every test fixture quoting it, and
// every comment discussing it — all of which live in this file.
func afterMarker(body, m string) (string, bool) {
	if strings.HasPrefix(body, m) {
		return strings.TrimSpace(strings.TrimPrefix(body, m)), true
	}
	name := strings.TrimSuffix(m, ":")
	if !strings.HasPrefix(body, name+"(") {
		return "", false
	}
	if i := strings.Index(body, "):"); i > 0 {
		return strings.TrimSpace(body[i+2:]), true
	}
	return "", false
}

// hasReason reports whether a note explains itself rather than only issuing an
// instruction. This is the entire filter that lets ordinary markers in without
// letting the noise in with them.
func hasReason(s string) bool {
	l := strings.ToLower(s)
	for _, w := range reasonWords {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

// sourceFor turns a marker into the source string a record carries, so `whence
// log` says where a decision was found. "ponytail:" stays "ponytail comment",
// which is what the records written before the marker set already say.
func sourceFor(m string) string {
	return strings.TrimSuffix(m, ":") + " comment"
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
