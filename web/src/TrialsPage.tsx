import type { ReactNode } from 'react'

import { DocPage, DocSection, P, type DocSectionMeta } from '@/components/DocPage'
import { cn } from '@/lib/utils'

/* A log, not an article. /notes argues one thing and is finished; this page
   gets a row appended every time whence is run against somebody else's code,
   and exists so that the runs are remembered rather than repeated. The table
   is the page — the prose around it only says what the columns cannot. */

const SECTIONS: DocSectionMeta[] = [
  { id: 'method', label: 'The method' },
  { id: 'runs', label: 'The runs' },
  { id: 'found', label: 'What each round found' },
  { id: 'ceiling', label: 'The ceiling' },
  { id: 'untested', label: 'Still untested' },
]

const M = ({ children }: { children: ReactNode }) => (
  <code className="font-mono text-[12.5px] text-silt">{children}</code>
)

const B = ({ children }: { children: ReactNode }) => (
  <b className="text-silt">{children}</b>
)

/** Copied from /notes rather than shared. Two pages is not a component
 *  library, and the day one of them needs a different column rule, a shared
 *  table would have to grow a flag to say so. */
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

export default function TrialsPage() {
  return (
    <DocPage
      current="trials"
      eyebrow="trials · updated 2026-08-16"
      title="Running whence against code that is not mine"
      lede="whence was written in one repository, by one person, and every rule in it was tuned against that repository's habits. A rule tuned on one codebase is a guess about all the others. So the tool gets pointed at large codebases nobody here wrote, the output is read by hand, and whatever turns out to be junk becomes the next fix. This page is the log of those runs."
      sections={SECTIONS}
    >
      <DocSection id="method" n={1} title="The method, which is deliberately dull">
        <P>
          Shallow-clone two or three large repositories into a temporary
          directory. Run <M>whence backfill</M> against each one. The command
          writes nothing without <M>--yes</M>, so every run is a dry run and no
          store is ever touched. Read the candidates by hand — all of them when
          there are three hundred, a sample when there are eight hundred — and
          sort them into records that state a decision and records that do not.
          Delete the clones. Write down what the junk had in common.
        </P>
        <P>
          The junk is the output. Each round produces a small number of rules,
          each rule is one filter, and the next round is run against the fixed
          binary so the fixes get tested by the same method that produced them.
          No round has yet failed to find something.
        </P>
      </DocSection>

      <DocSection id="runs" n={2} title="The runs">
        <Table
          cols={['repository', 'language', 'markers', 'candidates', 'round']}
          rows={[
            ['kubernetes/kubernetes', 'Go', '1,796', '92', '1'],
            ['microsoft/vscode', 'TypeScript', '372', '67', '1'],
            ['python/cpython', 'C, Python', '591', '151', '1'],
            ['torvalds/linux', 'C', '5,926', '517', '2'],
            ['rust-lang/rust', 'Rust', '1,993', '303', '2'],
            ['django/django', 'Python', '34', '3', '2'],
          ]}
        />
        <P>
          <B>Markers</B> counts the comments carrying a marker whence accepts.
          <B> Candidates</B> is how many of those also cleared the gate that
          asks whether the comment gives a reason. Round two ran against the
          binary round one produced. Linux, at two gigabytes, took fourteen
          seconds.
        </P>
      </DocSection>

      <DocSection id="found" n={3} title="What each round found">
        <P>
          <B>Round one</B> found three kinds of junk, and all three are now
          filtered. <M>XXX:</M> was trusted without a reason gate on the premise
          that nobody writes it about something obvious — CPython writes it the
          way other projects write <M>TODO</M>, and ninety-one of its
          hundred-and-fifty-one candidates arrived that way. Twenty of
          Kubernetes' ninety-two were one line emitted by a code generator into
          files where regenerating rewrites the anchor underneath it. And
          splitting a note at its first reason word stored one Kubernetes
          decision as, in full, the words <M>In order</M>.
        </P>
        <P>
          <B>Round two</B> found four more, against the fixed binary. The
          word-count guard that round one added counts punctuation as a word, so
          a headline reading <M>This must be a macro (</M> clears a six-word
          minimum with five real words. That split happened inside a
          parenthetical, which is a second bug wearing the first one's clothes.
          Rust's lint tests keep a generated <M>.fixed</M> twin beside each
          fixture, so twenty-two candidates arrived in identical pairs. And a
          question is still admitted as a decision — Linux offers{' '}
          <M>Is there any reason to assume differently?</M>, which qualifies
          because the word "reason" is on the list that proves a note explains
          itself.
        </P>
      </DocSection>

      <DocSection id="ceiling" n={4} title="The ceiling, which is the real result">
        <P>
          Django is the interesting row. Seventy-four megabytes of mature,
          heavily-commented Python produced <B>three records</B>. Not because
          the filters are wrong — the three are all decent — but because Django
          barely uses the markers whence reads. It has thirty-four of them in
          the entire repository, against seven hundred and eighty-five comment
          lines that carry a reason word and no marker at all.
        </P>
        <P>
          That ratio holds everywhere it has been measured: Linux has 5,926
          markers and roughly 38,000 unmarked comment lines that explain
          themselves; Rust, 1,993 against 8,651. Marker-gated harvesting reaches
          a small single-digit percentage of the reasoning a codebase has
          already written down. That is a deliberate trade — a missed note is
          recoverable by hand and a junk record in a shared store is not — but
          it means the honest promise on day one is not "your reasoning, now
          durable". It is "the handful of notes you flagged loudly enough".
        </P>
      </DocSection>

      <DocSection id="untested" n={5} title="What none of this has tested">
        <P>
          Anchoring was the gap here, and it is the one thing on this page that
          has since been closed. Every run above exercised the harvester and the
          sentence splitter, the easy half; whether a record stays attached
          while somebody else's code moves underneath it needed a different
          experiment, and in August 2026 it got one. Records were harvested at a
          commit a year back in rust-lang/rust and microsoft/vscode, then
          re-resolved across the history that followed: 198 of 238 still
          resolved. That run is written up in the notes.
        </P>
        <P>
          What is still untested is narrower, and worth naming rather than
          leaving implied. That experiment walked notes those repositories had
          already written, not records a person authored deliberately for their
          own code. vscode contributed ten of the records, so the result leans
          on effectively one codebase. And it measured a year already in the
          past — not a record written today and watched forward from here, which
          is the shape anyone installing this actually gets.
        </P>
      </DocSection>
    </DocPage>
  )
}
