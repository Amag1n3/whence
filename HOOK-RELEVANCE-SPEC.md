# Spec: make `whence hook pre` relevant

Handoff spec. Implement in this repo. Stdlib only, fail open, match the existing
comment voice (dense why-comments; `ponytail:` markers naming a ceiling and its
upgrade path). Untracked file — delete it once the work lands.

## Why this exists (the measurement)

Two ROF repos, one week of real use, `surfaced.jsonl` in both plus the session
transcripts. **19 real surfacings. One changed what the agent did.**

| Event | What surfaced | Outcome |
|---|---|---|
| 08-09 06:21 ×2 | records seeded 5 min earlier, same session | self-echo |
| 08-12 08:16–08:18 ×8 | `e0cc4b` + `4b3d68`, on 8 consecutive edits to one 3.6k-line file | agent's own words in the transcript: *"Whence just fired… Neither conflicts with what I'm doing."* |
| 08-12 08:58 ×4 | `683a93`, agent-written 30 min earlier, already contradicted by the edit in flight | negative; retracted same day |
| **08-12 15:50 ×2** | **`ddfb67` on `enterpriseProfileConfig.js`** | **the hit — see below** |
| 08-12 15:53–15:54 ×3 | the same two stale frontend records again | none |

**The hit, exactly.** An edit deleted the fallback hop inside
`resolveProfileForCase`. `ddfb67` surfaced; its `why` contains *"resolveProfileForCase
only calls this when there is no usable stamp, so stamp-first changes nothing for
that path."* Thirteen seconds later the agent re-read that region and rewrote the
comment above `resolveOrgIdForCase`, because the caller relationship it asserted
had just stopped existing. A stale-invariant catch — the exact category this tool
claims.

**The finding that decides the design:** `ddfb67` is anchored to lines 211–239.
The edit was at ~259–291. **They do not overlap.** It landed because the record
*named the function being edited*. Relevance here is identifier-shaped, not
line-shaped. Range overlap alone would have suppressed the only win.

## What blocked this (read before writing code)

`renderContext` in `main.go:196-203` carries this:

> Real relevance ranking (does this record concern the lines actually being
> changed?) needs a diff. […] but this is PreToolUse, which fires *before* the
> edit, so there is no diff here to rank against and **there cannot be**.

That is false, and it is why the feature was never built. The PreToolUse payload
carries the whole `tool_input`, and for `Edit` that includes **`old_string`** —
the pre-image, verbatim, present in the file *right now*. `locateSpan`
(`capture.go:413`) already turns exactly that into a line span. The ranking
signal was in the payload the whole time.

Replace that comment. A confident "cannot" in a comment is a load-bearing claim.

---

## Change 1 — gate on relevance, don't dump the file

**`hookIn` (`main.go:125`)** gains what the payload already sends:

```go
type hookIn struct {
	Cwd       string `json:"cwd"`
	SessionID string `json:"session_id"`
	ToolInput struct {
		FilePath   string `json:"file_path"`
		OldString  string `json:"old_string"`  // Edit — the pre-image
		NewString  string `json:"new_string"`  // Edit
		Content    string `json:"content"`     // Write
		ReplaceAll bool   `json:"replace_all"` // Edit
	} `json:"tool_input"`
}
```

**In `hookPre`**, after `Match` returns hits, split them into *on-target* and
*off-target*.

Edited span: `locateSpan(fileLinesWithin(abs, root), in.ToolInput.OldString)`.
Use `fileLinesWithin` (the guarded reader the hook path already uses), not
`fileLines`.

A record is **on-target** if either clause holds:

- **(a) span overlap** — its current anchor span, or its recorded span when the
  anchor is lost, intersects the edited span padded by 3 lines either side.
- **(b) name overlap** — any code-ish token drawn from `Decision + " " + Why`
  appears in `OldString + NewString` (or `Content` for Write).

**Code-ish token**, hand-rolled scan, no regexp:

- `[A-Za-z_][A-Za-z0-9_]*`, length ≥ 6, and
- contains an underscore, **or** an uppercase letter at any position after the
  first.

That second condition is the whole trick: it admits how these records actually
name code and excludes ordinary English. Validated on the live corpus —

| Token | Admitted | From |
|---|---|---|
| `resolveProfileForCase` | yes | `ddfb67` — this is the hit |
| `enterpriseProfileId` | yes | `ddfb67` |
| `respondentOwnsCase` | yes | `1f9dba` |
| `ERR_NGROK_6024` | yes | `e0cc4b` |
| `TimesNewRoman` | yes | `4b3d68` |
| `ngrok`, `heading`, `config`, `preview`, `interstitial` | no | prose |

