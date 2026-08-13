# whence

A Go CLI that captures the reasoning coding agents produce and discards,
anchors it to the code it concerns, and replays it before that code changes
again. Git remembers what changed; whence remembers why.

**Scope boundary, the one that matters: whence never judges code quality.**
It supplies evidence about decisions; it does not lint, score, or opine on a
diff. Any feature that starts rating code is out of scope — say so rather
than building it.

## Read before changing anything

The code cites `§7`, `§14`, `§17` and the like. **Those sections are not in
this repo.** They live in the decision log below, and a comment citing one is
load-bearing — read the section or you will re-litigate something that was
settled with reasons.

- `HANDOFF.md` — §4 is the thirteen invariants, each with the reasoning that
  produced it; §5 is what is deliberately not built; §6 has three ideas killed
  by measurement. Read §4 before touching the hook, `check`, or anything that
  produces or consumes an anchor.
- `~/Desktop/Amogh/10-Projects/whence/DECISIONS.md` — the decision log itself.

## Layout

Flat `package main` at the repo root — `main.go`, `anchor.go`, `author.go`,
`capture.go`, `check.go`, `record.go`, each with a `_test.go` beside it. New
code goes in the file that owns the concern; don't introduce a package
directory without asking.

- `commands/`, `hooks/`, `.claude-plugin/` — the Claude Code plugin
- `action.yml` — the GitHub Action
- `web/` — the whence.fyi site (Vite + React + Tailwind v4 + shadcn/ui)
- `.whence/` — the record store; this repo dogfoods the tool

## Hard constraints

- **Zero dependencies. `go.mod` is a module line and `go 1.22`, nothing
  else.** Standard library only. Do not add a require block. If you think
  something needs a dependency, stop and make the case first.
- **The read path — lookup, `whence hook`, and `whence check` — is deterministic
  and offline.** It makes no network or model calls, and the whence binary is
  never an API client. Write-side extraction remains an explicit open decision;
  the hook runs on every edit and must fail open and stay fast.
- **Amogh's explicit instruction controls cleanup.** He may ask to clean an
  artifact or to leave it alone; whatever he says goes. During an explicit
  cleanup run, flagging ignored artifacts is fine, but a rejection is final:
  stop, do not retry or bypass it, and leave the harmless local artifacts where
  they are.
- **`web/` is pnpm only.** Never npm, never yarn — there's a
  `pnpm-lock.yaml` and only that.
- Site copy is written from the code, not from `README.md`. The README
  states intent; the code is the product. This has been wrong twice.

## Build and verify

go build ./...
go test ./...

Every source file has tests beside it — add to the matching `_test.go`.

**The binary trap:** a `whence` on PATH is whatever was last installed, never
the working tree. Never verify an unreleased change by running `whence` —
build to a temp path and run that binary. Installing is `go install .` with
the dot, never `@latest` (that pulls GitHub and lags the tree). This has
already caused one false verification pass.

## Files with rules attached

- `.whence/records.jsonl` — **committed on purpose**, it travels with the
  repo. Don't gitignore it. (`records.json`, a single JSON array, is the legacy
  shape: still read, never written.)
- `.whence/surfaced.jsonl` — local telemetry, gitignored. Don't commit it.
- `/whence` — the built binary, gitignored. Don't commit it.
- `HANDOFF.md` — gitignored working notes, not on GitHub, and not backed up
  by git. Never delete or rewrite it wholesale.
- The decision log, `~/Desktop/Amogh/10-Projects/whence/DECISIONS.md` — **§5
  and §10 name the employer.** §5 places Sama on the India ODR market map and
  scopes work away from it; §10 records development leaning on
  employer-provided Claude access. Redact both before that file ships
  anywhere public. Audited 2026-08-02: nothing else in it mentions an
  employer or a competitive constraint, and all 16 sections cited from this
  repo resolve.

## Workflow

Work lands on `main` through PRs, and I open them. Stage and describe your
changes; don't commit, don't push, don't open the PR.

When asked for "a commit command", give exactly one shell command that does
the whole flow: commit → push → open PR → merge → delete the branch. Merge
with `gh pr merge --auto --squash --delete-branch`: `--auto` arms the merge
so CI (whence.yml runs on PRs) gets to pass before anything lands, `--squash`
keeps main one entry per PR, `--delete-branch` cleans up after the merge
completes. Skip pushing the result back to main afterwards — merged via the
PR.
