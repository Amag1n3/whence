import type { ReactNode } from 'react'

import { DocPage, DocSection, P, type DocSectionMeta } from '@/components/DocPage'
import { cn } from '@/lib/utils'

/* The article at FINDINGS.md, rendered as JSX. /notes is the article — not
   an index, not a post list. A second note is when a generalisation earns
   its keep. */

const SECTIONS: DocSectionMeta[] = [
  { id: 'filter', label: 'The filter' },
  { id: 'corpora', label: 'Across four corpora' },
  { id: 'unexpected', label: 'The part I did not expect' },
  { id: 'quality', label: 'When it fires, is it any good?' },
  { id: 'rule', label: 'One unspecified rule' },
  { id: 'absence', label: 'What did not happen' },
  { id: 'caveats', label: 'Caveats' },
  { id: 'conclusion', label: 'The conclusion' },
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
function Table({ cols, rows }: { cols: string[]; rows: ReactNode[][] }) {
  return (
    <div className="term-scroll max-w-full overflow-x-auto">
      <table className="w-max min-w-full border-collapse font-mono text-[12.5px] leading-[1.65]">
        <thead>
          <tr className="border-b border-white/[0.07] text-dim">
            {cols.map((c, i) => (
              <th
                key={c}
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
          51 records survived the gates. I dumped each one as it would actually
          be stored — the decision line, the why, the prose it came from, and
          the exact text the edit wrote — and judged all 51 by hand. Then I
          gave the same 51, and an identical prompt carrying none of my
          verdicts, to two other frontier models.
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

      <DocSection id="absence" n={6} title="What did not happen is the interesting part">
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
      </DocSection>

      <DocSection id="caveats" n={7} title="Caveats, because they bound everything above">
        <ul className="max-w-[56ch] list-disc space-y-3 pl-5 text-[14.5px] leading-[1.68] text-muted-foreground">
          <li>
            This measures <B>attribution</B> — does the record explain this
            edit — not <B>truth</B>. A record can pass and still be factually
            wrong about the world. Nobody has measured that.
          </li>
          <li>
            The corpus is <B>enriched</B>: 51 of 2,672 edits, 1.9%, selected by
            a filter that fires on the agent's most diagnosis-heavy moments.
            This measures records at their best.
          </li>
          <li>All of it is one engineer's sessions with one coding agent.</li>
          <li>
            Three LLMs grading LLM-written prose is a conflict of interest I
            cannot design away. Failures visible without domain knowledge get
            caught; failures requiring it are systematically undercounted. All
            three graders raised this independently.
          </li>
          <li>
            <B>A human tiebreak is pending.</B> I am reading a ten-record
            subset by hand, including both unanimous anchors. This page gets
            updated with that result, whether or not it agrees with me.
          </li>
        </ul>
      </DocSection>

      <DocSection id="conclusion" n={8} title="The conclusion I actually drew">
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
          whence is Go, zero dependencies, and its records are JSONL committed
          alongside your code. What I would most like to know is whether the
          anchoring survives a codebase that is not mine.
        </P>
      </DocSection>
    </DocPage>
  )
}
