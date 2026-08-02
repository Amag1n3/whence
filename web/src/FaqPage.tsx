import { Reveal } from '@/components/Reveal'
import { DocPage, DocSection, type DocSectionMeta } from '@/components/DocPage'
import { REPO } from '@/components/Chrome'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { FAQ } from '@/content/faq'

/* Its own page, because this is a different job from the landing page. The
   landing argues; this answers. Sixty-one rows sitting under the falsification
   panel smothered the one section doing the persuading, and a reader who arrives
   with a specific question should not have to scroll past the whole argument to
   reach it.
 *
 * The hero, the sticky index and the numbered sections moved into DocPage when
 * /install and /docs needed the same layout. This file kept only what is
 * actually particular to it: accordions. */

const COUNT = FAQ.reduce((n, c) => n + c.questions.length, 0)
const SECTIONS: DocSectionMeta[] = FAQ.map((c) => ({ id: c.id, label: c.label }))

export default function FaqPage() {
  return (
    <DocPage
      current="faq"
      eyebrow={`questions · ${COUNT} answers`}
      title="Everything else, answered"
      lede="Where the answer is that something is not built, not measured, or not settled, it says so. A tool that argues for writing down the reasons behind code should not be vague about its own."
      sections={SECTIONS}
    >
      {FAQ.map((cluster, i) => (
        <DocSection key={cluster.id} id={cluster.id} n={i + 1} title={cluster.label}>
          <Accordion
            type="multiple"
            /* Reference material, so several answers stay open at once. The
               first cluster opens on its first question — one row showing what
               an answer looks like, rather than a wall of closed headings the
               reader has to guess at. */
            defaultValue={i === 0 ? [`${cluster.id}-0`] : undefined}
            className="border-t border-white/[0.07]"
          >
            {cluster.questions.map((item, j) => (
              <AccordionItem key={item.q} value={`${cluster.id}-${j}`}>
                <AccordionTrigger>{item.q}</AccordionTrigger>
                <AccordionContent>{item.a}</AccordionContent>
              </AccordionItem>
            ))}
          </Accordion>
        </DocSection>
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
          — a question this page cannot answer is a gap in the argument, not a support
          request.
        </p>
      </Reveal>
    </DocPage>
  )
}
