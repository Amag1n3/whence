package main

// Capture, read-only.
//
// The premise this project rests on is that the agent HAD the reasoning and the
// session threw it away. That is half true, and the half that is false decides
// this command's whole shape.
//
// Claude Code already writes every session to
// ~/.claude/projects/<slug>/<session>.jsonl and keeps it. So there is nothing to
// intercept: no PostToolUse trail to maintain, no per-edit write path, no hook
// at all. The trail exists; what was missing was a reader. That is why this
// command exists before any hook does — an instrument costs nothing to be wrong
// about, and a hook writing records does not.
//
// What the transcript does NOT contain is the deliberation. Measured across
// every session in this project: 361 thinking blocks, 0 of them carrying text —
// the plaintext is dropped and only the signature is persisted. What survives is
// the assistant's prose to the user, the tool calls, and the user's own turns.
// So capture reads the EXPLANATION, never the weighing that produced it, and
// DECISIONS §9's doubt is therefore not a risk this might run into later. It is
// the starting condition: the only available source is the rationalisation.
//
// Which is exactly why nothing here writes. This prints what it found and stops.
// Deciding whether a stated reason is the real one is the open question, and the
// point of the command is to put enough real pairs in front of a human to answer
// it. A capture that wrote records would be answering it by assumption.
//
// ponytail: pairs each edit with the nearest preceding assistant message and the
// user turn before it. Filtering stays minimal: one label — "did a failed tool
// result precede this edit" (followsSurprise) — reported as a count in the
// footer. That is the cheap half of §22.5's surprise signal, and the rest, a
// real filter, should come from reading a few hundred true pairs, not from
// guessing now. Upgrade path is a model pass over these pairs — which breaks
// the stdlib-only rule, so it is a decision to take on purpose, not to drift
// into.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// transcriptEntry is one line of a session transcript. Only the fields capture
// reads are declared; the format carries a great deal more and is not ours.
type transcriptEntry struct {
	Type    string `json:"type"`
	IsMeta  bool   `json:"isMeta"`
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"` // a string, or an array of blocks
	} `json:"message"`
	Timestamp string `json:"timestamp"`
}

// contentBlock is one element of a message's content array.
type contentBlock struct {
	Type    string          `json:"type"`
	Text    string          `json:"text"`
	Name    string          `json:"name"`
	Content json.RawMessage `json:"content"` // tool_result payload; a string or blocks
	Input   struct {
		FilePath  string `json:"file_path"`
		NewString string `json:"new_string"` // Edit
		Content   string `json:"content"`    // Write
	} `json:"input"`
}

// moment is one edit and the reasoning standing next to it in the session.
type moment struct {
	At     string // RFC3339, as the transcript recorded it
	File   string // absolute, as the tool reported it
	Tool   string // Edit or Write
	Text   string // what the edit put into the file
	Asked  string // the user turn that preceded it
	Said   string // the assistant's last prose before the tool call
	Result string // the last tool result before the edit, capped
}

func captureCmd(args []string) {
	path := ""
	if len(args) > 0 {
		path = args[0]
	}
	if path == "" {
		p, err := newestTranscript()
		if err != nil {
			fmt.Fprintln(os.Stderr, "whence:", err)
			os.Exit(1)
		}
		path = p
	}

	all, err := readTrail(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "whence:", err)
		os.Exit(1)
	}
	ms, elsewhere := hereOnly(all)

	fmt.Println(path)
	if len(ms) == 0 {
		fmt.Println("\nno edits in this session")
		reportElsewhere(elsewhere)
		return
	}

	located, reasoned, surprised := 0, 0, 0
	for _, m := range ms {
		start, end := locateSpan(fileLines(m.File), m.Text)
		where := shortPath(m.File)
		if start != 0 {
			located++
			where = fmt.Sprintf("%s:%d-%d", where, start, end)
		} else {
			// Same rule as an anchor that cannot be found: say so rather than
			// point somewhere plausible. A block edited again later in the
			// session genuinely has no span to report.
			where += " (block changed again later; no current span)"
		}
		if hasReason(m.Said) {
			reasoned++
		}
		if followsSurprise(m.Result) {
			surprised++
		}

		fmt.Printf("\n  ● %s · %s · %s\n", m.At, m.Tool, where)
		if m.Asked != "" {
			fmt.Print(wrap("asked: "+m.Asked, "    ", "           "))
		}
		if m.Said != "" {
			fmt.Print(wrap("said:  "+m.Said, "    ", "           "))
		} else {
			fmt.Println("    said:  — nothing stated before this edit")
		}
	}

	// The last line is the instrument. DECISIONS §9 asks whether stated reasoning
	// is the real cause; before that can be asked, this says how often there is
	// any reasoning at all — and that split is the whole reading.
	//
	// The first version counted any non-empty message and reported 100% on three
	// real sessions, because "Now the code." is a non-empty message — which was
	// appendSurfaced's over-counting bug rebuilt in a new file.
	//
	// hasReason replaced it and undercounts instead, which is why this says WORD
	// rather than "stating a reason". It is a keyword test built for terse code
	// comments; prose carries a reason structurally, and "it promoted a guess to
	// a certainty, which is the one thing anchor.go exists to prevent" contains
	// no word from the list. It also false-positives in this repo specifically,
	// where "reason" is ordinary vocabulary rather than a marker.
	//
	// So the number is a floor and says so. Naming the conclusion when only the
	// proxy was measured is the same error as printing a confidence score that
	// was never taken — see the integrity rule in main.go.
	//
	// The surprise count is the §22.5 reading, separate from the reason-word
	// floor: it does not claim the reason is real, only that something
	// unexpected preceded the edit, which is the shape a real reason takes.
	fmt.Printf("\n%d edit(s) · %d with a current span · %d with a reason word (lexical floor), %d without · %d after a failed tool result (surprise signal)\n",
		len(ms), located, reasoned, len(ms)-reasoned, surprised)
	reportElsewhere(elsewhere)
	fmt.Println("nothing was written; capture proposes, it does not record")
}

