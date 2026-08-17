package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestAnchorSurvival measures whether records harvested at the oldest commit
// of a real repo still resolve after the history that followed. This is the
// question CORPUS-TEST-2026-08-16.md left open: that run exercised harvest
// and firstSentence only.
//
// Skips unless WHENCE_SURVIVAL_CLONE names an existing directory. go test
// must stay green and fast for anyone who never sets it — this is a
// measurement rig that happens to be executable, not a unit test.
func TestAnchorSurvival(t *testing.T) {
	clone := os.Getenv("WHENCE_SURVIVAL_CLONE")
	if clone == "" {
		t.Skip("WHENCE_SURVIVAL_CLONE unset")
	}
	st, err := os.Stat(clone)
	if err != nil || !st.IsDir() {
		t.Skipf("WHENCE_SURVIVAL_CLONE=%s is not an existing directory", clone)
	}
	clone, err = filepath.Abs(clone)
	if err != nil {
		t.Fatal(err)
	}
	// FindStore walks *up*. A clone inside this working tree writes its
	// harvested records into whence's own committed store.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if !outsideRoot(clone, cwd) {
		t.Fatalf("WHENCE_SURVIVAL_CLONE=%s is inside the whence tree — harvest would land in the committed store", clone)
	}

	// Tip has to be captured before we detach onto C0: after that checkout
	// HEAD *is* C0, and `C0..HEAD` is empty.
	tip, err := cloneGit(clone, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	tip = strings.TrimSpace(tip)
	c0, err := oldestCommit(clone)
	if err != nil {
		t.Fatalf("oldest commit: %v", err)
	}
	c0date, _ := cloneGit(clone, "log", "-1", "--format=%ci %s", c0)
	tipdate, _ := cloneGit(clone, "log", "-1", "--format=%ci %s", tip)
	fmt.Fprintf(os.Stderr, "survival: C0  %s %s", shortSHA(c0), c0date)
	fmt.Fprintf(os.Stderr, "survival: HEAD %s %s", shortSHA(tip), tipdate)
	if err := checkout(clone, c0); err != nil {
		t.Fatalf("checkout C0 %s: %v", c0, err)
	}

	store := filepath.Join(clone, storeDirName, recordsFileName)
	if err := os.MkdirAll(filepath.Dir(store), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	rs, hstat, err := harvestClone(clone, store)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "survival: harvested %d records at %s (%d files read, %d skipped, %d notes refused)\n",
		len(rs), shortSHA(c0), hstat.read, hstat.skipped, hstat.refused)
	for _, reason := range hstat.reasons {
		fmt.Fprintf(os.Stderr, "  refused  %s\n", reason)
	}

	type snap struct {
		File       string
		Start, End int
	}
	born := make([]snap, len(rs))
	for i, r := range rs {
		born[i] = snap{r.File, r.Start, r.End}
	}

	start := resolveAll(clone, rs)
	fmt.Fprintf(os.Stderr, "survival: at C0  %s\n", countStates(start))

	files := recordFiles(rs)
	shas, err := commitsTouching(clone, c0, tip, files)
	if err != nil {
		t.Fatal(err)
	}
	sampled, dropped := sampleEvenly(shas, 200)
	if dropped > 0 {
		fmt.Fprintf(os.Stderr, "survival: sampled %d of %d commits touching record files — dropped %d\n",
			len(sampled), len(shas), dropped)
	} else {
		fmt.Fprintf(os.Stderr, "survival: walking %d commits that touch record files\n", len(sampled))
	}

	type step struct {
		sha    string
		states []Anchor
	}
	var steps []step
	firstOrphan := make(map[string]string) // record id → sha it first went orphan
	newOrphansAt := map[string]int{}

	for i, sha := range sampled {
		if err := checkout(clone, sha); err != nil {
			t.Fatalf("checkout %s (%d/%d): %v", sha, i+1, len(sampled), err)
		}
		now := resolveAll(clone, rs)
		steps = append(steps, step{sha: sha, states: now})
		for j, r := range rs {
			if now[j].State != StateOrphaned {
				continue
			}
			if _, seen := firstOrphan[r.ID]; seen {
				continue
			}
			firstOrphan[r.ID] = sha
			newOrphansAt[sha]++
		}
		if (i+1)%25 == 0 || i+1 == len(sampled) {
			fmt.Fprintf(os.Stderr, "survival: %d/%d %s  %s\n", i+1, len(sampled), shortSHA(sha), countStates(now))
		}
	}

	end := start
	if len(steps) > 0 {
		end = steps[len(steps)-1].states
	}

	// Invariant 2: recorded Start/End are never rewritten.
	reloaded, err := Load(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded) != len(born) {
		t.Errorf("store changed size: harvested %d, now %d", len(born), len(reloaded))
	}
	for i := range reloaded {
		if i >= len(born) {
			break
		}
		if reloaded[i].File != born[i].File || reloaded[i].Start != born[i].Start || reloaded[i].End != born[i].End {
			t.Errorf("record %s File/Start/End rewritten: %s:%d-%d → %s:%d-%d",
				reloaded[i].ID, born[i].File, born[i].Start, born[i].End,
				reloaded[i].File, reloaded[i].Start, reloaded[i].End)
		}
	}

	fmt.Fprintf(os.Stderr, "survival: at end %s\n", countStates(end))
	printDrift(os.Stderr, rs, end)
	printWeak(os.Stderr, end)
	printSurvivors(os.Stderr, rs, end)
	printOrphanSpread(os.Stderr, newOrphansAt, firstOrphan, len(sampled))

	tax := classifyOrphans(clone, rs, end)
	printTaxonomy(os.Stderr, rs, end, firstOrphan, tax)

	fmt.Fprintf(os.Stderr, "survival: bounds  files_skipped=%d notes_refused=%d commits_available=%d commits_walked=%d commits_dropped=%d\n",
		hstat.skipped, hstat.refused, len(shas), len(sampled), dropped)
}

