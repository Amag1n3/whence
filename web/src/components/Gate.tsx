import { Reveal } from '@/components/Reveal'
import { cn } from '@/lib/utils'

/* The gate, and deliberately the only still object on the page.
 *
 * The hero terminal types itself out and the drift demo animates through four
 * states, because both are showing a process. This is showing a verdict. CI
 * output does not perform — it sits in a log and refuses to let you merge, and
 * making it move would be the page lying about what the experience is. The
 * restraint is the point: after two animated blocks, a block that just stops
 * and says `exit 1` lands harder than a third animation would. */

type Finding = {
  mark: '!' | '✗'
  tone: 'ochre' | 'cinnabar'
  where: string
  head: string
  body: string[]
}

/* Both findings are real shapes from check's output. The second is the one that
   justifies the whole evidence layer: the change never opened the file the
   record lives in. */
const FINDINGS: Finding[] = [
  {
    mark: '!',
    tone: 'ochre',
    where: 'src/auth/session.go:145',
    head: 'touches record [4f2a]',
    body: [
      'Namespace all three session keys to CHECKOUT_*.',
      'src/auth/session.go:142-148 · intact, exact range',
    ],
  },
  {
    mark: '✗',
    tone: 'cinnabar',
    where: 'dashboard/Header.tsx:88-94',
    head: 'this change removes the evidence for record [4f2a]',
    body: [
      'the record still anchors to session.go:142-148 and reads as current',
      'what made it true is gone, so the decision now rests on nothing.',
    ],
  },
]

export function Gate() {
  return (
    <Reveal>
      <div className="lit overflow-hidden rounded-xl border border-white/10 bg-terminal">
        {/* the command, and the only number that matters, on one line */}
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-white/[0.07] px-5 py-3">
          <span className="font-mono text-[12.5px] text-silt/85">
            <span className="text-ochre/70">$ </span>whence check --base origin/main
          </span>
          <span className="ml-auto rounded-full border border-cinnabar/40 bg-cinnabar/[0.09] px-2.5 py-0.5 font-mono text-[11px] tracking-wide text-cinnabar">
            exit 1
          </span>
        </div>

        <div className="divide-y divide-white/[0.05]">
          {FINDINGS.map((f) => (
            <div key={f.where} className="flex gap-4 px-5 py-5">
              <span
                className={cn(
                  'mt-px w-3 shrink-0 font-mono text-[13px] leading-[1.6]',
                  f.tone === 'cinnabar' ? 'text-cinnabar' : 'text-ochre',
                )}
                aria-hidden
              >
                {f.mark}
              </span>
              <div className="min-w-0">
                <p className="font-mono text-[12.5px] leading-[1.6] break-words">
                  <span className="text-silt/90">{f.where}</span>{' '}
                  <span className={f.tone === 'cinnabar' ? 'text-cinnabar' : 'text-ochre'}>
                    — {f.head}
                  </span>
                </p>
                {f.body.map((l) => (
                  <p
                    key={l}
                    className="mt-1.5 font-mono text-[12px] leading-[1.65] text-silt/45"
                  >
                    {l}
                  </p>
                ))}
              </div>
            </div>
          ))}
        </div>

        <div className="border-t border-white/[0.07] bg-white/[0.015] px-5 py-4">
          <p className="max-w-[76ch] text-[13.5px] leading-[1.6] text-muted-foreground">
            <b className="font-semibold text-silt">It reports coverage, never verdicts.</b>{' '}
            Decisions govern these lines — go and read them. Whether your change is right is
            not its call, and a tool that starts making that call is a code reviewer.
          </p>
        </div>
      </div>
    </Reveal>
  )
}
