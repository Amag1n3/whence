# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

**Two audiences, both real, neither subordinate.** Confirmed by the author
2026-08-03 against a forced single-primary question, which he rejected.

**Solo developers and small teams** who run a coding agent daily on their own
codebase. Their job is making their own agent less destructive: it has no memory
of last Tuesday, and it deletes load-bearing mess because it looks like mess.
They are usually one person deciding for themselves, install in minutes, and
never talk to procurement.

**Businesses adopting it across a team.** Same tool, same features, no crippled
tier — the free CLI is not a demo of a paid product. Their job is preventing a
decision made in an incident review from being quietly undone six months later by
someone who was not in the room, human or agent.

The site must serve both without optimising itself into one. A page written only
for the solo case reads as a hobby script to anyone evaluating adoption; a page
written only for the org case buries the two-minute install that is the actual
on-ramp.

## Product Purpose

Records **why** a piece of code is the way it is, anchors that record to the lines
it was about, and puts it back in front of a coding agent *before* it edits — via
a `PreToolUse` hook — and in front of a reviewer in CI.

Git remembers what changed. This remembers why. The premise: agents write a large
share of merged code, have no memory across sessions, and remove deliberate mess
because it reads as accidental mess. The reasoning existed at the time; the
session ended and only the diff survived.

**The product is the CI exit code.** Everything else is plumbing that leads to it.

Success is one number: times an agent proposed a change that contradicted a
recorded decision and whence caught it.

## Positioning

Anchored decision records, surfaced *before* the edit rather than after.

The mechanism a neighbouring product cannot truthfully copy: **a record knows
whether it still attaches to the code it was written about.** Per-line content
hashes mean a record survives reformatting, insertion, deletion and whole-block
moves, and reports itself orphaned rather than pointing confidently at the wrong
line. Position and content integrity are reported separately, and the percentage
is printed only where it was actually measured.

**It is not a code reviewer and must never become one.** It produces zero
opinions about code quality. That category is well funded and well served; a tool
that starts judging diffs has lost the thing that makes it different. This is the
project's stated failure mode, not a modesty claim.

## Operating Context

- Runs inside an agent session: a `PreToolUse` hook fires before every `Edit` and
  `Write`, measured at 6.3 ms, failing open on every error path.
- Runs in CI as a gate on a pull request, needing `fetch-depth: 0`.
- Runs at a terminal for lookup: `whence <file>:<line>`, `whence log`.
- The store is **committed to the repository** and travels with it, so records
  arrive by `git pull` and a fresh clone has them.
- Records are found by walking up from the edited **file**, the way git finds
  `.git` — never from the session's working directory.

## Capabilities and Constraints

**Shipped:** content-hash anchoring with exact / moved / altered / orphaned
states; evidence pointers that anchor and rot independently of the record;
`whence check` as a CI gate that exits 1 only for records a diff *damaged*, never
for merely touching them; human-vs-agent authorship with a confirmation step; a
committed retraction log; `add`, `rm`, `confirm`, `reground`, `reanchor`,
`backfill`; `capture` (reads a finished session and proposes — writes nothing);
a Claude Code plugin carrying the hook configuration.

**Deliberately not built:** capture that *writes* records; AST-path anchoring;
record signing. These are unbuilt for stated reasons, not backlog. Any surface
describing them says so in present tense.

**Constraints that are decisions:**

- Go standard library only. No runtime dependency, ever.
- Nothing reads git state except `check`'s diff.
- A record may never be grounded in another record — citogenesis is refused in
  code, not in documentation.
- `rm` writes to a retraction log and nothing else may, because that log counts
  how often a record turned out to be *wrong*.
- The binary is `whence`; zsh and ksh shadow it with a builtin. An rc alias or
  a symlink under another name are the two escapes.

**Undecided, and future work must not invent an answer:** nothing beyond the
monetisation shape recorded below.

## Brand Commitments

Confirmed binding by the author 2026-08-03. Future work must not break these:

1. **MIT licence, zero non-stdlib Go dependencies.** Stated as deliberate, not an
   accident of youth.
2. **The falsification criterion stays published, unsoftened.** The site says the
   repository gets archived if the catch count is zero after three months of real
   daily use. It is the most persuasive element on the page precisely because it
   is not softened, and it must not be turned into a "learn more" card.
3. **Records are data, never directives.** Anything rendering a record to an agent
   frames it as a historical note, never an instruction. The store is committed
   and arrives by `git pull`, which makes it a prompt-injection target.
4. **No per-developer AI-authorship attribution, ever.** Aggregate only. This is a
   developer tool, not a surveillance tool, and it will not become one.

