import { isValidElement, type ReactNode } from 'react'

import { FAQ } from '@/content/faq'
import { COMMANDS } from '@/content/commands'

/* Everything on the site, flattened into one searchable list.
 *
 * This module is imported dynamically by SearchDialog, never statically. It
 * pulls in faq.tsx (the largest content file in the project) and commands.tsx,
 * and a static import would put both into every page's bundle — including the
 * landing page, whose whole point is to be small. Loading it when the palette
 * first opens costs a few hundred milliseconds nobody notices, on an
 * interaction the reader deliberately started. */

export type Hit = {
  /** What the reader is looking for. Matched hardest. */
  title: string
  /** The prose under it. Matched too, so answers are findable by their content. */
  body: string
  /** Where the hit lives, for the group heading. */
  group: string
  href: string
}

/** Recursively pull the text out of a ReactNode.
 *
 *  The FAQ stores answers as JSX, not strings, so there is no text to match
 *  without walking the tree. Without this the filter can only see questions —
 *  and someone searching "orphaned" wants the answer that explains orphaning,
 *  which does not have the word in its heading.
 *
 *  Only reaches text that is literally in the tree as children. A child
 *  component rendering its own copy would be invisible here; nothing in the
 *  FAQ does that today, and the failure mode is a missed match rather than a
 *  wrong one. */
export function textOf(node: ReactNode): string {
  if (node == null || typeof node === 'boolean') return ''
  if (typeof node === 'string' || typeof node === 'number') return String(node)
  if (Array.isArray(node)) return node.map(textOf).join(' ')
  if (isValidElement(node)) {
    return textOf((node.props as { children?: ReactNode }).children)
  }
  return ''
}

const squash = (s: string) => s.replace(/\s+/g, ' ').trim()

export const INDEX: Hit[] = [
  /* The fragment is the accordion item's own value, not the cluster's — see
     itemId() in FaqPage, which reads it back and opens that one answer.
     Landing on a collapsed cluster heading and making the reader hunt is not
     a search result. */
  ...FAQ.flatMap((cluster) =>
    cluster.questions.map((item, j) => ({
      title: item.q,
      body: squash(textOf(item.a)),
      group: 'Questions',
      href: `/faq#${cluster.id}-${j}`,
    })),
  ),

  ...COMMANDS.flatMap((group) =>
    group.commands.map((c) => ({
      title: c.sig,
      body: squash([c.what, textOf(c.note), c.example ?? ''].join(' ')),
      group: 'Commands',
      href: `/docs#${group.id}`,
    })),
  ),

  /* Pages and their sections. Hand-written rather than derived: these live in
     JSX across five files with no shared data structure, and inventing one so
     a search index could read it would be a much larger change than typing
     eleven lines. If a section is renamed and this is not, search sends people
     to a heading that no longer exists — the same staleness the whole product
     is about, which is worth stating out loud rather than pretending the
     duplication is free. */
  { title: 'Install guide', body: 'Install the binary, go install, the Claude Code plugin, zsh and ksh builtin alias, wire up the PreToolUse hook by hand, backfill a non-empty store, gate CI', group: 'Pages', href: '/install' },
  { title: 'Why it exists', body: 'The mess was load-bearing, capture anchor surface, anchoring drift, the gate, what runs today, what it does with your code, the number that kills it', group: 'Pages', href: '/why' },
  { title: 'Commands reference', body: 'Every command, flags, exit codes, what a record is, anchoring states, the store', group: 'Pages', href: '/docs' },
  { title: 'Questions', body: 'Sixty-one answers about capture, anchoring, security, privacy, scope and project status', group: 'Pages', href: '/faq' },
  { title: 'Contact', body: 'Report a bug, open an issue, email, LinkedIn, X, Instagram', group: 'Pages', href: '/contact' },
  { title: 'Three LLMs graded the same 51 agent-written code rationales', body: '65% 75% 92% faithfulness capture stays off 2,672 edits 32.8% 33.0% 33.3% structural gate 30 unanimous good 3 unanimous bad 18 disputed read ten myself Claude 5/7 Grok 3/7 Kimi 2/7 seven judged entries', group: 'Pages', href: '/notes' },
  { title: 'Running whence against code that is not mine', body: 'trials corpus kubernetes vscode cpython linux rust django backfill dry run 310 candidates 823 candidates XXX as TODO generated files thin splits marker ceiling django three records anchoring untested', group: 'Pages', href: '/trials' },
]
