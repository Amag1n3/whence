import { lazy, Suspense } from 'react'
import { Check, Minus } from 'lucide-react'

import { Reveal } from '@/components/Reveal'
import { DocPage, DocSection, P, type DocSectionMeta } from '@/components/DocPage'
/* The only component left on the site that genuinely needs motion/react —
   AnimatePresence and layout animation, which CSS cannot do. Loading it
   lazily keeps its 119kB runtime off /why's critical path; it sits well below
   the fold and does not animate until scrolled to anyway. The placeholder
   reserves the frame so nothing shifts when it arrives. */
const DriftDemo = lazy(() =>
  import('@/components/DriftDemo').then((m) => ({ default: m.DriftDemo })),
)
import { Gate } from '@/components/Gate'
import { REPO } from '@/components/Chrome'
import { cn } from '@/lib/utils'

/* The argument, off the landing page. It used to be eleven layers of it there,
   with the install command at depth 07 — so the page asked every reader to sit
   through the whole case before it would tell them how to run the thing.
 *
 * It is also half the length it was. A reader who clicked "why it exists" has
 * agreed to hear the case, not to read an essay: the two demos do the work
 * three paragraphs used to, and every section that survived lost its second
 * and third paragraph. What was cut is not gone — /faq answers sixty-one
 * versions of it at length, which is the right place for the reader who
 * actually wants more. */

const SECTIONS: DocSectionMeta[] = [
  { id: 'problem', label: 'The problem' },
  { id: 'how', label: 'How it works' },
  { id: 'anchoring', label: 'Anchoring' },
  { id: 'gate', label: 'The gate' },
  { id: 'status', label: 'What runs today' },
  { id: 'handling', label: 'Your code' },
  { id: 'kill', label: 'The kill number' },
]

const STEPS: [string, string, string][] = [
  [
    'Capture',
    'not built',
    'Record the decision trail as the session runs. Redaction happens here, before anything is written to disk.',
  ],
  [
    'Anchor',
    'works',
    'Hash every significant line of the span, so the record follows the code instead of a line number.',
  ],
  [
    'Surface',
    'works',
    'In your terminal, in your agent’s context through a PreToolUse hook before it edits, and in CI as a gate.',
  ],
]

const LEDGER: [boolean, string, string][] = [
  [true, 'The PreToolUse hook', 'Records reach Claude Code before it edits, in 6ms. Fails open.'],
  [true, 'Content-hash anchoring', 'A record follows its code, decays when the code is rewritten, and reports itself orphaned rather than pointing at the wrong line.'],
  [true, 'whence check', 'The CI gate. Fails the build only for decisions a diff damaged.'],
  [true, 'Evidence', 'A record can point at what makes it true. That pointer is anchored too, so it can rot on its own.'],
  [true, 'whence backfill', 'Harvests decisions already written in your code, so day one is not an empty store.'],
  [false, 'Capture', 'Deciding which slice of a session is worth keeping is the actual hard part, and it is not built.'],
]

/* Three, down from five. The two that went — "hashes, paths and ranges, never
   file contents" and "a record no human has read says so" — are still
   answered on /faq at length. These three are the ones that would change
   someone's mind about running this on a private repo, which is the only job
   this section has. */
const COMMITMENTS: [string, string][] = [
  [
    'Redaction runs before the write, not before the share',
    'The store is meant to be committed, so anything capture picks up is public the moment you push — and git history keeps it.',
  ],
  [
    'Records are data, never directives',
    'They arrive by git pull, so anyone who can land a commit can put text in front of your agent. Every injected block is framed as history to be aware of, not instruction to follow.',
  ],
  [
    'Attribution stays aggregate',
    'No per-developer AI-authorship leaderboards. This is a developer tool and it will not become a surveillance tool.',
  ],
]

