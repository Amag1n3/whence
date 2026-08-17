package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
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
	// The target comes first — `whence add auth.go:142-148 -d "..."` reads the way
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

	// -s agent and -agent read identically at the call site, but only -agent
	// set Author. Every ROF record was written -s agent and stored as human
	// with a self-issued Verified date, so the §17 UNCHECKED guard was off.
	// Honour the label. Do not migrate existing records.
	rec, store, err := add(file, start, end, *decision, *why, *source, author(*asAgent || *source == "agent"), evidence)
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
		fmt.Fprintln(os.Stderr, "usage: whence confirm <id>")
		os.Exit(2)
	}
	store, root, rs := openStore()

	if rec, ok, err := promotePending(root, store, rs, args[0]); err != nil {
		fmt.Fprintln(os.Stderr, "whence:", err)
		os.Exit(1)
	} else if ok {
		fmt.Printf("confirmed [%s]\n", rec.ID)
		print1(rec)
		return
	}

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
			Anchor:  resolveAnchor(fileLinesWithin(filepath.Join(root, rs[i].File), root), rs[i]),
			Grounds: resolveEvidence(root, rs[i]),
		})
		return
	}
	fmt.Fprintf(os.Stderr, "whence: no record [%s] in %s\n", args[0], store)
	os.Exit(1)
}

// promotePending moves a pending record into the shared store after re-resolving
// its anchor. Only an intact span can be promoted: the recorded text has to be
// byte-identical to what the agent explained, possibly at a new line number.
// Anything else is a claim about text nobody has checked as it now stands, and
// stamping Verified on it would promote a guess to a stored certainty (§18.1).
// A revert is the orphaned case of the same rule — the §30 revert rule,
// proposed, not settled. Returns ok=false when id is not pending.
func promotePending(root, store string, rs []Record, id string) (Resolved, bool, error) {
	prs, err := Load(pendingFile(root))
	if err != nil {
		return Resolved{}, false, err
	}
	for i, r := range prs {
		if r.ID != id {
			continue
		}
		lines := fileLinesWithin(filepath.Join(root, r.File), root)
		a := resolveAnchor(lines, r)
		if a.State != StateExact && a.State != StateDrifted {
			return Resolved{}, true, fmt.Errorf("record [%s] cannot be promoted — the span now reads as %s. Write it in your own words with whence add, or drop it with whence rm %s", id, a.State, id)
		}
		if err := admit(root, r.File, r.Start, r.Decision, r.Why); err != nil {
			return Resolved{}, true, err
		}
		r.Verified = time.Now().Format("2006-01-02")
		if err := save(store, append(rs, r)); err != nil {
			return Resolved{}, true, err
		}
		if err := save(pendingFile(root), append(prs[:i], prs[i+1:]...)); err != nil {
			return Resolved{}, true, err
		}
		return Resolved{
			Record:  r,
			Anchor:  a,
			Grounds: resolveEvidence(root, r),
		}, true, nil
	}
	return Resolved{}, false, nil
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
			Anchor:  resolveAnchor(fileLinesWithin(filepath.Join(root, rs[i].File), root), rs[i]),
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
		lines := fileLinesWithin(abs, root)
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

		lines := fileLinesWithin(filepath.Join(root, f), root)
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
	fmt.Fprintln(os.Stderr, `usage: whence add <file>:<start>-<end> -d "decision" [-w "why"] [-s source]`)
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
	store, root, ok := FindStore(abs)
	if !ok {
		if store, root, err = createStore(); err != nil {
			return Resolved{}, "", err
		}
		fmt.Println("created", filepath.Join(root, storeDirName))
	}

	r, lines, err := makeRecord(abs, root, file, start, end, decision, why, source, author, evidence)
	if err != nil {
		return Resolved{}, "", err
	}

	rs, err := Load(store)
	if err != nil {
		return Resolved{}, "", err
	}
	if err := save(store, append(rs, r)); err != nil {
		return Resolved{}, "", err
	}
	return Resolved{
		Record:  r,
		Anchor:  resolveAnchor(lines, r),
		Grounds: resolveEvidence(root, r),
	}, store, nil
}

func pendingFile(root string) string {
	return filepath.Join(root, storeDirName, pendingLogName)
}

