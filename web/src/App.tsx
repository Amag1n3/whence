import { motion, useReducedMotion } from 'motion/react'
import { ArrowUpRight, Check, Minus } from 'lucide-react'
import type { MouseEvent, ReactNode } from 'react'

import { Reveal } from '@/components/Reveal'
import { Terminal } from '@/components/Terminal'
import { DriftDemo } from '@/components/DriftDemo'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'

const REPO = 'https://github.com/Amag1n3/whence'

/* One container, used everywhere. Wide enough to fill a laptop screen;
   every text block inside it is column-constrained so the measure never
   runs past ~70 characters. Width comes from structure, not from letting
   lines stretch. */
const SHELL = 'mx-auto w-full max-w-[1320px] px-6 sm:px-10'

/** Scroll to a section without writing a hash into the address bar. The href
 *  stays on the anchor so it still works without JS and reads correctly to a
 *  screen reader. No `behavior` argument on purpose — that defers to the CSS
 *  `scroll-behavior`, which the reduced-motion block already overrides. */
function jump(id: string) {
  return (e: MouseEvent<HTMLAnchorElement>) => {
    e.preventDefault()
    if (id === 'top') window.scrollTo({ top: 0 })
    else document.getElementById(id)?.scrollIntoView()
  }
}

/* ---------------------------------------------------------------- shell */

function Ambient() {
  const reduced = useReducedMotion()
  return (
    <div aria-hidden className="pointer-events-none fixed inset-0 overflow-hidden">
      <motion.div
        className="absolute -top-[30rem] left-1/2 h-[58rem] w-[58rem] -translate-x-1/2 rounded-full"
        style={{
          background:
            'radial-gradient(circle, oklch(0.782 0.106 146 / 0.13), transparent 62%)',
        }}
        animate={reduced ? undefined : { scale: [1, 1.09, 1], opacity: [0.75, 1, 0.75] }}
        transition={{ duration: 17, repeat: Infinity, ease: 'easeInOut' }}
      />
      <motion.div
        className="absolute top-[62rem] -right-52 h-[40rem] w-[40rem] rounded-full"
        style={{
          background:
            'radial-gradient(circle, oklch(0.833 0.118 77 / 0.085), transparent 65%)',
        }}
        animate={reduced ? undefined : { scale: [1, 1.14, 1], opacity: [0.6, 0.95, 0.6] }}
        transition={{ duration: 23, repeat: Infinity, ease: 'easeInOut', delay: 3 }}
      />
    </div>
  )
}

function Nav() {
  return (
    <header className="fixed inset-x-0 top-0 z-50 border-b border-white/[0.06] bg-background/70 backdrop-blur-xl">
      <nav className={cn(SHELL, 'flex h-14 items-center gap-6')}>
        <a
          href="#top"
          onClick={jump('top')}
          className="font-mono text-[13.5px] font-medium tracking-tight"
        >
          <span className="text-moss">●</span> whence
        </a>
        <div className="ml-auto flex items-center gap-6 font-mono text-[12.5px] text-muted-foreground">
          <a href="#how" onClick={jump('how')} className="transition-colors hover:text-foreground">
            how
          </a>
          <a
            href="#anchoring"
            onClick={jump('anchoring')}
            className="hidden transition-colors hover:text-foreground sm:inline"
          >
            anchoring
          </a>
          <a
            href="#start"
            onClick={jump('start')}
            className="transition-colors hover:text-foreground"
          >
            quickstart
          </a>
          <a
            href={REPO}
            className="flex items-center gap-0.5 transition-colors hover:text-foreground"
          >
            github <ArrowUpRight className="size-3.5" />
          </a>
        </div>
      </nav>
    </header>
  )
}

function Eyebrow({ children }: { children: ReactNode }) {
  return (
    <p className="font-mono text-[11px] tracking-[0.18em] text-moss uppercase">{children}</p>
  )
}

function Title({ children }: { children: ReactNode }) {
  return (
    <h2 className="mt-3 text-[clamp(1.75rem,2.9vw,2.5rem)] leading-[1.12]">{children}</h2>
  )
}

/** Two-column editorial section: heading rail on the left, content on the
 *  right. This is what fills a wide screen without stretching a sentence. */