**Monetisation:** the local CLI is free, always. End-to-end encrypted team sync
plus a dashboard is where a paid layer would sit. It is **not built**, and every
surface states it as intent in present tense — the same treatment capture and AST
anchoring already get. No pricing, no tiers, no waitlist copy.

**Voice:** argued, specific, and willing to publish what would disprove it. The
commit messages are the reference — long, explaining what was rejected and why,
never changelogs. Prose that hedges, sells, or uses "simply", "just",
"seamlessly", "powerful" or "robust" is off-voice.

## Evidence on Hand

Real, and future work must use these rather than inventing better ones:

- **`whence backfill` on a clean clone of `prometheus/prometheus`** — 9 records
  across 11 files in roughly 3 seconds, after excluding `TODO:` and `FIXME:`
  proposals. Measured 2026-08-09. The strongest demonstration the project has on
  a codebase nobody here wrote.
- **`whence backfill` on the Go standard library** — 50 records across 7,710
  files, down from 257 before `TODO:` and `FIXME:` were dropped from the marker
  set. Measured 2026-08-09.
- **The cost of that narrowing, measured rather than assumed.** A hand-read
  sample of 19 stdlib records before the change held 5 genuine decisions; 2 of
  those 5 survive it. Precision roughly doubled (~26% → ~54% of records being
  decisions) and recall on decisions dropped by more than half. `TODO:`
  sometimes carries a real explanation of current code — `runtime/mgcpacer.go`
  was one of the best records in the corpus and is now missed. The trade was
  taken deliberately: a store nobody reads is worth less than a store missing
  entries, and a missed note is recoverable by hand.
- **Comment culture varies about tenfold, and backfill's yield varies with it.**
  Records per 1,000 source files: Go stdlib 33 before the narrowing, a private
  Node backend 12, a private React frontend 3.5. Backfill is strongest in mature,
  review-heavy codebases and weakest in fast-moving product code.
- **Cross-file invariants are a real and unmodelled shape.** Two independent
  instances found on 2026-08-09 in unrelated corpora: `reflect/value.go:2895`
  ("These values must match ../runtime/select.go:/selectDir") and a private
  repo's "norm() here MUST stay identical to norm() in the ingest worker". A
  record anchors one span in one file, so the hook fires on the side that already
  carries the warning and stays silent on the side where the damage happens.
- **Hook latency: 6.3 ms per fire**, measured as 100 invocations in 0.629 s wall.
- **`whence capture` over this project's own sessions**: 361 thinking blocks with
  zero carrying text, so capture reads the stated explanation and never the
  deliberation. Reasoning is stated *before* each edit, which is not the shape
  post-hoc rationalisation takes.
- **The repository gates itself** in CI, and its store holds 10 records, all
  human-authored.
- Site live at whence.fyi; `go install github.com/Amag1n3/whence@latest` verified
  from a clean machine; tagged `v0.2.0`.

**Absences future work must not fabricate:** there are no users other than the
author, no testimonials, no adoption numbers, no benchmarks against competing
tools, and no third-party press. The falsification clock has not started, because
nobody else has run it.

## Product Principles

1. **Coverage, never verdicts.** Say *these lines are governed by these decisions,
   go and confirm*. Never say a change is wrong.
2. **Be loudly uncertain.** A tool that confidently points at the wrong line
   teaches you to distrust everything it says. Orphaned records claim no line
   number at all; measurements print only where they were measured.
3. **Fail open, always.** A broken or slow whence costs a developer a missing
   record and nothing more — never a blocked edit.
4. **Never hand a human data entry.** Before asking someone to write records,
   harvest what already exists in machine-readable form. That question is what
   produced `backfill`.
5. **Publish the thing that would kill it.** The unresolved doubt, the retraction
   count, the kill number. Credibility here comes from what the project admits,
   not from what it claims.

## Accessibility & Inclusion

**WCAG 2.2 AA is a stated target**, chosen by the author 2026-08-03 over
"sensible defaults, no formal standard".

Already held: `:focus-visible` rings, `prefers-reduced-motion` honoured for both
scroll behaviour and animation, real `<a href>` navigation that works without JS,
Radix primitives for keyboard and screen-reader semantics.

**Open obligations this commitment creates** — these are work, not decoration:

- Contrast of `dim` and `muted-foreground` against `basin` at body sizes. The
  mineral palette deliberately avoids white text, which is exactly where AA
  pressure lands.
- Interactive targets at 24×24 CSS pixels minimum (2.2's new criterion).
- The palette may have to move to satisfy this, and moving it is allowed —
  the commitment outranks the current values.
