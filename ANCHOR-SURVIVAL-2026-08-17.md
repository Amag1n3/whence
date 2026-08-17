# Anchors against a year of someone else's history — 2026-08-17

Answers the open thread from `CORPUS-TEST-2026-08-16.md`: that run exercised `harvest` and `firstSentence` only. This one harvests at the oldest commit a 12-month shallow clone actually has, then re-resolves every record the way `logAll` does (`Load` at `main.go:562`, `resolveAnchor(fileLinesWithin(...), r)` at `main.go:579`) across the commits that touch those files.

Nothing in `weakFloor`, `rareAt`, `resolveAnchor`, or `author.go`'s admission rules was changed. The number describes this binary.

Clones lived in `/tmp/whence-survival-{vscode,rust}` and were deleted after the run. Nothing was written to `~/Desktop/whence/.whence/`.

## Corpus

| repo | clone | C0 | HEAD | harvested at C0 | files read / skipped | notes refused | commits touching records | walked | dropped |
|---|---|---|---|---|---|---|---|---|---|
| microsoft/vscode | 552 MB | `7aea2b43` 2025-08-17 | `7ee8d716` 2026-08-17 | 10 | 2688 / 58 | 0 | 41 | 41 | 0 |
| rust-lang/rust | 599 MB | `cbfa17a9` 2025-08-17 | `6d656b1e` 2026-08-17 | 238 | 39685 / 15565 | 15 | 4731 | 200 | 4531 |

Round one harvested 67 vscode candidates against HEAD; this run harvests at C0, a year earlier, so 10 is not a contradiction of that 67. Round two's 303 rust candidates were likewise HEAD. Survival is only defined for records that existed at C0.

## 1. The survival curve

| repo | when | exact | drifted | weak | orphaned | n |
|---|---|---|---|---|---|---|
| vscode | C0 | 10 | 0 | 0 | 0 | 10 |
| vscode | end | 7 | 3 | 0 | 0 | 10 |
| rust | C0 | 238 | 0 | 0 | 0 | 238 |
| rust | end | 34 | 139 | 25 | 40 | 238 |

vscode: every record survived. Three moved (`conversationFeature.ts` +79, `stream.ts` +36, `multiFileEditQualityTelemetry.ts` +3) and stayed byte-identical. No orphans in 41 commits.

rust, non-orphan at end: 198 / 238 (83%). Intact (exact + drifted): 173 / 238 (73%). Weak: 25. Orphaned: 40 (17%).

Where the rust orphans appeared: steadily, not in one commit. First-seen across 32 of the 200 walked commits; the largest burst was +3 (`7c9c7ed5`, a codegen_gcc intrinsic rewrite). The curve from the 25-commit checkpoints:

| walked | exact | drifted | weak | orphaned |
|---|---|---|---|---|
| 25 | 129 | 101 | 3 | 5 |
| 50 | 82 | 134 | 10 | 12 |
| 75 | 65 | 140 | 11 | 22 |
| 100 | 52 | 146 | 14 | 26 |
| 125 | 44 | 144 | 21 | 29 |
| 150 | 39 | 143 | 24 | 32 |
| 175 | 35 | 143 | 24 | 36 |
| 200 | 34 | 139 | 25 | 40 |

41 records were *first* seen orphaned; 40 remain orphaned at HEAD. One recovered after a later sampled commit (a revert, or a sampled gap that skipped the restore and then the next sample had it back). First-seen is therefore a slightly high count of "ever lost".

## 2. Position drift

Recorded `Start` against resolved `Start`, survivors only. Invariant 2 held: reloading the store after the walk showed every `File`/`Start`/`End` unchanged.

| repo | survivors | unmoved | moved | median \|Δ\| | p90 | max | mean |
|---|---|---|---|---|---|---|---|
| vscode | 10 | 7 | 3 | 0 | 79 | 79 | 11.8 |
| rust | 198 | 34 | 164 | 19 | 181 | 1434 | 68.4 |

vscode's three moves are insertions above a still-identical block — the case `StateDrifted` exists for.

rust is the first measurement of how far a real year moves a record. 164 of 198 survivors are not on their recorded line. Median travel is 19 lines; one record (`src/tools/rust-analyzer/crates/rust-analyzer/src/config.rs`, "check alias first, to work around the VS Code where it pre-fills the defaults") travelled 362. The 1434 is a pair of identical notes in `compiler/rustc_hir_typeck/src/fn_ctxt/checks.rs` ("Because we might be re-arranging arguments…") that sat at 1319 and 1413 and now sit at 2751 and 2847 — the file grew under them, the hashes did not budge. A reading of "now at N, recorded at M" is doing work these numbers say is common, not rare.