type harvestStat struct {
	read, skipped, refused int
	reasons                []string
}

// harvestClone walks the clone the same way backfillCmd does — readSource,
// harvest, firstSentence — and writes through save. Flag parsing and the
// dry-run buffer stay in backfillCmd; this is the write path only, against
// a throwaway store inside the clone. Both corpus rounds harvested and
// wrote nothing (CORPUS-TEST-2026-08-16.md); this run has to write so
// there is something to re-resolve.
func harvestClone(root, store string) ([]Record, harvestStat, error) {
	var rs []Record
	var st harvestStat
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDir[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		lines := readSource(p)
		if lines == nil {
			st.skipped++
			return nil
		}
		st.read++
		for _, f := range harvest(lines) {
			decision, why := firstSentence(f.text)
			rel := filepath.ToSlash(Rel(root, p))
			if has(rs, rel, decision) {
				continue
			}
			r, _, err := makeRecord(p, root, p, f.start, f.end, decision, why, f.src, authorHuman, nil)
			if err != nil {
				st.refused++
				if len(st.reasons) < 20 {
					st.reasons = append(st.reasons, fmt.Sprintf("%s:%d %v", rel, f.start, err))
				}
				continue
			}
			rs = append(rs, r)
		}
		return nil
	})
	if err != nil {
		return nil, st, err
	}
	if err := save(store, rs); err != nil {
		return nil, st, err
	}
	return rs, st, nil
}

func resolveAll(root string, rs []Record) []Anchor {
	out := make([]Anchor, len(rs))
	for i, r := range rs {
		// Same call logAll makes at main.go:579.
		out[i] = resolveAnchor(fileLinesWithin(filepath.Join(root, r.File), root), r)
	}
	return out
}

func recordFiles(rs []Record) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range rs {
		if seen[r.File] {
			continue
		}
		seen[r.File] = true
		out = append(out, r.File)
	}
	sort.Strings(out)
	return out
}