**Rendering.** On-target records render exactly as they do today, unchanged.
Off-target records collapse to **one line total**, not one line each:

```
- 2 other record(s) on this file, none on these lines or names: 4b3d68, e0cc4b — run: whence src/Pages/RespondentOnboarding/OnboardingFlow.js
```

Never drop them silently. A tool that shows less than it saw is the failure this
project exists to complain about — but a pointer is not a dump.

**Fail open to today's behaviour** (render everything in full) when any of:
`OldString` empty (Write, NotebookEdit); `locateSpan` returns `0,0` (the
pre-image is no longer in the file); `ReplaceAll` is true (the pre-image occurs
more than once and `locateSpan` only finds the first).

Clause (a) is unexercised by this corpus — the one hit came through (b). Keep it
anyway: a record anchored to the exact lines being rewritten is the case this
tool was built for, and it costs three lines.

## Change 2 — stop repeating yourself within a session

Eight identical injections in 131 seconds is what teaches an agent to skim past
the block. On-target records are exempt — repetition on the lines you are
touching is the point. The **tail line** is suppressed after its first
appearance per `(session_id, file)`.

`appendSurfaced` (`main.go:308`) writes:

```json
{"at":"…","session":"ac8bfbb8-…","file":"…","records":["ddfb67"],"tail":["4b3d68","e0cc4b"]}
```

Before rendering, scan `surfaced.jsonl`; if any prior entry carries this
`session` and this `file`, omit the tail line. Read error, missing field, or
absent `session_id` → render the tail (fail open).

This also makes `surfaced.jsonl` finally answer "which session" — reconstructing
that from timestamps is what the measurement above cost an hour to do.

`ponytail:` note the ceiling — reads the whole surfaced log per edit, one line
per surfacing, fine until a store gets very large; bound it by date if that ever
shows up in the hook's latency.

## Change 3 — `-s agent` must imply `-agent`

`add` takes two flags that read identically at the call site:

- `-s agent` → `Source`, a free-text provenance label. Cosmetic.
- `-agent` → `Author`, which is what `unchecked()` (`record.go:80`) and the
  `UNCHECKED — an agent wrote this and no human has confirmed it` warning key
  on, and what `whence confirm` clears.

Every one of the 13 records in the ROF stores was written `-s agent` and carries
`"author":"human"` with a `verified` date `add` stamped on itself
(`author.go:448-449`). So `683a93` — agent-written, wrong within 30 minutes —
surfaced four times labelled `· agent`, with no UNCHECKED marker and a
self-issued verification date. The §17 self-feeding guard was silently off, and
the agent that wrote those records believed it was on.

Fix in `addCmd`: when `*source == "agent"` and `-agent` was not passed, treat the
record as agent-authored. Do not migrate existing records — rewriting history to
look better is not a fix.

## Non-goals — do not build these

No PostToolUse hook. No `Read` matcher. No blocking/deny path (the hit worked
fine as a post-write annotation; the write is uncommitted and 13 seconds from
correction). No scoring, no model call, no ranking beyond on-target/off-target.
No new dependencies. No record-format change beyond the two `surfaced.jsonl`
fields. No retro-migration. No changes to `check`, `capture`, or `backfill`.

## Tests (package `main`, stdlib `testing`, table-driven, matching the existing files)

1. `codeTokens` — the table above, both admitted and rejected.
2. On-target by name: `ddfb67`'s text against an `old_string` containing
   `const resolveProfileForCase = async (complain) => {` → on-target.
3. Off-target: `e0cc4b` and `4b3d68` against an `old_string` of
   `export default function OnboardingFlow() {` → both off-target, and assert the
   rendered output contains exactly one tail line naming both ids.
4. On-target by span, including the ±3 padding boundary and the anchor-lost
   fallback to the recorded span.
5. Tail suppression: two `hookPre` runs, same session + file → the second output
   has no tail line; a third run with a different session → tail returns.
6. Fail-open, each rendering everything as today: unparseable stdin, empty
   `old_string`, `old_string` absent from the file, `replace_all: true`.
7. `-s agent` without `-agent` → `Author == authorAgent`, `Verified == ""`.

The end-to-end check is a hand-fed payload, no live app required:

```sh
echo '{"session_id":"t1","cwd":"'"$PWD"'","tool_input":{"file_path":"'"$PWD"'/some.go","old_string":"…"}}' | ./whence hook pre
```

## Definition of done

`go test ./...` green, and on the corpus above the gate keeps `ddfb67` in full
and reduces the twelve `OnboardingFlow.js` surfacings to one tail line in the
first, none after.
