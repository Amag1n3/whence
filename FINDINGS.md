# Three LLMs graded the same 51 agent-written code rationales: 65%, 75%, 92%

*2026-08-15*

I built a small tool that records *why* a piece of code is the way it is, anchored
to the lines it concerns, and replays it before those lines change again. Git
remembers what changed; this remembers why. The motivating problem is agents
confidently undoing deliberate decisions — reverting a workaround, "simplifying" a
guard that existed for a reason — because the reasoning lived in a chat session
that no longer exists.

The obvious next step is to write those records automatically: after an agent edits
a file, take what it said just before the edit, and store that as the reason.

Before building it I wrote a condition into the project's decision log: **capture
stays off until there is a measured faithfulness rate.** Records are committed and
shared, so a wrong one is permanent and travels to everyone who clones the repo. I
wanted a number before letting a machine write into that.

So I went to measure it, and the measurement turned out to be more interesting than
the feature.

## The filter

The hook only fires when the agent's prose looks like a correction — a fixed list:
`real bug`, `turns out`, `false positive`, `mistake`, `wrong`, `flaw` and a few
more — **and** a backticked token from that prose appears in the edited text.

That list was built by reading three sessions of the tool's own repo. A sample of
one codebase, written by one person.

## Across 2,672 edits in four transcript corpora

| corpus | edits | passes marker | passes both gates | rate |
|---|---|---|---|---|
| the tool's own repo (where the list came from) | 552 | 88 | 29 | **5.25%** |
| private codebase A | 270 | 30 | 10 | 3.70% |
| private codebase B | 1620 | 125 | 41 | 2.53% |
| private codebase B, older sessions | 230 | 6 | 1 | 0.43% |

It under-fires away from home rather than over-firing, which is the safe direction.
A missed reason you can write by hand; a junk record in a shared repo you cannot.

## The part I did not expect

The filter has two gates: a **vocabulary** gate (does the prose contain a correction
word) and a **structural** gate (does a backticked token from the prose appear in
the edited text).

Survival through the structural gate, among records that passed the vocabulary gate:

- codebase B — 41/125 = **32.8%**
- the tool's own repo — 29/88 = **33.0%**
- codebase A — 10/30 = **33.3%**

Three codebases, three languages, three prose cultures, and a two-tenths-of-a-percent
spread. The vocabulary gate varies by more than 10×. The structural one does not vary
at all — it does not care whose English it is reading.

If you are building this kind of filter, that is where to build it.

One more: `wrong` alone accounted for **55–77%** of all vocabulary matches.
`strings.Contains(prose, "wrong")` matches "something went wrong", "the wrong file",
"worst case is a wrong number". It is ordinary English, not a marker.

## Then the actual question: when it fires, is the record any good?

81 records survived the gates across the four corpora; I graded 51 of them. I
dumped each one as it would actually be stored — the
decision line, the why, the prose it came from, and the exact text the edit wrote —
and had three frontier models judge them independently, each working from an
identical prompt that carried no verdicts from the others.

| grader | good | wrong | narration | rate |
|---|---|---|---|---|
| Claude Opus 5 | 33 | 17 | 1 | **65%** |
| Grok 4.6 | 38 | 11 | 2 | **75%** |
| Kimi K3 | 47 | 2 | 2 | **92%** |

Same evidence. The whole population rather than a sample, so there is no sampling
error available to explain it.

Per entry: **30 unanimous good, 3 unanimous bad, 18 disputed.** The defensible
statement is "somewhere between 59% and 94%", which is not a measurement.

## The spread is one unspecified rule, not noise

A record has two fields, split at the first sentence: `decision` gets sentence one,
`why` gets everything after it. Chat prose opens with an acknowledgement, so
`decision` frequently reads "Fair." or "All free." while the real reasoning sits
three paragraphs into `why`.

So: when the headline is junk but paragraph four happens to explain the edit, does
the record count?

One grader said yes — any true claim in the blob. Two said no — the dominant claim
has to match. Nobody specified it, all three answered it silently, and that single
choice is most of the 27 points.

## So I read ten of them myself

Ten records, picked to be maximally informative: eight the models disagreed on,
plus the two they were unanimous about as calibration anchors. Verdicts hidden until
after I'd marked mine.

| entry | **me** | Claude | Grok | Kimi |
|---|---|---|---|---|
| 019 | **good** | good | good | good |
| 010 | **good** | wrong | wrong | good |
| 013 | **narration** | wrong | wrong | good |
| 014 | **wrong** | wrong | narration | good |
| 021 | **wrong** | wrong | good | good |
| 023 | **wrong** | wrong | good | good |
| 027 | **good** | wrong | wrong | wrong |
| 030 | **can't tell** | wrong | wrong | good |
| 036 | *couldn't judge* | narration | good | good |
| 048 | *couldn't judge* | wrong | wrong | good |

