import type { MouseEvent, ReactNode } from 'react'

import { Reveal } from '@/components/Reveal'
import { Header, Footer, SHELL } from '@/components/Chrome'
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

export function DocPage({
  current,
  eyebrow,
  title,
  lede,
  sections,
  children,
}: {
  current: 'install' | 'docs' | 'faq'
  eyebrow: string
  title: string
  lede: ReactNode
  sections: DocSectionMeta[]
  children: ReactNode
}) {
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
                <ul className="mt-4 space-y-2.5">
                  {sections.map((s) => (
                    <li key={s.id}>
                      <a
                        href={`#${s.id}`}
                        onClick={jump(s.id)}
                        className="text-[13.5px] leading-[1.5] text-muted-foreground transition-colors hover:text-ochre"
                      >
                        {s.label}
                      </a>
                    </li>
                  ))}
                </ul>
              </nav>

              <div className="min-w-0 space-y-16">{children}</div>
            </div>
          </div>
        </div>
      </main>

      <Footer />
    </div>
  )
}

/** One numbered section. The index number is ochre and monospace for the same
 *  reason every number on this site is: monospace here is measurement. */
export function DocSection({
  id,
  n,
  title,
  children,
}: {
  id: string
  n: number
  title: string
  children: ReactNode
}) {
  return (
    <section id={id} className="scroll-mt-24">
      <Reveal>
        <div className="flex items-baseline gap-4">
          <span className="font-mono text-[11px] tracking-[0.18em] text-ochre">
            {String(n).padStart(2, '0')}
          </span>
          <h2 className="text-[clamp(1.3rem,2vw,1.6rem)] leading-[1.2]">{title}</h2>
        </div>
        <div className="mt-5 space-y-5">{children}</div>
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