// makeRecord builds a Record the same way add does — same anchor, same id, same
// refusals — without choosing a destination file. hookPost writes the result to
// pending.jsonl; add writes it to the shared store.
func makeRecord(abs, root, file string, start, end int, decision, why, source, author string, evidence []string) (Record, []string, error) {
	// A record can only concern a file inside its own repo — outside it there
	// is nothing to anchor to, and an outside file must not even be READ on a
	// crafted record's say-so (the anchor verdict is a one-bit hash oracle).
	// This check has to precede the read below: anchorSpan hashes the file, so
	// asking "is this mine to read" after reading it answers too late.
	if outsideRoot(abs, root) {
		return Record{}, nil, fmt.Errorf("whence: %s is outside the repo rooted at %s", file, root)
	}

	if err := admit(root, file, start, decision, why); err != nil {
		return Record{}, nil, err
	}

	lines := fileLinesWithin(abs, root)
	hashes, err := anchorSpan(lines, file, start, end)
	if err != nil {
		return Record{}, nil, err
	}

	ev, err := buildEvidence(root, evidence)
	if err != nil {
		return Record{}, nil, err
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
	return r, lines, nil
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
		if err := os.WriteFile(ign, []byte(surfacedLogName+"\n"+pendingLogName+"\n*.tmp\n"), 0o644); err != nil {
			return "", "", err
		}
	}
	// Union merge, for the same reason and in the same place as the .gitignore
	// above. The store is committed so that records travel with the repo (§14),
	// which means it gets merged — and two people adding unrelated decisions on
	// two branches both append at the end of the file, which is one region to
	// git and therefore a conflict on every parallel add. A store that conflicts
	// whenever two people use it is one a team stops writing to.
	//
	// Union takes both sides, which for an append-only log is simply correct.
	// The cost is narrow and worth naming: a record EDITED on both branches —
	// reanchor, confirm — survives twice, so one id shows up in `whence log`
	// twice. Visible when it happens, rare because records are appended far more
	// than they are changed, and cheaper than the alternative.
	attr := filepath.Join(dir, attributesName)
	if _, err := os.Stat(attr); os.IsNotExist(err) {
		if err := os.WriteFile(attr, []byte(recordsFileName+" merge=union\n"), 0o644); err != nil {
			return "", "", err
		}
	}
	// An empty line-delimited store is an empty file, not "[]".
	store = filepath.Join(dir, recordsFileName)
	if _, err := os.Stat(store); os.IsNotExist(err) {
		if err := os.WriteFile(store, nil, 0o644); err != nil {
			return "", "", err
		}
	}
	return store, cwd, nil
}

// save writes the store back through a temp file and a rename.
//
// The store IS the product. A half-written store loses decisions permanently,
// and this is the one place that rewrites the whole file, so the atomic version
// costs two lines and removes the only data-loss path in the tool.
//
// One record per line, compact, never indented. The store is committed and
// therefore merged, and git merges lines — so the record has to BE the line. An
// indented array put every record across a dozen lines and wrapped the whole
// thing in brackets, which meant two people adding unrelated decisions on two
// branches conflicted, and resolving it meant repairing JSON syntax by hand.
// With the record on one line, and the union driver createStore writes, the same
// merge takes both sides and needs nobody.
//
// A legacy array store is rewritten line-delimited on its first save, under its
// old name. Renaming a committed file out from under someone is not this
// function's call to make; Load reads either shape, so nothing breaks.
func save(path string, rs []Record) error {
	var b strings.Builder
	for _, r := range rs {
		line, err := json.Marshal(r)
		if err != nil {
			return err
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// newID derives a short id from the record's own content.
//
// Was 4 hex chars, matching the #4f2a shape in the README, on the grounds that
// nothing keyed off an id so a collision was cosmetic. `whence check` cites ids in
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
// Exists because `whence check` tells you an orphaned record must be re-anchored or
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
		fmt.Fprintln(os.Stderr, `usage: whence rm <id> [-w "why it was wrong"]`)
		os.Exit(2)
	}
	id := args[0]
	fl := flag.NewFlagSet("rm", flag.ExitOnError)
	reason := fl.String("w", "", "why this record is being retracted")
	if err := fl.Parse(args[1:]); err != nil {
		os.Exit(2)
	}

	store, root, rs := openStore()

	if gone, ok, err := dropPending(root, id); err != nil {
		fmt.Fprintln(os.Stderr, "whence:", err)
		os.Exit(1)
	} else if ok {
		fmt.Printf("removed [%s] %s:%d-%d — %s\n",
			gone.ID, gone.File, gone.Start, gone.End, gone.Decision)
		return
	}

	gone, err := removeRecord(store, root, rs, id, *reason)
	if err != nil {
		fmt.Fprintln(os.Stderr, "whence:", err)
		os.Exit(1)
	}
	fmt.Printf("removed [%s] %s:%d-%d — %s\n",
		gone.ID, gone.File, gone.Start, gone.End, gone.Decision)
	if *reason == "" {
		fmt.Println("no reason given. `-w \"...\"` next time — the retraction log is how you find out how often this tool is wrong.")
	}
}

func dropPending(root, id string) (Record, bool, error) {
	path := pendingFile(root)
	rs, err := Load(path)
	if err != nil {
		return Record{}, false, err
	}
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
		return Record{}, false, nil
	}
	if err := save(path, kept); err != nil {
		return Record{}, false, err
	}
	return *gone, true, nil
}

