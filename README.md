# whence

**Git remembers what changed. `whence` remembers why — and tells your AI agent before it changes it back.**

---

> [!NOTE]
> **Status: surfacing and anchoring work. Capture does not.**
>
> Real and tested: `whence <file>:<line>`, the `PreToolUse` hook, content-hash
> anchoring with drifted / weak / orphaned states, evidence pointers that anchor
> and rot independently of the record, `whence check` as a CI gate, human-vs-agent
> authorship with a confirmation step, a committed retraction log, `whence add`,
> `whence rm`, `whence confirm`, `whence reground`, `whence reanchor`, and `whence backfill` — which harvests decisions already
> written down as `HACK:` / `WORKAROUND:` / `ponytail:` comments, and as
> `NOTE:` / `TODO:` notes that give a reason — so a store is non-empty without
> anyone retyping anything.
>
> **Capture is not built.** Records are authored deliberately, because signal
> extraction is the hard problem and curated records are the spec for solving it.
>
> Anything below describing capture, AST-path anchoring or record signing is
> intended behaviour, not shipped behaviour.
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

  ● 2026-07-27 · code review, finding B5
    Never write shared session keys from this flow — namespace all three.
    "userToken", "userId" and "role" are all read by the admin
    dashboard — same origin, same app. Writing them here signs a
    staff user out mid-session and surfaces as a 200 with an
    error body, which is why it looks like a timeout.
    src/auth/session.go:147-153 (recorded at 142-148) · intact, moved  [4f2a]
```

That record was written about line 142 and answers for line 147, because the
code moved and the anchor followed it. When it stops following, it says so:

```console
$ whence src/auth/session.go

  ● 2026-07-27 · code review, finding B5 · 0% intact
    Never write shared session keys from this flow — namespace all three.
    src/auth/session.go:142-148 (recorded; anchor lost) · ORPHANED — anchor lost, needs a human  [4f2a]
```

The same record reaches your coding agent through a `PreToolUse` hook *before* it
edits, so it doesn't reintroduce the bug. And in CI:

```console
$ whence check --base origin/main

  · src/auth/session.go:145 — covered by record [4f2a] (2026-07-27), intact
    Never write shared session keys from this flow — namespace all three.
    why: "userToken", "userId" and "role" are all read by the admin dashboard.
    src/auth/session.go:142-148 · intact, exact range

  ! src/auth/session.go — this change erodes record [7d31] (2026-06-02)
    100% of the recorded block survived before this diff, 64% now
    Retry with backoff here; the provider rate-limits per-account, not per-key.
    src/auth/session.go:88-94 · altered
    if the rewrite kept the decision, re-point it with `why reanchor 7d31 ...`

  ✗ src/auth/session.go — record [9c1b] lost its anchor in this change
    it anchored at 88-94 before this diff; nothing matches now
    the decision is still on record; the code it described is gone.
    re-point it with `why reanchor 9c1b src/auth/session.go:<start>-<end>`,
    or retract it deliberately.

  2 recorded decision(s) damaged by this change, 1 more covered and intact.
  exit 1