function Split({
  id,
  eyebrow,
  title,
  aside,
  children,
}: {
  id?: string
  eyebrow: string
  title: string
  aside?: ReactNode
  children: ReactNode
}) {
  return (
    <section id={id} className={cn(SHELL, 'py-20 sm:py-28')}>
      <div className="grid gap-x-20 gap-y-9 lg:grid-cols-[20rem_minmax(0,1fr)]">
        <div className="lg:sticky lg:top-24 lg:self-start">
          <Reveal>
            <Eyebrow>{eyebrow}</Eyebrow>
            <Title>{title}</Title>
            {aside && (
              <div className="mt-5 max-w-[34ch] text-[14.5px] leading-[1.65] text-muted-foreground">
                {aside}
              </div>
            )}
          </Reveal>
        </div>
        <div>{children}</div>
      </div>
    </section>
  )
}

/** Full-bleed section: heading on top, content across the whole shell. For
 *  the two demos, which earn the width. */
function Stack({
  id,
  eyebrow,
  title,
  lede,
  children,
}: {
  id?: string
  eyebrow: string
  title: string
  lede?: ReactNode
  children: ReactNode
}) {
  return (
    <section id={id} className={cn(SHELL, 'py-20 sm:py-28')}>
      <Reveal>
        <div className="flex flex-wrap items-end justify-between gap-x-16 gap-y-5">
          <div>
            <Eyebrow>{eyebrow}</Eyebrow>
            <Title>{title}</Title>
          </div>
          {lede && (
            <p className="max-w-[52ch] text-[15.5px] leading-[1.7] text-muted-foreground">
              {lede}
            </p>
          )}
        </div>
      </Reveal>
      <div className="mt-10">{children}</div>
    </section>
  )
}

function Code({ children, caption }: { children: string; caption?: string }) {
  return (
    <div className="lit overflow-hidden rounded-lg border border-white/10 bg-terminal">
      {caption && (
        <div className="border-b border-white/[0.07] px-4 py-2 font-mono text-[11px] text-white/35">
          {caption}
        </div>
      )}
      <div className="term-scroll overflow-x-auto px-4 py-3.5">
        <pre className="min-w-max font-mono text-[12.5px] leading-[1.75] text-foreground/85">
          {children}
        </pre>
      </div>
    </div>
  )
}

/* ----------------------------------------------------------------- data */

const LEDGER = [
  { on: true, t: 'why <file>:<line>', d: 'Recorded decisions for a file, or for one line.' },
  {
    on: true,
    t: 'The PreToolUse hook',
    d: 'Records reach Claude Code before it edits. Fails open: a broken why costs you nothing but a missing record.',
  },
  {
    on: true,
    t: 'Store resolution by file',
    d: 'Walks up from the edited file the way git finds .git, so a session in one repo still resolves records for a sibling repo.',
  },
  {
    on: false,
    t: 'Capture',
    d: 'Records are hand-written. Pulling signal out of a session is the hard part.',
  },
  {
    on: false,
    t: 'Hybrid anchoring',
    d: 'Content hash, AST path, confidence decay, orphan states.',
  },
  { on: false, t: 'why check', d: 'The CI gate. The exit code is the product.' },
]

const STEPS = [
  {
    t: 'Capture',
    tag: 'not built',
    d: "Subscribe to the agent's hooks and record the decision trail as the session runs. Redaction happens here, before anything is written to disk — once a secret reaches a content-addressed store it may already be replicated.",
  },
  {
    t: 'Anchor',
    tag: 'exact ranges only',
    d: 'Bind each record to a file and a span. Today that span is a hand-written line range.',
  },
  {
    t: 'Surface',
    tag: 'works',
    d: 'From your terminal, and into a coding agent’s context through a PreToolUse hook before it edits. Phase 2 adds CI.',
  },
]

const NOTS: [string, string][] = [
  ['Not a code reviewer.', 'It has no opinions about code quality.'],
  ['Not an observability platform.', 'No traces, no evals, no token dashboards.'],
  ['Not a knowledge graph.', 'It answers one question about one line.'],
]

const COMMITMENTS: [string, string][] = [
  ['Hashes, paths and ranges — never file contents', 'unless you turn that on, per repo.'],
  [
    'Redaction runs before the write, not before the share',
    'The store is meant to be committed, so a bad capture is public the moment you push. Nothing reaches the store unredacted. The surfacing log — timestamps and absolute local paths — is gitignored on day one.',
  ],
  [
    'Records are data, never directives',
    'They arrive by git pull, so anyone who can land a commit can put text in front of your agent. Records are framed as history, and marked untrusted when they are not yours.',
  ],
  [
    'Attribution stays aggregate',
    'No per-developer AI-authorship leaderboards. This is a developer tool and it will not become a surveillance tool.',
  ],
]