func removeRecord(store, root string, rs []Record, id, reason string) (Record, error) {
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
		return Record{}, fmt.Errorf("no record [%s] in %s", id, store)
	}
	if err := save(store, kept); err != nil {
		return Record{}, err
	}
	if err := appendRetracted(root, *gone, reason); err != nil {
		if restoreErr := save(store, rs); restoreErr != nil {
			return Record{}, fmt.Errorf("could not write the retraction log (%v), and could not restore the record: %w", err, restoreErr)
		}
		return Record{}, fmt.Errorf("could not write the retraction log; record was kept: %w", err)
	}
	return *gone, nil
}

// appendRetracted logs a removal. Committed, unlike the surfacing log, because
// this one is evidence about the store's own accuracy and is worthless if it only
// exists on the machine that happened to do the deleting.
func appendRetracted(root string, r Record, reason string) error {
	f, err := os.OpenFile(filepath.Join(root, storeDirName, retractedLogName),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(f).Encode(map[string]any{
		"at":       time.Now().Format("2006-01-02"),
		"id":       r.ID,
		"file":     r.File,
		"decision": r.Decision,
		"author":   r.Author,
		"reason":   reason,
	}); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return nil
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
	fl := flag.NewFlagSet("backfill", flag.ExitOnError)
	// The store is committed and shared, so harvesting is content capture and
	// content capture is opt-in (§7.2). The default shows what would be stored
	// and writes nothing; --yes is the explicit act of committing it. A secret
	// sitting in a comment is the case this exists for — show it to a human
	// before it becomes a public commit, because git history is append-only.
	write := fl.Bool("yes", false, "write the harvested records to the store (default shows them without writing)")
	if err := fl.Parse(args); err != nil {
		os.Exit(2)
	}
	if err := validateBackfillArgs(fl.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "whence:", err)
		os.Exit(2)
	}
	dir := "."
	if fl.NArg() > 0 {
		dir = fl.Arg(0)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "whence:", err)
		os.Exit(1)
	}
	// The walk below ignores read errors, which is right for a subdirectory it
	// stumbles into and wrong for the root the user named: a typo there produced
	// "0 record(s) added", which reads as "your repo has nothing in it" rather
	// than "that path does not exist". Same rule as an anchor that cannot be
	// found — say so, do not report a confident zero.
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		fmt.Fprintf(os.Stderr, "whence: cannot read %s as a directory\n", dir)
		os.Exit(1)
	}

	added, skipped, shown := 0, 0, 0
	// The dry run buffers its finds and prints them only after the walk,
	// grouped by decision text, because the same sentence recurs: round two of
	// the corpus test found 72 repeated texts in linux and 36 in rust, and
	// "MUST NOT be called from interrupt context" sits above sixteen separate
	// drivers — every one of them true, so the data is right and the review
	// experience of approving one sentence sixteen times is wrong
	// (CORPUS-TEST-2026-08-16.md, round two). The cost of buffering is that a
	// dry run prints nothing until the walk ends — about 14 seconds on a
	// linux clone — which is the accepted trade for a report command.
	var dry []dryFind
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
			if !*write {
				// Dry run: hold what was found and where for the report after
				// the walk. The decision text is shown so a human can read it
				// for the secret this gate exists to catch before it is
				// committed.
				dry = append(dry, dryFind{rel: rel, start: f.start, end: f.end, decision: decision})
				shown++
				continue
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
	if !*write {
		lines, distinct := groupDryRuns(dry)
		for _, l := range lines {
			fmt.Println(l)
		}
		fmt.Printf("\n%d record(s) found (%d distinct texts), %d already present — nothing written. Rerun with --yes to store them.\n", shown, distinct, skipped)
		return
	}
	fmt.Printf("\n%d record(s) added, %d already present\n", added, skipped)
}

// dryFind is one harvest hit held for the dry-run report: where it was found,
// and the decision sentence a record would store.
type dryFind struct {
	rel        string
	start, end int
	decision   string
}

// groupDryRuns turns the buffered finds into the lines a dry run prints.
// Finds carrying an identical decision text collapse into one group — the
// decision once, with a count, then its locations indented beneath it — and a
// text found once keeps the single-line form it has always printed. Groups,
// and the locations within one, are in first-encounter order: the order the
// walk met them. Pure, so the report is testable without walking a
// filesystem; the count of groups rides along for the closing summary.
func groupDryRuns(finds []dryFind) (lines []string, distinct int) {
	var order []string
	groups := make(map[string][]dryFind)
	for _, f := range finds {
		if _, ok := groups[f.decision]; !ok {
			order = append(order, f.decision)
		}
		groups[f.decision] = append(groups[f.decision], f)
	}
	for _, d := range order {
		locs := groups[d]
		if len(locs) == 1 {
			f := locs[0]
			lines = append(lines, fmt.Sprintf("  would add  %s:%d-%d  %s", f.rel, f.start, f.end, d))
			continue
		}
		lines = append(lines, fmt.Sprintf("  would add  ×%d  %s", len(locs), d))
		for _, f := range locs {
			lines = append(lines, fmt.Sprintf("      %s:%d-%d", f.rel, f.start, f.end))
		}
	}
	return lines, len(order)
}

