package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderContextOmitsEmptyWhy(t *testing.T) {
	r := Resolved{
		Record: Record{Date: "2026-08-09", File: "main.go", Decision: "keep this path", Source: "NOTE comment"},
		Anchor: Anchor{State: StateExact, Start: 1, End: 1},
	}
	got := renderContext([]Resolved{r})
	if strings.Contains(got, "  why:") {
		t.Fatalf("empty why should not produce a why row:\n%s", got)
	}

	r.Why = "the caller depends on it"
	got = renderContext([]Resolved{r})
	if !strings.Contains(got, "  why: the caller depends on it\n") {
		t.Fatalf("non-empty why should produce a why row:\n%s", got)
	}
}

func TestCodeTokens(t *testing.T) {
	cases := []struct {
		tok  string
		want bool
	}{
		{"resolveProfileForCase", true},
		{"enterpriseProfileId", true},
		{"respondentOwnsCase", true},
		{"ERR_NGROK_6024", true},
		{"TimesNewRoman", true},
		{"ngrok", false},
		{"heading", false},
		{"config", false},
		{"preview", false},
		{"interstitial", false},
	}
	for _, tc := range cases {
		got := codeTokens(tc.tok)
		has := false
		for _, tok := range got {
			if tok == tc.tok {
				has = true
			}
		}
		if has != tc.want {
			t.Errorf("%q admitted=%v, want %v (tokens %q)", tc.tok, has, tc.want, got)
		}
	}
}

func TestOnTargetByName(t *testing.T) {
	r := Resolved{Record: Record{
		ID:       "ddfb67",
		Start:    211,
		End:      239,
		Decision: "stamp-first profile resolution",
		Why:      "resolveProfileForCase only calls this when there is no usable stamp, so stamp-first changes nothing for that path.",
	}}
	hay := "const resolveProfileForCase = async (complain) => {"
	if !onTarget(r, 259, 291, hay) {
		t.Fatal("ddfb67 names resolveProfileForCase and must be on-target even when the spans miss")
	}
}

func TestOffTargetRendersOneTailLine(t *testing.T) {
	off := []Resolved{
		{Record: Record{ID: "e0cc4b", Start: 80, End: 90, Decision: "map ERR_NGROK_6024", Why: "the tunnel page"}, Anchor: Anchor{Start: 80, End: 90}},
		{Record: Record{ID: "4b3d68", Start: 200, End: 210, Decision: "embed TimesNewRoman", Why: "legacy heading"}, Anchor: Anchor{Start: 200, End: 210}},
	}
	hay := "export default function OnboardingFlow() {"
	for _, r := range off {
		if onTarget(r, 10, 10, hay) {
			t.Errorf("%s should be off-target against OnboardingFlow", r.ID)
		}
	}
	got := renderContext(nil) + formatTail(off, "src/Pages/RespondentOnboarding/OnboardingFlow.js")
	if n := strings.Count(got, "other record(s) on this file"); n != 1 {
		t.Fatalf("want exactly one tail line, got %d:\n%s", n, got)
	}
	if !strings.Contains(got, "e0cc4b") || !strings.Contains(got, "4b3d68") {
		t.Fatalf("tail must name both ids:\n%s", got)
	}
	if strings.Contains(got, "map ERR_NGROK_6024") || strings.Contains(got, "embed TimesNewRoman") {
		t.Fatalf("off-target records must not dump in full:\n%s", got)
	}
}

