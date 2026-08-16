# backfill against three large foreign codebases — 2026-08-16

Answers the open thread "`captureMarkers` has never met a codebase other than
this one" (HANDOFF §7.7). Dry runs only: `--yes` was never passed, nothing was
written to any store, and the clones were deleted afterwards.

## Corpus

| repo | shallow clone | languages | markers present | candidates | wall clock |
|---|---|---|---|---|---|
| kubernetes/kubernetes | 396 MB | Go, shell, Dockerfile | 1,796 | 92 | 3.1 s |
| microsoft/vscode | 338 MB | TypeScript, PowerShell | 372 | 67 | 2.4 s |
| python/cpython | 188 MB | C, Python | 591 | 151 | 0.9 s |

Marker count is `grep` over the same eight prefixes `harvest` accepts, so the
candidate column is the fraction that also cleared the reason gate: 5% in k8s,
18% in vscode, 26% in cpython. Nothing crashed, no file tripped the 1 MB or
binary guards visibly, and every comment syntax in the corpus parsed — `//`,
`#`, `/* */`, `<#`, and shell.

## The one that changes a rule

**`XXX:` is in `alwaysMarkers`, and CPython uses `XXX` as its TODO marker.**

`alwaysMarkers` is the only path with no gate behind it — the premise is that
nobody writes `XXX:` about something obvious, so the marker itself is the
admission. That premise is per-project. In CPython, 91 of 151 candidates came
from `XXX:`, and reading them, the majority are open questions and unfinished
work, not decisions:

- `Lib/plistlib.py:316` — "is this test needed?"
- `Lib/_pyio.py:802` — "Should seek() be used"
- `Lib/test/libregrtest/save_env.py:270` — "Maybe add an allow-list here?"
- `Lib/test/test_site.py:495` — "implement"
- `Lib/test/test_functools.py:2018` — "Why can be not equal?"
- `Modules/arraymodule.c:2441` — "Is it possible to write a unit test for this?"

This is the same class of thing `TODO:`/`FIXME:` were excluded for, arriving
under a marker the exclusion does not cover. k8s and vscode are unaffected —
vscode's 42 `HACK:` records are almost all genuine workarounds with reasons, and
are the best output of the three.

## Three sources of records the repo did not author

1. **Generated code.** 20 of k8s's 92 are the identical protoc line "this should
   be embedded by value" in `*_grpc.pb.go`. Regenerating the file rewrites the
   anchor, and the decision belongs to protoc-gen-go, not to Kubernetes.
2. **Vendored third-party.** 11 of cpython's 151 are `Modules/expat/xmlparse.c`
   — upstream Expat's reasoning, harvested into CPython's store. `skipDir`
   covers `vendor/`, which is the Go convention and nothing else's.
3. **Commented-out code.** `Lib/imaplib.py:643`, `Lib/test/test_urlparse.py`
   (×3) and `Lib/test/test_email/test_headerregistry.py:119` store a disabled
   code block as the decision text, because the marker sits above it.

## Duplicate texts

63 of the 310 candidates (20%) repeat text another candidate already carries —
37 in k8s, 18 in cpython, 8 in vscode. Some is real (`test/integration/volume/persistent_volumes_test.go`
says "This test cannot run in parallel" above nine separate tests, and each one
is true), some is a copy-paste artifact (vscode vendors `parseBlock.ts` and
`glob.ts` twice). `has()` de-dupes on file + text, so same-text-different-file
is by design. At this rate the store is a fifth restatement.

## Headline truncation — ~5%, one of them unusable

`firstSentence` splits at the first reason word, which produces a fragment when
that word lands early in the sentence:

- `pkg/apis/scheduling/types.go:35` — "NOTE: In order **to avoid** conflict of
  names …, all the names must start with SystemPriorityClassPrefix" stores the
  decision as **"In order"**. The actual rule is entirely in `why`.
- `extensions/typescript-language-features/web/src/util/hrtime.ts:9` — "This
  check is added probably **because** it's missed without strictFunctionTypes"
  stores "This check is added probably".
- `test/e2e/network/netpol/kubemanager.go:181` — split on "otherwise" leaves the
  headline cut mid-clause and a one-word `why`.

