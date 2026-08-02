import type { ReactNode } from 'react'

import { Code } from '@/components/Code'
import { DocPage, DocSection, P, type DocSectionMeta } from '@/components/DocPage'
import { REPO } from '@/components/Chrome'
import { COMMANDS } from '@/content/commands'

/* Reference. The landing page argues, /install gets you running, this answers
   "what does that flag do" — a different visit from either. Ten commands were
   shipped and four were documented anywhere a reader could find them. */

const SECTIONS: DocSectionMeta[] = [
  ...COMMANDS.map((g) => ({ id: g.id, label: g.label })),
  { id: 'record', label: 'What a record is' },
  { id: 'anchoring', label: 'Anchoring' },
  { id: 'store', label: 'The store' },
]

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
 *  The signature is the heading, because that is what a reader scans for. */
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
  return (
    <div className="border-t border-white/[0.07] pt-6">
      <h3 className="font-mono text-[13.5px] leading-[1.6] font-medium break-words text-silt">
        {sig}
      </h3>
      <p className="mt-2.5 max-w-[56ch] text-[14.5px] leading-[1.68] text-silt/90">{what}</p>
      {note && (
        <p className="mt-2.5 max-w-[56ch] text-[14.5px] leading-[1.68] text-muted-foreground">
          {note}
        </p>
      )}
      {example && (
        <div className="mt-4">
          <Code>{example}</Code>
        </div>
      )}
    </div>
  )
}

export default function DocsPage() {
  return (
    <DocPage
      current="docs"
      eyebrow={`reference · ${COMMANDS.reduce((n, g) => n + g.commands.length, 0)} commands`}
      title="What each command does"
      lede={
        <>
          Transcribed from <M>whence --help</M>, which is the authority — if the binary
          and this page disagree, the binary is right and this page is stale. Not sure
          you have it installed yet? <A href="/install">Start there</A>.
        </>
      }
      sections={SECTIONS}
    >
      {COMMANDS.map((group, i) => (
        <DocSection key={group.id} id={group.id} n={i + 1} title={group.label}>
          <div className="space-y-7">
            {group.commands.map((c) => (
              <Entry key={c.sig} {...c} />
            ))}
          </div>
        </DocSection>
      ))}

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
              <span
                className={`font-mono text-[12.5px] ${
                  tone === 'ok' ? 'text-verdigris' : tone === 'bad' ? 'text-cinnabar' : 'text-dim'
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
    </DocPage>
  )
}