func validateBackfillArgs(args []string) error {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			return fmt.Errorf("unrecognised argument %q — flags must come before the directory", a)
		}
	}
	return nil
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
	//
	// That premise is per-project. XXX: sat here until the 2026-08-16 corpus
	// test (CORPUS-TEST-2026-08-16.md): CPython uses XXX the way other projects
	// use TODO, and 91 of its 151 candidates were unanswered questions under it
	// — "is this test needed?", "Should seek() be used", "implement". Do not
	// move it back: this list is the only path with no gate behind it, and in a
	// repo that treats XXX as a task marker there is nothing behind it at all.
	alwaysMarkers = []string{"ponytail:", "HACK:", "WORKAROUND:", "GOTCHA:"}

	// reasonMarkers are the comments that describe code already present. TODO and
	// FIXME are deliberately excluded: even when they give a reason, that reason
	// explains a proposed change rather than why the current code is this way.
	// The reason-word gate could not tell those apart — today's stdlib sample had
	// 14 task-shaped TODOs pass it — so narrowing the marker set is the smaller
	// honest filter. A missed task is noise avoided; a missing decision is still
	// recoverable by hand.
	//
	// XXX: is here rather than in alwaysMarkers because its meaning is
	// per-project: Kubernetes writes it as a decision, CPython writes it as a
	// task. The reason gate is what separates the two — see alwaysMarkers.
	reasonMarkers = []string{"NOTE:", "WARNING:", "CAVEAT:", "XXX:"}

	// reasonWords are what giving a reason sounds like. Deliberately a small,
	// boring list: every word here is one people write without thinking about it,
	// which is the point — a heuristic nobody has to learn.
	//
	// ponytail: conservative word list, so real reasons phrased around it are
	// missed — "the upstream 502s, so one attempt fails" has no word from this
	// list. Left narrow on purpose: a missed note is recoverable by hand, and a
	// garbage record in a committed shared store is not. Widen only against real
	// misses from a real repo, never by imagining phrasings.
	//
	// That trigger FIRED on 2026-08-04, against four employer repositories: three
	// genuine decisions were rejected — a cross-file invariant ("MUST stay
	// identical to"), a deliberate omission ("intentionally omitted for"), and a
	// guarantee ("ensures", "maintains"). The widening went to a SECOND AXIS
	// rather than into this list: those are commitment, not cause, and this list
	// is read for two jobs — admission, and via splitWords deciding where a reason
	// begins. "MUST" is fine evidence a note commits and marks no split point
	// whatsoever, so merging them would put a word that cannot split into the
	// splitting path. See commitmentWords. This list is UNCHANGED, and the same
	// rule still governs it: widen only against a fresh real miss.
	reasonWords = []string{
		"because", "so that", "otherwise", "since", "to avoid",
		"rather than", "instead of", "reason", "caused",
	}

	// commitmentWords are what a decision sounds like when it states no cause.
	//
	// A note can prove it is a decision two ways: by explaining WHY (reasonWords)
	// or by committing to something a reader must not undo. "norm() here MUST stay
	// identical to norm() in the ingest worker" gives no reason and is
	// unmistakably a decision — and is the shape whose loss costs most, since
	// editing one side of a cross-file invariant and not the other is the exact
	// regression whence exists to prevent.
	//
	// Admission-only — NEVER add these to splitWords: they mark that a note
	// commits, not where its explanation starts, and a note with no causal split
	// point is correctly left whole.
	//
	// Bare "must", "required", "never", "always" and "keep" are deliberately
	// absent: they appear in ordinary tasks, and "required" in particular would
	// admit `// TODO: Need to check if this is required`, one of 17 rejections
	// verified CORRECT on 2026-08-04. That is why "must" appears only in phrase
	// form. Checked by hand against that run: this list admits all three real
	// misses and none of the 17 correct rejections.
	commitmentWords = []string{
		"intentionally", "deliberately", "on purpose",
		"ensures", "guarantees", "maintains",
		"must stay", "must match", "must remain", "must be identical",
	}
)

// secretShapes are the prefixes of credentials that end up pasted into
// comments. The list is deliberately the well-known token formats, not an
// entropy guess: a false positive here refuses a legitimate record into a
// shared store, which is annoying; a false negative commits a key, which is
// unrecoverable. Both args point the same way, so the bias is to recall on the
// shapes that are unambiguous, and to say "rephrase" rather than guess harder.
//
// ponytail: prefix/format match only, no entropy scoring and no per-provider
// checksum. That misses rotated or homemade secret formats; widen only against
// a real leak that had a shape this list does not carry, never by imagining.
var secretShapes = []string{
	"sk-", "sk_live_", "sk_test_", // OpenAI / Stripe-style secret keys
	"ghp_", "gho_", "ghu_", "github_pat_", // GitHub tokens
	"glpat-",                                    // GitLab
	"xoxb-", "xoxp-", "xoxa-", "xoxr-", "xoxs-", // Slack
	"AKIA", "ASIA", // AWS access key ids
	"AIza",       // Google API keys
	"-----BEGIN", // PEM private keys and certificates
	"eyJ",        // JWT header
}

