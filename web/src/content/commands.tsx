import type { ReactNode } from 'react'

/* The command surface, taken from `usage()` in main.go. Same standing rule as
   Terminal and DriftDemo: this is a transcription of what the binary does, so
   it becomes a lie the moment main.go changes without it. Eleven commands as
   of v0.3.1 — if `whence --help` lists one that is not here, here is wrong.

   Do not write the count into page copy. It said "ten" in three places while
   this list held eleven, and the only reason /docs was right is that its
   eyebrow computes the number instead of stating it. */

export type Command = {
  /** The invocation, as you would type it. */
  sig: string
  /** One line: what it does. */
  what: string
  /** Why it exists, or the thing people get wrong about it. Optional. */
  note?: ReactNode
  /** A worked example. Optional — the trivial commands do not need one. */
  example?: string
}

export type CommandGroup = { id: string; label: string; commands: Command[] }

export const COMMANDS: CommandGroup[] = [
  {
    id: 'reading',
    label: 'Reading',
    commands: [
      {
        sig: 'whence <file>[:<line>]',
        what: 'Show recorded decisions for a file, or for one line of it.',
        note: (
          <>
            The bare form. No subcommand, because looking something up is the thing you
            do most and it should cost the fewest keystrokes.
          </>
        ),
        example: `whence src/auth/session.go:145
whence src/auth/session.go`,
      },
      {
        sig: 'whence log',
        what: 'List every record in the nearest store.',
        note: (
          <>
            The store is found by walking up from the current directory, so this answers
            for the repo you are standing in.{' '}
            <b className="text-silt">no .whence/records.jsonl found</b> means there is no
            store yet, not that the store is empty.
          </>
        ),
      },
    ],
  },
  {
    id: 'writing',
    label: 'Writing',
    commands: [
      {
        sig: 'whence add <file>:<a>-<b> -d "decision" -w "why" [-s source] [-e evidence]',
        what: 'Record a decision and anchor it to those lines.',
        note: (
          <>
            <b className="text-silt">Use this rather than editing the file by hand.</b>{' '}
            The per-line content hashes that let a record survive the code moving are
            computed here; hand-writing the JSON gets you a line number and none of what
            follows from it. <code className="font-mono text-silt">-e</code> is optional
            and repeatable, and takes anything checkable — a{' '}
            <code className="font-mono text-silt">file:line</code> (anchored, so its rot
            is detectable), a command, a commit, a link. Never another record.
          </>
        ),
        example: `whence add src/auth/session.go:142-148 \\
  -d "Namespace all three session keys to CHECKOUT_*." \\
  -w "The admin dashboard reads them on the same origin." \\
  -e dashboard/Header.tsx:88-94`,
      },
      {
        sig: 'whence backfill [dir]',
        what: 'Harvest decisions already written in your comments.',
        note: (
          <>
            <code className="font-mono text-silt">HACK:</code>,{' '}
            <code className="font-mono text-silt">WORKAROUND:</code>,{' '}
            <code className="font-mono text-silt">XXX:</code>,{' '}
            <code className="font-mono text-silt">GOTCHA:</code> and{' '}
            <code className="font-mono text-silt">ponytail:</code> always;{' '}
            <code className="font-mono text-silt">NOTE:</code>,{' '}
            <code className="font-mono text-silt">WARNING:</code> and{' '}
            <code className="font-mono text-silt">CAVEAT:</code> only where the note gives
            a reason. The usual first command after installing, because a store with no
            records is a tool that does nothing.
          </>
        ),
      },
      {
        sig: 'whence rm <id> [-w why]',
        what: 'Retract one record, logging why it was wrong.',
        note: (
          <>
            A retraction, not a delete — the record was wrong and that is itself worth
            knowing. Contrast with{' '}
            <code className="font-mono text-silt">reground</code> and{' '}
            <code className="font-mono text-silt">reanchor</code>, where the claim still
            stands.
          </>
        ),
      },
    ],
  },
  {
    id: 'maintaining',
    label: 'Maintaining',
    commands: [
      {
        sig: 'whence reanchor <id> <file>:<a>-<b>',
        what: 'Re-point a record at the lines it is about now.',
        note: (
          <>
            For after the block it described was rewritten past recognition.{' '}
            <b className="text-silt">You name the span</b>, rather than the tool guessing
            — where a degraded record currently points is a guess, not an answer, and
            having the tool guess twice does not make it right.
          </>
        ),
      },
      {
        sig: 'whence reground <id> -e <ref> [-e ...]',
        what: "Re-point a record's evidence.",
        note: (
          <>
            Not a retraction: the claim stands, only what backs it up has moved. Use when{' '}
            <code className="font-mono text-silt">check</code> reports evidence gone but
            the decision is still true.
          </>
        ),
      },
      {
        sig: 'whence confirm <id>',
        what: 'Record that a human has checked an agent-written record.',
        note: (
          <>
            <b className="text-silt">
              Records an agent wrote are not trusted until a human says so.
            </b>{' '}
            This is the command that says so, and the distinction is load-bearing —
            anything able to write the store can put text in front of an agent.
          </>
        ),
      },
    ],
  },
  {
    id: 'checking',
    label: 'Checking',
    commands: [
      {
        sig: 'whence check [-base rev]',
        what: 'Report the decisions covering a diff.',
        note: (
          <>
            <b className="text-silt">Exits 1 only for the ones the change damaged</b> —
            eroded, orphaned, evidence gone. Records that merely cover the changed lines
            print and pass, because a gate that fires on proximity fails on{' '}
            <code className="font-mono text-silt">gofmt</code> and gets switched off. In
            CI this needs <code className="font-mono text-silt">fetch-depth: 0</code>; it
            reads the base revision of the store and of every cited file.
          </>
        ),
        example: `whence check -base origin/main`,
      },
      {
        sig: 'whence capture <session.jsonl>',
        what: 'Read a finished Claude Code session and show each edit beside what was said before it.',
        note: (
          <>
            Reads the explicitly named finished session. There is no default: guessing
            the newest file can select the live session.{' '}
            <b className="text-silt">Writes nothing.</b> Whether a stated reason is the
            real one is the open question this project has not answered, so a human reads
            these and decides rather than the tool recording them automatically.
          </>
        ),
      },
      {
        sig: 'whence hook pre',
        what: 'Called by Claude Code. Reads a hook payload on stdin, writes context on stdout.',
        note: (
          <>
            You do not run this by hand except to test the wiring.{' '}
            <b className="text-silt">Every error path exits 0 having printed nothing</b>,
            which Claude Code reads as "no opinion" — a broken whence must cost you a
            missing record and nothing else, never a blocked edit.
          </>
        ),
      },
    ],
  },
]
