import { useEffect, useRef, useState } from 'react'
import { AnimatePresence, motion, useInView, useReducedMotion } from 'motion/react'
import { cn } from '@/lib/utils'

/* The signature of the page. Anchoring is the one genuinely hard problem in
   this project, and it is invisible in prose — a record's line number is
   wrong the moment anyone edits above it. So show the code moving, and show
   what the anchor does about it. */

type Line = { id: string; text: string; anchored?: boolean }

type Stage = {
  key: string
  label: string
  caption: string
  lines: Line[]
  start: number
  integrity: number
  state: string
  tone: 'verdigris' | 'ochre' | 'cinnabar'
}

/* Synthetic, matching the record the hero terminal shows. Same bug class as
   the review it came from; invented service, because a public page is no
   place for an employer's file paths. */
const HEAD: Line[] = [{ id: 'h1', text: 'func persist(s Session) {' }]
const TAIL: Line[] = [{ id: 't1', text: '}' }]

const ANCHORED: Line[] = [
  { id: 'a1', text: '\t// CHECKOUT_* namespace — see .whence [b5]', anchored: true },
  { id: 'a2', text: '\tstore.Set("CHECKOUT_userToken", s.Token)', anchored: true },
  { id: 'a3', text: '\tstore.Set("CHECKOUT_userID",    s.ID)', anchored: true },
  { id: 'a4', text: '\tstore.Set("CHECKOUT_role",      s.Role)', anchored: true },
]

const INSERTED: Line[] = [
  { id: 'n1', text: 'store := sessionStore(ctx)' },
  { id: 'n2', text: '' },
]

const STAGES: Stage[] = [
  {
    key: 'recorded',
    label: 'recorded',
    caption: 'A decision is written down and pinned to lines 141–144. Everything agrees.',
    lines: [...HEAD, ...ANCHORED, ...TAIL],
    start: 140,
    integrity: 1,
    state: 'intact, exact range',
    tone: 'verdigris',
  },
  {
    key: 'moved',
    label: 'the code moves',
    caption:
      'Someone adds two lines above. Every line number in the record is now wrong — but the content still hashes the same, so the anchor follows it down.',
    lines: [...INSERTED, ...HEAD, ...ANCHORED, ...TAIL],
    start: 140,
    integrity: 1,
    state: 'intact, moved',
    tone: 'verdigris',
  },
  {
    key: 'edited',
    label: 'one line changes',
    caption:
      'Someone drops the namespace on one of the three keys. Three of the four anchored lines still hash; one does not. So it is still found — and reported as no longer trustworthy, at a number that says how much survived.',
    lines: [
      ...INSERTED,
      ...HEAD,
      ANCHORED[0],
      ANCHORED[1],
      ANCHORED[2],
      { id: 'e1', text: '\tstore.Set("role",               s.Role)', anchored: true },
      ...TAIL,
    ],
    start: 140,
    integrity: 0.75,
    state: 'altered',
    tone: 'ochre',
  },
  {
    key: 'rewritten',
    label: 'the code is rewritten',
    caption:
      'The block is refactored away. Nothing hashes — not at the recorded lines, not anywhere else in the file. The record is surfaced as orphaned, loudly, and claims no line number at all rather than quietly pointing at whatever now sits on 143.',
    lines: [
      ...INSERTED,
      ...HEAD,
      { id: 'r1', text: '\tpersistNamespaced(s)', anchored: true },
      ...TAIL,
    ],
    start: 140,
    integrity: 0,
    state: 'ORPHANED — needs a human',
    tone: 'cinnabar',
  },
]

/* Four stages now, so each holds a little less — long enough to read the
   caption, short enough that the whole cycle is watchable in one sitting. */
const DWELL = 3800