```

That exit code is the product. Everything else is plumbing that leads to it.

**`check` reports coverage, never verdicts.** It says *these lines are governed by
these decisions, go and confirm* — it does not decide that your change is wrong.
Judging a diff is code review, that category is well served, and a tool that
starts doing it has lost the thing that makes it different.

**Only damage to a record fails the build.** The last two findings are the ones no
reviewer would catch unaided: neither change contradicted anything in writing,
they wore a decision away and then severed it from the code it described. The
first finding is different — the diff passed through covered lines and the record
came through whole — so it prints and the build passes.

That split is load-bearing, not a convenience. A touch cannot be *satisfied*: you
read the record, you agree with it, you change nothing, and re-running reports the
identical finding. And a touch that is not also an erosion is in practice a
whitespace change, because any edit with text in it moves a content hash and
surfaces as erosion instead. So failing on touches means failing on `gofmt`, which
means failing on every pull request, which means the gate gets switched off —
taking the two findings that justify the tool with it.

The obvious alternative, an "I checked this" flag, is refused. A gate you clear by
pasting a command on every pull request is one you stop reading before you clear,
and it then reports that a human checked something when no human did.

A record introduced by the same diff is not reported. Otherwise every pull
request that records a decision fails its own gate, which is how a team learns to
switch the gate off.

## Install

```console
$ go install github.com/Amag1n3/whence@latest
```

Go 1.22+. `@latest` resolves to the newest tag — v0.2.0 at time of writing.

Building from source works the same way and is the better option if you want
the `why` symlink or intend to edit the code:

```console
$ git clone https://github.com/Amag1n3/whence && cd whence
$ go build -o whence .
$ mv whence ~/.local/bin/                        # anywhere on $PATH
$ ln -s ~/.local/bin/whence ~/.local/bin/why     # optional short form
```

### One note about zsh and ksh

**zsh and ksh have a `whence` builtin**, and builtins take precedence over
`$PATH`. Those two shells — and only those two — need one line in your
`~/.zshrc` or `~/.kshrc`:

```sh
alias whence='command whence'
```

Aliases are resolved before builtins, so that is the whole fix. `command whence`
is also how you reach the binary in a one-off without the alias.

If you use zsh's `whence` builtin and would rather not shadow it, the `why`
symlink above runs the same binary under a name nothing competes for. **Every
example in this README works with either name.**

bash, fish, nushell and every non-interactive context — hooks, CI, scripts —
need nothing. They have no such builtin.

### Wire up the hook

This is the part that matters. Without it `whence` is a lookup tool you have to
remember to run; with it, records reach the agent before it edits.

**The short way — the Claude Code plugin:**

```console
/plugin marketplace add Amag1n3/whence
/plugin install whence@whence
```

Restart Claude Code, then run `/whence:setup` to check the wiring. The plugin
carries the hook configuration and finds the binary itself, looking at
`$WHENCE_BIN`, then `$GOBIN`/`$GOPATH/bin`, then `~/.local/bin`, `/usr/local/bin`
and `/opt/homebrew/bin`, then `$PATH`. It cannot install the binary for you —
`go install` above is still step one — and if it finds nothing it stays silent
rather than erroring on every edit.

**The manual way**, if you would rather not install a plugin, or you are wiring
up an agent that is not Claude Code. In `~/.claude/settings.json` (all projects)
or `.claude/settings.json` (this repo only):

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "timeout": 5,
            "command": "/absolute/path/to/whence hook pre"
          }
        ]
      }
    ]
  }
}
```

**Use an absolute path.** Hooks do not reliably inherit the `$PATH` from your
shell profile, and a hook that cannot find its binary fails silently — which is
also how a working hook with nothing to say behaves, so you would never notice.
`command -v whence` gives you the path to paste. The builtin is not in play here
— hooks are not your interactive shell, so no alias is needed.

`timeout: 5` is belt-and-braces. The hook measures 6.3ms on a warm store, and it
exits 0 having printed nothing on every error path, so a broken `whence` costs you
a missing record and nothing else. Restart Claude Code after editing settings.

Verify it fired: `.whence/surfaced.jsonl` gains a line each time records are put
in front of an agent.

### Get a non-empty store

A store with no records is a tool that does nothing, so start by harvesting what
is already written down:

```console
$ whence backfill              # decision comments under the current directory
$ whence log                   # what you now have
```

Backfill recognises two classes of comment, because the markers people actually
write split cleanly in two:

| Marker | Harvested |
|---|---|
| `HACK:` `WORKAROUND:` `XXX:` `GOTCHA:` `ponytail:` | **Always.** The word is itself the admission that a choice was made against a constraint. |
| `NOTE:` `TODO:` `TODO(owner):` `FIXME:` `WARNING:` `CAVEAT:` | **Only when the note gives a reason** — *because*, *so that*, *since*, *otherwise*, *to avoid*, *rather than*. |

That second rule is the whole difference between a store worth reading and one
full of `TODO: fix this`. A decision says why; a task only says what. The reason
may appear anywhere in the comment block, not just the first line.

The word list is deliberately narrow, so a reason phrased around it is missed. A
missed note you can add by hand; a garbage record in a committed, shared store
you cannot take back.

Then add them as you go, at the moment you make the call:

```console
$ whence add src/auth/session.go:142-148 \
    -d "Namespace all three session keys to CHECKOUT_*." \
    -w "The staff dashboard reads them on the same origin." \
    -e src/dashboard/session.go:40-44
```