## 3. The orphan taxonomy

**File-level events are not hash failures.** `hashLine` is unsalted (`anchor.go:110`) so a block that moved to another file stays comparable; following it is listed as unbuilt roadmap, and `check` already treats a rename as a removal. They are split off here on purpose.

| kind | vscode | rust | hash-relevant? |
|---|---|---|---|
| file deleted | 0 | 3 | no |
| file renamed | 0 | 5 | no |
| block deleted | 0 | 21 | yes |
| block rewritten past `weakFloor` | 0 | 11 | yes |
| anything else | 0 | 0 | yes |

### File-level (8), not a candidate for tree-sitter

Deleted, path gone, recorded span not found elsewhere:

- `compiler/rustc_ast_lowering/src/delegation.rs:191-194` (`af0170`) — file gone.
- `compiler/rustc_next_trait_solver/src/solve/eval_ctxt/canonical.rs:183-188` (`3ff7ba`) — file gone.
- `compiler/rustc_metadata/src/errors.rs:664-667` (`3fbeb4`) — path gone. The NOTE "this suggests using rustup…" still sits in `compiler/rustc_metadata/src/diagnostics.rs:480`, but the recorded hash *sequence* (comment plus the following `diag.help`) is not intact there. Classified deleted by the exact-sequence rule; see bounds.

Renamed, exact recorded sequence still in the tree:

- `compiler/rustc_hir_analysis/src/errors/wrong_number_of_generic_args.rs:1056-1064` → `…/diagnostics/wrong_number_of_generic_args.rs` (`032d42`)
- `compiler/rustc_mir_transform/src/coroutine.rs:873-878` → `…/coroutine/layout.rs` (`2e6142`)
- `library/coretests/tests/floats/mod.rs:731-735` → `library/coretests/tests/num/floats.rs` (`aa5222`)
- `src/tools/rust-analyzer/crates/rust-analyzer/src/diagnostics/to_proto.rs:27-28` → `…/flycheck_to_proto.rs` (`8c7006`)
- `tests/rustdoc/extern/extern-html-root-url.rs:4-5` → `tests/rustdoc-html/extern/extern-html-root-url.rs` (`3f7613`)

### Hash-relevant (32)

Block deleted — file lives, best overlap 0 (21). Hand-checked against HEAD:

