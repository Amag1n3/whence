import type { ReactNode } from 'react'

import { Code } from '@/components/Code'
import { DocPage, DocSection, P, type DocSectionMeta } from '@/components/DocPage'
import { REPO } from '@/components/Chrome'

/* The install story, which used to be four cards at the bottom of the landing
   page. It had drifted into teaching the manual settings.json edit — the one
   step the plugin exists to remove — because the plugin shipped after the
   landing page was written and nothing pointed back at it.
 *
 * Order is the order a first install actually happens in, and the plugin comes
 * before the manual JSON for the same reason: the short path first, the escape
 * hatch after. */

const SECTIONS: DocSectionMeta[] = [
  { id: 'binary', label: 'Install the binary' },
  { id: 'shell', label: 'zsh and ksh' },
  { id: 'hook', label: 'Wire up the hook' },
  { id: 'manual', label: 'Without the plugin' },
  { id: 'store', label: 'Get a non-empty store' },
  { id: 'verify', label: 'Check it fired' },
  { id: 'ci', label: 'Gate CI on it' },
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

export default function InstallPage() {
  return (
    <DocPage
      current="install"
      eyebrow="install · about five minutes"
      title="Getting it running"
      lede={
        <>
          Two commands get you a working install. The rest of this page is the detail
          behind them — the shell builtin that shadows the binary, what to do if you
          would rather not install a plugin, and how to tell a silent hook from a
          broken one.
        </>
      }
      sections={SECTIONS}
    >
      <DocSection id="binary" n={1} title="Install the binary">
        <Code>{`go install github.com/Amag1n3/whence@latest`}</Code>
        <P>
          Go 1.22+. <M>@latest</M> resolves to the newest tag. This step cannot be
          skipped and no plugin can do it for you — the plugin carries hook
          configuration, not a compiled binary.
        </P>
        <P>
          Building from source works the same way, and is the better option if you want
          the <M>why</M> symlink or intend to edit the code:
        </P>
        <Code>{`git clone ${REPO} && cd whence
go build -o whence .
mv whence ~/.local/bin/                        # anywhere on $PATH
ln -s ~/.local/bin/whence ~/.local/bin/why     # optional short form`}</Code>
        <P>
          If you have installed before, check which copy you are actually reaching. A
          hand-placed binary in <M>~/.local/bin</M> shadowing a newer <M>go install</M>{' '}
          build cost a full day on this project. <M>command -v whence</M> settles it.
        </P>
      </DocSection>

      <DocSection id="shell" n={2} title="One note about zsh and ksh">
        <P>
          <b className="text-silt">zsh and ksh ship their own `whence` builtin</b>, and
          builtins beat <M>$PATH</M>. Those two shells — and only those two — need one
          line in <M>~/.zshrc</M> or <M>~/.kshrc</M>:
        </P>
        <Code>{`alias whence='command whence'`}</Code>
        <P>
          Aliases resolve before builtins, so that is the whole fix. <M>command whence</M>{' '}
          also reaches the binary in a one-off without the alias, and the <M>why</M>{' '}
          symlink runs the same binary under a name nothing competes for.
        </P>
        <P>
          bash, fish, nushell and every non-interactive context — hooks, CI, scripts —
          need nothing. They have no such builtin, which is also why this never affects
          the hook.
        </P>
      </DocSection>

      <DocSection id="hook" n={3} title="Wire up the hook">
        <P>
          This is the part that matters. Without it whence is a lookup tool you have to
          remember to run; with it, records reach the agent before it edits. The
          Claude Code plugin is the short path:
        </P>
        <Code caption="in Claude Code">{`/plugin marketplace add Amag1n3/whence`}</Code>
        <Code caption="then">{`/plugin install whence@whence`}</Code>
        <P>
          Restart Claude Code — hook configuration is read at startup — then run{' '}
          <M>/whence:setup</M>, which checks each link in the chain and tells you which
          one is missing.
        </P>
        <P>
          The plugin finds the binary itself, looking at <M>$WHENCE_BIN</M>, then{' '}
          <M>$GOBIN</M> and <M>$GOPATH/bin</M>, then <M>~/.local/bin</M>,{' '}
          <M>/usr/local/bin</M> and <M>/opt/homebrew/bin</M>, then <M>$PATH</M>. The Go
          path is preferred over <M>~/.local/bin</M> deliberately: that ordering is the
          day-long shadowing bug written down as code, so reinstalling is what decides
          which binary runs. Find nothing and it stays silent rather than erroring on
          every edit.
        </P>
      </DocSection>

      <DocSection id="manual" n={4} title="Without the plugin">
        <P>
          If you would rather not install a plugin, or you are wiring up an agent that
          is not Claude Code, the hook is ordinary configuration. In{' '}
          <M>~/.claude/settings.json</M> for all projects, or <M>.claude/settings.json</M>{' '}
          for one repo:
        </P>
        <Code caption=".claude/settings.json">{`{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "timeout": 5,
            "command": "/absolute/path/to/whence hook pre"
          }
        ]
      }
    ]
  }
}`}</Code>
        <P>
          <b className="text-silt">Use an absolute path.</b> Hooks do not reliably
          inherit <M>$PATH</M> from your shell profile, and a hook that cannot find its
          binary fails silently — which is also exactly how a working hook with nothing
          to say behaves. That is the trap this whole page is arranged around.{' '}
          <M>command -v whence</M> gives you the path to paste. No alias is needed here;
          a hook is not your interactive shell.
        </P>
        <P>
          <M>timeout: 5</M> is belt-and-braces. The hook measures 6.3ms on a warm store,
          and every error path exits 0 having printed nothing, so a broken whence costs
          you a missing record and nothing else. Restart Claude Code after editing
          settings.
        </P>
      </DocSection>

      <DocSection id="store" n={5} title="Get a non-empty store">
        <P>
          A store with no records is a tool that does nothing, so start by harvesting
          what the codebase already wrote down rather than authoring records by hand:
        </P>
        <Code>{`whence backfill`}</Code>
        <P>
          That reads <M>HACK:</M>, <M>WORKAROUND:</M>, <M>XXX:</M>, <M>GOTCHA:</M> and{' '}
          <M>ponytail:</M> comments always, and <M>NOTE:</M>, <M>TODO:</M>, <M>FIXME:</M>
          , <M>WARNING:</M>, <M>CAVEAT:</M> only where the note gives a reason. Read what
          it found and delete anything you disagree with — the store is committed and
          shared, so a bad record costs more than a missing one.
        </P>
        <P>
          Records live in <M>.whence/records.jsonl</M>, found by walking up from the
          file. Commit it; that is the point.
        </P>
      </DocSection>

      <DocSection id="verify" n={6} title="Check it fired">
        <P>
          A silent hook and a broken hook look identical from the outside. That is the
          cost of the fail-open rule, and the only honest check is the log:
        </P>
        <Code>{`ls -la .whence/surfaced.jsonl`}</Code>
        <P>
          That file gains a line every time records are put in front of an agent. If it
          does not exist after editing a file that has records, the hook is not reaching
          the binary. To test the path directly without waiting for an edit:
        </P>
        <Code>{`echo '{"cwd":"'"$PWD"'","tool_input":{"file_path":"'"$PWD"'/some/file.go"}}' \\
  | whence hook pre`}</Code>
        <P>
          JSON on stdout means the whole chain works. Empty output means no records match
          that file, which is not a failure.
        </P>
      </DocSection>

      <DocSection id="ci" n={7} title="Gate CI on it">
        <P>
          <M>whence check</M> compares a diff against the store and exits 1{' '}
          <b className="text-silt">only</b> for records the change damaged — eroded,
          orphaned, or evidence deleted. Records that merely cover the changed lines
          print and pass, because a gate that fires on proximity fails on <M>gofmt</M>{' '}
          and gets switched off within a week.
        </P>
        <Code caption=".github/workflows/whence.yml">{`check:
  if: github.event_name == 'pull_request'
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
      with:
        fetch-depth: 0
    - uses: actions/setup-go@v5
      with:
        go-version: '1.22'
    - run: go install github.com/Amag1n3/whence@latest
    - run: whence check -base origin/\${{ github.base_ref }}`}</Code>
        <P>
          <b className="text-silt">
            <M>fetch-depth: 0</M> is not optional.
          </b>{' '}
          <M>check</M> reads the base revision of the store and of every cited file. A
          shallow clone has neither, so the gate would pass by finding nothing — green
          for the wrong reason, which is worse than red.
        </P>
        <P>
          <M>setup-go</M> puts <M>$GOPATH/bin</M> on the path, which is what makes{' '}
          <M>whence</M> callable on the next line. whence's own repository builds from
          source instead, for the obvious reason.
        </P>
      </DocSection>

      <div className="border-t border-white/[0.07] pt-7">
        <P>
          Installed and running? <A href="/docs">The command reference</A> covers all ten
          commands, the record format, and what anchoring does when the code moves.
          Anything unclear or wrong here is <A href={REPO}>an issue worth opening</A> —
          an install someone cannot complete is a gap in the argument, not a support
          request.
        </P>
      </div>
    </DocPage>
  )
}
