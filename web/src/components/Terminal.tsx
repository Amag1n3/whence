import { useEffect, useRef, useState } from 'react'
import { motion, useInView, useReducedMotion } from 'motion/react'
import { cn } from '@/lib/utils'

const CMD = 'why src/auth/session.go:1455'

type Out = { text: string; tone?: 'moss' | 'dim' | 'plain' }

/** Verbatim shape of print1() in main.go. If the CLI's output changes, this
 *  is a lie — which matters more than it looking good. */
const OUTPUT: Out[] = [
  { text: '' },
  { text: '  ● 2026-07-27 · code review, finding B5', tone: 'moss' },
  { text: '    Never write shared session keys from the checkout flow —' },
  { text: '    namespace all three to CHECKOUT_*.' },
  { text: '    "userToken", "userId" and "role" are all read by the admin', tone: 'dim' },
  { text: '    dashboard on the same origin. Writing them here signs a', tone: 'dim' },
  { text: '    staff user out mid-session, and it surfaces as HTTP 200', tone: 'dim' },
  { text: '    with {"message":"API TIMEOUT"} — which looks like a network', tone: 'dim' },
  { text: '    problem and is not one.', tone: 'dim' },
  { text: '    session.go:1450-1465  [b5]', tone: 'dim' },
]

const TYPE_MS = 24
const OUT_STAGGER = 88
const HOLD_MS = 7000

export function Terminal({ className }: { className?: string }) {
  const ref = useRef<HTMLDivElement>(null)
  const inView = useInView(ref, { once: false, margin: '-10% 0px' })
  const reduced = useReducedMotion()

  const [typed, setTyped] = useState(0)
  const [shown, setShown] = useState(0)
  const [run, setRun] = useState(0)

  useEffect(() => {
    if (reduced) {
      setTyped(CMD.length)
      setShown(OUTPUT.length)
      return
    }
    if (!inView) return

    setTyped(0)
    setShown(0)

    const timers: number[] = []
    let i = 0
    const typer = window.setInterval(() => {
      i += 1
      setTyped(i)
      if (i < CMD.length) return

      window.clearInterval(typer)
      OUTPUT.forEach((_, idx) => {
        timers.push(window.setTimeout(() => setShown(idx + 1), 340 + idx * OUT_STAGGER))
      })
      timers.push(
        window.setTimeout(
          () => setRun((r) => r + 1),
          340 + OUTPUT.length * OUT_STAGGER + HOLD_MS,
        ),
      )
    }, TYPE_MS)

    return () => {
      window.clearInterval(typer)
      timers.forEach(window.clearTimeout)
    }
  }, [run, inView, reduced])

  const typing = typed < CMD.length

  return (
    <div
      ref={ref}
      className={cn(
        'lit overflow-hidden rounded-xl border border-white/10 bg-terminal',
        className,
      )}
    >
      <div className="flex items-center gap-3 border-b border-white/[0.07] px-4 py-2.5">
        <span className="font-mono text-[11px] tracking-wide text-white/35">
          ~/dev/storefront
        </span>
        <span className="ml-auto flex items-center gap-1.5 font-mono text-[10.5px] text-white/30">
          <span className="size-1.5 rounded-full bg-moss/80" />
          .whence/records.json
        </span>
      </div>

      <div className="term-scroll overflow-x-auto px-4 py-4 sm:px-5 sm:py-5">
        <pre className="min-w-max font-mono text-[12px] leading-[1.72] sm:text-[13px]">
          <span className="text-moss/70">$ </span>
          <span className="text-foreground/90">{CMD.slice(0, typed)}</span>
          {typing && (
            <span className="ml-px inline-block h-[1.05em] w-[7px] translate-y-[2px] animate-pulse bg-moss/80" />
          )}
          {'\n'}
          {OUTPUT.slice(0, shown).map((l, idx) => (
            <motion.span
              key={`${run}-${idx}`}
              initial={reduced ? false : { opacity: 0, x: -4 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.24, ease: 'easeOut' }}
              className={cn(
                'block',
                l.tone === 'moss' && 'text-moss',
                l.tone === 'dim' && 'text-white/45',
                !l.tone && 'text-foreground/85',
              )}
            >
              {l.text || ' '}
            </motion.span>
          ))}
        </pre>
      </div>
    </div>
  )
}