Gone from the file and from the tree: `e290fe` GIMPLE/`EMBED_LTO_BITCODE` (`write.rs:30-33`), `1642c7` `current_func` (`declare.rs:111-114`), `30f0d7` GIMPLE cast (`intrinsic/mod.rs:984-988`), `927f68` `& 0xFF for uchar` (`intrinsic/mod.rs:1021-1022`), `302c28` autoderef normalize (`autoderef.rs:83-86`; a *copy* of the NOTE exists in rust-analyzer's `hir-ty/src/autoderef.rs:201`, not a move of this file), `e8e83a` forbid generic `Self` (`hir_ty_lowering/mod.rs:1980-1982`), `082359` `string_deref_patterns` (`thir/pattern/mod.rs:664-668`), `cea28f` `Self` in an anonymous constant (`ident.rs:1291-1295`), `3cdc88` represent as tuple (`v0.rs:494-496`), `77a672` builtins-test Cargo.toml benchmarks NOTE, `de48a1` `Tag` enum (`rpc.rs:64-67`), `4a023a` `tempdir` (`builder/tests.rs:47-48`), `8652a6` python2 for x.py (`tidy/Dockerfile:45-47`; the file now ends at line 37 and runs python3), `4144b0` variant_field (`collect_intra_doc_links.rs:676-678`), `8c8ac1` dummy rlib path (`miri.rs:140-143`), `5e0db0` `rhs_ty` (`hir-ty/.../expr.rs:1353-1356`), `c45034` substitution (`expr.rs:1371-1373`).

File lives as a stub or sibling; the recorded lines moved to another path in the same crate (the unsalted-hash case, but the original path is not gone, so this is not "file renamed"):

- `46d518` `context.rs:557-559` → same HACK now at `compiler/rustc_middle/src/ty/trait_def.rs:187`
- `19471f` `src/bootstrap/src/bin/main.rs:82-84` → `src/bootstrap/src/cli_main.rs:113` (`main.rs` is now an 8-line stub)
- `dd8943` `main.rs:143-146` → `cli_main.rs:174`
- `04cd20` `intern/src/lib.rs:121-122` → `intern/src/intern.rs:164` (and a twin in `intern_slice.rs:127`); `lib.rs` is now a module root

Block rewritten past `weakFloor` (11), with the Integrity the scan actually computed:

| id | recorded | Integrity | what HEAD shows |
|---|---|---|---|
| `5c80ef` | `lto.rs:515-519` | 0.400 | NOTE still at `:257`; `mem::forget(tmp_path)` became a live `temp_dir = Some(tmp_path)` |
| `78d040` | `intrinsic/mod.rs:465-468` | 0.250 | NOTE gone; `count_leading_zeroes` refactored into `count_zeroes` |
| `543592` | `opaque_hidden_inferred_bound.rs:100-101` | 0.500 | HACK still at `:105`; the `if let ty::Alias(ty::Opaque, …)` was rewritten |
| `131d3e` | `coverage/hir_info.rs:27-28` | 0.500 | HACK wording replaced; `if tcx.is_synthetic_mir(def_id)` still at `:30` |
| `514022` | `libm/Cargo.toml:34-38` | 0.200 | HACK gone; `force-soft-floats = []` remains, marked DEPRECATED |
| `7da30e` | `compile.rs:1819-1827` | 0.444 | NOTE still at `:1981`, rewrapped across more lines; `filtered_files` still follows |
| `85317f` | `test.rs:3178-3180` | 0.333 | NOTE gone from the file |
| `ff3a10` | `search_index.rs:123-124` | 0.500 | HACK still at `:1301`; the key tuple moved into a `fn key` helper |
| `e3d8c2` | `runtest.rs:2083-2084` | 0.500 | HACK still at `:2204`; `self.revision` became `self.variant.revision()` |
| `77ea9b` | `tree_borrows/mod.rs:354-357` | 0.250 | NOTE gone; `ty_is_freeze` still computed at `:344` under a different `precise_interior_mut` branch |
| `771747` | `source_analyzer.rs:599-601` | 0.500 | HACK still at `:883` and `:910`; `TyBuilder::subst_for_def` became `GenericArgs::new_from_slice` |

## 4. The judgement

Would an AST path have held each hash-relevant orphan? From the actual diff, not in general.

**No, for every block-deleted record whose comment is gone (17 of 21).** An AST path to a deleted node is an orphan too. Tree-sitter does not resurrect a comment rustc deleted.

**No, for the four block-deleted records whose comment moved to another file** (`46d518`, `19471f`, `dd8943`, `04cd20`). Tree-sitter is per-file. The original file still parses; the recorded lines are not in it. Following those is the unsalted-hash / `git blame -C -M` roadmap item, not an AST item.

**No, for 5 of the 11 rewritten-past-floor records**, because the decision text itself was deleted or the file is TOML (`78d040`, `514022`, `85317f`, `77ea9b`, and `131d3e` whose HACK was rewritten into a different claim). An AST path to `count_leading_zeroes` or `extract_hir_info` would land on a function whose recorded reason is no longer there.

**Yes, for 6 of the 11 rewritten-past-floor records**, in the narrow sense that the HACK/NOTE is still in the same function and an AST path to that function would still resolve: `5c80ef`, `543592`, `7da30e`, `ff3a10`, `e3d8c2`, `771747`. What killed the hash is the *rest* of the recorded span — harvest anchors comment-plus-the-next-non-comment-line (`author.go:1372-1379`), so a two-line record of `HACK:` plus `if let` orphans when the `if let` is rewritten, even though the HACK is untouched. `7da30e` is the purest hash-only miss: the NOTE was rewrapped, and per-line hashes treat a different line break as different content.

Six cases in a year of rustc, none in vscode, where an AST path would have held a comment the hashes dropped. That does not justify a CGo dependency. **No, the hashes hold.** The ponytail at `anchor.go:26` can keep waiting.

What the 8 file-level events *would* justify scheduling is following a block across a rename — five clean ones, plus the three stub-file moves, plus `3fbeb4`'s file split. That is already on the roadmap and is not this task.

## 5. `weakFloor = 0.60` against real data

`anchor.go:59` called this "picked by eye, not measured — the knob to calibrate once real repos produce real orphans". This run is that.

StateWeak at end, rust only (vscode produced none):

| Integrity | n |
|---|---|
| [0.60, 0.65) | 0 |
| [0.65, 0.70) | 10 |
| [0.70, 0.75) | 0 |
| [0.75, 0.80) | 5 |
| [0.80, 0.85) | 9 |
| [0.85, 0.90) | 1 |
| [0.90, 1.00) | 0 |

n=25, min=0.667, median=0.750, max=0.857. Two clusters: ten records piled on 0.667, nine on ~0.80. Nothing sits against the floor.

The rewritten-past-floor orphans sit at 0.200, 0.250, 0.250, 0.333, 0.400, 0.444, and five at 0.500. Then a gap from 0.50 to 0.667, and Weak begins.

0.60 sat in that gap, not through the middle of a cluster. Raising it to 0.70 would swallow the ten 0.667 Weaks (a third of the Weak set) and would not pick up any additional orphan. Lowering it to 0.50 would promote the five 0.500 orphans to Weak — those are the "HACK still here, following line rewritten" cases above. The floor as picked by eye happened to land in the one empty band the data produced. Left alone.

## Defects noticed, not fixed

Round two found four this way. This run found more. None of these were changed.

1. **`secretEntropy` treats a GitHub issue URL as a credential.** 12 of rust's 15 refused notes are real `NOTE:`/`HACK:` comments that cite `https://github.com/rust-lang/rust/issues/NNNNN`. `entropyToken` requires length ≥ 32, mixed classes, not hex — a URL is all three. Verified by running the same classifier on `https://github.com/rust-lang/rust/issues/75100` (true, 46 chars, 3 classes). The refused sites include `compiler/rustc/src/main.rs:31` (jemalloc / `global_allocator`), `compiler/rustc_hir_analysis/src/check/check.rs:220` (don't infer `impl Trait` for rustdoc), `library/stdarch/.../macros.rs:112` (don't use `&self.0`). These are the notes the corpus called near-uniformly real. The other 3 refusals are unidentifiable 3-line spans in `codegen_gcc/src/intrinsic/mod.rs` — `anchorSpan` working as designed.

2. **`firstSentence` stored a line of code as `why`.** `1642c7`'s decision is the NOTE; its `why` is `self.current_func.borrow_mut() = Some(func);` — the declaration harvest included in the span, which has no reason-word split and leaked into the second half. Same shape as round one's commented-out-code-as-decision, other half of the record.

3. **A two-line harvest is comment plus one statement**, so rewriting the statement orphans a still-present HACK (`543592`, `e3d8c2`, `771747`, `ff3a10`). That is harvest's documented choice (`author.go:1372-1379`), and it is why six "AST would hold" cases exist. The hashes did what they were told with the span they were given.

4. **One recovered orphan** (41 first-seen, 40 at end). Sampling can skip a revert; a record can also come back for real. The end-state table is the one that answers the question.

## Bounds imposed

- **Commits sampled away:** rust 4531 of 4731 (96%) dropped, 200 kept, first and last of `C0..HEAD -- <record files>` retained. vscode walked all 41. A first-seen SHA on rust is the first *sampled* commit after the loss, not necessarily the commit that caused it.
- **Files skipped by `readSource`:** vscode 58, rust 15565 (binaries, >1 MB, expected-output extensions, generated-file marker). The walk itself also skipped `skipDir` names (`.git`, `vendor`, `third_party`, …) without counting those files as skipped.
- **Notes refused at write:** vscode 0, rust 15 (12 entropy-on-URL, 3 unidentifiable). They never entered the store, so they are not in the survival numbers.
- **Harvest is at C0, not HEAD.** vscode 10 vs the corpus's 67; rust 238 vs 303. Survival cannot be blamed on a smaller, older input without saying so.
- **Shallow since 12 months.** C0 is the oldest commit the clone has, not the repo's root. vscode reported 13 shallow roots; rust's C0 date is 2025-08-17, matching `--shallow-since`. Full history was not fetched. Clone sizes were 552 MB and 599 MB, larger than the 338 / 459 the task quoted from the previous rounds.
- **Rename detection requires the exact recorded hash sequence** in some other file. A file that moved *and* whose following line changed (`3fbeb4` `errors.rs` → `diagnostics.rs`) reads as deleted.
- **Both clones succeeded.** Nothing failed to clone.
- **200-commit cap printed, not silent.**

## Disagreements, built anyway

- Following a rename would have recovered more records than tree-sitter (5 clean renames + 4 intra-crate moves + 1 file-split, against 6 "AST of the same function would still see the comment"). The spec said to measure, not to build that. Measured.
- A window scan keyed on whole-comment text, rather than per-line hashes, would have held `7da30e` (rewrap) and most of the six "HACK still here" cases. That is a different algorithm. Not built.
- `weakFloor` as a function of block size would treat a 2-line HACK+`if` differently from a 9-line NOTE. The data that would motivate it is exactly those six two-line records. Knob left at 0.60.
- `secretEntropy` should ignore `https://` tokens. Not this task.

## What this does not answer

- Survival of records harvested at HEAD and walked forward from *today*. This is C0-to-HEAD on a year of already-written notes.
- A third corpus round. Admission quality is still the two rounds already done; the 12 URL refusals are a write-path defect, not a harvest-precision number.
- Whether following renames is worth scheduling. The count is 5 clean + 4 stub-file + 1 split, and that is the input, not the work.
- Anything in `capture.go`.
