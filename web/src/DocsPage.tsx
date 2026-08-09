import { useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { ChevronRight, Search, X } from 'lucide-react'

import { Code } from '@/components/Code'
import { DocPage, DocSection, P, type DocSectionMeta } from '@/components/DocPage'
import { REPO } from '@/components/Chrome'
import { COMMANDS, type Command } from '@/content/commands'

/* Reference. The landing page argues, /install gets you running, this answers
   "what does that flag do" — a different visit from either. Ten commands were
   shipped and four were documented anywhere a reader could find them. */

const TOTAL = COMMANDS.reduce((n, g) => n + g.commands.length, 0)

const A = ({ href, children }: { href: string; children: ReactNode }) => (
  <a
    href={href}
    className="text-ochre underline-offset-4 transition-colors hover:underline"
  >
    {children}
  </a>
)

const M = ({ children }: { children: ReactNode }) => (
  <code className="font-mono text-[13px] text-silt">{children}</code>
)

/** One command: signature, what it does, why, and an example if it earns one.
 *
 *  Collapsed is NOT hidden. The signature and the one-line description stay
 *  visible at all times — those are what a reader scans for, and folding them
 *  away would trade a long page for a page you cannot skim, which is worse.
 *  Only the rationale and the worked example fold, and a command with neither
 *  is not a disclosure at all: it renders as a plain row, because a control
 *  that opens onto nothing is a small lie about there being more to read. */
function Entry({
  sig,
  what,
  note,
  example,
}: {
  sig: string
  what: string
  note?: ReactNode
  example?: string
}) {
  const head = (
    <>
      <h3 className="font-mono text-[13.5px] leading-[1.6] font-medium break-words text-silt">
        {sig}
      </h3>
      <p className="mt-1.5 max-w-[56ch] text-[14.5px] leading-[1.68] text-muted-foreground">
        {what}
      </p>
    </>
  )

  if (!note && !example) {
    return <div className="border-t border-white/[0.09] py-4 pl-7">{head}</div>
  }

  return (
    <details className="group border-t border-white/[0.09]">
      <summary className="flex cursor-pointer list-none items-start gap-3 py-4 [&::-webkit-details-marker]:hidden">
        <ChevronRight className="mt-1 size-4 shrink-0 text-dim transition-transform duration-200 group-open:rotate-90" />
        <span className="min-w-0">{head}</span>
      </summary>
      <div className="pb-6 pl-7">
        {note && (
          <p className="max-w-[56ch] text-[14.5px] leading-[1.68] text-muted-foreground">
            {note}
          </p>
        )}
        {example && (
          <div className="mt-4">
            <Code>{example}</Code>
          </div>
        )}
      </div>
    </details>
  )
}

/* Matches on the signature, the one-line description and the example — the
   three fields that are plain strings. `note` is a ReactNode and cannot be
   searched without rendering it to text, so it is deliberately not covered:
   a search that silently misses a third of the prose would be worse than one
   whose scope is stated. The empty state says so out loud. */
const matches = (c: Command, q: string) =>
  c.sig.toLowerCase().includes(q) ||
  c.what.toLowerCase().includes(q) ||
  (c.example?.toLowerCase().includes(q) ?? false)

export default function DocsPage() {
  const [query, setQuery] = useState('')
  const q = query.trim().toLowerCase()

  const groups = useMemo(
    () =>
      q
        ? COMMANDS.map((g) => ({ ...g, commands: g.commands.filter((c) => matches(c, q)) }))
            .filter((g) => g.commands.length > 0)
        : COMMANDS,
    [q],
  )

  /* The index tracks the filter, so it never offers a jump to a section the
     search has emptied. Memoised because DocPage subscribes a scroll listener
     keyed on this array's identity — a fresh array every keystroke would tear
     the listener down and rebuild it on every keystroke. */
  const sections = useMemo<DocSectionMeta[]>(
    () => [
      ...groups.map((g) => ({ id: g.id, label: g.label })),
      ...(q
        ? []
        : [
            { id: 'record', label: 'What a record is' },
            { id: 'anchoring', label: 'Anchoring' },
            { id: 'store', label: 'The store' },
          ]),
    ],
    [groups, q],
  )

  const hits = groups.reduce((n, g) => n + g.commands.length, 0)

  return (
    <DocPage
      current="docs"
      eyebrow={`reference · ${TOTAL} commands`}
      title="What each command does"
      lede={
        <>
          Transcribed from <M>whence --help</M>, which is the authority — if the binary
          and this page disagree, the binary is right and this page is stale. Not sure
          you have it installed yet? <A href="/install">Start there</A>.
        </>
      }
      sections={sections}
    >
      <div>
        <div className="relative">
          <Search className="pointer-events-none absolute top-1/2 left-3.5 size-4 -translate-y-1/2 text-dim" />
          <input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Filter commands — try check, or --base"
            aria-label="Filter commands"
            className="h-10 w-full border border-white/10 bg-white/[0.02] pr-10 pl-10 font-mono text-[13px] text-silt transition-colors outline-none placeholder:text-dim hover:border-white/20 focus-visible:border-white/30"
          />
          {query && (
            <button
              type="button"
              onClick={() => setQuery('')}
              aria-label="Clear filter"
              className="absolute top-1/2 right-2.5 inline-flex size-6 -translate-y-1/2 items-center justify-center text-dim transition-colors hover:text-silt"
            >
              <X className="size-4" />
            </button>
          )}
        </div>
        {/* Only announced while filtering — a permanent count is noise. */}
        <p aria-live="polite" className="mt-2.5 font-mono text-[11px] text-dim">
          {q ? `${hits} of ${TOTAL} commands` : ' '}
        </p>
      </div>

      {groups.map((group, i) => (
        <DocSection key={group.id} id={group.id} n={i + 1} title={group.label}>
          <div>
            {group.commands.map((c) => (
              <Entry key={c.sig} {...c} />
            ))}
          </div>
        </DocSection>
      ))}

      {q && hits === 0 && (
        <p className="border-t border-white/[0.09] pt-7 text-[14.5px] leading-[1.7] text-muted-foreground">
          No command matches <span className="font-mono text-silt">{query}</span>. The
          filter reads signatures, descriptions and examples — not the longer notes
          inside each entry, so try the command name itself, or{' '}
          <button
            type="button"
            onClick={() => setQuery('')}
            className="text-silt underline underline-offset-4"
          >
            clear the filter
          </button>{' '}
          and use the page.
        </p>
      )}

      {/* Concept sections, hidden while filtering. The filter searches
          commands; leaving three essays below a "2 of 10 commands" count
          would read as unfiltered results. */}
      {!q && (
        <>
      <DocSection id="record" n={COMMANDS.length + 1} title="What a record is">
        <P>
          One JSON object per line in <M>.whence/records.jsonl</M>. One line per record so
          the file merges — two branches each adding a record produce two added lines, not
          a conflict, which is the whole reason it is not a single JSON document.
        </P>
        <P>
          A record carries the decision, the reason, the lines it is anchored to, the
          content hashes of those lines, an optional source, and optional evidence. The
          id is short hex — six characters, so collisions get likely somewhere around a
          few thousand records.
        </P>
        <P>
          <b className="text-silt">Records are data, never directives.</b> Everything the
          hook injects is prefixed with a line saying these are historical notes and not
          instructions to follow. That framing is what stops anything able to write the
          store from being able to give your agent orders.
        </P>
      </DocSection>

      <DocSection id="anchoring" n={COMMANDS.length + 2} title="Anchoring">
        <P>
          A record pinned to a line number is wrong the first time someone adds an import
          above it. So a record stores a content hash per anchored line, and finds those
          lines again by scanning a window around where they used to be. No AST, no
          tree-sitter — that is a deliberate ceiling, and the trigger for revisiting it is
          a real repository producing orphans this cannot explain.
        </P>
        <P>
          A resolved anchor reads as one of five states. These are the display strings
          themselves, not a summary of them — the CLI, the context injected into your
          agent and this page share one vocabulary on purpose, because a second mapping
          layer would only be somewhere for them to drift apart.
        </P>
        <div className="space-y-3">
          {(
            [
              ['intact, exact range', 'ok', 'Every anchored line hashes the same, right where it was recorded.'],
              ['intact, moved', 'ok', 'Same content, different line numbers. The window scan found it; nothing is wrong.'],
              ['line range only, unverified', 'neutral', 'A record with no content hashes to check — hand-written, rather than added by the CLI.'],
              ['altered', 'bad', 'The anchored block is partly rewritten. The record may still be true, but something it described has changed.'],
              ['ORPHANED — anchor lost, needs a human', 'bad', 'The content cannot be found at all. The record has lost its subject, and only a person can say where it went.'],
            ] as const
          ).map(([state, tone, desc]) => (
            <div key={state} className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
              {/* Same opacity ladder as Gate and DriftDemo: 'ok' and 'bad'
                  were a green and a red, which are the same white now, so
                  severity is brightness. ORPHANED also shouts in caps, which
                  is doing at least as much of the work. */}
              <span
                className={`font-mono text-[12.5px] ${
                  tone === 'bad' ? 'text-silt' : tone === 'ok' ? 'text-silt/55' : 'text-dim'
                }`}
              >
                {state}
              </span>
              <span className="max-w-[52ch] text-[14.5px] leading-[1.68] text-muted-foreground">
                {desc}
              </span>
            </div>
          ))}
        </div>
        <P>
          <b className="text-silt">A state is not a verdict on a diff.</b>{' '}
          <M>check</M> reports something different — what a particular change did — and
          only three of its four findings fail: a record <b className="text-silt">eroded</b>{' '}
          by the diff, an <b className="text-silt">anchor lost</b> to it, or{' '}
          <b className="text-silt">evidence deleted</b> by it. A diff merely{' '}
          <b className="text-silt">touching</b> covered lines is informational and exits 0.
        </P>
        <P>
          <M>reanchor</M> resolves an orphan once you know where the code went. A record
          already orphaned before your change is never your pull request's failure —
          failing CI for it would only teach people to pass <M>--no-verify</M>.
        </P>
      </DocSection>

      <DocSection id="store" n={COMMANDS.length + 3} title="The store">
        <Code>{`.whence/
├── records.jsonl     the records — commit this
├── surfaced.jsonl    what was put in front of an agent, and when
├── .gitattributes    merge strategy for the append-only line format
└── .gitignore        keeps surfaced.jsonl local`}</Code>
        <P>
          Found by walking up from the file being looked at, so a monorepo can hold one
          store at the root or one per package and both work.{' '}
          <M>records.jsonl</M> is committed and shared — that is the point of it. A bad
          record therefore costs more than a missing one, which is the argument for
          reading what <M>backfill</M> harvested rather than trusting it.
        </P>
        <P>
          <M>surfaced.jsonl</M> is local and gitignored. It counts surfacings, not caught
          contradictions, so it over-counts badly as a measure of whether any of this
          works — most surfacings are purely informational.
        </P>
      </DocSection>

      <div className="border-t border-white/[0.07] pt-7">
        <P>
          Something missing or wrong here? <A href={REPO}>Open an issue</A>. A command
          this page documents incorrectly is worse than one it omits.
        </P>
      </div>
        </>
      )}
    </DocPage>
  )
}