// secretShape reports whether text contains a credential-shaped token. The
// caller never prints the match — naming the shape would re-transcribe the
// secret it is trying to keep out of a committed store.
func secretShape(text string) bool {
	for _, s := range secretShapes {
		if strings.Contains(text, s) {
			return true
		}
	}
	return false
}

// admit is the write-time scan (§14 layer 3). First net: this repo's own
// .env / .env.* / gitignored-config values. Second net: prefix and entropy
// shapes. The match is never printed — naming it would re-transcribe the secret.
// Residual, stated honestly: nothing here catches paraphrase. Nor a secret that
// travels as an unprefixed path segment of an otherwise plain URL —
// entropyToken skips those so that a note citing an issue is not refused
// (ANCHOR-SURVIVAL-2026-08-17.md, defect 1), and a webhook path carries no
// prefix secretShapes knows. Deliberately not widened for it: that list admits
// a shape on a real leak, never on an imagined one.
func admit(root, file string, start int, decision, why string) error {
	if secretShape(decision) || secretShape(why) || secretEntropy(decision) || secretEntropy(why) {
		return fmt.Errorf("whence: %s:%d — the text looks like it holds a credential (a key or token shape); refusing to commit it to a shared store. Rephrase without the secret", file, start)
	}
	vals := envSecrets(root)
	if holdsValue(decision, vals) || holdsValue(why, vals) {
		return fmt.Errorf("whence: %s:%d — the text looks like it holds a credential (a key or token shape); refusing to commit it to a shared store. Rephrase without the secret", file, start)
	}
	return nil
}

func holdsValue(text string, vals []string) bool {
	for _, v := range vals {
		if strings.Contains(text, v) {
			return true
		}
	}
	return false
}

// envSecrets extracts values from the repo's own secret files. Reads .gitignore
// as a text file — does not ask git anything (invariant 8).
func envSecrets(root string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(v string) {
		if !usableSecret(v) || seen[v] {
			return
		}
		seen[v] = true
		out = append(out, v)
	}
	for _, p := range envFiles(root) {
		for _, v := range dotenvValues(p) {
			add(v)
		}
	}
	return out
}

func envFiles(root string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if seen[p] || outsideRoot(p, root) {
			return
		}
		st, err := os.Stat(p)
		if err != nil || st.IsDir() {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	if matches, err := filepath.Glob(filepath.Join(root, ".env")); err == nil {
		for _, p := range matches {
			add(p)
		}
	}
	if matches, err := filepath.Glob(filepath.Join(root, ".env.*")); err == nil {
		for _, p := range matches {
			add(p)
		}
	}
	b, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		if strings.HasSuffix(line, "/") || strings.ContainsAny(line, "*?[") {
			continue
		}
		add(filepath.Join(root, filepath.Clean(line)))
	}
	return out
}

func dotenvValues(path string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		_, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		if n := len(v); n >= 2 && ((v[0] == '"' && v[n-1] == '"') || (v[0] == '\'' && v[n-1] == '\'')) {
			v = v[1 : n-1]
		}
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// usableSecret drops values that would fire on every other record: ports,
// booleans, NODE_ENV=development. High precision, near-zero false positives.
func usableSecret(v string) bool {
	if len(v) < 8 {
		return false
	}
	if boringEnv[strings.ToLower(v)] {
		return false
	}
	for _, r := range v {
		if r < '0' || r > '9' {
			return true
		}
	}
	return false
}

var boringEnv = map[string]bool{
	"true": true, "false": true,
	"development": true, "production": true, "test": true,
	"localhost": true, "debug": true, "info": true, "warn": true, "error": true,
	"utf-8": true,
}

// secretEntropy is the second net: a long mixed-class token that is not a hex
// hash. Prefix shapes are secretShape's job. ponytail: 32-char floor and hex
// skip; homemade short secrets miss, add a real miss when one lands.
func secretEntropy(text string) bool {
	for _, tok := range strings.FieldsFunc(text, func(r rune) bool {
		return r <= ' ' || r == '"' || r == '\'' || r == '`'
	}) {
		if entropyToken(tok) {
			return true
		}
	}
	return false
}

func entropyToken(s string) bool {
	// A plain http/https URL is a citation, not a credential: 12 of rust's 15
	// refused notes in the 2026-08-17 survival run were NOTE/HACK comments
	// citing a GitHub issue — exactly the records worth keeping, dropped on
	// length and character classes (ANCHOR-SURVIVAL-2026-08-17.md, defect 1).
	// Only the shape that cannot carry a secret is skipped: userinfo, a query
	// string or a fragment is how a credential genuinely travels in a URL
	// (https://user:pass@host, ?api_key=…), so those still fall through to the
	// class check below (§7).
	if u, err := url.Parse(s); err == nil &&
		(u.Scheme == "http" || u.Scheme == "https") &&
		u.User == nil && u.RawQuery == "" && u.Fragment == "" {
		return false
	}
	if len(s) < 32 {
		return false
	}
	hex := true
	var lower, upper, digit, other bool
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digit = true
		case r >= 'a' && r <= 'f':
			lower = true
		case r >= 'g' && r <= 'z':
			lower = true
			hex = false
		case r >= 'A' && r <= 'F':
			upper = true
		case r >= 'G' && r <= 'Z':
			upper = true
			hex = false
		default:
			other = true
			hex = false
		}
	}
	if hex {
		return false
	}
	n := 0
	if lower {
		n++
	}
	if upper {
		n++
	}
	if digit {
		n++
	}
	if other {
		n++
	}
	return n >= 3
}

