import { useEffect, useState, type RefObject } from 'react'

/* The two things this site actually used motion/react for outside of
   DriftDemo: "is it on screen" and "does this person want animation". Both
   are native, and importing a 119kB animation runtime for them put it on
   every page. */

/** IntersectionObserver as a hook.
 *
 *  `once` disconnects after the first intersection, for reveals that should
 *  not replay on every scroll-by. Leave it false for anything that restarts
 *  when it comes back into view. */
export function useInView(
  ref: RefObject<Element | null>,
  { once = true, margin = '0px' }: { once?: boolean; margin?: string } = {},
) {
  const [inView, setInView] = useState(false)

  useEffect(() => {
    const el = ref.current
    if (!el) return
    const io = new IntersectionObserver(
      ([entry]) => {
        setInView(entry.isIntersecting)
        if (entry.isIntersecting && once) io.disconnect()
      },
      { rootMargin: margin },
    )
    io.observe(el)
    return () => io.disconnect()
  }, [ref, once, margin])

  return inView
}

/** True when the reader has asked for less motion.
 *
 *  Read once on mount rather than subscribed to: this gates whether an
 *  animation runs at all, and someone flipping the OS setting mid-visit does
 *  not need the half-finished typing animation to restart. Purely visual
 *  motion is handled in CSS by index.css, which clamps every transition and
 *  animation under the same query — this is only for the cases where the
 *  animation is JavaScript driving state. */
export function usePrefersReducedMotion() {
  const [reduced, setReduced] = useState(false)
  useEffect(() => {
    setReduced(window.matchMedia('(prefers-reduced-motion: reduce)').matches)
  }, [])
  return reduced
}
