import { useEffect, useState } from 'react'
import type { MouseEvent, ReactNode } from 'react'
import { motion, useReducedMotion } from 'motion/react'
import { Plus } from 'lucide-react'

import { Reveal } from '@/components/Reveal'
import { Header, Footer, SHELL, type PageKey } from '@/components/Chrome'
import { Toaster } from '@/components/ui/sonner'
import { cn } from '@/lib/utils'

/* The shell every reference page shares: hero, a sticky index, numbered
   sections. Extracted when /install and /docs arrived — FaqPage had already
   built this layout once, and three copies of it would have drifted the way
   the header did before Chrome existed.
 *
 * No strata rail and no Layer, deliberately, and the reasoning is FaqPage's:
 * the rail means depth-is-time and belongs to the core-sample conceit on the
 * landing page. Reference material is not a layer of anything. The contents
 * index is its analogue — same position, same monospace, different claim. */

export type DocSectionMeta = { id: string; label: string }

/** Same rule as every other page: scroll, never write a hash into the URL. */
const jump = (id: string) => (e: MouseEvent<HTMLAnchorElement>) => {
  e.preventDefault()
  document.getElementById(id)?.scrollIntoView()
}

/** Which section is being read: the deepest one whose top has crossed a third
 *  of the viewport.
 *
 *  Not IntersectionObserver, which answers "what is on screen" — on a page
 *  with short sections that is three or four of them at once, and the last
 *  section of a document never wins because the page stops scrolling before
 *  it reaches the middle. "Deepest one already passed" has neither problem
 *  and needs no thresholds tuned. Same rule the strata rail used.
 *
 *  Reads getBoundingClientRect rather than offsetTop so it does not depend on
 *  which ancestor happens to be positioned. */
function useActiveSection(sections: DocSectionMeta[]) {
  const [active, setActive] = useState(sections[0]?.id ?? '')

  useEffect(() => {
    const measure = () => {
      const line = window.innerHeight * 0.33
      let current = sections[0]?.id ?? ''
      sections.forEach((s) => {
        const el = document.getElementById(s.id)
        if (el && el.getBoundingClientRect().top <= line) current = s.id
      })
      // Same value short-circuits in React, so this does not re-render per tick.
      setActive(current)
    }

    /* Coalesced to one measurement per frame. getBoundingClientRect forces
       layout, and the unthrottled version ran it once per section per scroll
       event — sixty-one of them on /faq, several times a frame. That is the
       other half of what felt choppy, and no amount of easing on the marker
       would have hidden it. */
    let frame = 0
    const onScroll = () => {
      if (frame) return
      frame = window.requestAnimationFrame(() => {
        frame = 0
        measure()
      })
    }

    measure()
    window.addEventListener('scroll', onScroll, { passive: true })
    window.addEventListener('resize', onScroll)
    return () => {
      if (frame) window.cancelAnimationFrame(frame)
      window.removeEventListener('scroll', onScroll)
      window.removeEventListener('resize', onScroll)
    }
  }, [sections])

  return active
}