// splitWords are the subset of reasonWords that mark WHERE the reason
// starts, for splitting a one-sentence note into its two halves.
//
// "reason" and "caused" are deliberately absent. Both occur as an ordinary
// noun and verb in the middle of a clause — "the reason code is duplicated",
// "the leak caused by this" — where cutting at them yields two fragments and
// no decision. They remain fine evidence that a note explains itself, which
// is a different question from where the explanation begins.
var splitWords = []string{
	"because", "so that", "otherwise", "since", "to avoid",
	"rather than", "instead of",
}

// ponytail: vendored trees are only caught when they are named vendor/ or
// third_party/. CPython vendors Expat at Modules/expat/ with no marker of any
// kind, and 11 of its 151 candidates were upstream's reasoning
// (CORPUS-TEST-2026-08-16). No heuristic catches that shape; invent one only
// when a second real case lands.
var skipDir = map[string]bool{
	".git": true, storeDirName: true, "node_modules": true,
	"dist": true, "build": true, "vendor": true, "third_party": true, ".next": true,
}

// found is one harvested note: the comment block plus the declaration it sits
// above, which is the code the note is actually about.
type found struct {
	start, end int
	text       string
	src        string // which marker produced it, recorded as the record's source
}

// expectedOutputExts are the well-known expected-output formats: files a test
// framework rewrites on every run, like the machine-written .fixed twin Rust's
// lint tests keep beside each .rs fixture — 22 of the round-two corpus
// candidates arrived as identical .rs/.fixed pairs (CORPUS-TEST-2026-08-16).
// A record anchored in one is anchored to something regenerated on every run,
// which is the generated-file defect wearing an extension nobody would guess.
// A named list of known formats, grown against real cases only — never by
// imagining extensions.
var expectedOutputExts = []string{".fixed", ".stderr", ".stdout", ".snap", ".golden"}

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
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	// Expected-output fixtures are the same refusal as generated files below:
	// the test framework rewrites them, so the anchor is rewritten with them.
	ext := filepath.Ext(p)
	for _, e := range expectedOutputExts {
		if ext == e {
			return nil
		}
	}
	// A generated file's comments are the generator's reasoning, not the
	// repo's, and regenerating rewrites the anchor under them — 20 of
	// Kubernetes' 92 candidates were one identical protoc line living in
	// *_grpc.pb.go files (CORPUS-TEST-2026-08-16). The standard marker line
	// is the convention; filename patterns like *.pb.go are not, and miss
	// every generator that does not use that suffix.
	if generated(lines) {
		return nil
	}
	return lines
}