// hereOnly splits edits into those belonging to the store capture is reporting
// for and those belonging somewhere else.
//
// A session edits whatever it is pointed at — another repo, a vault note, a
// settings file. Those are real edits and they are not this store's business:
// records resolve by FILE (DECISIONS §12), so an edit under a different root
// could only ever become a record in a different store. Measured on one real
// session, 11 of 15 edits were to a file on the Desktop with no store above it
// at all.
//
// They are counted rather than dropped in silence. A tool that quietly shows
// fewer edits than the session made is telling you it saw everything, which is
// the failure this whole project is about.
// Roots are compared with os.SameFile rather than by string. A transcript
// records the path the tool was handed while the cwd comes back resolved, and on
// this machine that is the difference between /var and /private/var — every edit
// in the session's own repo counted as somewhere else, silently.
func hereOnly(ms []moment) (here []moment, elsewhere int) {
	var root os.FileInfo
	if cwd, err := os.Getwd(); err == nil {
		if _, r, ok := FindStore(filepath.Join(cwd, "x")); ok {
			root, _ = os.Stat(r)
		}
	}
	if root == nil {
		return ms, 0 // no store here to be outside of
	}
	for _, m := range ms {
		_, r, ok := FindStore(m.File)
		if !ok {
			elsewhere++
			continue
		}
		fi, err := os.Stat(r)
		if err != nil || !os.SameFile(root, fi) {
			elsewhere++
			continue
		}
		here = append(here, m)
	}
	return here, elsewhere
}

func reportElsewhere(n int) {
	if n > 0 {
		fmt.Printf("%d edit(s) in other repos, not shown\n", n)
	}
}

// newestTranscript finds the most recent session for the current directory.
//
// Claude Code names a project directory after the absolute path with every
// separator replaced by a dash. Resolving from cwd is correct HERE and nowhere
// else in this tool: a transcript belongs to a session, and a session is rooted
// at a directory. Records belong to files, which is why FindStore walks up from
// a file instead — see DECISIONS §12.
func newestTranscript() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".claude", "projects", strings.ReplaceAll(cwd, string(filepath.Separator), "-"))

	des, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("no sessions recorded for %s", cwd)
	}
	type cand struct {
		path string
		mod  int64
	}
	var cs []cand
	for _, d := range des {
		if d.IsDir() || filepath.Ext(d.Name()) != ".jsonl" {
			continue
		}
		info, err := d.Info()
		if err != nil {
			continue
		}
		cs = append(cs, cand{filepath.Join(dir, d.Name()), info.ModTime().UnixNano()})
	}
	if len(cs) == 0 {
		return "", fmt.Errorf("no sessions recorded for %s", cwd)
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].mod > cs[j].mod })
	return cs[0].path, nil
}

// readTrail walks a transcript and returns every edit with the reasoning that
// stood next to it.
//
// Streamed with a json.Decoder rather than scanned by line: a single transcript
// entry can carry a whole file's contents, and bufio.Scanner caps a token at
// 64KB and reports the overrun as end-of-input. Silently stopping halfway
// through a session is the failure mode this tool exists to complain about.
func readTrail(path string) ([]moment, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		ms         []moment
		asked      string
		said       string
		lastResult string
		dec        = json.NewDecoder(f)
		writes     = map[string]bool{"Edit": true, "Write": true, "NotebookEdit": true}
	)
	for {
		var e transcriptEntry
		if err := dec.Decode(&e); err != nil {
			break // a truncated final line is normal for a live session
		}
		if e.IsMeta || len(e.Message.Content) == 0 {
			continue
		}

		// A user turn arrives as a plain string. The array form is a tool result
		// being handed back, which is machinery rather than anything anyone said.
		if e.Type == "user" {
			var s string
			if json.Unmarshal(e.Message.Content, &s) == nil && strings.TrimSpace(s) != "" {
				asked, said = s, ""
			} else if res := toolResultText(e.Message.Content); res != "" {
				// The surprise signal (§22.5): what happened right before the
				// edit. A tool result is machinery, not words, but its failure
				// is the most reliable marker there is.
				lastResult = res
			}
			continue
		}
		if e.Type != "assistant" {
			continue
		}

		var bs []contentBlock
		if err := json.Unmarshal(e.Message.Content, &bs); err != nil {
			continue
		}
		for _, b := range bs {
			switch {
			case b.Type == "text" && strings.TrimSpace(b.Text) != "":
				said = b.Text
			case b.Type == "tool_use" && writes[b.Name]:
				text := b.Input.NewString
				if text == "" {
					text = b.Input.Content
				}
				if b.Input.FilePath == "" || text == "" {
					continue
				}
				ms = append(ms, moment{
					At:     e.Timestamp,
					File:   b.Input.FilePath,
					Tool:   b.Name,
					Text:   text,
					Asked:  asked,
					Said:   said,
					Result: lastResult,
				})
			}
		}
	}
	return ms, nil
}

