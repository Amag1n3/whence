import { ArrowUpRight, Check, Minus } from 'lucide-react'
import type { MouseEvent, ReactNode } from 'react'

import { Reveal } from '@/components/Reveal'
import { Terminal } from '@/components/Terminal'
import { DriftDemo } from '@/components/DriftDemo'
import { StrataRail, type Stratum } from '@/components/StrataRail'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

const REPO = 'https://github.com/Amag1n3/whence'
const SHELL = 'mx-auto w-full max-w-[1280px] px-6 sm:px-10'

/* The core, top to bottom. Order here is the order on the page and in the
   rail — depth is the one thing both have to agree on. */
const STRATA: Stratum[] = [
  { id: 'surface', label: 'surface' },
  { id: 'problem', label: 'the problem' },
  { id: 'status', label: 'what runs today' },
  { id: 'how', label: 'how it works' },
  { id: 'anchoring', label: 'anchoring' },
  { id: 'start', label: 'quickstart' },
  { id: 'scope', label: 'what it isn’t' },
  { id: 'handling', label: 'your code' },
  { id: 'falsification', label: 'the kill number' },
]

/** Scroll without writing a hash into the address bar. The href stays so it
 *  works without JS and reads correctly to a screen reader. No `behavior`
 *  argument — that defers to the CSS, which reduced-motion already handles. */
function scrollToId(id: string) {
  if (id === 'surface') window.scrollTo({ top: 0 })
  else document.getElementById(id)?.scrollIntoView()
}
const jump = (id: string) => (e: MouseEvent<HTMLAnchorElement>) => {
  e.preventDefault()
  scrollToId(id)
}

/* --------------------------------------------------------------- pieces */

function Header() {
  return (
    <header className="fixed inset-x-0 top-0 z-50 border-b border-white/[0.06] bg-basin/75 backdrop-blur-xl lg:pl-14">
      <div className={cn(SHELL, 'flex h-14 items-center')}>
        <a
          href="#surface"
          onClick={jump('surface')}
          className="font-mono text-[13.5px] font-medium tracking-tight"
        >
          <span className="text-ochre">●</span> whence
        </a>
        <a
          href={REPO}
          className="ml-auto flex items-center gap-0.5 font-mono text-[12.5px] text-muted-foreground transition-colors hover:text-silt"
        >
          github <ArrowUpRight className="size-3.5" />
        </a>
      </div>
    </header>
  )
}

/** A layer of the core. The header reads like a core log: depth marker,
 *  layer name, what it is. */
function Layer({
  id,
  depth,
  eyebrow,
  title,
  lede,
  children,
}: {
  id: string
  depth: string
  eyebrow: string
  title: string
  lede?: ReactNode
  children: ReactNode
}) {
  return (
    <section id={id} className="border-t border-white/[0.07]">
      <div className={cn(SHELL, 'py-16 sm:py-24')}>
        <Reveal>
          <div className="flex flex-wrap items-start gap-x-16 gap-y-6">
            <span className="font-mono text-[11px] leading-[1.9] tracking-[0.18em] text-ochre">
              {depth}
            </span>
            <div className="min-w-0 flex-1">
              <p className="font-mono text-[11px] tracking-[0.2em] text-dim uppercase">
                {eyebrow}
              </p>
              <h2 className="mt-2.5 max-w-[19ch] text-[clamp(1.65rem,2.7vw,2.3rem)] leading-[1.15]">
                {title}
              </h2>
            </div>
            {lede && (
              <p className="max-w-[46ch] text-[15.5px] leading-[1.7] text-muted-foreground">
                {lede}
              </p>
            )}
          </div>
        </Reveal>
        <div className="mt-11">{children}</div>
      </div>
    </section>
  )
}

