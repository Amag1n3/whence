import { ArrowUpRight, Check, Minus } from 'lucide-react'
import type { MouseEvent, ReactNode } from 'react'

import { Reveal } from '@/components/Reveal'
import { Terminal } from '@/components/Terminal'
import { DriftDemo } from '@/components/DriftDemo'
import { Gate } from '@/components/Gate'
import { StrataRail, type Stratum } from '@/components/StrataRail'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

const REPO = 'https://github.com/Amag1n3/whence'
const SHELL = 'mx-auto w-full max-w-[1280px] px-6 sm:px-10'

/* The core, top to bottom. Order here is the order on the page and in the
   rail — depth is the one thing both have to agree on. */
const STRATA: Stratum[] = [
  { id: 'surface', label: 'surface' },
  { id: 'gap', label: 'the cost' },
  { id: 'problem', label: 'the problem' },
  { id: 'status', label: 'what runs today' },
  { id: 'how', label: 'how it works' },
  { id: 'anchoring', label: 'anchoring' },
  { id: 'gate', label: 'the gate' },
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

/** Two dated markers and the distance between them. The page argues about
 *  amnesia everywhere else in the abstract; this is the one place it costs
 *  something. Symmetric, so the closing line is centred — the only centred
 *  text on an otherwise strictly left-aligned page, and it earns it. */
function TheGap() {
  return (
    <section id="gap" className="border-t border-white/[0.07]">
      <div className={cn(SHELL, 'py-16 sm:py-24')}>
        <Reveal>
          <div className="flex items-center gap-4">
            <span className="size-2 shrink-0 rounded-full bg-ochre" />
            <span className="h-px flex-1 bg-gradient-to-r from-ochre/45 to-white/[0.08]" />
            <span className="font-mono text-[11px] tracking-[0.16em] text-dim uppercase">
              three days
            </span>
            <span className="h-px flex-1 bg-gradient-to-r from-white/[0.08] to-cinnabar/45" />
            <span className="size-2 shrink-0 rounded-full bg-cinnabar" />
          </div>

          <div className="mt-5 flex items-start justify-between gap-10">
            <div className="max-w-[26ch]">
              <p className="font-mono text-[11px] tracking-[0.16em] text-ochre">2026-07-27</p>
              <p className="mt-2 text-[14.5px] leading-[1.55] text-muted-foreground">
                The decision gets made, and written down in a code review.
              </p>
            </div>
            <div className="max-w-[26ch] text-right">
              <p className="font-mono text-[11px] tracking-[0.16em] text-cinnabar">
                2026-07-30
              </p>
              <p className="mt-2 text-[14.5px] leading-[1.55] text-muted-foreground">
                The same bug is worked out again from scratch. It takes an evening.
              </p>
            </div>
          </div>

          <p className="mx-auto mt-14 max-w-[24ch] text-center text-[clamp(1.5rem,3vw,2.35rem)] leading-[1.18]">
            The note existed. It was not in front of anyone.
          </p>
        </Reveal>
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
  {
    on: true,
    t: 'The PreToolUse hook',
    d: 'Records reach Claude Code before it edits, in 6ms. Fails open — a broken whence costs you nothing but a missing record.',
  },
  {
    on: true,
    t: 'Content-hash anchoring',
    d: 'A record follows its code as it moves, decays when the code is rewritten, and reports itself orphaned rather than pointing at the wrong line.',
  },
  {
    on: true,
    t: 'whence check',
    d: 'The CI gate. Prints the decisions covering a diff; fails the build only for the ones it damaged — worn away, cut loose, or left standing on deleted evidence.',
  },
  {
    on: true,
    t: 'Evidence',
    d: 'A record can point at what makes it true. Point at code and that pointer is anchored too, so it can rot on its own.',
  },
  {
    on: true,
    t: 'whence backfill',
    d: 'Harvests decisions already written in your code — HACK:, WORKAROUND:, and NOTE:/TODO: notes that give a reason. Day one is not an empty store.',
  },
  {
    on: false,
    t: 'Capture',
    d: 'Records are authored deliberately. Deciding which slice of a session is worth keeping is the actual hard part, and it is not built.',
  },
]

const STEPS = [
  {
    t: 'Capture',
    tag: 'not built',
    d: "Subscribe to the agent's hooks and record the decision trail as the session runs. Redaction happens here, before anything is written to disk — once a secret reaches a content-addressed store it may already be replicated.",
  },
  {
    t: 'Anchor',
    tag: 'works',
    d: 'Hash every significant line of the span. Reindent and reformat freely — nothing moves. Insert above it and the record follows down, at no cost — a block that moved is not a block that changed. Only rewriting it costs anything, and it falls until the record calls itself orphaned and claims no line at all.',
  },
  {
    t: 'Surface',
    tag: 'works',
    d: 'From your terminal, into a coding agent’s context through a PreToolUse hook before it edits, and in CI as a gate that fails the build.',
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
    'They arrive by git pull, so anyone who can land a commit can put text in front of your agent. Every injected block is framed as history to be aware of, not instruction to follow.',
  ],
  [
    'A record no human has read says so',
    'A person writing one is a deliberate act of attention; an agent writing one is none. Agent-authored records are marked unchecked until somebody confirms them — including in the block your agent receives. A record can also never cite another record as its evidence, which is the link that would let one wrong record prop up the next.',
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
                  datum · surfacing, anchoring and the CI gate run · capture does not
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

          <TheGap />

          {/* ----------------------------------------------------- problem */}
          <Layer id="problem" depth="02" eyebrow="the problem" title="The mess was load-bearing">
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
            depth="03"
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
          <Layer id="how" depth="04" eyebrow="how it works" title="Capture, anchor, surface">
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
            depth="05"
            eyebrow="the hard part"
            title="A record that loses its anchor is a diary entry"
            lede="A decision is about code, and code moves. Line 142 today is line 187 tomorrow, and in a different file next week. A tool that confidently points at the wrong line teaches you to distrust everything it says — so a record that cannot find its anchor says so."
          >
            <Reveal>
              <DriftDemo />
            </Reveal>
          </Layer>

          {/* -------------------------------------------------------- gate */}
          <Layer
            id="gate"
            depth="06"
            eyebrow="the gate"
            title="The exit code is the product"
            lede="Everything else leads here. In CI, a diff is compared against the decisions that govern the lines it touches — and the build fails only for the ones it quietly wore away or severed."
          >
            <Gate />
          </Layer>

          {/* --------------------------------------------------- quickstart */}
          <Layer
            id="start"
            depth="07"
            eyebrow="quickstart"
            title="Try it in five minutes"
            lede="Four steps, about five minutes. You decide what gets recorded; the tool works out how to make it survive."
          >
            <div className="grid gap-x-14 gap-y-10 xl:grid-cols-2">
              <Reveal>
                <h3 className="mb-3 text-[16px]">
                  <span className="mr-2.5 font-mono text-[11.5px] text-dim">01</span>
                  Install it
                </h3>
                <Code>{`go install github.com/Amag1n3/whence@latest

# zsh and ksh only — both shadow it with a builtin
echo "alias whence='command whence'" >> ~/.zshrc`}</Code>
                <p className="mt-3 max-w-[56ch] text-[14.5px] leading-[1.65] text-muted-foreground">
                  <b className="text-silt">whence</b> everywhere — the project, the repo,
                  the binary. zsh and ksh are the one exception: both ship a{' '}
                  <code className="font-mono text-silt">whence</code> builtin that beats{' '}
                  <code className="font-mono">$PATH</code>, and an alias is resolved before
                  a builtin, so that one line is the whole fix. bash, fish and every hook
                  or CI job need nothing.
                </p>
              </Reveal>

              <Reveal delay={0.05}>
                <h3 className="mb-3 text-[16px]">
                  <span className="mr-2.5 font-mono text-[11.5px] text-dim">02</span>
                  Record a decision
                </h3>
                <Code>{`whence add src/auth/session.go:142-148 \\
  -d "Namespace all three session keys to CHECKOUT_*." \\
  -w "The admin dashboard reads them on the same origin." \\
  -e dashboard/Header.tsx:88-94`}</Code>
                <p className="mt-3 max-w-[56ch] text-[14.5px] leading-[1.65] text-muted-foreground">
                  Writes <code className="font-mono text-silt">.whence/records.jsonl</code> —
                  commit it, that is the point. The line hashes that let the record survive
                  the code moving are computed here; hand-writing the file gets you a line
                  number and nothing that follows it. <code className="font-mono">-e</code>{' '}
                  is optional and repeatable.
                </p>
              </Reveal>

              <Reveal delay={0.1}>
                <h3 className="mb-3 text-[16px]">
                  <span className="mr-2.5 font-mono text-[11.5px] text-dim">03</span>
                  Read it back
                </h3>
                <Code>{`whence src/auth/session.go:145   # one line
whence log                       # everything in the nearest store`}</Code>
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
        "hooks": [{ "type": "command", "command": "/abs/path/to/whence hook pre" }]
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
          <Layer id="scope" depth="08" eyebrow="scope" title="What it isn’t">
            <DefList items={NOTS} />
          </Layer>

          <Layer
            id="handling"
            depth="09"
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
                          10
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
                        Agreed before a line of code existed.{' '}
                        <code className="font-mono">whence check</code> is what makes it
                        measurable, and it now runs. The surfacing log counts how often
                        records were shown, which over-counts badly, and is deliberately not
                        reported as the number above.
                      </p>
                      <p>
                        And the number cannot see its own worst failure. If an agent’s
                        stated reason is a story told afterwards rather than the actual
                        cause, this preserves confident nonsense durably — and a store full
                        of nonsense produces <em className="not-italic text-silt">more</em>{' '}
                        catches, not fewer, so the count rises while the tool rots. That is
                        why retractions are logged too, and why capture stays off until the
                        faithfulness of stated reasoning has actually been measured.
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