// toolResultText pulls the text out of a tool_result block. The content is
// either a string or an array of blocks; whatever is not text is dropped,
// because the surprise signal lives in what a tool actually reported.
func toolResultText(raw json.RawMessage) string {
	var bs []contentBlock
	if err := json.Unmarshal(raw, &bs); err != nil {
		return ""
	}
	for _, b := range bs {
		if b.Type != "tool_result" || len(b.Content) == 0 {
			continue
		}
		var s string
		if json.Unmarshal(b.Content, &s) == nil {
			return s
		}
		var inner []contentBlock
		if err := json.Unmarshal(b.Content, &inner); err != nil {
			continue
		}
		var sb strings.Builder
		for _, t := range inner {
			if t.Type == "text" {
				sb.WriteString(t.Text)
			}
		}
		return sb.String()
	}
	return ""
}

// followsSurprise reports whether a tool result looked like a failure.
//
// §22.5's finding, measured: every reason-bearing statement in the corpus
// follows something unexpected — a failing test, a wrong fixture, --help
// walking a path. Announcements introduce runs of edits executing an
// already-agreed plan. So the cheap, reliable filter is not vocabulary in the
// prose (that is hasReason's job, and it lies about prose); it is what the
// tool result that preceded the edit said.
//
// ponytail: marker list on machine output, where the words mean what they
// say — "FAIL", "panic", "exit status" are not ordinary vocabulary in tool
// output the way "reason" is in prose. Widen only against real misses from a
// real session, never by imagining phrasings.
func followsSurprise(res string) bool {
	for _, m := range surpriseMarkers {
		if strings.Contains(res, m) {
			return true
		}
	}
	return false
}

// surpriseMarkers are what a failed tool result looks like. Narrow on
// purpose, in the spirit of reasonWords: a miss is recoverable (the human
// reads the pair anyway), and a list that fires on every tool result would
// reclassify the whole corpus as surprise, which is the same lie the
// over-counting footer told.
var surpriseMarkers = []string{
	"FAIL", "panic", "exit status", "Error", "error:",
}

// locateSpan reports where text sits in lines now, or 0,0 if it does not.
//
// Whole-block exact match, and no fallback. resolveAnchor has a window scan for
// a block that has since been damaged, and it is right there — but it answers a
// different question. That one re-finds a span somebody already chose to record.
// This one is proposing a span nobody has vouched for yet, and a best guess laid
// under a proposal reads as a measurement. If the block moved on, the honest
// answer is that there is nothing here to point at.
func locateSpan(lines []string, text string) (start, end int) {
	want := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	for len(want) > 0 && strings.TrimSpace(want[0]) == "" {
		want = want[1:]
	}
	for len(want) > 0 && strings.TrimSpace(want[len(want)-1]) == "" {
		want = want[:len(want)-1]
	}
	if len(want) == 0 || len(want) > len(lines) {
		return 0, 0
	}
	for i := 0; i+len(want) <= len(lines); i++ {
		if sameLines(lines[i:i+len(want)], want) {
			return i + 1, i + len(want)
		}
	}
	return 0, 0
}

func sameLines(a, b []string) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// shortPath renders a path relative to the store's root when it is inside one,
// so output reads like the rest of the tool rather than like absolute noise.
func shortPath(abs string) string {
	if _, root, ok := FindStore(abs); ok {
		return filepath.ToSlash(Rel(root, abs))
	}
	return abs
}

const (
	wrapWidth = 74
	wrapCap   = 420 // enough to judge whether a reason is a reason
)

// wrap prints prose at a readable width under a hanging indent. Collapses
// newlines: an agent's message is markdown with headings and code fences in it,
// and reproducing that here would bury the one line worth reading.
func wrap(s, first, hang string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > wrapCap {
		s = s[:wrapCap] + "…"
	}
	var b strings.Builder
	indent, width := first, wrapWidth
	for len(s) > width {
		cut := strings.LastIndex(s[:width], " ")
		if cut <= 0 {
			cut = width
		}
		fmt.Fprintf(&b, "%s%s\n", indent, s[:cut])
		s = strings.TrimSpace(s[cut:])
		indent, width = hang, wrapWidth-len(hang)+len(first)
	}
	fmt.Fprintf(&b, "%s%s\n", indent, s)
	return b.String()
}