func oldestCommit(dir string) (string, error) {
	// --max-count is applied before --reverse, so `rev-list --reverse -1`
	// is just HEAD. First line of the reversed walk is the oldest commit
	// the shallow clone actually has.
	out, err := cloneGit(dir, "rev-list", "--reverse", "HEAD")
	if err != nil {
		return "", err
	}
	sha, _, _ := strings.Cut(out, "\n")
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return "", fmt.Errorf("no commits in %s", dir)
	}
	return sha, nil
}

func commitsTouching(dir, c0, tip string, files []string) ([]string, error) {
	if len(files) == 0 {
		return nil, nil
	}
	args := []string{"log", "--reverse", "--format=%H", c0 + ".." + tip, "--"}
	args = append(args, files...)
	out, err := cloneGit(dir, args...)
	if err != nil {
		return nil, err
	}
	var shas []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			shas = append(shas, line)
		}
	}
	return shas, nil
}

func checkout(dir, sha string) error {
	// Plain checkout, never -f / clean / reset --hard: the store lives in
	// an untracked .whence/ and those forms delete the measurement.
	_, err := cloneGit(dir, "checkout", "-q", sha)
	return err
}

// cloneGit runs one git command in the clone and returns stdout. History
// has no stdlib equivalent; this is the wrapper the task asked for and
// nothing more.
func cloneGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out), nil
}

// sampleEvenly keeps first and last and spaces the rest, so a cap of 200
// still reads as a walk from C0 to HEAD. A silent cap is the failure
// mode the task named.
func sampleEvenly(shas []string, cap int) (kept []string, dropped int) {
	if len(shas) <= cap {
		return shas, 0
	}
	if cap <= 1 {
		return []string{shas[len(shas)-1]}, len(shas) - 1
	}
	kept = make([]string, 0, cap)
	seen := map[string]bool{}
	for i := 0; i < cap; i++ {
		sha := shas[i*(len(shas)-1)/(cap-1)]
		if seen[sha] {
			continue
		}
		seen[sha] = true
		kept = append(kept, sha)
	}
	return kept, len(shas) - len(kept)
}

func TestSampleEvenly(t *testing.T) {
	var in []string
	for i := 0; i < 10; i++ {
		in = append(in, fmt.Sprintf("%02d", i))
	}
	got, dropped := sampleEvenly(in, 200)
	if dropped != 0 || len(got) != 10 {
		t.Fatalf("under the cap should keep all, got %d dropped %d", len(got), dropped)
	}
	got, dropped = sampleEvenly(in, 4)
	if dropped != 6 || len(got) != 4 {
		t.Fatalf("cap 4 of 10: got %v dropped %d", got, dropped)
	}
	if got[0] != "00" || got[len(got)-1] != "09" {
		t.Errorf("must keep first and last, got %v", got)
	}
	got, dropped = sampleEvenly(nil, 200)
	if len(got) != 0 || dropped != 0 {
		t.Errorf("empty in: got %v dropped %d", got, dropped)
	}
}

func countStates(as []Anchor) string {
	var exact, drifted, weak, orphan, line int
	for _, a := range as {
		switch a.State {
		case StateExact:
			exact++
		case StateDrifted:
			drifted++
		case StateWeak:
			weak++
		case StateOrphaned:
			orphan++
		case StateLineOnly:
			line++
		}
	}
	return fmt.Sprintf("exact=%d drifted=%d weak=%d orphaned=%d line-only=%d (n=%d)",
		exact, drifted, weak, orphan, line, len(as))
}

func printDrift(w *os.File, rs []Record, as []Anchor) {
	var dists []int
	for i, a := range as {
		if a.State == StateOrphaned || a.Start == 0 {
			continue
		}
		d := a.Start - rs[i].Start
		if d < 0 {
			d = -d
		}
		dists = append(dists, d)
	}
	if len(dists) == 0 {
		fmt.Fprintln(w, "survival: drift  no surviving records")
		return
	}
	sort.Ints(dists)
	sum := 0
	moved := 0
	for _, d := range dists {
		sum += d
		if d > 0 {
			moved++
		}
	}
	fmt.Fprintf(w, "survival: drift  n=%d moved=%d unmoved=%d min=%d median=%d p90=%d max=%d mean=%.1f\n",
		len(dists), moved, len(dists)-moved, dists[0], dists[len(dists)/2],
		dists[(len(dists)*9)/10], dists[len(dists)-1], float64(sum)/float64(len(dists)))
}

