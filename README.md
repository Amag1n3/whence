# whence

**Git remembers what changed. `whence` remembers why — and tells your AI agent before it changes it back.**

---

> [!WARNING]
> **Status: design stage. Nothing is built yet.**
>
> This README is written first, on purpose — it's the design document and the
> spec at the same time. Every command below describes intended behaviour, not
> shipped behaviour. Don't `go install` this expecting it to work.
>
> Started 2026-07-31.

---

## The problem

Everyone who has worked on a team has this story: someone new opens a file, finds
code that looks messy or redundant, cleans it up, and takes production down. The
mess was load-bearing. It was written that way after an incident, and the reason
was never written anywhere the newcomer would look.

That used to happen occasionally, at human speed.

Now every team has an infinitely fast engineer with total amnesia. Coding agents
write a large share of merged code and they do exactly the same thing — see
verbose retry logic, judge it redundant, simplify it — many times a day, with no
memory of last Tuesday, let alone last year's outage.

Here's what makes it fixable: **the agent had the reasoning.** It weighed options,
hit constraints, rejected approaches. Then the session ended and all of it was
discarded. What reached the repository was a diff.

`whence` captures that reasoning as it happens, anchors it to the lines it was
about, and puts it back in front of the next agent before it edits.

## What it does

```console
$ whence src/auth/session.go:142

  ● 2026-07-27 · code-review · confidence 0.94
    Don't write shared session keys from this flow.
    "userToken", "userId" and "role" are all read by the admin
    dashboard — same origin, same app. Writing them here signs a
    staff user out mid-session and surfaces as a 200 with an
    error body, which is why it looks like a timeout.
    → namespace all three
    anchored via content-hash · source: code-review log
```

The same record reaches your agent through MCP *before* it edits, so it doesn't
reintroduce the bug. And in CI:

```console
$ whence check --base origin/main

  ✗ src/auth/session.go:145
    writes localStorage["role"] — contradicts record #4f2a (2026-07-27)
    "namespace all three keys"

  1 violation. exit 1
```

That exit code is the product. Everything else is plumbing that leads to it.

## How it works

1. **Capture.** Subscribes to your agent's hooks (`PostToolUse`, `Stop`, …) and
   records the decision trail as the session runs. Redaction happens here, before
   anything is written.
2. **Anchor.** Binds each record to a file, a line range, a content hash and a
   tree-sitter AST path — so it survives reformatting, drift and most refactors.
3. **Surface.** `whence <file>:<line>` from the terminal, an MCP server for the
   agent, and `whence check` as a CI gate.

Records live in `.cairn`-style local storage at `.whence/` — SQLite, gitignored,
never committed.

### Anchoring is the hard part

A decision is *about code*, and code moves. Line 142 today is line 187 tomorrow
and in a different file next week. **A record that loses its anchor is a diary
entry. A record that keeps it is a control.**

So anchoring is hybrid — line ranges *plus* a normalised content hash *plus* an
AST path — with a confidence score that decays as the code drifts. When
confidence drops far enough, the record is surfaced as **orphaned** rather than
silently pointed at the wrong line.

A tool that confidently points at the wrong line teaches you to distrust
everything it says. Being loudly uncertain is a feature.

## What this is not

- **Not a code reviewer.** It produces zero opinions about code quality. That
  category is well served and well funded; this is a different job.
- **Not an observability platform.** It doesn't do traces, evals or token spend.
- **Not a knowledge graph.** It answers one question about one line.

## Roadmap

| Phase | Scope |
|---|---|
| **0** | Claude Code hook → local SQLite → `whence <file>:<line>` and `whence log`. Naive line-range anchoring. |
| **1** | Hybrid anchoring, confidence decay, orphan surfacing, and **backfill** from git history and existing ADR/review docs. |
| **2** | MCP server feeding records into agent context. `whence check` CI gate. |
| **3** | End-to-end encrypted team sync + dashboard: AI-authorship attribution, per-commit cost, violation history. |

Phase 0 has to be useful on its own. Backfill in Phase 1 isn't polish — a
decision store with three records is worthless, so day one has to be non-empty.

## Security posture

This tool captures what an agent saw and did, which makes it sensitive by
default. Design commitments, not afterthoughts:

- **Hashes, paths and ranges — never file contents** by default. Content capture
  is opt-in per repo.
- **Redaction at capture, not at write.** Once a secret reaches a
  content-addressed store it may already be replicated.
- **Refuses to run if the store is git-tracked.** Ships a `.gitignore` entry on
  `init`.
- **Records are data, never directives.** Feeding records into agent context
  makes the store a prompt-injection target — anything able to write `.whence/`
  could otherwise inject authoritative-looking "project history" the agent obeys.
  Records are signed per author; unsigned or external records are marked
  untrusted at the point of display.
- **Attribution is aggregate-only by default.** No per-developer AI-authorship
  leaderboards. This is a developer tool, not a surveillance tool, and it will
  not become one.

## Honest open question

If an agent's stated reasoning is post-hoc rationalisation rather than actual
cause, `whence` preserves confident nonsense — durably. There's no mitigation for
this yet, and it's the most serious doubt about the premise. It gets tested on
real captured sessions before Phase 1 work begins.

**Falsification criterion:** count the times an agent proposed a change that
contradicted a recorded decision and `whence` caught it. If that's zero after
three months of genuine daily use, the idea is wrong and this repo gets archived.
The counter ships in Phase 0.

## License

TBD — will be permissive (MIT or Apache-2.0) for the CLI.
