import type { MouseEvent } from 'react'

import { Reveal } from '@/components/Reveal'
import { Header, Footer, REPO, SHELL } from '@/components/Chrome'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { FAQ } from '@/content/faq'
import { cn } from '@/lib/utils'

/* Its own page, because this is a different job from the landing page. The
   landing argues; this answers. Sixty-one rows sitting under the falsification
   panel smothered the one section doing the persuading, and a reader who arrives
   with a specific question should not have to scroll past the whole argument to
   reach it.
 *
 * No strata rail here on purpose. The rail means depth-is-time and belongs to
 * the core-sample conceit; reference material is not a layer of anything. The
 * cluster index is its analogue: same position, same monospace, different claim. */

const COUNT = FAQ.reduce((n, c) => n + c.questions.length, 0)

/** Same rule as the landing page: scroll, never write a hash into the URL. */
const jump = (id: string) => (e: MouseEvent<HTMLAnchorElement>) => {
  e.preventDefault()
  document.getElementById(id)?.scrollIntoView()
}

export default function FaqPage() {
  return (
    <div className="grain relative min-h-screen">
      <Header />

      <main>
        <section className={cn(SHELL, 'pt-32 pb-14 sm:pt-40 sm:pb-20')}>
          <Reveal>
            <p className="font-mono text-[11px] tracking-[0.2em] text-dim uppercase">
              questions · {COUNT} answers
            </p>
            <h1 className="mt-3 max-w-[19ch] text-[clamp(2.1rem,4.6vw,3.4rem)] leading-[1.08]">
              Everything else, answered
            </h1>
            <p className="mt-6 max-w-[54ch] text-[15.5px] leading-[1.7] text-muted-foreground">
              Where the answer is that something is not built, not measured, or not settled,
              it says so. A tool that argues for writing down the reasons behind code should
              not be vague about its own.
            </p>
          </Reveal>
        </section>

        <div className="border-t border-white/[0.07]">
          <div className={cn(SHELL, 'py-14 sm:py-20')}>
            <div className="grid gap-x-16 gap-y-14 lg:grid-cols-[13rem_minmax(0,1fr)] lg:items-start">
              {/* The index. Sticky on large screens, an ordinary list below —
                  position:sticky inside a short viewport just pins a block over
                  the content it is meant to index. */}
              <nav className="lg:sticky lg:top-24" aria-label="Question groups">
                <p className="font-mono text-[11px] tracking-[0.2em] text-dim uppercase">
                  contents
                </p>
                <ul className="mt-4 space-y-2.5">
                  {FAQ.map((cluster) => (
                    <li key={cluster.id}>
                      <a
                        href={`#${cluster.id}`}
                        onClick={jump(cluster.id)}
                        className="text-[13.5px] leading-[1.5] text-muted-foreground transition-colors hover:text-ochre"
                      >
                        {cluster.label}
                      </a>
                    </li>
                  ))}
                </ul>
              </nav>

              <div className="min-w-0 space-y-16">
                {FAQ.map((cluster, i) => (
                  <section key={cluster.id} id={cluster.id} className="scroll-mt-24">
                    <Reveal>
                      <div className="flex items-baseline gap-4">
                        <span className="font-mono text-[11px] tracking-[0.18em] text-ochre">
                          {String(i + 1).padStart(2, '0')}
                        </span>
                        <h2 className="text-[clamp(1.3rem,2vw,1.6rem)] leading-[1.2]">
                          {cluster.label}
                        </h2>
                      </div>
                      <Accordion
                        type="multiple"
                        /* Reference material, so several answers stay open at once.
                           The first cluster opens on its first question — one row
                           showing what an answer looks like, rather than a wall of
                           closed headings the reader has to guess at. */
                        defaultValue={i === 0 ? [`${cluster.id}-0`] : undefined}
                        className="mt-5 border-t border-white/[0.07]"
                      >
                        {cluster.questions.map((item, j) => (
                          <AccordionItem key={item.q} value={`${cluster.id}-${j}`}>
                            <AccordionTrigger>{item.q}</AccordionTrigger>
                            <AccordionContent>{item.a}</AccordionContent>
                          </AccordionItem>
                        ))}
                      </Accordion>
                    </Reveal>
                  </section>
                ))}

                <Reveal>
                  <p className="max-w-[56ch] border-t border-white/[0.07] pt-7 text-[14.5px] leading-[1.7] text-muted-foreground">
                    Something not answered here, or answered wrongly?{' '}
                    <a
                      href={REPO}
                      className="text-ochre underline-offset-4 transition-colors hover:underline"
                    >
                      Open an issue
                    </a>{' '}
                    — a question this page cannot answer is a gap in the argument, not a
                    support request.
                  </p>
                </Reveal>
              </div>
            </div>
          </div>
        </div>
      </main>

      <Footer />
    </div>
  )
}