func printSurvivors(w *os.File, rs []Record, as []Anchor) {
	for i, a := range as {
		if a.State == StateOrphaned {
			continue
		}
		delta := a.Start - rs[i].Start
		fmt.Fprintf(w, "  KEEP %s  %s:%d-%d → %d-%d  Δ%+d  %s  integrity=%.3f  %s\n",
			rs[i].ID, rs[i].File, rs[i].Start, rs[i].End, a.Start, a.End, delta, a.State, a.Integrity, clip(rs[i].Decision, 100))
	}
}

func printWeak(w *os.File, as []Anchor) {
	var vals []float64
	for _, a := range as {
		if a.State == StateWeak {
			vals = append(vals, a.Integrity)
		}
	}
	if len(vals) == 0 {
		fmt.Fprintln(w, "survival: weakFloor  no StateWeak records at end")
		return
	}
	sort.Float64s(vals)
	// 0.05 buckets from weakFloor up. The floor itself is 0.60; anything
	// below it is an orphan, not Weak.
	buckets := make([]int, 8)
	for _, v := range vals {
		i := int((v - weakFloor) / 0.05)
		if i < 0 {
			i = 0
		}
		if i >= len(buckets) {
			i = len(buckets) - 1
		}
		buckets[i]++
	}
	fmt.Fprintf(w, "survival: weakFloor  n=%d min=%.3f median=%.3f max=%.3f\n",
		len(vals), vals[0], vals[len(vals)/2], vals[len(vals)-1])
	for i, n := range buckets {
		lo := weakFloor + float64(i)*0.05
		fmt.Fprintf(w, "  [%.2f, %.2f): %d\n", lo, lo+0.05, n)
	}
}

func printOrphanSpread(w *os.File, at map[string]int, first map[string]string, steps int) {
	if len(first) == 0 {
		fmt.Fprintln(w, "survival: orphans appeared in 0 commits")
		return
	}
	type hit struct {
		sha string
		n   int
	}
	var hits []hit
	for sha, n := range at {
		hits = append(hits, hit{sha, n})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].n > hits[j].n })
	fmt.Fprintf(w, "survival: orphans appeared across %d of %d walked commits (total first-seen %d)\n",
		len(hits), steps, len(first))
	show := 5
	if len(hits) < show {
		show = len(hits)
	}
	for _, h := range hits[:show] {
		fmt.Fprintf(w, "  %s  +%d\n", shortSHA(h.sha), h.n)
	}
}

// orphanKind is one bucket of the taxonomy the tree-sitter question needs.
// fileDeleted and fileRenamed are split off from the rest on purpose: a
// missing path is not a hash failure. hashLine is unsalted (anchor.go:110)
// so following a block across files stays possible; that is listed as
// unbuilt roadmap, not as a defect. Only the last three kinds are
// candidates for "orphans the hashes cannot explain".
type orphanKind int

const (
	kindDeleted orphanKind = iota
	kindRenamed
	kindBlockGone
	kindRewritten
	kindOther
)

func (k orphanKind) String() string {
	switch k {
	case kindDeleted:
		return "file deleted"
	case kindRenamed:
		return "file renamed"
	case kindBlockGone:
		return "block deleted"
	case kindRewritten:
		return "block rewritten past weakFloor"
	default:
		return "anything else"
	}
}

type orphanClass struct {
	kind orphanKind
	note string
	at   string // other path, when renamed
}