export function DocPage({
  current,
  eyebrow,
  title,
  lede,
  sections,
  children,
}: {
  current: PageKey
  eyebrow: string
  title: string
  lede: ReactNode
  sections: DocSectionMeta[]
  children: ReactNode
}) {
  const active = useActiveSection(sections)
  const reduced = useReducedMotion()

  return (
    <div className="grain relative min-h-screen">
      <Header current={current} />

      <main>
        <section className={cn(SHELL, 'pt-32 pb-14 sm:pt-40 sm:pb-20')}>
          <Reveal>
            <p className="font-mono text-[11px] tracking-[0.2em] text-dim uppercase">
              {eyebrow}
            </p>
            <h1 className="mt-3 max-w-[19ch] text-[clamp(2.1rem,4.6vw,3.4rem)] leading-[1.08]">
              {title}
            </h1>
            <p className="mt-6 max-w-[54ch] text-[15.5px] leading-[1.7] text-muted-foreground">
              {lede}
            </p>
          </Reveal>
        </section>

        <div className="border-t border-white/[0.07]">
          <div className={cn(SHELL, 'py-14 sm:py-20')}>
            <div className="grid gap-x-16 gap-y-14 lg:grid-cols-[13rem_minmax(0,1fr)] lg:items-start">
              {/* Sticky on large screens, an ordinary list below — position:sticky
                  inside a short viewport just pins a block over the content it is
                  meant to index. */}
              <nav className="lg:sticky lg:top-24" aria-label="Contents">
                <p className="font-mono text-[11px] tracking-[0.2em] text-dim uppercase">
                  contents
                </p>
                {/* One hairline running the length of the index, with the
                    active item's own 2px segment sitting on top of it at
                    -ml-px. The rail is the position indicator: you can see
                    how far down the document you are without reading a
                    single label. Square ends, no dot — the marker is a
                    measurement tick, same as every other number here. */}
                <ul className="mt-4 space-y-0.5 border-l border-white/10">
                  {sections.map((s) => {
                    const isActive = active === s.id
                    return (
                      <li key={s.id} className="relative">
                        {/* One marker for the whole list, not a border that
                            flips on each item. A border-colour transition
                            cross-fades in place — the bar blinks out at the
                            old row and blinks in at the new one, which is
                            what read as choppy. Sharing a layoutId makes
                            motion treat it as the same element moving, so it
                            travels the distance instead. */}
                        {isActive && (
                          <motion.span
                            layoutId="doc-index-marker"
                            className="absolute inset-y-0 -left-px w-0.5 bg-ochre"
                            transition={
                              reduced
                                ? { duration: 0 }
                                : { type: 'spring', stiffness: 420, damping: 34, mass: 0.6 }
                            }
                          />
                        )}
                        <a
                          href={`#${s.id}`}
                          onClick={jump(s.id)}
                          aria-current={isActive ? 'location' : undefined}
                          className={cn(
                            'flex min-h-6 items-center py-1.5 pl-4 text-[13.5px] leading-[1.5] transition-colors duration-300',
                            isActive
                              ? 'text-silt'
                              : 'text-muted-foreground hover:text-silt',
                          )}
                        >
                          {s.label}
                        </a>
                      </li>
                    )
                  })}
                </ul>
              </nav>

              <div className="min-w-0 space-y-16">{children}</div>
            </div>
          </div>
        </div>
      </main>

      <Footer />

      {/* Mounted here rather than per page: every page that renders a
          copyable <Code> block goes through DocPage, and the landing and
          contact pages have none. One Toaster, no duplicates. */}
      <Toaster position="top-right" />
    </div>
  )
}

/** One numbered section. The index number is monospace for the same reason
 *  every number on this site is: monospace here is measurement.
 *
 *  `collapsible` folds the body behind the heading, for sections that are a
 *  branch rather than a step — the reader who is not gating CI should not
 *  scroll through how to. Native <details>, not Radix: it keeps its content
 *  in the DOM, so find-in-page still reaches inside, and Chromium expands it
 *  automatically on a match. Radix unmounts closed content and would have
 *  made three sections of /install invisible to browser search. */
export function DocSection({
  id,
  n,
  title,
  hint,
  collapsible = false,
  children,
}: {
  id: string
  n: number
  title: string
  /** Says WHEN this applies. Only meaningful on a collapsible section. */
  hint?: string
  collapsible?: boolean
  children: ReactNode
}) {
  const head = (
    <>
      <span className="font-mono text-[11px] tracking-[0.18em] text-ochre">
        {String(n).padStart(2, '0')}
      </span>
      <h2 className="text-[clamp(1.3rem,2vw,1.6rem)] leading-[1.2]">{title}</h2>
    </>
  )

  if (!collapsible) {
    return (
      <section id={id} className="scroll-mt-24">
        <Reveal>
          <div className="flex items-baseline gap-4">{head}</div>
          <div className="mt-5 space-y-5">{children}</div>
        </Reveal>
      </section>
    )
  }

  return (
    <section id={id} className="scroll-mt-24">
      <Reveal>
        {/* Wears the accordion's clothes — same rotating plus marker, same
            hover lift — but stays a native <details>. Radix wraps its trigger
            in its own heading element, which would bury the numbered <h2>
            these sections are anchored and scroll-spied by. Matching the
            accordion visually costs nothing; matching it structurally would
            cost the page its heading outline. */}
        <details className="group">
          <summary className="-mx-4 flex cursor-pointer list-none items-baseline gap-4 rounded-none px-4 py-2 transition-colors hover:bg-white/[0.035] [&::-webkit-details-marker]:hidden">
            {head}
            <span className="ml-auto flex shrink-0 items-center gap-2 self-center">
              {hint && (
                <span className="hidden font-mono text-[11px] tracking-wide text-dim sm:block">
                  {hint}
                </span>
              )}
              <Plus className="size-4 text-dim transition-transform duration-200 group-open:rotate-45" />
            </span>
          </summary>
          <div className="mt-5 space-y-5">{children}</div>
        </details>
      </Reveal>
    </section>
  )
}

/** Body copy, capped where the design system caps it. Saves every page
 *  repeating the same four utility classes and getting one of them wrong. */
export function P({ children }: { children: ReactNode }) {
  return (
    <p className="max-w-[56ch] text-[14.5px] leading-[1.68] text-muted-foreground">
      {children}
    </p>
  )
}
