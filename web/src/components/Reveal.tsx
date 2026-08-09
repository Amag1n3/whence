import { useRef } from 'react'
import type { ReactNode } from 'react'

import { useInView } from '@/lib/motion'
import { cn } from '@/lib/utils'

/** Scroll-in reveal. Once only — a page that re-animates on every scroll-by
 *  is a page that fights the reader.
 *
 *  This was motion/react. It is on every page, and a 620ms fade-and-rise does
 *  not need a 124kB animation runtime: IntersectionObserver and a CSS
 *  transition do exactly the same thing. Same distance, same duration, same
 *  curve — `ease-settle` is the token holding the array that was inline here.
 *
 *  Reduced motion is handled globally: index.css clamps every transition to
 *  0.001ms under prefers-reduced-motion, so the element still lands visible,
 *  just instantly. */
export function Reveal({
  children,
  delay = 0,
  className,
}: {
  children: ReactNode
  delay?: number
  className?: string
}) {
  const ref = useRef<HTMLDivElement>(null)
  // Margin matches motion's old viewport setting: fire before the top edge.
  const shown = useInView(ref, { once: true, margin: '-80px' })

  return (
    <div
      ref={ref}
      style={delay ? { transitionDelay: `${delay}s` } : undefined}
      className={cn(
        /* `translate`, not `transform`: Tailwind v4 compiles translate-y-* to
           the standalone translate property, so a transition declared on
           transform animates the fade and jumps the rise. */
        'transition-[opacity,translate] duration-[620ms] ease-settle',
        shown ? 'translate-y-0 opacity-100' : 'translate-y-[18px] opacity-0',
        className,
      )}
    >
      {children}
    </div>
  )
}