export default function WhyPage() {
  return (
    <DocPage
      current="why"
      eyebrow="why it exists"
      title="The reason is worth more than the diff"
      lede="Coding agents write a large share of merged code and remember none of why they wrote it. This is the case for keeping that reasoning, and the honest line between what runs today and what does not."
      sections={SECTIONS}
    >
      {/* ------------------------------------------------------------ 01 */}
      <DocSection id="problem" n={1} title="The mess was load-bearing">
        <P>
          Everyone has this story. Someone opens a file, finds code that looks
          redundant, tidies it up, and takes production down — the mess was written
          that way after an incident, and the reason was never recorded anywhere they
          would look. That used to happen occasionally, at human speed. Now every team
          has an infinitely fast engineer with total amnesia.
        </P>

        {/* Two dated markers and the distance between them: the one place on the
            site where the amnesia costs something specific. */}
        <div className="pt-4">
          <div className="flex items-center gap-4">
            {/* Left marker dim, right marker full — the two dates were an
                amber-to-red run, and in monochrome the same "it got worse"
                reading has to come from brightness. */}
            <span className="size-2 shrink-0 rounded-full bg-silt/45" />
            <span className="h-px flex-1 bg-gradient-to-r from-white/25 to-white/[0.08]" />
            <span className="font-mono text-[11px] tracking-[0.16em] text-dim uppercase">
              three days
            </span>
            <span className="h-px flex-1 bg-gradient-to-r from-white/[0.08] to-white/70" />
            <span className="size-2 shrink-0 rounded-full bg-silt" />
          </div>
          <div className="mt-5 flex items-start justify-between gap-10">
            <div className="max-w-[24ch]">
              <p className="font-mono text-[11px] tracking-[0.16em] text-silt/45">2026-07-27</p>
              <p className="mt-2 text-[13.5px] leading-[1.55] text-muted-foreground">
                The decision gets made, and written down in a code review.
              </p>
            </div>
            <div className="max-w-[24ch] text-right">
              <p className="font-mono text-[11px] tracking-[0.16em] text-silt">2026-07-30</p>
              <p className="mt-2 text-[13.5px] leading-[1.55] text-muted-foreground">
                The same bug is worked out again from scratch. It takes an evening.
              </p>
            </div>
          </div>
        </div>

        <p className="max-w-[42ch] border-l-2 border-ochre pl-6 text-[18px] leading-[1.5] font-medium text-silt">
          The agent <em className="text-ochre not-italic">had</em> the reasoning. Then the
          session ended and all of it was thrown away. What reached the repository was a
          diff.
        </p>
      </DocSection>

      {/* ------------------------------------------------------------ 02 */}
      <DocSection id="how" n={2} title="Capture, anchor, surface">
        <div className="grid gap-8 sm:grid-cols-3">
          {STEPS.map(([t, tag, d], i) => (
            <div key={t} className="border-t border-white/[0.09] pt-5">
              <span className="font-mono text-[11px] text-dim">
                {String(i + 1).padStart(2, '0')}
              </span>
              <h3 className="mt-2 flex flex-wrap items-center gap-2 text-[18px]">
                {t}
                <span
                  className={cn(
                    'border px-2 py-0.5 font-mono text-[10px] tracking-wide uppercase',
                    tag === 'works'
                      ? 'border-verdigris/35 text-verdigris'
                      : 'border-white/12 text-dim',
                  )}
                >
                  {tag}
                </span>
              </h3>
              <p className="mt-3 text-[13.5px] leading-[1.6] text-muted-foreground">{d}</p>
            </div>
          ))}
        </div>
      </DocSection>

      {/* ------------------------------------------------------------ 03 */}
      <DocSection id="anchoring" n={3} title="A record that loses its anchor is a diary entry">
        <P>
          Line 142 today is line 187 tomorrow, and in a different file next week. A tool
          that confidently points at the wrong line teaches you to distrust everything it
          says — so a record that cannot find its anchor says so.
        </P>
        <Suspense
          fallback={
            <div className="lit min-h-[290px] rounded-xl border border-white/10 bg-terminal" />
          }
        >
          <DriftDemo />
        </Suspense>
      </DocSection>

      {/* ------------------------------------------------------------ 04 */}
      <DocSection id="gate" n={4} title="The exit code is the product">
        <P>
          In CI, a diff is compared against the decisions governing the lines it touches
          — and the build fails only for the ones it quietly wore away or severed.
        </P>
        <Gate />
      </DocSection>

      {/* ------------------------------------------------------------ 05 */}
      <DocSection id="status" n={5} title="What actually runs today">
        <div className="grid gap-px overflow-hidden rounded-lg border border-white/[0.08] bg-white/[0.06] sm:grid-cols-2">
          {LEDGER.map(([on, t, d]) => (
            <div key={t} className="h-full bg-basin/85 p-5">
              <div className="flex items-center gap-2.5">
                {on ? (
                  <Check className="size-4 shrink-0 text-verdigris" strokeWidth={2.5} />
                ) : (
                  <Minus className="size-4 shrink-0 text-dim/70" strokeWidth={2.5} />
                )}
                <span className={cn('font-mono text-[13.5px]', on ? 'text-silt' : 'text-dim')}>
                  {t}
                </span>
              </div>
              <p className="mt-2.5 text-[13.5px] leading-[1.6] text-muted-foreground">{d}</p>
            </div>
          ))}
        </div>
        <P>
          It is also not a code reviewer, not an observability platform and not a
          knowledge graph. It answers one question about one line.
        </P>
      </DocSection>

      {/* ------------------------------------------------------------ 06 */}
      <DocSection id="handling" n={6} title="What it does with your code">
        <P>This reads what an agent saw and did, which makes it sensitive by default.</P>
        <div className="grid gap-x-12 gap-y-px sm:grid-cols-2">
          {COMMITMENTS.map(([t, d]) => (
            <div key={t} className="border-t border-white/[0.09] py-4">
              <b className="text-[14.5px] font-semibold text-silt">{t}</b>{' '}
              <span className="text-[14.5px] leading-[1.6] text-muted-foreground">{d}</span>
            </div>
          ))}
        </div>
      </DocSection>

      {/* ------------------------------------------------------------ 07 */}
      <DocSection id="kill" n={7} title="The number that kills it">
        {/* The kill clause used to be the one red sentence on the site. With
            no hue left, emphasis inverts instead: the setup drops to muted
            and the clause keeps full-strength foreground, so the brightest
            thing in the panel is still the sentence that ends the project. */}
        <div className="border border-white/30 bg-white/[0.04] p-6 sm:p-8">
          <p className="max-w-[50ch] text-[19px] leading-[1.5] text-muted-foreground">
            Count the times an agent proposed a change that contradicted a recorded
            decision and <span className="font-mono text-[0.86em] text-silt">whence</span>{' '}
            caught it.{' '}
            <span className="font-medium text-silt">
              If that is zero after three months of real daily use, the idea is wrong and
              this repo gets archived.
            </span>
          </p>
          <p className="mt-5 max-w-[56ch] text-[14.5px] leading-[1.68] text-muted-foreground">
            Agreed before a line of code existed. The number also cannot see its own worst
            failure: if a stated reason is a story told afterwards rather than the actual
            cause, this preserves confident nonsense durably — and a store full of nonsense
            produces <em className="not-italic text-silt">more</em> catches, not fewer. That
            is why retractions are logged, and why capture stays off until the
            faithfulness of stated reasoning has been measured.
          </p>
        </div>
      </DocSection>

      <Reveal>
        <p className="max-w-[56ch] border-t border-white/[0.07] pt-7 text-[14.5px] leading-[1.7] text-muted-foreground">
          The long version of all of this —{' '}
          <a
            href="/faq"
            className="text-ochre underline-offset-4 transition-colors hover:underline"
          >
            sixty-one answers
          </a>{' '}
          — or{' '}
          <a
            href={REPO}
            className="text-ochre underline-offset-4 transition-colors hover:underline"
          >
            the source
          </a>
          .
        </p>
      </Reveal>
    </DocPage>
  )
}