export function DriftDemo() {
  const ref = useRef<HTMLDivElement>(null)
  const inView = useInView(ref, { once: false, margin: '-20% 0px' })
  const reduced = useReducedMotion()
  const [i, setI] = useState(0)
  const [held, setHeld] = useState(false)

  useEffect(() => {
    if (reduced || held || !inView) return
    const t = window.setTimeout(() => setI((n) => (n + 1) % STAGES.length), DWELL)
    return () => window.clearTimeout(t)
  }, [i, held, inView, reduced])

  const stage = STAGES[i]
  /* Was three hues; monochrome collapses those to one white, so decay is an
     opacity ladder instead — the record gets brighter as it gets worse. The
     integrity percentage and the state label next to it were always the
     literal signal, so nothing is lost that the colour alone was carrying. */
  const toneText =
    stage.tone === 'cinnabar'
      ? 'text-silt'
      : stage.tone === 'ochre'
        ? 'text-silt/85'
        : 'text-silt/55'
  const toneBg =
    stage.tone === 'cinnabar' ? 'bg-silt' : stage.tone === 'ochre' ? 'bg-silt/85' : 'bg-silt/55'

  return (
    <div
      ref={ref}
      className="lit grid overflow-hidden rounded-xl border border-white/10 bg-terminal lg:grid-cols-[minmax(0,1fr)_21rem]"
    >
      {/* the file */}
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5 border-b border-white/[0.07] px-4 py-2.5">
          <span className="font-mono text-[11px] text-silt/35">
            src/auth/session.go
          </span>
          <span
            className={cn(
              'ml-auto font-mono text-[11px] transition-colors duration-500',
              toneText,
            )}
          >
            {stage.state}
          </span>
        </div>

        <div className="term-scroll min-h-[210px] overflow-x-auto py-4">
          <div className="min-w-max font-mono text-[12px] leading-[1.85] sm:text-[12.5px]">
            <AnimatePresence initial={false}>
              {stage.lines.map((l, idx) => (
                <motion.div
                  key={l.id}
                  layout={!reduced}
                  initial={reduced ? false : { opacity: 0, x: -8 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={reduced ? undefined : { opacity: 0, x: 8 }}
                  transition={{ duration: 0.38, ease: [0.16, 0.8, 0.28, 1] }}
                  className={cn(
                    'flex items-baseline gap-4 border-l-2 pr-6 transition-colors duration-500',
                    l.anchored
                      ? stage.tone === 'cinnabar'
                        ? 'border-white bg-white/[0.08]'
                        : stage.tone === 'ochre'
                          ? 'border-white/50 bg-white/[0.05]'
                          : 'border-white/25 bg-white/[0.03]'
                      : 'border-transparent',
                  )}
                >
                  <span className="w-14 shrink-0 pl-2 text-right text-silt/25 tabular-nums">
                    {stage.start + idx}
                  </span>
                  <span className={cn(l.anchored ? 'text-silt/90' : 'text-silt/45')}>
                    {l.text || ' '}
                  </span>
                </motion.div>
              ))}
            </AnimatePresence>
          </div>
        </div>
      </div>

      {/* side rail: what the anchor is doing, and why */}
      <div className="flex flex-col gap-5 border-t border-white/[0.07] bg-white/[0.015] p-5 lg:border-t-0 lg:border-l">
        <div>
          <div className="flex items-baseline justify-between">
            <span className="font-mono text-[11px] text-silt/35">record [b5]</span>
            <span className={cn('font-mono text-[15px] tabular-nums', toneText)}>
              {`${Math.round(stage.integrity * 100)}%`}
            </span>
          </div>
          <div className="mt-2 h-1 overflow-hidden rounded-full bg-white/10">
            <motion.div
              className={cn('h-full rounded-full', toneBg)}
              animate={{ width: `${stage.integrity * 100}%` }}
              transition={{ duration: 0.7, ease: [0.16, 0.8, 0.28, 1] }}
            />
          </div>
          <p className="mt-1.5 font-mono text-[10.5px] tracking-wide text-silt/30">
            intact
          </p>
        </div>

        <div className="flex flex-col gap-1">
          {STAGES.map((s, idx) => (
            <button
              key={s.key}
              onClick={() => {
                setI(idx)
                setHeld(true)
              }}
              className={cn(
                'rounded-md px-2.5 py-1.5 text-left font-mono text-[11px] tracking-wide transition-colors',
                idx === i
                  ? 'bg-white/10 text-foreground'
                  : 'text-silt/35 hover:bg-white/[0.04] hover:text-silt/65',
              )}
            >
              {String(idx + 1).padStart(2, '0')}  {s.label}
            </button>
          ))}
        </div>

        <AnimatePresence mode="wait">
          <motion.p
            key={stage.key}
            initial={reduced ? false : { opacity: 0, y: 4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={reduced ? undefined : { opacity: 0, y: -4 }}
            transition={{ duration: 0.3 }}
            className="text-[13px] leading-[1.65] text-silt/55"
          >
            {stage.caption}
          </motion.p>
        </AnimatePresence>
      </div>
    </div>
  )
}
