import type { ReactNode } from 'react'

import { DocPage, DocSection, P, type DocSectionMeta } from '@/components/DocPage'
import { Reveal } from '@/components/Reveal'
import { cn } from '@/lib/utils'

/* Two dated writeups on one page, not an index and not a third HTML
   entry. FINDINGS.md (2026-08-15) owns the hero; the survival run
   (ANCHOR-SURVIVAL-2026-08-17.md) follows it. A second note is when a
   generalisation earns its keep. */

const SECTIONS: DocSectionMeta[] = [
  { id: 'filter', label: 'The filter' },
  { id: 'corpora', label: 'Across four corpora' },
  { id: 'unexpected', label: 'The part I did not expect' },
  { id: 'quality', label: 'When it fires, is it any good?' },
  { id: 'rule', label: 'One unspecified rule' },
  { id: 'human', label: 'So I read ten myself' },
  { id: 'absence', label: 'What did not happen' },
  { id: 'caveats', label: 'Caveats' },
  { id: 'conclusion', label: 'The conclusion' },
  { id: 'survival', label: 'A year of rustc' },
  { id: 'curve', label: 'The survival curve' },
  { id: 'orphans', label: 'The forty orphans' },
  { id: 'treesitter', label: 'Tree-sitter' },
  { id: 'limits', label: 'What this does not answer' },
]

const M = ({ children }: { children: ReactNode }) => (
  <code className="font-mono text-[12.5px] text-silt">{children}</code>
)

const B = ({ children }: { children: ReactNode }) => (
  <b className="text-silt">{children}</b>
)

/** Columns stay columns. A stacked-card table would break the comparison
 *  the numbers exist to make. On a phone the row scrolls; the first column
 *  stays put so the label still names the figures beside it. */