func classifyOrphans(root string, rs []Record, as []Anchor) []orphanClass {
	out := make([]orphanClass, len(rs))
	var missing []int
	for i, a := range as {
		if a.State != StateOrphaned {
			continue
		}
		p := filepath.Join(root, rs[i].File)
		if _, err := os.Stat(p); err != nil {
			missing = append(missing, i)
			continue
		}
		if a.Integrity > 0 && a.Integrity < weakFloor {
			out[i] = orphanClass{kind: kindRewritten, note: fmt.Sprintf("integrity=%.3f", a.Integrity)}
			continue
		}
		if a.Integrity == 0 {
			out[i] = orphanClass{kind: kindBlockGone, note: "file lives, overlap 0"}
			continue
		}
		out[i] = orphanClass{kind: kindOther, note: fmt.Sprintf("state=%s integrity=%.3f start=%d", a.State, a.Integrity, a.Start)}
	}
	if len(missing) == 0 {
		return out
	}
	elsewhere := findElsewhere(root, rs, missing)
	for _, i := range missing {
		if dest, ok := elsewhere[i]; ok {
			out[i] = orphanClass{kind: kindRenamed, note: "content still in tree", at: dest}
			continue
		}
		out[i] = orphanClass{kind: kindDeleted, note: "path gone, content not found elsewhere"}
	}
	return out
}

// findElsewhere walks the tree once looking for each missing record's hash
// sequence. One walk, not one per orphan: rust's tree is tens of thousands
// of files and the unsalted hashes exist precisely so this comparison is
// possible (anchor.go:110).
func findElsewhere(root string, rs []Record, missing []int) map[int]string {
	want := make(map[int][]string, len(missing))
	skip := map[string]bool{}
	for _, i := range missing {
		want[i] = rs[i].Lines
		skip[rs[i].File] = true
	}
	found := map[int]string{}
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if err == nil && d.IsDir() && skipDir[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if len(found) == len(want) {
			return fs.SkipAll
		}
		rel := filepath.ToSlash(Rel(root, p))
		if skip[rel] {
			return nil
		}
		sig := significant(fileLinesWithin(p, root))
		if len(sig) == 0 {
			return nil
		}
		for i, lines := range want {
			if _, ok := found[i]; ok {
				continue
			}
			h := len(lines)
			if h == 0 || len(sig) < h {
				continue
			}
			for j := 0; j+h <= len(sig); j++ {
				if sameWindow(sig[j:j+h], lines) {
					found[i] = rel
					break
				}
			}
		}
		return nil
	})
	return found
}

func printTaxonomy(w *os.File, rs []Record, as []Anchor, first map[string]string, tax []orphanClass) {
	var nDel, nRen, nBlk, nRew, nOth int
	for i, a := range as {
		if a.State != StateOrphaned {
			continue
		}
		switch tax[i].kind {
		case kindDeleted:
			nDel++
		case kindRenamed:
			nRen++
		case kindBlockGone:
			nBlk++
		case kindRewritten:
			nRew++
		default:
			nOth++
		}
	}
	fmt.Fprintf(w, "survival: taxonomy  file-level (not a hash failure): deleted=%d renamed=%d\n", nDel, nRen)
	fmt.Fprintf(w, "survival: taxonomy  hash-relevant: block-deleted=%d rewritten-past-floor=%d other=%d\n", nBlk, nRew, nOth)

	for i, a := range as {
		if a.State != StateOrphaned {
			continue
		}
		c := tax[i]
		fmt.Fprintf(w, "  ORPHAN %s  %s:%d-%d  %s  integrity=%.3f  first=%s  %s",
			rs[i].ID, rs[i].File, rs[i].Start, rs[i].End, c.kind, a.Integrity, shortSHA(first[rs[i].ID]), c.note)
		if c.at != "" {
			fmt.Fprintf(w, " → %s", c.at)
		}
		fmt.Fprintln(w)
		// Decision text is what a human needs to open the right comment.
		fmt.Fprintf(w, "           %s\n", clip(rs[i].Decision, 160))
	}
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func shortSHA(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}