function DefList({ items }: { items: [string, string][] }) {
  return (
    <div className="grid gap-x-14 gap-y-px sm:grid-cols-2">
      {items.map(([t, d], i) => (
        <Reveal key={t} delay={i * 0.05}>
          <div className="border-t border-white/[0.09] py-5">
            <b className="text-[15.5px] font-semibold text-foreground">{t}</b>{' '}
            <span className="text-[15.5px] leading-[1.6] text-muted-foreground">{d}</span>
          </div>
        </Reveal>
      ))}
    </div>
  )
}

/* ----------------------------------------------------------------- page */

export default function App() {
  return (
    <div className="grain relative min-h-screen" id="top">
      <Ambient />
      <Nav />

      <main className="relative">
        {/* ------------------------------------------------------- hero */}
        <section className={cn(SHELL, 'pt-32 pb-16 sm:pt-44 sm:pb-24')}>
          <Reveal>
            <Badge
              variant="outline"
              className="mb-8 gap-2 border-white/12 bg-white/[0.04] py-1.5 pr-3.5 pl-2.5 font-mono text-[11.5px] font-normal"
            >
              <span className="size-1.5 rounded-full bg-moss" />
              <span className="text-muted-foreground">
                <span className="text-foreground">Phase 0 works.</span> Capture is not built.
              </span>
            </Badge>
          </Reveal>

          <div className="grid gap-x-20 gap-y-8 lg:grid-cols-[minmax(0,1fr)_22rem] lg:items-end">
            <Reveal delay={0.06}>
              <h1 className="max-w-[18ch] text-[clamp(2.6rem,6.4vw,4.9rem)] leading-[1.02] font-semibold">
                Git remembers what changed.{' '}
                <span className="font-mono text-[0.85em] font-medium text-moss">why</span>{' '}
                remembers the reason.
              </h1>
            </Reveal>

            <Reveal delay={0.12}>
              <div className="lg:pb-2">
                <p className="max-w-[46ch] text-[17px] leading-[1.65] text-muted-foreground">
                  Your coding agent works out why the code has to be like this, then throws
                  it away when the session ends. This keeps it, pinned to the lines it was
                  about, and hands it back to the next agent before it edits them.
                </p>
                <div className="mt-7 flex flex-wrap gap-3">
                  <Button asChild size="lg" className="rounded-full font-medium">
                    <a href="#start" onClick={jump('start')}>
                      Try it in five minutes
                    </a>
                  </Button>
                  <Button
                    asChild
                    size="lg"
                    variant="outline"
                    className="rounded-full border-white/12 bg-white/[0.03] font-medium hover:bg-white/[0.07]"
                  >
                    <a href={REPO}>
                      Read the source <ArrowUpRight className="size-4" />
                    </a>
                  </Button>
                </div>
              </div>
            </Reveal>
          </div>

          <Reveal delay={0.26} className="mt-14">
            <Terminal />
          </Reveal>
        </section>

        {/* ---------------------------------------------------- problem */}
        <Split eyebrow="the problem" title="The mess was load-bearing">
          <div className="grid gap-x-14 gap-y-5 text-[16.5px] leading-[1.72] text-muted-foreground xl:grid-cols-2">
            <Reveal>
              <p className="max-w-[58ch]">
                Everyone has this story. Someone opens a file, finds code that looks
                redundant, tidies it up, and takes production down. The mess was there for
                a reason — written that way after an incident — and the reason was never
                recorded anywhere they would look.
              </p>
            </Reveal>
            <Reveal delay={0.06}>
              <p className="max-w-[58ch]">
                That used to happen occasionally, at human speed. Now every team has an
                infinitely fast engineer with total amnesia. Coding agents write a large
                share of merged code and they do exactly this, several times a day, with no
                memory of last Tuesday.
              </p>
            </Reveal>
          </div>
          <Reveal delay={0.1}>
            <p className="mt-12 max-w-[46ch] border-l-2 border-moss pl-7 text-[20px] leading-[1.5] font-medium text-foreground sm:text-[23px]">
              The agent <em className="text-moss not-italic">had</em> the reasoning. Then the
              session ended and all of it was thrown away. What reached the repository was a
              diff.
            </p>
          </Reveal>
        </Split>

        {/* ----------------------------------------------------- ledger */}
        <Stack
          eyebrow="status"
          title="What actually runs today"
          lede="Built in the open. Here is the line between what works and what is still intent."
        >
          <div className="grid gap-px overflow-hidden rounded-xl border border-white/[0.08] bg-white/[0.06] sm:grid-cols-2 lg:grid-cols-3">
            {LEDGER.map((r, i) => (
              <Reveal key={r.t} delay={i * 0.04}>
                <div
                  className={cn(
                    'h-full bg-background/80 p-6 transition-colors',
                    r.on ? 'hover:bg-moss/[0.05]' : 'hover:bg-white/[0.02]',
                  )}
                >
                  <div className="flex items-center gap-2.5">
                    {r.on ? (
                      <Check className="size-4 shrink-0 text-moss" strokeWidth={2.5} />
                    ) : (
                      <Minus className="size-4 shrink-0 text-white/25" strokeWidth={2.5} />
                    )}
                    <span
                      className={cn(
                        'font-mono text-[13px]',
                        r.on ? 'text-foreground' : 'text-white/45',
                      )}
                    >
                      {r.t}
                    </span>
                  </div>
                  <p className="mt-2.5 text-[14.5px] leading-[1.6] text-muted-foreground">
                    {r.d}
                  </p>
                </div>
              </Reveal>
            ))}
          </div>
        </Stack>

        {/* -------------------------------------------------------- how */}
        <Stack id="how" eyebrow="how it works" title="Capture, anchor, surface">
          <div className="grid gap-10 md:grid-cols-3 md:gap-14">
            {STEPS.map((s, i) => (
              <Reveal key={s.t} delay={i * 0.08}>
                <div className="border-t border-white/[0.09] pt-6">
                  <span className="font-mono text-[11px] text-white/30">
                    {String(i + 1).padStart(2, '0')}
                  </span>
                  <h3 className="mt-2.5 flex flex-wrap items-center gap-2 text-[18px]">
                    {s.t}
                    <span
                      className={cn(
                        'rounded-full border px-2 py-0.5 font-mono text-[10px] tracking-wide uppercase',
                        s.tag === 'works'
                          ? 'border-moss/30 text-moss'
                          : 'border-white/12 text-white/35',
                      )}
                    >
                      {s.tag}
                    </span>
                  </h3>
                  <p className="mt-3 max-w-[46ch] text-[14.5px] leading-[1.65] text-muted-foreground">
                    {s.d}
                  </p>
                </div>
              </Reveal>
            ))}
          </div>
        </Stack>

        {/* --------------------------------------------------- anchoring */}
        <Stack
          id="anchoring"
          eyebrow="the hard part"
          title="A record that loses its anchor is a diary entry"
          lede="A decision is about code, and code moves. Line 142 today is line 187 tomorrow, and in a different file next week. A tool that confidently points at the wrong line teaches you to distrust everything it says — so a record that cannot find its anchor says so."
        >
          <Reveal>
            <DriftDemo />
          </Reveal>
        </Stack>

        {/* ------------------------------------------------------- start */}
        <Split
          id="start"
          eyebrow="quickstart"
          title="Try it in five minutes"
          aside={<p>Four steps. Records are hand-written for now.</p>}
        >
          <div className="grid gap-x-14 gap-y-10 xl:grid-cols-2">
            <Reveal>
              <h3 className="mb-3 text-[16px]">
                <span className="mr-2.5 font-mono text-[11.5px] text-white/30">01</span>
                Build the binary
              </h3>
              <Code>{`git clone https://github.com/Amag1n3/whence
cd whence && go build -o why .`}</Code>
              <p className="mt-3 max-w-[56ch] text-[14.5px] leading-[1.65] text-muted-foreground">
                The project is <b className="text-foreground">whence</b>; the binary is{' '}
                <code className="font-mono text-foreground">why</code>, because{' '}
                <code className="font-mono text-foreground">whence</code> is a zsh and ksh
                builtin and builtins beat <code className="font-mono">$PATH</code>.
              </p>
            </Reveal>

            <Reveal delay={0.05}>
              <h3 className="mb-3 text-[16px]">
                <span className="mr-2.5 font-mono text-[11.5px] text-white/30">02</span>
                Write a record
              </h3>
              <Code caption=".whence/records.json — commit this, it is the point">{`[
  {
    "id":         "b5",
    "date":       "2026-07-27",
    "source":     "code review, finding B5",
    "file":       "src/auth/session.go",
    "line_start": 142,
    "line_end":   148,
    "decision":   "Namespace all three session keys.",
    "why":        "The admin dashboard reads them on the same origin..."
  }
]`}</Code>
            </Reveal>

            <Reveal delay={0.1}>
              <h3 className="mb-3 text-[16px]">
                <span className="mr-2.5 font-mono text-[11.5px] text-white/30">03</span>
                Read it back
              </h3>
              <Code>{`why src/auth/session.go:145   # one line
why log                       # everything in the nearest store`}</Code>
            </Reveal>

            <Reveal delay={0.15}>
              <h3 className="mb-3 text-[16px]">
                <span className="mr-2.5 font-mono text-[11.5px] text-white/30">04</span>
                Put it in front of your agent
              </h3>
              <Code caption=".claude/settings.json">{`{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [{ "type": "command", "command": "why hook pre" }]
      }
    ]
  }
}`}</Code>
              <p className="mt-3 max-w-[56ch] text-[14.5px] leading-[1.65] text-muted-foreground">
                Claude Code now sees the record before it touches the file — passed as
                history to be aware of, never as instructions to follow.
              </p>
            </Reveal>
          </div>
        </Split>

        {/* -------------------------------------------------- boundaries */}
        <Split eyebrow="scope" title="What it isn’t">
          <DefList items={NOTS} />
        </Split>

        <Split
          eyebrow="commitments"
          title="What it does with your code"
          aside={<p>This reads what an agent saw and did, which makes it sensitive by default.</p>}
        >
          <DefList items={COMMITMENTS} />
        </Split>

        {/* -------------------------------------------------------- kill */}
        <section className={cn(SHELL, 'pb-10')}>
          <Reveal>
            <div className="rounded-2xl border border-oxide/25 bg-oxide/[0.05] p-8 sm:p-12">
              <div className="grid gap-x-20 gap-y-8 lg:grid-cols-[minmax(0,1fr)_26rem]">
                <div>
                  <p className="font-mono text-[11px] tracking-[0.18em] text-oxide uppercase">
                    falsification
                  </p>
                  <h2 className="mt-3 mb-6 max-w-[18ch] text-[clamp(1.75rem,2.9vw,2.5rem)] leading-[1.12]">
                    The number that kills it
                  </h2>
                  <p className="max-w-[50ch] text-[19px] leading-[1.55] text-foreground sm:text-[21px]">
                    Count the times an agent proposed a change that contradicted a recorded
                    decision and{' '}
                    <span className="font-mono text-[0.88em] text-moss">why</span> caught it.{' '}
                    <span className="text-oxide">
                      If that is zero after three months of real daily use, the idea is wrong
                      and this repo gets archived.
                    </span>
                  </p>
                </div>
                <div className="space-y-4 text-[14.5px] leading-[1.68] text-muted-foreground lg:pt-1">
                  <Separator className="bg-white/[0.08] lg:hidden" />
                  <p>
                    Agreed before a line of code existed. The counter needs{' '}
                    <code className="font-mono">why check</code> to mean anything — today’s
                    log counts how often records were shown, which over-counts, and is not
                    reported as the number above.
                  </p>
                  <p>
                    The other open doubt: if an agent’s stated reasoning is a post-hoc
                    rationalisation rather than the actual cause, this preserves confident
                    nonsense — durably. That gets tested against real captured sessions
                    before any Phase 1 work starts.
                  </p>
                </div>
              </div>
            </div>
          </Reveal>
        </section>
      </main>

      <footer className="relative mt-20 border-t border-white/[0.07]">
        <div
          className={cn(
            SHELL,
            'flex flex-wrap items-center gap-x-7 gap-y-2 py-9 font-mono text-[12.5px] text-white/35',
          )}
        >
          <span className="text-white/55">
            <span className="text-moss">●</span> whence
          </span>
          <a href={REPO} className="transition-colors hover:text-foreground">
            github
          </a>
          <a
            href="mailto:amogh@whence.fyi"
            className="transition-colors hover:text-foreground"
          >
            amogh@whence.fyi
          </a>
          <span className="sm:ml-auto">started 2026-07-31</span>
        </div>
      </footer>
    </div>
  )
}