// generated reports whether a file declares itself machine-made with Go's
// standard marker line, `^// Code generated .* DO NOT EDIT\.$`, expressed as
// prefix plus suffix rather than a regexp because that is all the pattern is.
// Only the head is examined: the marker is defined to sit in the comment
// block before the first non-comment, non-blank line, and the cap keeps a
// pathological all-comment file from being scanned to its end.
func generated(lines []string) bool {
	for i, l := range lines {
		if i >= 50 {
			return false
		}
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "// Code generated ") && strings.HasSuffix(t, " DO NOT EDIT.") {
			return true
		}
		if t != "" && !strings.HasPrefix(t, "//") && !strings.HasPrefix(t, "/*") &&
			!strings.HasPrefix(t, "*") && !strings.HasPrefix(t, "#") {
			return false
		}
	}
	return false
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
		//
		// Two independent ways to qualify, so the condition is widened here rather
		// than inside hasReason: a function called hasReason that returned true
		// for "MUST" would be a lie in its own name.
		//
		// The question rule lives here for the same reason. capture.go reads
		// hasReason as a deliberately-labelled lexical floor on how often stated
		// reasoning exists; refusing questions inside it would move that number
		// without the thing it measures having changed.
		if text != "" && (!needsReason || hasReason(text) || hasCommitment(text)) && !asksQuestion(text) {
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

// hasCommitment reports whether a note commits to something without saying why.
//
// The second admission axis, kept separate from hasReason on purpose. capture.go
// reads hasReason as a deliberately-labelled lexical floor on how often stated
// reasoning exists at all; folding commitment into it would move that number
// without the thing it measures having changed.
func hasCommitment(s string) bool {
	l := strings.ToLower(s)
	for _, w := range commitmentWords {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

// asksQuestion reports whether a note's headline is a question. A question is
// not a decision: Linux's "Is there any reason to assume differently?"
// qualified only because "reason" is on the admission list, and CPython's "is
// this test needed?" is the same shape (CORPUS-TEST-2026-08-16, round two).
//
// The headline is the same text firstSentence would store as the decision, so
// a rhetorical question mid-note is fine — "We cannot use the fast path here.
// Why? The lock is already held." does not END on its question.
func asksQuestion(text string) bool {
	d, _ := firstSentence(text)
	return strings.HasSuffix(strings.TrimSpace(d), "?")
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
//
// The first ". " is not always a sentence break. A real note read "...might
// belong to other services (e.g. <names>), but since they are used directly..."
// and the cut landed inside the abbreviation: the decision ended mid-parenthesis
// and stated nothing, while the why carried the entire note. That is the failure
// splitAtReason's comment describes — the gate and the store disagreeing about
// the same comment — one level down, with the cut point wrong rather than absent.
//
// So an abbreviation boundary is refused and the scan CONTINUES to the next
// candidate, because a note that merely mentions "e.g." mid-sentence still has a
// legitimate later break. Only if every candidate is an abbreviation does this
// fall through to splitAtReason, and then to the whole note. It never returns an
// empty decision.
func firstSentence(s string) (decision, why string) {
	for at := 0; at+1 < len(s); {
		i := strings.Index(s[at:], ". ")
		if i < 0 {
			break
		}
		i += at
		if endsInAbbreviation(s[:i]) {
			at = i + 1
			continue
		}
		d, w := s[:i+1], strings.TrimSpace(s[i+2:])
		// A why that is a statement of code explains nothing — storing nothing
		// is better than storing a statement. See codeWhy.
		if codeWhy(w) {
			return d, ""
		}
		return d, w
	}
	if d, w, ok := splitAtReason(s); ok {
		// Same guard as the sentence split above: commentBody's `*` rule feeds
		// both branches, so a folded declaration reaches this one too whenever
		// the note has no sentence break to split at.
		if codeWhy(w) {
			return d, ""
		}
		return d, w
	}
	return s, ""
}

// codeWhy reports whether a would-be why half is a statement of code rather
// than reasoning. The declaration a note sits above can join the note text
// when the line opens with `*` — commentBody reads a bare `*` as a block-
// comment continuation, so rust's `*self.current_func.borrow_mut() = ...` was
// folded into the note and the ". " split stored the statement as the why
// (ANCHOR-SURVIVAL-2026-08-17.md, defect 2, record 1642c7).
//
// codeish (main.go) settles it, over identifier-shaped words of 6+ letters: a
// code statement's long words are all identifiers (current_func, borrow_mut),
// while a prose why always carries one plain long word codeish refuses
// (because, called, function). Inherits codeish's blind spot verbatim — a
// code statement whose long words are plain lowercase keeps its why — which
// is the direction that loses nothing.
func codeWhy(s string) bool {
	long := 0
	for i := 0; i < len(s); {
		if !identStart(s[i]) {
			i++
			continue
		}
		j := i + 1
		for j < len(s) && identCont(s[j]) {
			j++
		}
		if j-i >= 6 {
			long++
			if !codeish(s[i:j]) {
				return false
			}
		}
		i = j
	}
	return long > 0
}

// abbreviations are the tokens that end in a period without ending a sentence.
//
// Deliberately small and boring, in the spirit of reasonWords: these are the ones
// people write without thinking, and a longer list is a sentence tokeniser, which
// is a dependency and a new class of wrong answer. A single letter is here for
// initials ("J. Smith"), where the failure is the same and the decision would
// otherwise be one character.
var abbreviations = []string{"e.g", "i.e", "etc", "cf", "vs", "no"}

// endsInAbbreviation reports whether the text immediately before a period is an
// abbreviation rather than the end of a sentence.
//
// The token is lowercased for comparison only, never indexed — splitAtReason
// refuses non-ASCII outright because case folding can change byte length and move
// every offset, and that hazard is real here too. Every index used by the caller
// comes from the original string.
func endsInAbbreviation(before string) bool {
	tok := before
	if i := strings.LastIndexAny(tok, " 	"); i >= 0 {
		tok = tok[i+1:]
	}
	tok = strings.TrimLeft(tok, "([{\"'`")
	if tok == "" {
		return false
	}
	if r := []rune(tok); len(r) == 1 && unicode.IsLetter(r[0]) {
		return true // an initial: "J. Smith"
	}
	low := strings.ToLower(tok)
	for _, a := range abbreviations {
		if low == a {
			return true
		}
	}
	return false
}

// splitAtReason cuts a one-sentence note at the word that admitted it.
//
// A reason-bearing note is only harvested when it says WHY, and people write that
// inline at least as often as in a second sentence: "change to 0 here to avoid
// pointer mangling" carries both halves in one breath. firstSentence finds no
// boundary in that and puts everything in the decision — so the note is admitted
// for giving a reason and then stored with none, which is the gate and the store
// disagreeing about the same comment.
//
// The earliest reason word wins, so the why carries the whole justification
// rather than its tail.
//
// It cuts on a narrower list than admission uses — see splitWords. Admitting a
// note and cutting one are different questions: any of the reason words is
// evidence the note explains itself, but only some of them mark where the
// explanation starts.
//
// Two more refusals, on the same asymmetry the word list itself is chosen on: a
// note left whole is recoverable by hand, a note cut into fragments is a garbage
// record in a committed store. A note that OPENS on its reason ("since X, do Y")
// would leave no decision. And non-ASCII case folding can change byte length,
// which moves every offset — so the split is abandoned rather than applied to
// indices that may no longer mean anything.
//
// A third refusal, from the 2026-08-16 corpus test: a cut that leaves either
// half too thin to mean anything. Real output stored the decision as "In order"
// (cut at "to avoid", the whole rule pushed into why) and produced a one-word
// why (cut at "otherwise"). So the split is refused when the decision half is
// fewer than 6 words or the why half fewer than 4. The thresholds are tuned to
// the asymmetry, not derived: an unsplit note is a correct record with a longer
// headline, while a fragment headline is a wrong record printed at the moment
// `check` is trying to stop an agent. 6/4 was chosen over a looser 4/3 on
// purpose — 4/3 still passes "In order to avoid | ..." — so do not loosen them
// back without a real miss that 6/4 caused and 4/3 would not.
//
// Round two of the same test showed the scan and the counter under those
// thresholds both leaking. "This must be a macro (rather than a function with
// trait bounds) because ..." cut at "rather than" — INSIDE the brackets — and
// stored "This must be a macro (" as the decision, which also passed the floor
// only because strings.Fields scores a bare "(" as a word. So an occurrence of
// a split word inside an unclosed (, [, { or backtick pair is skipped, and if
// every occurrence is inside one there is no split at all; and the halves are
// counted by wordCount, which a punctuation-only token does not fool.
func splitAtReason(s string) (decision, why string, ok bool) {
	l := strings.ToLower(s)
	if len(l) != len(s) {
		return "", "", false
	}
	at := -1
	for _, w := range splitWords {
		// The earliest occurrence outside any bracket wins for this word — see
		// above. A word at index 0 opens the note on its reason and is refused,
		// as before.
		for i := strings.Index(l, w); i > 0; {
			if !insideBrackets(l[:i]) {
				if at < 0 || i < at {
					at = i
				}
				break
			}
			n := strings.Index(l[i+1:], w)
			if n < 0 {
				break
			}
			i += n + 1
		}
	}
	if at < 0 {
		return "", "", false
	}
	d := strings.TrimRight(strings.TrimSpace(s[:at]), " ,;—-")
	w := strings.TrimSpace(s[at:])
	if wordCount(d) < 6 || wordCount(w) < 4 {
		return "", "", false
	}
	return d, w, true
}

// insideBrackets reports whether the end of s sits inside an unclosed (, [, {
// or backtick pair, counting from the start of the string. Plain byte counting
// is enough: splitAtReason has already refused non-ASCII, so bytes and offsets
// agree. A closer at zero depth closes nothing — a stray ")" does not cancel a
// bracket that opens after it.
func insideBrackets(s string) bool {
	depth, ticks := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case '`':
			ticks++
		}
	}
	return depth > 0 || ticks%2 != 0
}

// wordCount counts the tokens carrying at least one letter or digit.
// strings.Fields alone scores a bare "(" as a word, which let a five-word
// decision plus a stray bracket pass the six-word floor — rust/zerocopy's
// "This must be a macro (" (CORPUS-TEST-2026-08-16, round two). The 6/4
// thresholds were right; the counter under them was not.
func wordCount(s string) int {
	n := 0
	for _, tok := range strings.Fields(s) {
		for _, r := range tok {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				n++
				break
			}
		}
	}
	return n
}

func has(rs []Record, file, decision string) bool {
	for _, r := range rs {
		if samePath(r.File, file) && r.Decision == decision {
			return true
		}
	}
	return false
}
