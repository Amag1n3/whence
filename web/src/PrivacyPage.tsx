import { Reveal } from '@/components/Reveal'
import { Header, Footer, REPO, SHELL, EMAIL } from '@/components/Chrome'
import { cn } from '@/lib/utils'

/* The shortest privacy page that is still true.
 *
 * There is no data to have a policy about: the site is static files on
 * Cloudflare Pages and the CLI's read path makes no network calls at all, so
 * the honest document is a list of the things that do not exist — no
 * accounts, no cookies, no analytics — rather than a legal notice describing
 * how data is handled. Rows, not prose, because "what do you collect" is a
 * lookup question.
 *
 * The claims here are structural, not aspirational: if anyone ever adds an
 * analytics script or a network call to the read path, this page becomes a
 * lie and must change in the same commit. */

const ROWS: { label: string; truth: string }[] = [
  {
    label: 'This site',
    truth:
      'Static files on Cloudflare Pages. No accounts, no cookies, no analytics, no tracking pixels, no A/B tests. Cloudflare sees the request your browser makes to fetch a page, the way any web server does; nothing here reads or stores it.',
  },
  {
    label: 'The CLI',
    truth:
      'The read path — lookup, hook, and check — is deterministic and offline. It makes no network calls and no model calls. Records live in .whence/ inside the repository they describe and travel only if you commit them, which is the point.',
  },
  {
    label: 'The only uploads',
    truth:
      'Two commands touch the network by design: capture transcribes a session you point it at, and the GitHub Action runs in your own CI. Both act on data you already have, in infrastructure you already run.',
  },
]

export default function PrivacyPage() {
  return (
    <div className="grain relative min-h-screen">
      <Header current="privacy" />

      <main>
        <section className={cn(SHELL, 'pt-32 pb-14 sm:pt-40 sm:pb-20')}>
          <Reveal>
            <p className="font-mono text-[11px] tracking-[0.2em] text-dim uppercase">
              privacy
            </p>
            <h1 className="mt-3 max-w-[19ch] text-[clamp(2.1rem,4.6vw,3.4rem)] leading-[1.08]">
              There is nothing to collect
            </h1>
            <p className="mt-6 max-w-[54ch] text-[15.5px] leading-[1.7] text-muted-foreground">
              No accounts, no cookies, no analytics — not as a promise but as a
              description of the architecture. A static site and an offline CLI
              have nothing to gather.
            </p>
          </Reveal>
        </section>

        <div className="border-t border-white/[0.07]">
          <div className={cn(SHELL, 'py-14 sm:py-20')}>
            <div className="border-t border-white/[0.09]">
              {ROWS.map((r, i) => (
                <Reveal key={r.label} delay={i * 0.06}>
                  <div className="-mx-4 grid gap-x-14 gap-y-3 border-b border-white/[0.09] px-4 py-7 lg:grid-cols-[12rem_minmax(0,1fr)]">
                    <span className="font-display text-[18px] font-semibold">
                      {r.label}
                    </span>
                    <p className="max-w-[62ch] text-[14.5px] leading-[1.68] text-muted-foreground">
                      {r.truth}
                    </p>
                  </div>
                </Reveal>
              ))}
            </div>

            <Reveal>
              <p className="mt-10 max-w-[56ch] text-[14.5px] leading-[1.7] text-muted-foreground">
                If that ever changes, this page changes in the same commit.
                Questions:{' '}
                <a
                  href={`mailto:${EMAIL}`}
                  className="text-ochre underline-offset-4 transition-colors hover:underline"
                >
                  {EMAIL}
                </a>{' '}
                or the{' '}
                <a
                  href={`${REPO}/issues`}
                  className="text-ochre underline-offset-4 transition-colors hover:underline"
                >
                  issue tracker
                </a>
                .
              </p>
            </Reveal>
          </div>
        </div>
      </main>

      <Footer />
    </div>
  )
}
