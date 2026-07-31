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
  confidence: number
  state: string
  tone: 'moss' | 'honey' | 'oxide'
}

const HEAD: Line[] = [{ id: 'h1', text: 'const persist = (session) => {' }]
const TAIL: Line[] = [{ id: 't1', text: '}' }]

const ANCHORED: Line[] = [
  { id: 'a1', text: '  // CHECKOUT_* namespace — see .whence [b5]', anchored: true },
  { id: 'a2', text: '  ls.setItem("CHECKOUT_userToken", session.token)', anchored: true },
  { id: 'a3', text: '  ls.setItem("CHECKOUT_userId",    session.id)', anchored: true },
  { id: 'a4', text: '  ls.setItem("CHECKOUT_role",      session.role)', anchored: true },
]

const STAGES: Stage[] = [
  {
    key: 'recorded',
    label: 'recorded',
    caption:
      'A decision is written down and pinned to lines 1451–1454. Everything agrees.',
    lines: [...HEAD, ...ANCHORED, ...TAIL],
    start: 1450,
    confidence: 1,
    state: 'anchored · exact range',
    tone: 'moss',
  },
  {
    key: 'moved',
    label: 'the code moves',
    caption:
      'Someone adds two lines above. Every line number in the record is now wrong — but the content still hashes the same, so the anchor follows it down.',
    lines: [
      { id: 'n1', text: 'const ls = window.localStorage' },
      { id: 'n2', text: '' },
      ...HEAD,
      ...ANCHORED,
      ...TAIL,
    ],
    start: 1450,
    confidence: 0.91,
    state: 'anchored · content hash',
    tone: 'moss',
  },
  {
    key: 'rewritten',
    label: 'the code is rewritten',
    caption:
      'The block is refactored away. Nothing hashes, no AST path matches. The record is surfaced as orphaned — loudly — instead of quietly pointing at whatever now sits on line 1453.',
    lines: [
      { id: 'n1', text: 'const ls = window.localStorage' },
      { id: 'n2', text: '' },
      ...HEAD,
      { id: 'r1', text: '  persistNamespaced(session)', anchored: true },
      ...TAIL,
    ],
    start: 1450,
    confidence: 0.24,
    state: 'orphaned · needs a human',
    tone: 'oxide',
  },
]

const DWELL = 4200

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
  const toneText =
    stage.tone === 'oxide' ? 'text-oxide' : stage.tone === 'honey' ? 'text-honey' : 'text-moss'
  const toneBg =
    stage.tone === 'oxide' ? 'bg-oxide' : stage.tone === 'honey' ? 'bg-honey' : 'bg-moss'

  return (
    <div
      ref={ref}
      className="lit grid overflow-hidden rounded-xl border border-white/10 bg-terminal lg:grid-cols-[minmax(0,1fr)_21rem]"
    >
      {/* the file */}
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5 border-b border-white/[0.07] px-4 py-2.5">
          <span className="font-mono text-[11px] text-white/35">
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
                      ? stage.tone === 'oxide'
                        ? 'border-oxide bg-oxide/[0.08]'
                        : 'border-moss bg-moss/[0.07]'
                      : 'border-transparent',
                  )}
                >
                  <span className="w-14 shrink-0 pl-2 text-right text-white/25 tabular-nums">
                    {stage.start + idx}
                  </span>
                  <span className={cn(l.anchored ? 'text-foreground/90' : 'text-white/45')}>
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
            <span className="font-mono text-[11px] text-white/35">record [b5]</span>
            <span className={cn('font-mono text-[15px] tabular-nums', toneText)}>
              {stage.confidence.toFixed(2)}
            </span>
          </div>
          <div className="mt-2 h-1 overflow-hidden rounded-full bg-white/10">
            <motion.div
              className={cn('h-full rounded-full', toneBg)}
              animate={{ width: `${stage.confidence * 100}%` }}
              transition={{ duration: 0.7, ease: [0.16, 0.8, 0.28, 1] }}
            />
          </div>
          <p className="mt-1.5 font-mono text-[10.5px] tracking-wide text-white/30">
            confidence
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
                  : 'text-white/35 hover:bg-white/[0.04] hover:text-white/65',
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
            className="text-[13px] leading-[1.65] text-white/55"
          >
            {stage.caption}
          </motion.p>
        </AnimatePresence>
      </div>
    </div>
  )
}
