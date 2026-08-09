import { Reveal } from '@/components/Reveal'
import { cn } from '@/lib/utils'

/* The gate, and deliberately the only still object on the page.
 *
 * The hero terminal types itself out and the drift demo animates through four
 * states, because both are showing a process. This is showing a verdict. CI
 * output does not perform — it sits in a log and refuses to let you merge, and
 * making it move would be the page lying about what the experience is. The
 * restraint is the point: after two animated blocks, a block that just stops
 * and says `exit 1` lands harder than a third animation would.
 *
 * What the rows have to carry now is that the three findings are NOT equal.
 * Only two of them failed this build. So the two that did get the treatment
 * DriftDemo already uses for "this is the line under discussion" — a solid
 * left rule and a wash of its own tone — and the one that did not gets neither,
 * and sits at the top where it is read first and dismissed first. Same tone
 * mapping as the drift demo and the CLI, because there is one vocabulary for
 * anchor state across all three: verdigris intact, ochre altered, cinnabar
 * lost. A second mapping would only be somewhere for them to drift apart. */

type Finding = {
  mark: '·' | '!' | '✗'
  tone: 'verdigris' | 'ochre' | 'cinnabar'
  /* Whether this finding is why the build failed. Three of these render; two
     are true, and the summary strip counts them rather than asserting a number
     the rows do not add up to. */
  blocks: boolean
  where: string
  head: string
  body: string[]
}

/* All three are real shapes from check's output, in the order it prints them.
   The last is the one that justifies the whole evidence layer: the change never
   opened the file the record lives in. */
const FINDINGS: Finding[] = [
  {
    mark: '·',
    tone: 'verdigris',
    blocks: false,
    where: 'src/auth/session.go:145',
    head: 'covered by record [b5], intact',
    body: [
      'Namespace all three session keys to CHECKOUT_*.',
      'src/auth/session.go:142-148 · intact, exact range',
    ],
  },
  {
    mark: '!',
    tone: 'ochre',
    blocks: true,
    where: 'src/net/retry.go',
    head: 'this change erodes record [7d31]',
    body: [
      '100% of the recorded block survived before this diff, 64% now',
      'Retry with backoff here; the provider rate-limits per-account, not per-key.',
      'src/net/retry.go:88-94 · altered',
    ],
  },
  {
    mark: '✗',
    tone: 'cinnabar',
    blocks: true,
    where: 'dashboard/Header.tsx:88-94',
    head: 'this change removes the evidence for record [4f2a]',
    body: [
      'the record still anchors to src/auth/token.go:31-38 and reads as current',
      'what made it true is gone, so the decision now rests on nothing.',
    ],
  },
]

/* Tailwind cannot build a class name from a variable, so the mapping is
   spelled out rather than interpolated.

   These were three hues. On a monochrome page three hues collapse into one
   white and the rows stop being distinguishable, so severity is an opacity
   ladder now: a finding gets brighter the worse it is. The ✓/✗ marks and the
   prose were always carrying the actual meaning — the colour was
   reinforcement, and this keeps the reinforcement without the colour. */
const TEXT = {
  verdigris: 'text-silt/55',
  ochre: 'text-silt/80',
  cinnabar: 'text-silt',
} as const

const RULE = {
  verdigris: 'border-white/20 bg-white/[0.02]',
  ochre: 'border-white/45 bg-white/[0.04]',
  cinnabar: 'border-white bg-white/[0.07]',
} as const

export function Gate() {
  const damaged = FINDINGS.filter((f) => f.blocks).length
  const intact = FINDINGS.length - damaged

  return (
    <Reveal>
      <div className="lit overflow-hidden rounded-xl border border-white/10 bg-terminal">
        {/* the command, and the only number that matters, on one line */}
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-white/[0.07] px-5 py-3">
          <span className="font-mono text-[12.5px] text-silt/85">
            <span className="text-ochre/70">$ </span>whence check --base origin/main
          </span>
          {/* Inverted rather than tinted. With no red available, the loudest
              thing a monochrome page can do is flip to a light fill — and
              this is the one element on the site that has earned it. */}
          <span className="ml-auto bg-silt px-2.5 py-0.5 font-mono text-[11px] font-medium tracking-wide text-basin">
            exit 1
          </span>
        </div>

        <div className="divide-y divide-white/[0.05]">
          {FINDINGS.map((f) => (
            <div
              key={f.where}
              className={cn(
                'flex gap-4 border-l-2 py-5 pr-5 pl-4',
                f.blocks ? RULE[f.tone] : 'border-transparent',
              )}
            >
              <span
                className={cn(
                  'mt-px w-3 shrink-0 font-mono text-[13px] leading-[1.6]',
                  TEXT[f.tone],
                  f.blocks ? '' : 'opacity-70',
                )}
                aria-hidden
              >
                {f.mark}
              </span>
              <div className="min-w-0">
                <p className="font-mono text-[12.5px] leading-[1.6] break-words">
                  <span className={f.blocks ? 'text-silt/90' : 'text-silt/60'}>
                    {f.where}
                  </span>{' '}
                  <span className={TEXT[f.tone]}>— {f.head}</span>
                </p>
                {f.body.map((l) => (
                  <p
                    key={l}
                    className={cn(
                      'mt-1.5 font-mono text-[12px] leading-[1.65]',
                      f.blocks ? 'text-silt/45' : 'text-silt/30',
                    )}
                  >
                    {l}
                  </p>
                ))}
              </div>
            </div>
          ))}
        </div>

        {/* check's own summary line, which is where the count is reconciled */}
        <div className="border-t border-white/[0.07] bg-white/[0.02] px-5 py-3.5">
          <p className="font-mono text-[12px] leading-[1.6] text-silt/60">
            <span className="text-cinnabar">{damaged} recorded decisions damaged</span> by
            this change, {intact} more covered and intact.
          </p>
        </div>

        <div className="border-t border-white/[0.07] bg-white/[0.015] px-5 py-4">
          <p className="max-w-[76ch] text-[13.5px] leading-[1.6] text-muted-foreground">
            <b className="font-semibold text-silt">Only damage fails the build.</b> The
            first finding is the diff passing through lines a decision covers, with the
            record still whole — it prints and exits 0. A gate that failed there would fail
            on reformatting, so it would fail on every pull request, so it would get
            switched off, taking the two findings below it — the ones no reviewer catches
            unaided — along with it.
          </p>
          <p className="mt-3 max-w-[76ch] text-[13.5px] leading-[1.6] text-muted-foreground">
            <b className="font-semibold text-silt">And it reports coverage, never
            verdicts.</b>{' '}
            Whether your change is right is not its call. A tool that starts making that
            call is a code reviewer.
          </p>
        </div>
      </div>
    </Reveal>
  )
}
