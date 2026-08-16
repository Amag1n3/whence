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
