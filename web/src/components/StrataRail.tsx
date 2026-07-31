import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { motion, useReducedMotion, useScroll, useSpring } from 'motion/react'
import { cn } from '@/lib/utils'

export type Stratum = { id: string; label: string }

/* The page's only navigation, and the thing it should be remembered by.
 *
 * A core sample read top to bottom: every band's height is its section's
 * true share of the document, so the rail is a depth gauge and a table of
 * contents at the same time. Nothing here is decorative — band height
 * encodes length, position encodes order, and the fill encodes how far
 * down you have read. */
export function StrataRail({
  strata,
  onJump,
}: {
  strata: Stratum[]
  onJump: (id: string) => void
}) {
  const [heights, setHeights] = useState<number[]>([])
  const [active, setActive] = useState(0)
  const reduced = useReducedMotion()
  const railRef = useRef<HTMLElement>(null)

  const { scrollYProgress } = useScroll()
  const fill = useSpring(scrollYProgress, { stiffness: 140, damping: 32, mass: 0.4 })

  // Real section heights, remeasured whenever the document reflows.
  useLayoutEffect(() => {
    const measure = () =>
      setHeights(strata.map((s) => document.getElementById(s.id)?.offsetHeight ?? 0))
    measure()
    const ro = new ResizeObserver(measure)
    ro.observe(document.body)
    return () => ro.disconnect()
  }, [strata])

  // Active layer = the deepest one whose top has passed a third of the viewport.
  useEffect(() => {
    const onScroll = () => {
      const line = window.scrollY + window.innerHeight * 0.33
      let i = 0
      strata.forEach((s, idx) => {
        const el = document.getElementById(s.id)
        if (el && el.offsetTop <= line) i = idx
      })
      setActive(i)
    }
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [strata])

  return (
    <nav
      ref={railRef}
      aria-label="Sections"
      className="fixed inset-y-0 left-0 z-40 hidden w-14 border-r border-white/[0.07] bg-basin/70 backdrop-blur-sm lg:block"
    >
      {/* deposition: the core fills in top to bottom on load */}
      <motion.div
        className="flex h-full flex-col"
        initial={reduced ? false : { clipPath: 'inset(0 0 100% 0)' }}
        animate={{ clipPath: 'inset(0 0 0% 0)' }}
        transition={{ duration: 0.9, ease: [0.16, 0.8, 0.28, 1], delay: 0.1 }}
      >
        {strata.map((s, i) => {
          const passed = i < active
          const isActive = i === active
          return (
            <button
              key={s.id}
              onClick={() => onJump(s.id)}
              style={{
                flexGrow: heights[i] || 1,
                flexBasis: 0,
                // varying density, the way real bands differ
                backgroundColor: `oklch(0.858 0.018 78 / ${0.014 + (i % 3) * 0.011})`,
              }}
              className={cn(
                'group relative flex min-h-9 items-center justify-center border-b border-white/[0.05] transition-colors',
                isActive && 'bg-ochre/[0.10]!',
              )}
            >
              <span
                className={cn(
                  'font-mono text-[10px] tabular-nums transition-colors',
                  isActive
                    ? 'text-ochre'
                    : passed
                      ? 'text-silt/40 group-hover:text-silt/70'
                      : 'text-dim/60 group-hover:text-silt/70',
                )}
              >
                {String(i).padStart(2, '0')}
              </span>

              {/* label flies out on hover rather than sitting rotated and unreadable */}
              <span className="pointer-events-none absolute left-full z-10 ml-3 rounded-sm border border-white/10 bg-sediment px-2 py-1 font-mono text-[10.5px] whitespace-nowrap text-silt opacity-0 shadow-lg transition-opacity group-hover:opacity-100">
                {s.label}
              </span>
            </button>
          )
        })}
      </motion.div>

      {/* how far down the core you have read */}
      <motion.div
        style={{ scaleY: fill }}
        className="absolute inset-y-0 right-0 w-px origin-top bg-ochre"
      />
    </nav>
  )
}