function Table({ cols, rows }: { cols: ReactNode[]; rows: ReactNode[][] }) {
  return (
    <div className="term-scroll max-w-full overflow-x-auto">
      <table className="w-max min-w-full border-collapse font-mono text-[12.5px] leading-[1.65]">
        <thead>
          <tr className="border-b border-white/[0.07] text-dim">
            {cols.map((c, i) => (
              <th
                key={i}
                scope="col"
                className={cn(
                  'py-2 pr-5 font-normal whitespace-nowrap',
                  i === 0
                    ? 'sticky left-0 bg-basin pr-6 text-left'
                    : 'text-right',
                )}
              >
                {c}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr key={i} className="border-b border-white/[0.07] text-muted-foreground">
              {row.map((cell, j) => (
                <td
                  key={j}
                  className={cn(
                    'py-2.5 pr-5 whitespace-nowrap',
                    j === 0
                      ? 'sticky left-0 bg-basin pr-6 text-left text-silt'
                      : 'text-right',
                  )}
                >
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export default function NotesPage() {
  return (
    <DocPage
      current="notes"
      eyebrow="notes · 2026-08-15"
      title="Three LLMs graded the same 51 agent-written code rationales: 65%, 75%, 92%"
      lede="I built a small tool that records why a piece of code is the way it is, anchored to the lines it concerns, and replays it before those lines change again. Git remembers what changed; this remembers why. The motivating problem is agents confidently undoing deliberate decisions — reverting a workaround, “simplifying” a guard that existed for a reason — because the reasoning lived in a chat session that no longer exists."
      sections={SECTIONS}
    >
      <div className="space-y-5">
        <P>
          The obvious next step is to write those records automatically: after an
          agent edits a file, take what it said just before the edit, and store
          that as the reason.
        </P>
        <P>
          Before building it I wrote a condition into the project's decision log:{' '}
          <B>capture stays off until there is a measured faithfulness rate.</B>{' '}
          Records are committed and shared, so a wrong one is permanent and
          travels to everyone who clones the repo. I wanted a number before
          letting a machine write into that.
        </P>
        <P>
          So I went to measure it, and the measurement turned out to be more
          interesting than the feature.
        </P>
      </div>

      <DocSection id="filter" n={1} title="The filter">
        <P>
          The hook only fires when the agent's prose looks like a correction — a
          fixed list: <M>real bug</M>, <M>turns out</M>, <M>false positive</M>,{' '}
          <M>mistake</M>, <M>wrong</M>, <M>flaw</M> and a few more —{' '}
          <B>and</B> a backticked token from that prose appears in the edited
          text.
        </P>
        <P>
          That list was built by reading three sessions of the tool's own repo.
          A sample of one codebase, written by one person.
        </P>
      </DocSection>

      <DocSection id="corpora" n={2} title="Across 2,672 edits in four transcript corpora">
        <Table
          cols={['corpus', 'edits', 'passes marker', 'passes both gates', 'rate']}
          rows={[
            ['the tool\'s own repo (where the list came from)', '552', '88', '29', <B>5.25%</B>],
            ['private codebase A', '270', '30', '10', '3.70%'],
            ['private codebase B', '1620', '125', '41', '2.53%'],
            ['private codebase B, older sessions', '230', '6', '1', '0.43%'],
          ]}
        />
        <P>
          It under-fires away from home rather than over-firing, which is the
          safe direction. A missed reason you can write by hand; a junk record
          in a shared repo you cannot.
        </P>
      </DocSection>

      <DocSection id="unexpected" n={3} title="The part I did not expect">
        <P>
          The filter has two gates: a <B>vocabulary</B> gate (does the prose
          contain a correction word) and a <B>structural</B> gate (does a
          backticked token from the prose appear in the edited text).
        </P>
        <P>
          Survival through the structural gate, among records that passed the
          vocabulary gate:
        </P>
        <Table
          cols={['corpus', 'survived', 'of', 'rate']}
          rows={[
            ['codebase B', '41', '125', <B>32.8%</B>],
            ["the tool's own repo", '29', '88', <B>33.0%</B>],
            ['codebase A', '10', '30', <B>33.3%</B>],
          ]}
        />
        <P>
          Three codebases, three languages, three prose cultures, and a
          two-tenths-of-a-percent spread. The vocabulary gate varies by more
          than 10×. The structural one does not vary at all — it does not care
          whose English it is reading.
        </P>
        <P>If you are building this kind of filter, that is where to build it.</P>
        <P>
          One more: <M>wrong</M> alone accounted for <B>55–77%</B> of all
          vocabulary matches.{' '}
          <M>strings.Contains(prose, "wrong")</M> matches "something went
          wrong", "the wrong file", "worst case is a wrong number". It is
          ordinary English, not a marker.
        </P>
      </DocSection>

      <DocSection id="quality" n={4} title="Then the actual question: when it fires, is the record any good?">
        <P>
          81 records survived the gates across the four corpora; I graded 51 of
          them. I dumped each one as it would actually be stored — the
          decision line, the why, the prose it came from, and
          the exact text the edit wrote — and had three frontier models judge
          them independently, each working from an identical prompt that
          carried no verdicts from the others.
        </P>
        <Table
          cols={['grader', 'good', 'wrong', 'narration', 'rate']}
          rows={[
            ['Claude Opus 5', '33', '17', '1', <B>65%</B>],
            ['Grok 4.6', '38', '11', '2', <B>75%</B>],
            ['Kimi K3', '47', '2', '2', <B>92%</B>],
          ]}
        />
        <P>
          Same evidence. The whole population rather than a sample, so there is
          no sampling error available to explain it.
        </P>
        <Table
          cols={['verdict', 'count']}
          rows={[
            ['unanimous good', <B>30</B>],
            ['unanimous bad', <B>3</B>],
            ['disputed', <B>18</B>],
          ]}
        />
        <P>
          The defensible statement is "somewhere between 59% and 94%", which is
          not a measurement.
        </P>
      </DocSection>

      <DocSection id="rule" n={5} title="The spread is one unspecified rule, not noise">
        <P>
          A record has two fields, split at the first sentence: <M>decision</M>{' '}
          gets sentence one, <M>why</M> gets everything after it. Chat prose
          opens with an acknowledgement, so <M>decision</M> frequently reads
          "Fair." or "All free." while the real reasoning sits three paragraphs
          into <M>why</M>.
        </P>
        <P>
          So: when the headline is junk but paragraph four happens to explain
          the edit, does the record count?
        </P>
        <P>
          One grader said yes — any true claim in the blob. Two said no — the
          dominant claim has to match. Nobody specified it, all three answered
          it silently, and that single choice is most of the 27 points.
        </P>
      </DocSection>

      <DocSection id="human" n={6} title="So I read ten of them myself">
        <P>
          Ten records, picked to be maximally informative: eight the models
          disagreed on, plus the two they were unanimous about as calibration
          anchors. Verdicts hidden until after I'd marked mine.
        </P>
        <Table
          cols={['entry', <B>me</B>, 'Claude', 'Grok', 'Kimi']}
          rows={[
            ['019', <B>good</B>, 'good', 'good', 'good'],
            ['010', <B>good</B>, 'wrong', 'wrong', 'good'],
            ['013', <B>narration</B>, 'wrong', 'wrong', 'good'],
            ['014', <B>wrong</B>, 'wrong', 'narration', 'good'],
            ['021', <B>wrong</B>, 'wrong', 'good', 'good'],
            ['023', <B>wrong</B>, 'wrong', 'good', 'good'],
            ['027', <B>good</B>, 'wrong', 'wrong', 'wrong'],
            ['030', <B>can't tell</B>, 'wrong', 'wrong', 'good'],
            ['036', <em>couldn't judge</em>, 'narration', 'good', 'good'],
            ['048', <em>couldn't judge</em>, 'wrong', 'wrong', 'good'],
          ]}
        />
        <P>
          Three of the ten I did not return a verdict on at all — 030 I marked
          "can't tell", 036 and 048 I left blank. So the comparison below runs
          over the <B>seven</B> I could actually call.
        </P>
        <P>
          <B>Agreement on direction, over those seven</B> — good versus
          not-good, collapsing <M>narration</M> into not-good: Claude 5/7, Grok
          3/7, Kimi 2/7. The human lands nearest the strictest grader and
          furthest from the most permissive one, which points the real rate at
          65–75% rather than 92%. Seven entries is a small n and I am not
          claiming more than a direction.
        </P>
        <P>
          On exact verdict match instead it is 4/7, 1/7, 2/7, and Grok rather
          than Kimi is furthest from me. I had to choose that rule to state
          these numbers at all, and the choice moves them — which is the
          section above happening again, in my own arithmetic.
        </P>
        <P>Four things came out of it that the models could not have produced.</P>
        <P>
          <B>I inverted one of the anchors.</B> All three graders called 027 a
          failure; I called it good. The reason is disqualifying rather than
          reassuring: I was in that session. The record reads{' '}
          <em>"All four done. Scrollbar — the tab strip used the platform
          default"</em>{' '}
          and I remember which four. Someone finding those lines in two years
          has none of that. My verdict is evidence the record works for a
          reader who was present — precisely the reader this tool is not built
          for.
        </P>
        <P>
          <B>I used a criterion none of the three thought to apply.</B> On 023
          the record cites a zip of design exports to justify a change.
          Attribution is fine. But the zip will not exist later, so the record
          rots into an assertion with a dead reference. All three models asked{' '}
          <em>does this reason explain this edit</em>. None asked{' '}
          <em>will this still mean anything once what it points at is gone</em>{' '}
          — which, for a tool whose whole pitch is records that outlive the
          session, is arguably the more important question. It is not in the
          gates at all.
        </P>
        <P>
          <B>I never stated the rule.</B> The prompt asked every grader to fix
          its position on the thin-decision ambiguity up front and apply it
          consistently; all three did, and their choices explain most of the
          spread. I left the field blank and answered by feel — "feels real",
          "feels wrong". The distinction all three models resolved explicitly
          is not one a human appears to make at all.
        </P>
        <P>
          <B>And I could not judge three of the ten.</B> Not "found them
          borderline" — could not evaluate them from what the record shows, on
          my own repo and my employer's code, with full context. That is the
          uncomfortable one, and it is the next section.
        </P>
      </DocSection>

      <DocSection id="absence" n={7} title="What did not happen is the interesting part">
        <P>
          Across 51 records,{' '}
          <B>not one case of an agent stating a false reason for what it did.</B>{' '}
          The failure mode people worry about — post-hoc rationalisation, a
          model narrating a plausible story it did not act on — did not appear.
        </P>
        <P>
          Every failure was a <em>true</em> reason attached to the{' '}
          <em>wrong edit</em>, because one assistant message routinely produces
          several file changes and the reason belongs to only one of them.
        </P>
        <P>
          That is mechanical, not epistemic. Which is good news, because
          mechanical is fixable — and both other graders independently proposed
          the same fix, better than mine: store only the paragraph that carries
          the marker <B>and</B> names this edit, rather than the whole message.
        </P>
        <P>
          That fix is now in: a captured reason is the one paragraph where both
          gates land on the same prose, and zero or two-or-more qualifying
          paragraphs write nothing at all. Every number above was graded before
          it, so it re-measures none of them.
        </P>
      </DocSection>

      <DocSection id="caveats" n={8} title="Caveats, because they bound everything above">
        <ul className="max-w-[56ch] list-disc space-y-3 pl-5 text-[14.5px] leading-[1.68] text-muted-foreground">
          <li>
            This measures <B>attribution</B> — does the record explain this
            edit — not <B>truth</B>. A record can pass and still be factually
            wrong about the world. Nobody has measured that.
          </li>
          <li>
            The corpus is <B>enriched</B>: 81 of 2,672 edits, 3.0%, passed both
            gates, selected by a filter that fires on the agent's most
            diagnosis-heavy moments, and the 51 I graded are drawn from those
            81. Which 51, and why not the other 30, I did not record. This
            measures records at their best.
          </li>
          <li>All of it is one engineer's sessions with one coding agent.</li>
          <li>
            Three LLMs grading LLM-written prose is a conflict of interest I
            cannot design away. Failures visible without domain knowledge get
            caught; failures requiring it are systematically undercounted. All
            three graders raised this independently.
          </li>
          <li>
            The human tiebreak is <B>seven judged entries</B>, which is enough
            for a direction and not enough for a number. It also went through
            one round of correction: the first version of this page counted an
            abstention as agreement and reported 6/8·4/8·2/8. One of the
            graders caught it.
          </li>
        </ul>
      </DocSection>

      <DocSection id="conclusion" n={9} title="The conclusion I actually drew">
        <P>
          The number my own falsification condition demanded does not exist at
          useful precision, and the disagreement is definitional rather than
          statistical.
        </P>
        <P>
          So automatic capture stays off. And the thing keeping the store
          honest turns out not to be the filter at all — it is that nothing
          reaches the shared, committed store without a human approving it.
          Records the hook writes land in a local, gitignored queue instead,
          and a human promotes them one at a time.
        </P>
        <P>
          That was designed as a safety net. The measurement says it is the
          actual mechanism.
        </P>
        <P>
          Which is where the last result bites.{' '}
          <B>I could not judge three of the ten records I read</B> — on my own
          repository, in code I had written or supervised, with the full prose
          and the exact diff in front of me. If the safeguard against a bad
          record is a human approving it, and the human cannot evaluate roughly
          a third of the queue, then for that third the safeguard is not a
          filter. It is a coin flip, or an entry that sits unreviewed forever.
        </P>
        <P>
          I do not have an answer to that yet. It is a better problem than the
          one I set out to measure, and I would rather publish it than round it
          off.
        </P>
        <P>
          whence is Go, zero dependencies, and its records are JSONL committed
          alongside your code. What I would most like to know is whether the
          anchoring survives a codebase that is not mine.
        </P>
      </DocSection>

      <div className="border-t border-white/[0.07]" />

      <section id="survival" className="scroll-mt-24">
        <Reveal>
          <p className="font-mono text-[11px] tracking-[0.2em] text-dim uppercase">
            notes · 2026-08-17
          </p>
          <h2 className="mt-3 max-w-[19ch] text-[clamp(1.65rem,2.7vw,2.3rem)] leading-[1.15]">
            198 of 238 rustc records still resolved after a year. 173 of them fully intact
          </h2>
          <p className="mt-6 max-w-[54ch] text-[15.5px] leading-[1.7] text-muted-foreground">
            The last note ended by asking whether the anchoring survives a
            codebase that is not mine. I harvested the reason-bearing{' '}
            <M>HACK</M> and <M>NOTE</M> comments rustc and VS Code already
            had on 2025-08-17, then re-resolved each record the way the
            lookup path does, across the commits that touched those files,
            up to 2026-08-17. Nothing in the resolver was changed. The
            number describes this binary.
          </p>
        </Reveal>
      </section>

      <DocSection id="curve" n={1} title="The survival curve">
        <Table
          cols={['repo', 'when', 'exact', 'drifted', 'altered', 'orphaned']}
          rows={[
            ['microsoft/vscode', 'C0', '10', '0', '0', '0'],
            ['microsoft/vscode', 'HEAD', '7', '3', '0', '0'],
            ['rust-lang/rust', 'C0', '238', '0', '0', '0'],
            ['rust-lang/rust', 'HEAD', '34', '139', '25', '40'],
          ]}
        />
        <P>
          The run recorded the middle two states as drifted and weak. Weak is
          the altered state on this site: the block is still in the file, and
          enough of it has been rewritten that the match is no longer exact.
        </P>
        <P>
          vscode: every record survived. Three moved and stayed
          byte-identical. There is nothing to learn from a column of ten that
          a column of one would not have shown; it is in the table so the
          method is visible, not because it bears any weight.
        </P>
        <P>
          rust, still resolved at HEAD — not orphaned:{' '}
          <B>198 / 238</B>. Intact, meaning exact or drifted:{' '}
          <B>173 / 238</B>. Altered: 25. Orphaned: 40.
        </P>
        <P>
          Harvest was at C0, a year earlier than the trial that produced 67
          vscode candidates and 303 rust ones against HEAD. Survival is only
          defined for records that already existed at C0, so 10 and 238 are
          not a contradiction of those earlier counts.
        </P>
      </DocSection>

      <DocSection id="orphans" n={2} title="The forty orphans">
        <P>
          File-level events are not hash failures. A block that moved to
          another file stays comparable — the line hashes are unsalted — but
          following it is unbuilt, and <M>check</M> already treats a rename as
          a removal.
        </P>
        <Table
          cols={['what happened', 'rust', 'whose miss']}
          rows={[
            ['the decision itself is gone', '25', 'nobody\'s — correct'],
            ['recorded lines still in the tree, under another path', '9', 'unbuilt — follow a rename'],
            ['comment still in the same function; the hashes dropped it', '6', 'anchoring'],
          ]}
        />
        <P>
          Of the 40 rust orphans, <B>25 are the decision itself being gone</B>{' '}
          — the comment deleted from a living file, the file deleted, or the
          claim rewritten into a different one. That is the correct
          behaviour. <B>9 need following a record across a rename</B>: five
          clean renames, and four moves into a sibling file while the
          original path stayed behind as a stub. That is listed as unbuilt
          roadmap. <B>6 are anchoring&rsquo;s own miss.</B>
        </P>
        <P>
          Anchoring&rsquo;s real miss rate is <B>6 of 238</B>, and one of
          those six is a pure line-rewrap: the <M>NOTE</M> is still there, the
          line that followed it is still there, and per-line hashes treat a
          different line break as different content.
        </P>
      </DocSection>

      <DocSection id="treesitter" n={3} title="Tree-sitter, and the deferral it does not reopen">
        <P>
          Would an AST path have held each hash-relevant orphan? From the
          actual diff, not in general.
        </P>
        <P>
          No, for every deleted comment. An AST path to a deleted node is an
          orphan too. Tree-sitter does not resurrect a comment rustc deleted.
        </P>
        <P>
          No, for a comment that moved to another file. Tree-sitter is
          per-file. The original file still parses; the recorded lines are
          not in it. Following those is the rename item, not an AST item.
        </P>
        <P>
          Yes, for the six — in the narrow sense that the <M>HACK</M> or{' '}
          <M>NOTE</M> is still in the same function, and an AST path to that
          function would still resolve. What killed the hash is the rest of
          the recorded span. Harvest anchors the comment plus the next
          non-comment line, so a two-line record of <M>HACK:</M> plus{' '}
          <M>if let</M> orphans when the <M>if let</M> is rewritten, even
          though the <M>HACK</M> is untouched. The rewrap is the purest
          hash-only miss.
        </P>
        <P>
          Six cases in a year of rustc, none in vscode, where an AST path
          would have held a comment the hashes dropped. That does not
          justify a CGo dependency. <B>The hashes hold.</B> The standing
          deferral — wait until a real repository produces orphans the
          hashes cannot explain — is closed, not reopened.
        </P>
      </DocSection>

      <DocSection id="limits" n={4} title="What this does not answer">
        <ul className="max-w-[56ch] list-disc space-y-3 pl-5 text-[14.5px] leading-[1.68] text-muted-foreground">
          <li>
            Effectively <B>one repo</B>. vscode&rsquo;s n is 10 and
            decorative; every record there survived, and a sample that small
            cannot show otherwise.
          </li>
          <li>
            <B>One grader.</B> I classified the orphans against HEAD. Nobody
            independently re-graded them.
          </li>
          <li>
            The clones were deleted after the run. The per-case
            classification cannot be re-checked without a re-run.
          </li>
          <li>
            These are notes rustc wrote before whence existed, harvested
            from comments, not records a human authored deliberately.
          </li>
        </ul>
        <P>
          It also does not answer survival of records harvested today and
          walked forward from today. This is C0-to-HEAD on a year of
          already-written notes.
        </P>
      </DocSection>
    </DocPage>
  )
}