Three of the ten I did not return a verdict on at all — 030 I marked "can't tell",
036 and 048 I left blank. So the comparison below runs over the **seven** I could
actually call.

**Agreement on direction, over those seven** — good versus not-good, collapsing
`narration` into not-good: Claude 5/7, Grok 3/7, Kimi 2/7. The human lands nearest
the strictest grader and furthest from the most permissive one, which points the
real rate at 65–75% rather than 92%. Seven entries is a small n and I am not
claiming more than a direction.

On exact verdict match instead it is 4/7, 1/7, 2/7, and Grok rather than Kimi is
furthest from me. I had to choose that rule to state these numbers at all, and the
choice moves them — which is the section above happening again, in my own
arithmetic.

Four things came out of it that the models could not have produced.

**I inverted one of the anchors.** All three graders called 027 a failure; I called
it good. The reason is disqualifying rather than reassuring: I was in that session.
The record reads *"All four done. Scrollbar — the tab strip used the platform
default"* and I remember which four. Someone finding those lines in two years has
none of that. My verdict is evidence the record works for a reader who was
present — precisely the reader this tool is not built for.

**I used a criterion none of the three thought to apply.** On 023 the record cites a
zip of design exports to justify a change. Attribution is fine. But the zip will not
exist later, so the record rots into an assertion with a dead reference. All three
models asked *does this reason explain this edit*. None asked *will this still mean
anything once what it points at is gone* — which, for a tool whose whole pitch is
records that outlive the session, is arguably the more important question. It is not
in the gates at all.

**I never stated the rule.** The prompt asked every grader to fix its position on
the thin-decision ambiguity up front and apply it consistently; all three did, and
their choices explain most of the spread. I left the field blank and answered by
feel — "feels real", "feels wrong". The distinction all three models resolved
explicitly is not one a human appears to make at all.

**And I could not judge three of the ten.** Not "found them borderline" — could not
evaluate them from what the record shows, on my own repo and my employer's code,
with full context. That is the uncomfortable one, and it is the next section.

## What did not happen is the interesting part

Across 51 records, **not one case of an agent stating a false reason for what it
did.** The failure mode people worry about — post-hoc rationalisation, a model
narrating a plausible story it did not act on — did not appear.

Every failure was a *true* reason attached to the *wrong edit*, because one
assistant message routinely produces several file changes and the reason belongs to
only one of them.

That is mechanical, not epistemic. Which is good news, because mechanical is
fixable — and both other graders independently proposed the same fix, better than
mine: store only the paragraph that carries the marker **and** names this edit,
rather than the whole message.

That fix is now in: a captured reason is the one paragraph where both gates land
on the same prose, and zero or two-or-more qualifying paragraphs write nothing at
all. Every number above was graded before it, so it re-measures none of them.

## Caveats, because they bound everything above

- This measures **attribution** — does the record explain this edit — not **truth**.
  A record can pass and still be factually wrong about the world. Nobody has
  measured that.
- The corpus is **enriched**: 81 of 2,672 edits, 3.0%, passed both gates, selected
  by a filter that fires on the agent's most diagnosis-heavy moments, and the 51 I
  graded are drawn from those 81. Which 51, and why not the other 30, I did not
  record. This measures records at their best.
- All of it is one engineer's sessions with one coding agent.
- Three LLMs grading LLM-written prose is a conflict of interest I cannot design
  away. Failures visible without domain knowledge get caught; failures requiring it
  are systematically undercounted. All three graders raised this independently.
- The human tiebreak is **seven judged entries**, which is enough for a direction
  and not enough for a number. It also went through one round of correction: the
  first version of this page counted an abstention as agreement and reported
  6/8·4/8·2/8. One of the graders caught it.

## The conclusion I actually drew

The number my own falsification condition demanded does not exist at useful
precision, and the disagreement is definitional rather than statistical.

So automatic capture stays off. And the thing keeping the store honest turns out not
to be the filter at all — it is that nothing reaches the shared, committed store
without a human approving it. Records the hook writes land in a local, gitignored
queue instead, and a human promotes them one at a time.

That was designed as a safety net. The measurement says it is the actual mechanism.

Which is where the last result bites. **I could not judge three of the ten records I
read** — on my own repository, in code I had written or supervised, with the full
prose and the exact diff in front of me. If the safeguard against a bad record is a
human approving it, and the human cannot evaluate roughly a third of the queue, then
for that third the safeguard is not a filter. It is a coin flip, or an entry that
sits unreviewed forever.

I do not have an answer to that yet. It is a better problem than the one I set out
to measure, and I would rather publish it than round it off.

---

whence is Go, zero dependencies, and its records are JSONL committed alongside your
code. What I would most like to know is whether the anchoring survives a codebase
that is not mine.
