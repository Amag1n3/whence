import { motion, useReducedMotion } from 'motion/react'
import type { ReactNode } from 'react'

/** Scroll-in reveal. Once only — a page that re-animates on every scroll-by
 *  is a page that fights the reader. */
export function Reveal({
  children,
  delay = 0,
  className,
}: {
  children: ReactNode
  delay?: number
  className?: string
}) {
  const reduced = useReducedMotion()
  if (reduced) return <div className={className}>{children}</div>

  return (
    <motion.div
      className={className}
      initial={{ opacity: 0, y: 18 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: '-80px' }}
      transition={{ duration: 0.62, delay, ease: [0.16, 0.8, 0.28, 1] }}
    >
      {children}
    </motion.div>
  )
}