Commit `.whence/records.jsonl`. That is the point — records travel with the repo.

### Gate CI on it

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0   # check reads the BASE revision of the store and of cited files
- uses: actions/setup-go@v5
  with: { go-version: '1.22' }
- run: go build -o whence . && ./whence check -base origin/${{ github.base_ref }}
```

Exit 1 means this pull request damaged a recorded decision — wore part of one
away, cut one loose from its code, or deleted the evidence one rested on. It is
not a verdict on the change: re-point each with `whence reanchor`, or retract it
deliberately, then re-run. Decisions merely *covering* the lines you changed print
and exit 0. See [`.github/workflows/whence.yml`](.github/workflows/whence.yml),
which is this repo gating on itself.

## How it works

1. **Capture.** Subscribes to your agent's hooks (`PostToolUse`, `Stop`, …) and
   records the decision trail as the session runs. Redaction happens here, before
   anything is written.
2. **Anchor.** Binds each record to a file, a line range, a content hash and a
   tree-sitter AST path — so it survives reformatting, drift and most refactors.
3. **Surface.** `whence <file>:<line>` from the terminal, the same records injected
   into a coding agent's context before it edits, and `whence check` as a CI gate.

Records live in `.whence/records.jsonl`, found by walking up from the file the way
git finds `.git` — so a session rooted in one repo still resolves records for a
file edited in a sibling repo.

> The project, the repo, the binary and the domain are all **whence**. zsh and
> ksh shadow it with a builtin of the same name — [one line of shell rc
> fixes that](#one-note-about-zsh-and-ksh).

### Anchoring is the hard part

A decision is *about code*, and code moves. Line 142 today is line 187 tomorrow
and in a different file next week. **A record that loses its anchor is a diary
entry. A record that keeps it is a control.**

So anchoring is hybrid: line ranges *plus* a hash per significant line of the
recorded span. Reformatting and reindentation change nothing. Code inserted above
moves the record and costs it nothing — a block that travelled 400 lines and
still hashes identically is not less certainly the same block. Only rewriting the
block itself costs anything.

**Position and content are reported separately, never as one number.** Where a
record points is shown as line numbers; how much of it survives is shown as a
percentage, and only where that percentage was actually measured. A record that
is intact says so, whether it moved or not. When enough of it is gone, the record
is surfaced as **orphaned** and claims no line number at all, rather than being
silently pointed at the wrong one.

### Evidence: what someone else could check

A record mixes two very different kinds of claim. *"These three keys are read by
the staff dashboard"* can be checked by going and looking. *"Which looks like a
network problem and is not one"* cannot — it's a judgement. Left in one paragraph,
the judgement borrows credibility from the fact sitting next to it.

So a record can carry **evidence**: pointers to things checkable without consulting
another record.

```console
$ whence add session.go:2-4 -d "Namespace all three session keys to CHECKOUT_*." \
    -w "The staff dashboard reads them on the same origin." \
    -e dashboard.go:2-3 -e "go test ./auth/..."
```

A pointer at code gets **its own anchor**, which is the reason evidence is a
pointer and never a copied snippet — a copy goes stale silently, which is the
disease. So when the cited code is deleted, whence says so on its own:

```console
$ whence check --base origin/main

  ✗ dashboard.go:2-3 — this change removes the evidence for record [390a35]
    the record still anchors to session.go:2-4 and reads as current
    Namespace all three session keys to CHECKOUT_*.
    what made it true is gone, so the decision now rests on nothing.