About 16 of 310 are broken this way. It matters more than the count suggests,
because the headline is what `check` prints at the moment an agent is about to
break something. Not a bug in every case: `Modules/expat/xmlparse.c:6147` reads
"We are avoiding MALLOC(..) here to" only because upstream wrote "here to so
that" — the split is correct and the typo is Expat's.

## Rough precision, hand-judged

| repo | genuine decisions | driver |
|---|---|---|
| vscode | ~87% | `HACK:` culture; the target case |
| k8s | ~74% | drops to this only because of the 20 protoc lines |
| cpython | ~60% | `XXX:`-as-TODO |

Judged by one reader in one pass, on repos I do not maintain — precision here is
softer than the FINDINGS.md numbers, and no second grader saw it.

## Incidental

`kubernetes/kubernetes` contains two `ponytail:` comments, in
`pkg/kubelet/oom/oom_counter.go`. Both are good records — a global with its
reason, and a known unbounded map with its upgrade path.

## What this does not answer

Anchoring. This exercised `harvest` and `firstSentence` only; whether an anchor
survives a rebase in someone else's repo is still untested, and is the question
the write-up says matters most.

---

# Round two — linux, rust, django

Same method, run against the binary round one produced, so the three filters
above are under test as much as anything else. Dry runs, nothing written, clones
deleted.

| repo | clone | markers | candidates | wall clock |
|---|---|---|---|---|
| torvalds/linux | 2.0 GB | 5,926 | 517 | 14.2 s |
| rust-lang/rust | 459 MB | 1,993 | 303 | 5.9 s |
| django/django | 74 MB | 34 | 3 | 0.8 s |

Round one's filters held: no generated file slipped through in 823 candidates,
and `XXX:` now has to earn its way in. Linux parses `.c`, `.h`, `.S`, `.dtsi`,
`.py` and `.sh`; Rust's `HACK(nox):` and `NOTE(FractalFir):` attributed forms are
matched correctly. Quality on Rust is the best seen so far — its `HACK:` notes
are near-uniformly real decisions with reasons.

## Four new defects

1. **`strings.Fields` counts punctuation as a word**, so round one's own
   six-word floor is porous. `rust/zerocopy/src/macros.rs` stores the decision
   as `This must be a macro (` — five real words plus a stray paren. Three
   instances in 823.
2. **The split lands inside a parenthetical.** That same note reads "This must
   be a macro (rather than a function with trait bounds) because …", and the cut
   happens at "rather than", inside the brackets. The identical note two hundred
   lines up splits correctly, only because it happens to contain a full stop and
   never reaches `splitAtReason`.
3. **Expected-output fixtures.** Rust's lint tests keep a generated `.fixed`
   twin beside each `.rs` fixture, so 22 candidates arrived as identical pairs —
   the same class as round one's generated files, in a file extension nobody
   would guess.
4. **A question is admitted as a decision.** Linux's
   `arch/powerpc/.../ppc476.c` gives "Is there any reason to assume
   differently?", which qualifies *because* the word "reason" is on the
   admission list. CPython's "is this test needed?" and "Why can be not equal?"
   are the same shape. A question is not a decision, and this is one cheap rule.

## Django is the result worth keeping

Seventy-four megabytes of mature Python produced **three records**. All three are
fine. The repository simply does not write the markers whence reads — 34 in the
whole tree, against 785 comment lines that carry a reason word and no marker.

The same ratio holds elsewhere: Linux, 5,926 markers against ~38,000 unmarked
reason-bearing comment lines; Rust, 1,993 against 8,651. Marker-gated harvesting
reaches low single-digit percentages of the reasoning a codebase has already
written down. That is the deliberate trade — a missed note is recoverable, a junk
record in a shared store is not — but the honest day-one promise is "the notes
you flagged loudly", not "your reasoning, made durable".

## Duplicates, again

72 repeated texts in linux, 36 in rust. Linux's "MUST NOT be called from
interrupt context" appears 16 times across 16 drivers, and each one is true. The
data is not wrong; the review experience is. Whoever promotes these from the
queue sees sixteen separate approvals for one sentence.
