import { ArrowUpRight } from 'lucide-react'

import { Reveal } from '@/components/Reveal'
import { Terminal } from '@/components/Terminal'
import { Header, Footer, REPO, SHELL } from '@/components/Chrome'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { cn } from '@/lib/utils'

/* The landing page is an introduction and a set of doors, nothing else. It
   used to carry the entire argument — eleven layers, two interactive demos and
   the falsification panel — with the links to /install, /docs and /faq buried
   at depths 07 and 10. A reader who wanted the install command had to scroll
   past the whole case for the project to find it. The argument moved to /why,
   where the people who want it can go looking. */

const DOORS: {
  href: string
  label: string
  blurb: string
  external?: boolean
  /** Spans two columns, so the five cards fill their rows exactly. */
  wide?: boolean
}[] = [
  {
    href: '/install',
    label: 'Install guide',
    blurb:
      'Two commands, then the plugin that puts records in front of your agent. zsh and ksh need one more line — both ship a whence builtin that beats $PATH.',
  },
  {
    href: '/docs',
    label: 'Commands',
    blurb:
      'Every command, with flags, exit codes and what each one prints. whence check is the CI gate.',
  },
  {
    href: '/why',
    label: 'Why it exists',
    blurb:
      'The problem, how a record survives the code moving underneath it, what runs today, and the number that would kill the project.',
  },
  {
    href: '/faq',
    label: 'Questions',
    blurb:
      'Sixty-one answers — what capture does and does not do, what this reads off your machine, what happens when a record goes stale.',
  },
  {
    href: REPO,
    label: 'GitHub',
    blurb: 'Source, issues, releases. Built in the open, Go, MIT.',
    external: true,
    wide: true,
  },
]

export default function App() {
  return (
    <div className="grain relative min-h-screen">
      <Header />

      <main>
        {/* ------------------------------------------------------- surface */}
        <section className={cn(SHELL, 'pt-32 pb-20 sm:pt-40 sm:pb-28')}>
          <div className="grid gap-x-16 gap-y-12 lg:grid-cols-[minmax(0,27rem)_minmax(0,1fr)] lg:items-center">
            <div>
              <Reveal>
                <h1 className="max-w-[15ch] text-[clamp(2.4rem,5vw,3.9rem)] leading-[1.04]">
                  Git remembers what changed.{' '}
                  <span className="font-mono text-[0.82em] font-medium text-ochre">
                    whence
                  </span>{' '}
                  remembers the reason.
                </h1>
              </Reveal>

              <Reveal delay={0.12}>
                <p className="mt-7 max-w-[46ch] text-[15.5px] leading-[1.7] text-muted-foreground sm:text-[19px] sm:leading-[1.6]">
                  Your coding agent works out why the code has to be like this, then
                  throws it away when the session ends. This keeps it, pinned to the
                  lines it was about, and commits it alongside them — so the next agent
                  gets it before it edits, on any machine that cloned the repo.
                </p>
              </Reveal>

              {/* datum: the reference horizon. Kept from the old hero because a
                  landing page that does not say what is unbuilt is a landing
                  page that is lying by omission. */}
              <Reveal delay={0.18}>
                <div className="mt-7 flex items-center gap-3.5">
                  <span className="h-px w-8 shrink-0 bg-ochre" />
                  <span className="font-mono text-[11px] leading-[1.6] tracking-[0.14em] text-dim uppercase">
                    surfacing, anchoring and the CI gate run · capture does not
                  </span>
                </div>
              </Reveal>

              <Reveal delay={0.24}>
                <div className="mt-9 flex flex-wrap items-center gap-x-6 gap-y-3">
                  {/* Stock registry button, no overrides. It was a rounded-full
                      pill in ochre, which is the shape and the colour every
                      generated landing page reaches for. */}
                  <Button asChild size="lg">
                    <a href="/install">Install in five minutes</a>
                  </Button>
                  <a
                    href="/why"
                    className="inline-flex min-h-6 items-center gap-1 font-mono text-[12.5px] text-muted-foreground transition-colors hover:text-silt"
                  >
                    why it exists <ArrowUpRight className="size-3.5" />
                  </a>
                </div>
              </Reveal>
            </div>

            <Reveal delay={0.3}>
              <Terminal aside={false} />
            </Reveal>
          </div>
        </section>

        {/* --------------------------------------------------------- doors */}
        {/* Real gaps between real cards. This grid was briefly a 1px-gap
            mosaic — cards on a lighter container, the gap being the container
            showing through. With five cards in a three-column grid the sixth
            cell had nothing on top of it, so the container showed through a
            whole card-sized rectangle and the section looked broken. Gapped
            cards cannot do that: an empty cell is just page. The last card
            spans two columns so the row fills exactly at every breakpoint. */}
        <section id="doors" className="border-t border-white/[0.07]">
          <div className={cn(SHELL, 'py-16 sm:py-24')}>
            <Reveal>
              <h2 className="font-mono text-[11px] font-normal tracking-[0.2em] text-dim uppercase">
                where to go
              </h2>
            </Reveal>

            <div className="mt-9 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {DOORS.map((d, i) => (
                <Reveal
                  key={d.href}
                  delay={i * 0.05}
                  className={cn(d.wide && 'sm:col-span-2')}
                >
                  <a href={d.href} className="group block h-full">
                    <Card className="h-full gap-3 transition-colors group-hover:border-white/20 group-hover:bg-secondary/40">
                      <CardHeader>
                        <CardTitle className="flex items-center gap-1.5 font-display text-[18px]">
                          {d.label}
                          {d.external && (
                            <ArrowUpRight className="size-4 text-dim transition-colors group-hover:text-silt" />
                          )}
                        </CardTitle>
                      </CardHeader>
                      <CardContent>
                        <p className="max-w-[46ch] text-[14.5px] leading-[1.6] text-muted-foreground">
                          {d.blurb}
                        </p>
                      </CardContent>
                    </Card>
                  </a>
                </Reveal>
              ))}
            </div>
          </div>
        </section>
      </main>

      <Footer />
    </div>
  )
}