func TestOnTargetBySpan(t *testing.T) {
	// Edit 10-12, padded ±3 → 7-15.
	cases := []struct {
		name         string
		recS, recE   int
		ancS, ancE   int
		editS, editE int
		want         bool
	}{
		{"overlaps edit", 10, 12, 10, 12, 10, 12, true},
		{"pad lower bound", 5, 7, 5, 7, 10, 12, true},
		{"below pad", 5, 6, 5, 6, 10, 12, false},
		{"pad upper bound", 15, 16, 15, 16, 10, 12, true},
		{"above pad", 16, 17, 16, 17, 10, 12, false},
		{"lost anchor is on-target", 1, 2, 0, 0, 10, 12, true},
		{"current span wins over recorded", 10, 12, 40, 42, 10, 12, false},
	}
	for _, tc := range cases {
		r := Resolved{
			Record: Record{Start: tc.recS, End: tc.recE},
			Anchor: Anchor{Start: tc.ancS, End: tc.ancE},
		}
		if got := onTarget(r, tc.editS, tc.editE, ""); got != tc.want {
			t.Errorf("%s: onTarget=%v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestHookPreOnTargetByNameAndOneTail(t *testing.T) {
	dir, abs, named, other := hookRepo(t)
	payload := hookJSON(t, "s1", abs, "const resolveProfileForCase = async (complain) => {", "", false)
	ctx := hookContext(t, runHookPre(t, payload))
	if !strings.Contains(ctx, named.Decision) {
		t.Fatalf("named record must render in full:\n%s", ctx)
	}
	if strings.Contains(ctx, other.Decision) {
		t.Fatalf("off-target record must not dump in full:\n%s", ctx)
	}
	if n := strings.Count(ctx, "other record(s) on this file"); n != 1 {
		t.Fatalf("want one tail line, got %d:\n%s", n, ctx)
	}
	if !strings.Contains(ctx, other.ID) {
		t.Fatalf("tail must name %s:\n%s", other.ID, ctx)
	}
	_ = dir
}

func TestHookPreSuppressesTailPerSession(t *testing.T) {
	_, abs, named, other := hookRepo(t)
	old := "export default function OnboardingFlow() {"
	first := hookContext(t, runHookPre(t, hookJSON(t, "s1", abs, old, "", false)))
	if n := strings.Count(first, "other record(s) on this file"); n != 1 {
		t.Fatalf("first fire should show the tail, got %d:\n%s", n, first)
	}
	if !strings.Contains(first, named.ID) || !strings.Contains(first, other.ID) {
		t.Fatalf("tail must name both ids:\n%s", first)
	}
	second := runHookPre(t, hookJSON(t, "s1", abs, old, "", false))
	if second != "" && strings.Contains(hookContext(t, second), "other record(s) on this file") {
		t.Fatalf("same session+file must not repeat the tail:\n%s", second)
	}
	third := hookContext(t, runHookPre(t, hookJSON(t, "s2", abs, old, "", false)))
	if !strings.Contains(third, "other record(s) on this file") {
		t.Fatalf("a new session must see the tail again:\n%s", third)
	}
}

func TestHookPreFailsOpen(t *testing.T) {
	_, abs, named, other := hookRepo(t)
	old := "export default function OnboardingFlow() {"
	cases := []struct {
		name    string
		payload string
		empty   bool
	}{
		{"unparseable", "not json", true},
		{"empty old_string", hookJSON(t, "f1", abs, "", "", false), false},
		{"old_string absent from file", hookJSON(t, "f2", abs, "this text is not in the file at all", "", false), false},
		{"replace_all", hookJSON(t, "f3", abs, old, "", true), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runHookPre(t, tc.payload)
			if tc.empty {
				if strings.TrimSpace(got) != "" {
					t.Fatalf("unparseable must print nothing, got %q", got)
				}
				return
			}
			ctx := hookContext(t, got)
			if !strings.Contains(ctx, named.Decision) || !strings.Contains(ctx, other.Decision) {
				t.Fatalf("fail-open must render every record in full:\n%s", ctx)
			}
			if strings.Contains(ctx, "other record(s) on this file") {
				t.Fatalf("fail-open must not collapse to a tail:\n%s", ctx)
			}
		})
	}
}

func hookRepo(t *testing.T) (dir, abs string, named, other Resolved) {
	t.Helper()
	dir = t.TempDir()
	chdir(t, dir)
	lines := make([]string, 40)
	for i := range lines {
		lines[i] = fmt.Sprintf("var unique_line_%02d = %d", i+1, i+1)
	}
	lines[2] = "const resolveProfileForCase = async (complain) => {"
	lines[3] = "	return null"
	lines[4] = "}"
	lines[9] = "export default function OnboardingFlow() {"
	lines[10] = "	return null"
	lines[11] = "}"
	writeFile(t, dir, "flow.js", lines)

	var err error
	named, _, err = add("flow.js", 20, 21, "stamp-first profile resolution",
		"resolveProfileForCase only calls this when there is no usable stamp, so stamp-first changes nothing for that path.",
		"manual", authorHuman, nil)
	if err != nil {
		t.Fatal(err)
	}
	other, _, err = add("flow.js", 30, 31, "map ERR_NGROK_6024 on the tunnel page",
		"the ngrok interstitial", "manual", authorHuman, nil)
	if err != nil {
		t.Fatal(err)
	}
	return dir, filepath.Join(dir, "flow.js"), named, other
}

func hookJSON(t *testing.T, session, abs, old, neu string, replaceAll bool) string {
	t.Helper()
	in := hookIn{Cwd: filepath.Dir(abs), SessionID: session}
	in.ToolInput.FilePath = abs
	in.ToolInput.OldString = old
	in.ToolInput.NewString = neu
	in.ToolInput.ReplaceAll = replaceAll
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func runHookPre(t *testing.T, payload string) string {
	t.Helper()
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(inW, payload); err != nil {
		t.Fatal(err)
	}
	inW.Close()

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldIn, oldOut := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = inR, outW
	defer func() { os.Stdin, os.Stdout = oldIn, oldOut }()

	hookPre()
	outW.Close()
	b, err := io.ReadAll(outR)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestVersionLine(t *testing.T) {
	for _, c := range []struct {
		version string
		dirty   bool
		want    string
	}{
		// A tagged `go install` build: the module version and nothing else.
		{"v0.3.1", false, "whence v0.3.1"},
		// `go build` after v0.3.1 stamps a pseudo-version, which already
		// carries the revision — so no second copy of it here.
		{"v0.3.2-0.20260817184129-5d8953281fb3", false, "whence v0.3.2-0.20260817184129-5d8953281fb3"},
		// Go stamps its own +dirty from the same setting the flag comes from.
		// Trimmed, so the state is spelled out once rather than twice.
		{"v0.3.2-0.20260817184129-5d8953281fb3+dirty", true, "whence v0.3.2-0.20260817184129-5d8953281fb3 (uncommitted changes)"},
		// Built in a way that carries no build info at all.
		{"", false, "whence — build information not available"},
	} {
		if got := versionLine(c.version, c.dirty); got != c.want {
			t.Errorf("versionLine(%q, %v) = %q, want %q", c.version, c.dirty, got, c.want)
		}
	}
}

func hookContext(t *testing.T, raw string) string {
	t.Helper()
	var out hookOut
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("hook output is not JSON: %v\n%s", err, raw)
	}
	return out.HookSpecificOutput.AdditionalContext
}