```

That change never opened `session.go`. The record is still perfectly anchored and
still reads as current. What went is the reason it was true — and no reviewer would
have caught it.

**A record may never be grounded in another record.** Cross-reference them for
reading all you like; citing one as grounds is refused. That single link is how one
wrong record makes the next look credible — Wikipedia calls its version
*citogenesis*, and the rule is the same one: link to another article freely, never
use it as your source.

Both halves of a record rot on their own, so both have a verb that fixes them
without a retraction. `whence reground` re-points the evidence when the grounds
move; `whence reanchor <id> <file>:<a>-<b>` re-points the record itself when the
block it described gets rewritten and the decision still holds. Neither is a
retraction — `whence rm` writes to a log that counts records which turned out to
be *wrong*, and doing routine bookkeeping there would destroy the one number
measuring whether the store can be trusted.

`reanchor` makes you name the lines. Where a degraded record currently points is
a best-match window of a fixed number of significant lines, so it sits a line or
two off the real block about as often as not — re-hashing it would store a guess
as a certainty, which is the one failure the anchoring design exists to prevent.

Records with no evidence are normal, not second-class. Most decisions are judgement
under constraints and have no artifact behind them. The field exists to stop the
unfalsifiable half from passing as the checkable half, not to demand proof of
everything.

### Anchoring: what is not built

The AST path from the design notes is not built. Per-line hashes already survive
insertion, deletion, reindentation and whole-block moves; tree-sitter buys the
remainder for the price of a CGo dependency, and waits until a real repo produces
orphans the hashes can't explain.

A tool that confidently points at the wrong line teaches you to distrust
everything it says. Being loudly uncertain is a feature.

## Records an agent wrote are not trusted until a human says so

A human writing a record is one deliberate act of attention. An agent writing one
is zero. So agent-authored records are marked, and stay marked until somebody
confirms them — including in the block injected into the next agent's context:

```
  anchor: intact, exact range
  source: capture
  UNCHECKED — an agent wrote this and no human has confirmed it
```

Wikipedia's rule is that content must be *verifiable*, with no requirement that
anyone verified it when it went in. That gap is what circular citation exploits.
This closes it for the only records that have it.

`whence log` ends with the number that matters:

```
10 records · 10 human, 0 agent · 0 unchecked · 0 orphaned
```

Self-feeding stores degrade, and the rare cases go first — which here are exactly
the records that justify the tool. A share trending toward agent-written and
unchecked is the warning, and it shows up long before the damage.

`whence rm` writes to a committed `.whence/retracted.jsonl` rather than deleting
quietly, because a store full of confident nonsense produces *more* CI hits, not
fewer. The count of times a record turned out to be wrong is the only number that
sees that failure.

## What this is not

- **Not a code reviewer.** It produces zero opinions about code quality. That
  category is well served and well funded; this is a different job.
- **Not an observability platform.** It doesn't do traces, evals or token spend.
- **Not a knowledge graph.** It answers one question about one line.

## Roadmap

| Phase | Scope |
|---|---|
| **0** | ✅ Claude Code `PreToolUse` hook → records → `whence <file>:<line>` and `whence log`. |
| **1** | ✅ Content-hash anchoring, confidence decay, orphan surfacing, evidence pointers, `whence add`, `whence rm`, and backfill from decision comments. Still open: backfill from git history and ADR/review docs, AST paths, capture, record signing. |
| **2** | ✅ `whence check` as a CI gate, comparing a diff against records. |
| **3** | End-to-end encrypted team sync + dashboard: AI-authorship attribution, per-commit cost, violation history. |

Phase 0 has to be useful on its own. Backfill in Phase 1 isn't polish — a
decision store with three records is worthless, so day one has to be non-empty.

## Security posture

This tool captures what an agent saw and did, which makes it sensitive by
default. Design commitments, not afterthoughts:

- **Hashes, paths and ranges — never file contents** by default. Content capture
  is opt-in per repo.
- **Redaction at capture, not at write.** The store is committed, so a bad
  capture is public the moment you push — and once a secret reaches a
  content-addressed store it may already be replicated.
- **The store is committed on purpose.** Records travel with the repo; a fresh
  clone has them, which is the whole point. Only the surfacing log
  (`.whence/surfaced.jsonl`) is gitignored — it holds timestamps and absolute
  local paths — by a `.gitignore` written inside `.whence/` when the store is
  created, so the decision stays in the directory it concerns instead of editing a
  repo-level file the tool does not own. Anyone using this alone who wants records
  kept local can gitignore the store themselves; nothing in the tool reads git
  state.
- **Records are data, never directives.** Feeding records into agent context
  makes the store a prompt-injection target, and records arrive by `git pull` —
  anyone able to land a commit could otherwise inject authoritative-looking
  "project history" the agent obeys. Records are signed per author; unsigned or
  external records are marked untrusted at the point of display.
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

[MIT](LICENSE).
