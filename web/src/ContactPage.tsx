import { ArrowUpRight } from 'lucide-react'

import { Reveal } from '@/components/Reveal'
import { Header, Footer, REPO, SHELL, EMAIL } from '@/components/Chrome'
import { cn } from '@/lib/utils'

/* A routing table, not a contact form.
 *
 * One person maintains this, so the useful thing a contact page can do is say
 * which door a given message should go through — a bug in an issue is public,
 * searchable and gets fixed; the same bug in an email is a private todo that
 * rots. The rows are ordered by where most messages should actually go, which
 * is deliberately not the email.
 *
 * No form. A form implies a queue with an SLA behind it, and there isn't one. */

type Channel = {
  label: string
  /** What this door is for. The row's whole job. */
  use: string
  href: string
  /** Shown in mono — the address itself is data. */
  handle: string
  external?: boolean
}

const CHANNELS: Channel[] = [
  {
    label: 'Issues',
    use: 'Bugs, wrong behaviour, a record that anchored somewhere it should not have. Also feature requests. Public and searchable, so the next person with the same problem finds the answer instead of sending it again.',
    href: `${REPO}/issues`,
    handle: 'github.com/Amag1n3/whence/issues',
    external: true,
  },
  {
    label: 'Email',
    use: 'Everything that is not a bug — security reports, questions the FAQ did not answer, or anything you would rather not put in a public tracker.',
    href: `mailto:${EMAIL}`,
    handle: EMAIL,
  },
]

/* Secondary by design. The two rows above are where work should go; these are
   a footnote, set inline in mono rather than as more full-width rows, so the
   page cannot be mistaken for a links-in-bio. Order is professional-first.

   The section is guarded on length, so emptying this array removes the block
   rather than leaving dead links behind — a contact page with a 404 on it is
   worse than one without the link at all. */
const SOCIALS: { label: string; href: string }[] = [
  { label: 'LinkedIn', href: 'https://www.linkedin.com/in/amag1n3/' },
  { label: 'X', href: 'https://x.com/TyagiAmogh' },
  { label: 'Instagram', href: 'https://www.instagram.com/tyagi_amogh/' },
]

export default function ContactPage() {
  return (
    <div className="grain relative min-h-screen">
      <Header current="contact" />

      <main>
        <section className={cn(SHELL, 'pt-32 pb-14 sm:pt-40 sm:pb-20')}>
          <Reveal>
            <p className="font-mono text-[11px] tracking-[0.2em] text-dim uppercase">
              contact
            </p>
            <h1 className="mt-3 max-w-[19ch] text-[clamp(2.1rem,4.6vw,3.4rem)] leading-[1.08]">
              Which door to use
            </h1>
            <p className="mt-6 max-w-[54ch] text-[15.5px] leading-[1.7] text-muted-foreground">
              whence is built and maintained by one person. There is no support queue
              behind this page, so the fastest answer is usually the most public one.
            </p>
          </Reveal>
        </section>

        <div className="border-t border-white/[0.07]">
          <div className={cn(SHELL, 'py-14 sm:py-20')}>
            <div className="border-t border-white/[0.09]">
              {CHANNELS.map((c, i) => (
                <Reveal key={c.label} delay={i * 0.06}>
                  <a
                    href={c.href}
                    className="group -mx-4 grid gap-x-14 gap-y-3 border-b border-white/[0.09] px-4 py-7 transition-colors hover:bg-white/[0.035] lg:grid-cols-[12rem_minmax(0,1fr)]"
                  >
                    <div className="flex items-center gap-1.5">
                      {/* The hover used to be group-hover:text-ochre. Once the
                          palette went monochrome, ochre and silt became the
                          same white and this row's hover state was white on
                          white — invisible. Hover is a background lift now,
                          which does not depend on there being a second hue. */}
                      <span className="font-display text-[18px] font-semibold">
                        {c.label}
                      </span>
                      {c.external && (
                        <ArrowUpRight className="size-4 text-dim transition-colors group-hover:text-ochre" />
                      )}
                    </div>
                    <div className="min-w-0">
                      <p className="max-w-[62ch] text-[14.5px] leading-[1.68] text-muted-foreground">
                        {c.use}
                      </p>
                      <p className="mt-3 font-mono text-[12.5px] break-all text-silt/70 transition-colors group-hover:text-ochre">
                        {c.handle}
                      </p>
                    </div>
                  </a>
                </Reveal>
              ))}
            </div>

            {SOCIALS.length > 0 && (
              <Reveal>
                <div className="mt-12">
                  <p className="font-mono text-[11px] tracking-[0.2em] text-dim uppercase">
                    elsewhere
                  </p>
                  <div className="mt-4 flex flex-wrap items-center gap-x-7 gap-y-2">
                    {SOCIALS.map((s) => (
                      <a
                        key={s.label}
                        href={s.href}
                        className="inline-flex min-h-6 items-center gap-1 font-mono text-[12.5px] text-muted-foreground transition-colors hover:text-ochre"
                      >
                        {s.label}
                        <ArrowUpRight className="size-3.5" />
                      </a>
                    ))}
                  </div>
                </div>
              </Reveal>
            )}

            <Reveal>
              <p className="mt-10 max-w-[56ch] text-[14.5px] leading-[1.7] text-muted-foreground">
                Before either: the{' '}
                <a
                  href="/faq"
                  className="text-ochre underline-offset-4 transition-colors hover:underline"
                >
                  questions page
                </a>{' '}
                answers sixty-one of these, including most of what arrives by email.
              </p>
            </Reveal>
          </div>
        </div>
      </main>

      <Footer />
    </div>
  )
}