function Code({ children, caption }: { children: string; caption?: string }) {
  return (
    <div className="lit overflow-hidden rounded-md border border-white/10 bg-terminal">
      {caption && (
        <div className="border-b border-white/[0.07] px-4 py-2 font-mono text-[11px] text-dim">
          {caption}
        </div>
      )}
      <div className="term-scroll overflow-x-auto px-4 py-3.5">
        <pre className="min-w-max font-mono text-[12.5px] leading-[1.75] text-silt/85">
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
    'The store is meant to be committed, so anything capture picks up — an API key the agent read, a token in a log it was debugging — is public the moment you push, and git history keeps it. Nothing reaches the store unredacted.',
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
            <b className="text-[15.5px] font-semibold text-silt">{t}</b>{' '}
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
    <div className="grain relative min-h-screen">
      <StrataRail strata={STRATA} onJump={scrollToId} />
      <Header />

      <div className="lg:pl-14">
        <main>
          {/* ----------------------------------------------------- surface */}
          <section id="surface" className={cn(SHELL, 'pt-32 pb-16 sm:pt-40 sm:pb-24')}>
            <Reveal>
              <h1 className="max-w-[17ch] text-[clamp(2.4rem,6vw,4.4rem)] leading-[1.04]">
                Git remembers what changed.{' '}
                <span className="font-mono text-[0.82em] font-medium text-ochre">why</span>{' '}
                remembers the reason.
              </h1>
            </Reveal>

            {/* datum: the reference horizon everything below is measured from */}
            <Reveal delay={0.1}>
              <div className="mt-9 flex items-center gap-4">
                <span className="h-px w-10 bg-ochre" />
                <span className="font-mono text-[11px] tracking-[0.16em] text-dim uppercase">
                  datum · phase 0 · surfacing works · capture not built
                </span>
                <span className="hidden h-px flex-1 bg-white/[0.08] sm:block" />
              </div>
            </Reveal>

            <div className="mt-10 grid gap-x-20 gap-y-8 lg:grid-cols-[minmax(0,1fr)_20rem] lg:items-end">
              <Reveal delay={0.16}>
                <p className="max-w-[54ch] text-[17.5px] leading-[1.65] text-muted-foreground sm:text-[19px]">
                  Your coding agent works out why the code has to be like this, then throws
                  it away when the session ends. This keeps it, pinned to the lines it was
                  about, and commits it alongside them — so the next agent gets it before it
                  edits, on any machine that cloned the repo.
                </p>
              </Reveal>
              <Reveal delay={0.22}>
                <Button asChild size="lg" className="rounded-full font-medium lg:w-full">
                  <a href="#start" onClick={jump('start')}>
                    Try it in five minutes
                  </a>
                </Button>
              </Reveal>
            </div>

            <Reveal delay={0.3} className="mt-14">
              <Terminal />
            </Reveal>
          </section>

          {/* ----------------------------------------------------- problem */}
          <Layer id="problem" depth="01" eyebrow="the problem" title="The mess was load-bearing">
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
                  share of merged code and they do exactly this, several times a day, with
                  no memory of last Tuesday.
                </p>
              </Reveal>
            </div>
            <Reveal delay={0.1}>
              <p className="mt-12 max-w-[46ch] border-l-2 border-ochre pl-7 text-[20px] leading-[1.5] font-medium text-silt sm:text-[23px]">
                The agent <em className="text-ochre not-italic">had</em> the reasoning. Then
                the session ended and all of it was thrown away. What reached the repository
                was a diff.
              </p>
            </Reveal>
          </Layer>

          {/* ------------------------------------------------------ status */}
          <Layer
            id="status"
            depth="02"
            eyebrow="status"
            title="What actually runs today"
            lede="Built in the open. Here is the line between what works and what is still intent."
          >
            <div className="grid gap-px overflow-hidden rounded-lg border border-white/[0.08] bg-white/[0.06] sm:grid-cols-2 lg:grid-cols-3">
              {LEDGER.map((r, i) => (
                <Reveal key={r.t} delay={i * 0.04}>
                  <div
                    className={cn(
                      'h-full bg-basin/85 p-6 transition-colors',
                      r.on ? 'hover:bg-verdigris/[0.05]' : 'hover:bg-white/[0.02]',
                    )}
                  >
                    <div className="flex items-center gap-2.5">
                      {r.on ? (
                        <Check className="size-4 shrink-0 text-verdigris" strokeWidth={2.5} />
                      ) : (
                        <Minus className="size-4 shrink-0 text-dim/70" strokeWidth={2.5} />
                      )}
                      <span
                        className={cn(
                          'font-mono text-[13px]',
                          r.on ? 'text-silt' : 'text-dim',
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
          </Layer>

          {/* --------------------------------------------------------- how */}
          <Layer id="how" depth="03" eyebrow="how it works" title="Capture, anchor, surface">
            <div className="grid gap-10 md:grid-cols-3 md:gap-14">
              {STEPS.map((s, i) => (
                <Reveal key={s.t} delay={i * 0.08}>
                  <div className="border-t border-white/[0.09] pt-6">
                    <span className="font-mono text-[11px] text-dim">
                      {String(i + 1).padStart(2, '0')}
                    </span>
                    <h3 className="mt-2.5 flex flex-wrap items-center gap-2 text-[18px]">
                      {s.t}
                      <span
                        className={cn(
                          'rounded-full border px-2 py-0.5 font-mono text-[10px] tracking-wide uppercase',
                          s.tag === 'works'
                            ? 'border-verdigris/35 text-verdigris'
                            : 'border-white/12 text-dim',
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
          </Layer>

          {/* --------------------------------------------------- anchoring */}
          <Layer
            id="anchoring"
            depth="04"
            eyebrow="the hard part"
            title="A record that loses its anchor is a diary entry"
            lede="A decision is about code, and code moves. Line 142 today is line 187 tomorrow, and in a different file next week. A tool that confidently points at the wrong line teaches you to distrust everything it says — so a record that cannot find its anchor says so."
          >
            <Reveal>
              <DriftDemo />
            </Reveal>
          </Layer>

          {/* --------------------------------------------------- quickstart */}
          <Layer
            id="start"
            depth="05"
            eyebrow="quickstart"
            title="Try it in five minutes"
            lede="Four steps. Records are hand-written for now."
          >
            <div className="grid gap-x-14 gap-y-10 xl:grid-cols-2">
              <Reveal>
                <h3 className="mb-3 text-[16px]">
                  <span className="mr-2.5 font-mono text-[11.5px] text-dim">01</span>
                  Build the binary
                </h3>
                <Code>{`git clone https://github.com/Amag1n3/whence
cd whence && go build -o why .`}</Code>
                <p className="mt-3 max-w-[56ch] text-[14.5px] leading-[1.65] text-muted-foreground">
                  The project is <b className="text-silt">whence</b>; the binary is{' '}
                  <code className="font-mono text-silt">why</code>, because{' '}
                  <code className="font-mono text-silt">whence</code> is a zsh and ksh
                  builtin and builtins beat <code className="font-mono">$PATH</code>.
                </p>
              </Reveal>

              <Reveal delay={0.05}>
                <h3 className="mb-3 text-[16px]">
                  <span className="mr-2.5 font-mono text-[11.5px] text-dim">02</span>
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
                  <span className="mr-2.5 font-mono text-[11.5px] text-dim">03</span>
                  Read it back
                </h3>
                <Code>{`why src/auth/session.go:145   # one line
why log                       # everything in the nearest store`}</Code>
              </Reveal>

              <Reveal delay={0.15}>
                <h3 className="mb-3 text-[16px]">
                  <span className="mr-2.5 font-mono text-[11.5px] text-dim">04</span>
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
          </Layer>

          {/* -------------------------------------------------------- scope */}
          <Layer id="scope" depth="06" eyebrow="scope" title="What it isn’t">
            <DefList items={NOTS} />
          </Layer>

          <Layer
            id="handling"
            depth="07"
            eyebrow="commitments"
            title="What it does with your code"
            lede="This reads what an agent saw and did, which makes it sensitive by default."
          >
            <DefList items={COMMITMENTS} />
          </Layer>

          {/* ------------------------------------------------ falsification */}
          <section id="falsification" className="border-t border-white/[0.07]">
            <div className={cn(SHELL, 'py-16 sm:py-24')}>
              <Reveal>
                <div className="rounded-xl border border-cinnabar/25 bg-cinnabar/[0.05] p-8 sm:p-12">
                  <div className="grid gap-x-20 gap-y-8 lg:grid-cols-[minmax(0,1fr)_26rem]">
                    <div>
                      <div className="flex items-center gap-4">
                        <span className="font-mono text-[11px] tracking-[0.18em] text-cinnabar">
                          08
                        </span>
                        <span className="font-mono text-[11px] tracking-[0.2em] text-cinnabar/70 uppercase">
                          falsification
                        </span>
                      </div>
                      <h2 className="mt-3 mb-6 max-w-[18ch] text-[clamp(1.65rem,2.7vw,2.3rem)] leading-[1.15]">
                        The number that kills it
                      </h2>
                      <p className="max-w-[50ch] text-[19px] leading-[1.55] text-silt sm:text-[21px]">
                        Count the times an agent proposed a change that contradicted a
                        recorded decision and{' '}
                        <span className="font-mono text-[0.86em] text-ochre">why</span> caught
                        it.{' '}
                        <span className="text-cinnabar">
                          If that is zero after three months of real daily use, the idea is
                          wrong and this repo gets archived.
                        </span>
                      </p>
                    </div>
                    <div className="space-y-4 border-t border-white/[0.08] pt-6 text-[14.5px] leading-[1.68] text-muted-foreground lg:border-t-0 lg:pt-1">
                      <p>
                        Agreed before a line of code existed. The counter needs{' '}
                        <code className="font-mono">why check</code> to mean anything —
                        today’s log counts how often records were shown, which over-counts,
                        and is not reported as the number above.
                      </p>
                      <p>
                        The other open doubt: if an agent’s stated reasoning is a post-hoc
                        rationalisation rather than the actual cause, this preserves
                        confident nonsense — durably. That gets tested against real captured
                        sessions before any Phase 1 work starts.
                      </p>
                    </div>
                  </div>
                </div>
              </Reveal>
            </div>
          </section>
        </main>

        <footer className="border-t border-white/[0.07]">
          <div
            className={cn(
              SHELL,
              'flex flex-wrap items-center gap-x-7 gap-y-2 py-9 font-mono text-[12.5px] text-dim',
            )}
          >
            <span className="text-silt/70">
              <span className="text-ochre">●</span> whence
            </span>
            <a href={REPO} className="transition-colors hover:text-silt">
              github
            </a>
            <a href="mailto:amogh@whence.fyi" className="transition-colors hover:text-silt">
              amogh@whence.fyi
            </a>
            <span className="sm:ml-auto">started 2026-07-31</span>
          </div>
        </footer>
      </div>
    </div>
  )
}
