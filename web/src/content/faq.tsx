import type { ReactNode } from 'react'

/* The FAQ, as data.
 *
 * Kept out of App.tsx for the same reason LEDGER and COMMITMENTS are consts:
 * the page file should read as a page. This one is long enough to bury the
 * layout if it sat inline.
 *
 * Every answer here has to be true of the code as it stands. Capture, AST-path
 * anchoring and record signing are NOT built, and the questions about them say
 * so in the present tense rather than describing them as nearly-here. This file
 * is the easiest place in the project to accidentally promise something that
 * does not exist — the README carries the same warning about itself. */

export type Question = { q: string; a: ReactNode }
export type Cluster = { id: string; label: string; questions: Question[] }

const C = ({ children }: { children: string }) => (
  <code className="font-mono text-[0.92em] text-silt">{children}</code>
)

export const FAQ: Cluster[] = [
  {
    id: 'what',
    label: 'What it is',
    questions: [
      {
        q: 'What does this do that git blame does not?',
        a: (
          <>
            <C>git blame</C> tells you who changed a line and which commit it arrived in.
            That is the what. It cannot tell you the option that was rejected, the incident
            the code was written after, or the constraint that made the obvious version
            wrong — because none of that was ever in the diff. A record holds the reasoning
            and stays attached to the lines it was about, so the question &ldquo;why is this
            like this&rdquo; has somewhere to be answered.
          </>
        ),
      },
      {
        q: 'Why not write a comment?',
        a: (
          <>
            Write the comment. <C>whence backfill</C> harvests comments that already carry a
            reason, so this is not an either/or. What a comment cannot do is reach an agent
            that is about to edit a different part of the file, survive being deleted by
            someone who thought it was noise, or fail a build when the code it described was
            worn away. A record is a comment that is anchored, checkable and enforced.
          </>
        ),
      },
      {
        q: 'Why not an ADR directory?',
        a: (
          <>
            ADRs record decisions at the level of the system, and nothing connects them to
            lines. Six months later the decision is in <C>docs/adr/0012.md</C> and the code
            it governs is in four files nobody has linked to it, so the ADR is read when
            somebody already suspects it exists. Records point at spans, which is why they
            can be surfaced automatically rather than remembered.
          </>
        ),
      },
      {
        q: 'Who is this for?',
        a: (
          <>
            Anyone whose codebase is being edited by a coding agent often enough that
            &ldquo;it has no memory of last Tuesday&rdquo; has become an actual cost. It is
            useful solo — the store is a local file and needs nobody else to adopt it — and
            more useful on a team, because the store is committed and travels with the
            clone.
          </>
        ),
      },
      {
        q: 'What is the product, exactly?',
        a: (
          <>
            The CI exit code. Everything else — the lookup command, the hook, the anchoring
            — is plumbing that leads to a gate which can fail a build when a change damaged
            a recorded decision. If that gate is not worth having, the rest is a diary.
          </>
        ),
      },
    ],
  },

  {
    id: 'honesty',
    label: 'Does it work',
    questions: [
      {
        q: 'What would make you archive this project?',
        a: (
          <>
            Count the times an agent proposed a change that contradicted a recorded decision
            and this caught it. If that number is zero after three months of genuine daily
            use, the idea is wrong and the repo gets archived. That criterion was agreed
            before any code existed, so it cannot be quietly revised once there is something
            to defend.
          </>
        ),
      },
      {
        q: 'Has anyone other than the author run it?',
        a: (
          <>
            No. That is the honest status: the falsification clock above has not started,
            because a tool with one user and one repo cannot produce the number. Treat every
            claim on this page as verified on one codebase.
          </>
        ),
      },
      {
        q: 'How do I know a record is not an agent making something up?',
        a: (
          <>
            Records carry who wrote them. An agent-authored record is marked{' '}
            <C>UNCHECKED</C> until a human runs <C>confirm</C> on it, and the mark travels
            into the context block the next agent receives. Right now every record in this
            project&rsquo;s own store is human-written, because nothing writes records
            automatically yet — which is a schedule accident, not a control.
          </>
        ),
      },
      {
        q: 'What if the agent\u2019s stated reason is a plausible story rather than the real cause?',
        a: (
          <>
            Then this preserves confident nonsense durably, feeds it to the next agent as
            project history, and that agent writes a record leaning on the first. There is
            no mitigation for this. Marking records unchecked narrows it and does not close
            it, signing would prove who wrote a record rather than whether it is true, and
            the research on stated reasoning suggests the problem gets worse with more
            capable models, not better. It is the most serious doubt about the premise and
            it is why capture reads without writing.
          </>
        ),
      },
      {
        q: 'Could the kill number rise while the tool is rotting?',
        a: (
          <>
            Yes, and that is the flaw in it. A store full of wrong records produces{' '}
            <em className="text-silt not-italic">more</em> catches, because it flags people
            for contradicting things that were never true. So retractions are logged to a
            committed file rather than deleted quietly — how often a record turned out to be
            wrong is the only number that can see that failure, and it is deliberately kept
            separate from anything routine.
          </>
        ),
      },
      {
        q: 'Is the surfacing log the same as the catch count?',
        a: (
          <>
            No, and it must not be read as one. <C>.whence/surfaced.jsonl</C> counts how
            often records were put in front of an agent, which over-counts badly — most
            surfacings are informational and change nothing. The real number comes from{' '}
            <C>check</C>.
          </>
        ),
      },
    ],
  },

  {
    id: 'anchoring',
    label: 'Anchoring',
    questions: [
      {
        q: 'What happens when the code moves?',
        a: (
          <>
            The record follows it and costs nothing. Each record stores one hash per
            significant line of its span; when the recorded lines no longer match, the file
            is scanned for that exact hash sequence, and finding it means the block is
            byte-identical and somewhere else. The state reads <C>intact, moved</C>, and the
            display shows both where it is now and where it was recorded, because
            &ldquo;now at 187, recorded at 142&rdquo; is read faster than &ldquo;moved
            45&rdquo;.
          </>
        ),
      },
      {
        q: 'What happens when I reformat or reindent?',
        a: (
          <>
            Nothing moves. Lines are trimmed and blank lines are dropped before hashing, so
            whitespace changes are invisible to the anchor. This is also why a
            reformatting-only diff produces a touch rather than damage in CI.
          </>
        ),
      },
      {
        q: 'What happens when the block is rewritten?',
        a: (
          <>
            The anchor degrades to <C>altered</C> and reports how much of the recorded
            content survived, as a percentage. The measure is weighted by how rare each line
            is in the file, so a block rewritten down to its <C>func</C> line and its
            closing brace scores near zero rather than the 50% an unweighted count would
            give it. Below 60% survival the record is called orphaned instead.
          </>
        ),
      },
      {
        q: 'What happens when the code is deleted?',
        a: (
          <>
            The record is surfaced as <C>ORPHANED</C> and claims no line number at all —
            not its old one, not a nearby guess. A tool that confidently points at the wrong
            line teaches you to distrust everything it says, so being loudly uncertain is
            the intended behaviour. The decision text is still there to read; what is gone
            is the claim about where it applies.
          </>
        ),
      },
      {
        q: 'Why two separate readings instead of one confidence score?',
        a: (
          <>
            Because position and content are independent, and one number cannot say which
            one moved. A block can be byte-identical 400 lines away, or sitting exactly
            where it was recorded and half rewritten — opposite situations that a single
            score collapses. An earlier version charged a flat 0.90 for having moved; every
            record in a file people work in converged on that value within days, and a
            reading they all share tells you nothing about any of them.
          </>
        ),
      },
      {
        q: 'Why does the percentage not print on every record?',
        a: (
          <>
            It prints only where it was measured. The exact and moved states are proven
            byte-identical by the match itself, so there is nothing left to compute —
            printing <C>1.00</C> there would dress a constant up as a reading. The number
            appears on altered records because that is where a scan actually produced one.
          </>
        ),
      },
      {
        q: 'What if my span is all boilerplate?',
        a: (
          <>
            Recording it is refused at authoring time. A span whose every line occurs all
            over the file — closing braces, bare returns — cannot be found again by content,
            and searching for it anyway produces a confident answer about whichever
            lookalike is nearest. Failing when you are looking at the file and can widen the
            span is worth far more than failing at lookup six months later.
          </>
        ),
      },
      {
        q: 'Is tree-sitter / AST-path anchoring in there?',
        a: (
          <>
            No, and it is not scheduled. Per-line hashes already survive insertion,
            deletion, reindentation and whole-block moves, which is most real drift. AST
            paths buy the remainder — a block lifted into a different function, a signature
            rewritten around unchanged statements — for the price of a CGo dependency, and
            they wait until a real repository produces orphans the hashes cannot explain.
          </>
        ),
      },
      {
        q: 'Are the line hashes a copy of my code?',
        a: (
          <>
            No. Each is the first eight hex characters of a SHA-256 of the trimmed line, and
            a hash of a line is not the line. They are deliberately not salted with the file
            path, so a block that moves between files stays comparable — and the store is
            committed next to the code it describes anyway, which is where the plaintext
            already is.
          </>
        ),
      },
    ],
  },

  {
    id: 'gate',
    label: 'The gate',
    questions: [
      {
        q: 'What does whence check actually fail on?',
        a: (
          <>
            Three things, all of them damage: a recorded block that anchored cleanly in the
            base revision and is partly gone now, a record whose anchor was intact before
            the diff and does not resolve at all after it, and evidence that the diff
            deleted. Everything else prints and exits 0.
          </>
        ),
      },
      {
        q: 'Why does merely touching a recorded line not fail the build?',
        a: (
          <>
            Because a touch cannot be satisfied. You read the record, you agree with it, you
            change nothing, you re-run, and the identical finding comes back — there is
            nowhere to put the agreement. A touch that is not also an erosion is in practice
            a whitespace change, since any edit with text in it moves a hash and surfaces as
            erosion instead. Failing on touches means failing on <C>gofmt</C>, which means
            failing on every pull request, which means the gate gets switched off and takes
            the two findings nobody catches by eye with it.
          </>
        ),
      },
      {
        q: 'Then add a flag to acknowledge a finding?',
        a: (
          <>
            That was proposed and refused. A gate you clear by pasting a command on every
            pull request is one you stop reading before you clear, and it then reports that
            a human checked something when no human did. A rubber-stamped gate is worse than
            no gate, because it launders unchecked claims as checked ones.
          </>
        ),
      },
      {
        q: 'Does it decide whether my change is wrong?',
        a: (
          <>
            No, and it must not start. It reports coverage — these lines are governed by
            these decisions, go and confirm — and never a verdict. Judging a diff is code
            review; that category is well served and well funded, and a tool that drifts
            into it has lost the thing that makes it different.
          </>
        ),
      },
      {
        q: 'Does a pull request that adds a record fail its own gate?',
        a: (
          <>
            No. Records introduced by the same diff are not reported. Otherwise every pull
            request that documents a decision fails because it documented it, which is how a
            team learns to switch the gate off.
          </>
        ),
      },
      {
        q: 'What is evidence, and why can a record never cite another record?',
        a: (
          <>
            Evidence is a pointer to something checkable without consulting another record:
            a place in the code, a command anyone can run, a commit, an external artifact.
            Pointing at code is strongest because that pointer gets its own anchor and can
            rot on its own — so <C>check</C> can report that a record still reads as current
            while the thing that made it true was deleted, in a diff that never opened the
            record&rsquo;s file. Citing another record is refused at write time, because
            that single link is how one wrong record makes the next look credible. Wikipedia
            calls its version citogenesis and its rule is the same.
          </>
        ),
      },
      {
        q: 'Why does CI need fetch-depth: 0?',
        a: (
          <>
            <C>check</C> compares against the base revision — the base copy of the file and
            of the store — to tell &ldquo;this diff broke the anchor&rdquo; apart from
            &ldquo;the anchor was already broken&rdquo;. A shallow clone does not have the
            base to read, so the comparison cannot be made.
          </>
        ),
      },
    ],
  },

  {
    id: 'capture',
    label: 'Capture',
    questions: [
      {
        q: 'Capture is described first everywhere. Is it built?',
        a: (
          <>
            Not the half that writes. Records are still authored by hand or harvested from
            comments that already exist. Deciding which slice of a session is worth keeping
            is the hard problem, and curated records are the specification for solving it —
            shipping a naive version that writes plausible-sounding reasoning into a
            committed shared store would manufacture exactly the failure this project is
            most worried about.
          </>
        ),
      },
      {
        q: 'Then what does whence capture do?',
        a: (
          <>
            It reads. Given a finished Claude Code session it prints every edit beside the
            span it touched, the request that prompted it, and what was said immediately
            before it — then stops. Nothing reaches the store, and the last line of the
            output says so. It is the instrument the writing half is waiting on: the
            question of whether stated reasoning is the actual cause has to be answered by
            somebody reading real pairs, and a capture that wrote records would be answering
            it by assumption.
          </>
        ),
      },
      {
        q: 'Why does it read a file instead of hooking the session?',
        a: (
          <>
            Because the file already exists. Claude Code writes every session to disk and
            keeps it, so there was nothing to intercept — no second hook, no trail to
            maintain, and no cost added to any edit. What was missing was a reader. The
            deliberation is not in there, though: thinking blocks persist with their text
            dropped, so what survives is the explanation given to the user rather than the
            weighing that produced it. That is a limit of the source, not a detail of the
            implementation.
          </>
        ),
      },
      {
        q: 'What has to be true before capture ships?',
        a: (
          <>
            A measured faithfulness rate for stated reasoning, and a human-authored fraction
            that cannot reach zero. The standing rule is that humans author the rare and
            load-bearing records while capture handles the routine, never the reverse —
            self-feeding stores lose their rare cases first, and the rare load-bearing
            exception is the entire product.
          </>
        ),
      },
      {
        q: 'Has anything been measured yet?',
        a: (
          <>
            A little, and it cuts both ways. Reasoning is stated <em>before</em> each edit
            rather than after, which is not the shape post-hoc rationalisation takes — a
            claim made in advance can be checked against what the edit did. But counting how
            often a reason is given turns out to be harder than the counting: a keyword test
            reports a floor and misses reasons carried by sentence structure instead of by a
            conjunction, so the honest figure is a range, not a number. That is one project
            read by its own author, which is the weakest possible evidence and still more
            than existed before.
          </>
        ),
      },
      {
        q: 'Would you ship a model to extract records?',
        a: (
          <>
            No. Tested against this repository&rsquo;s full commit history with a cold
            reader: extraction quality was adequate and no quotes were fabricated, so there
            is no capability gap for a specialised model to fill. The commitment that came
            out of it is that this must never become an API client — it can emit candidates
            and something else can do the reading.
          </>
        ),
      },
      {
        q: 'Can it backfill from commit messages?',
        a: (
          <>
            Not as tried, and the attempt was abandoned on measurement. About 84% of
            proposals could not name a file, and of those that did, several named logs, a
            deleted file, or a path that never existed — two of sixty-two were anchorable.
            Worse, git history is a decision stream: it produced both &ldquo;rename the
            binary&rdquo; and &ldquo;rename it back&rdquo; with nothing marking which one is
            live, so adding all of them yields a store that contradicts itself.
          </>
        ),
      },
    ],
  },

  {
    id: 'security',
    label: 'Security and privacy',
    questions: [
      {
        q: 'What does it send anywhere?',
        a: (
          <>
            Nothing. There is no network code in the binary — no HTTP client, no telemetry,
            no update check, no account. The only external process it runs is{' '}
            <C>git</C>, in <C>check</C>, to read a diff.
          </>
        ),
      },
      {
        q: 'Why is the store committed instead of gitignored?',
        a: (
          <>
            Because a record only helps the next person if it is there when they clone. The
            whole claim is that reasoning should travel with the code the way the code
            travels, so <C>.whence/records.jsonl</C> is committed on purpose. Nothing in the
            tool reads git state, so anyone who wants their records local can gitignore the
            file themselves and everything keeps working.
          </>
        ),
      },
      {
        q: 'Records go into agent context. Is that not a prompt-injection vector?',
        a: (
          <>
            It is, and it is treated as one. Records arrive by <C>git pull</C>, so anyone
            who can land a commit can put text in front of your agent. Every injected block
            is framed as historical notes for information, explicitly not instructions to
            follow. That framing is prompt-only filtering, which the memory-poisoning
            literature finds insufficient on its own — it is kept because it is free and it
            addresses obedience, not because it settles the problem.
          </>
        ),
      },
      {
        q: 'Will it commit a secret my agent saw?',
        a: (
          <>
            Nothing is captured automatically today, so the only way a secret enters the
            store is if you type it into a record or it sits in a comment that{' '}
            <C>backfill</C> harvests. Backfill stores the comment text verbatim, so a
            comment containing a live key would be copied into a file meant to be committed
            — worth knowing before running it across an unfamiliar repository. Redaction
            belongs at capture, before the write, because git history is effectively
            append-only and deleting the file later is not recovery.
          </>
        ),
      },
      {
        q: 'whence capture reads my session transcripts. Where does that go?',
        a: (
          <>
            To your terminal, and nowhere else. It opens a session file your agent already
            wrote, on your own disk, prints what it found and exits — it does not copy the
            transcript, write to the store, or reach the network. The binary contains no
            HTTP client at all. It does print what you typed and what the agent replied, so
            treat the output like any other terminal session before pasting it somewhere.
            Edits to files outside the store you are asking about are counted and not shown.
          </>
        ),
      },
      {
        q: 'Is there per-developer AI-authorship reporting?',
        a: (
          <>
            No, and there will not be. Attribution stays aggregate by default. This is a
            developer tool and a leaderboard of who let an agent write what is the fastest
            way to be banned by the team that has to install it.
          </>
        ),
      },
      {
        q: 'Are records signed?',
        a: (
          <>
            No. Signing lands with capture, because it only matters once records stop being
            written by a human. It is also worth being clear about what it would buy:
            signatures prove who wrote a record, never whether it is true, so they answer
            the deliberate-forgery threat and do nothing about the sincere-mistake one.
          </>
        ),
      },
    ],
  },

  {
    id: 'setup',
    label: 'Setup and day-to-day',
    questions: [
      {
        q: 'Does it slow my agent down?',
        a: (
          <>
            The hook measures 6.3ms per fire — 100 invocations in 0.629 seconds — and most
            of that is process startup rather than the file read. That is why there is no
            daemon and no cache: a persistent process would win back almost nothing.
          </>
        ),
      },
      {
        q: 'What happens if it breaks or hangs?',
        a: (
          <>
            The hook fails open on every path. Every error exits 0 having printed nothing,
            so a broken or slow install costs you a missing record and never a blocked edit.
            It runs synchronously before every edit in every session, which makes that the
            only acceptable behaviour.
          </>
        ),
      },
      {
        q: 'Why does typing whence do nothing in my shell?',
        a: (
          <>
            zsh and ksh ship a <C>whence</C> builtin, and builtins beat <C>$PATH</C>.
            Aliases are resolved before builtins, so <C>alias whence=&apos;command
            whence&apos;</C> in your rc file is the entire fix, and <C>command whence</C>{' '}
            works for a one-off. If you would rather not shadow a builtin you use,
            symlink the binary to any other name. bash, fish, hooks and CI need
            nothing — they have no such builtin.
          </>
        ),
      },
      {
        q: 'Does it work outside Claude Code?',
        a: (
          <>
            The lookup command and the CI gate work anywhere; they are a binary and a git
            diff. The hook integration is written for Claude Code&rsquo;s{' '}
            <C>PreToolUse</C> payload specifically, so other agents would need their own
            adapter. None is built, and building one before somebody asks would be adding a
            payload shape to test for demand that has not been observed.
          </>
        ),
      },
      {
        q: 'Does it work outside Go?',
        a: (
          <>
            Yes. Anchoring hashes lines, not syntax, so it is language-agnostic by
            construction. Backfill recognises comments by prefix, covering <C>//</C>,{' '}
            <C>#</C> and block-comment continuations — a run across a large open-source
            repository harvested records from TypeScript files as well as Go. A language
            that comments some other way is not harvested, which is silent but absent rather
            than silently wrong.
          </>
        ),
      },
      {
        q: 'Where does it look for the store?',
        a: (
          <>
            It walks up from the file being edited to the nearest <C>.whence/</C>, the way
            git finds <C>.git</C> — never from the session&rsquo;s working directory. A
            session rooted in one repository routinely edits a file in a sibling one, and
            resolving from the current directory would search the wrong store and match
            nothing, silently.
          </>
        ),
      },
      {
        q: 'My store is empty. Do I have to write records by hand?',
        a: (
          <>
            Start with <C>backfill</C>, which harvests decisions already sitting in your
            code. <C>HACK</C>, <C>WORKAROUND</C>, <C>XXX</C> and <C>GOTCHA</C> are taken
            always, because the word is itself the admission that a choice was made against
            a constraint. <C>NOTE</C>, <C>TODO</C>, <C>FIXME</C>, <C>WARNING</C> and{' '}
            <C>CAVEAT</C> are taken only when the note gives a reason, which is the whole
            difference between a store worth reading and one full of &ldquo;fix this
            later&rdquo;. A decision says why; a task only says what.
          </>
        ),
      },
      {
        q: 'Does backfill find every reason in my code?',
        a: (
          <>
            No. The list of reason words is deliberately narrow and does miss real phrasings
            around it. That is the intended trade: a missed note you can add by hand, and a
            garbage record in a committed shared store you cannot take back.
          </>
        ),
      },
      {
        q: 'reground, reanchor, rm — which one do I want?',
        a: (
          <>
            <C>reground</C> when the evidence moved but the claim is unchanged.{' '}
            <C>reanchor</C> when the code the record described was rewritten and the
            decision is still true of what replaced it. <C>rm</C> only when the record was{' '}
            <em className="text-silt not-italic">wrong</em>. The split exists because{' '}
            <C>rm</C> writes to a committed retraction log that counts how often a record
            turned out to be false — filing routine bookkeeping there would destroy the one
            number that can see a store degrading.
          </>
        ),
      },
      {
        q: 'Why does reanchor make me name the lines?',
        a: (
          <>
            Because where a degraded record currently points is a best-match window of a
            fixed number of significant lines, so it sits a line or two off the real block
            about as often as not. Re-hashing that window would store a guess as a
            certainty, which is the exact failure the anchoring design exists to prevent.
            The command prints the window, so agreeing costs a paste — the point is that
            agreeing is an act somebody performs.
          </>
        ),
      },
      {
        q: 'What happens when two people add records on two branches?',
        a: (
          <>
            They merge. Records are one compact line each in <C>records.jsonl</C>, and a{' '}
            <C>.gitattributes</C> beside the store marks it <C>merge=union</C> so the merge
            takes both sides. Both halves are required — measured in a scratch repository,
            line-delimited without the union driver still conflicts. The accepted cost, named
            rather than discovered later: a record <em className="text-silt not-italic">
            edited</em> on both branches survives twice, so one id can appear in the log
            twice.
          </>
        ),
      },
    ],
  },

  {
    id: 'scope',
    label: 'Scope and comparisons',
    questions: [
      {
        q: 'Is this an AI code reviewer?',
        a: (
          <>
            No, and the failure mode of the project is becoming one. It produces zero
            opinions about code quality. Review tools generate judgements about a diff;
            this reports which recorded decisions govern the lines a diff touches, and
            leaves the judgement to you.
          </>
        ),
      },
      {
        q: 'Is it an observability platform?',
        a: <>No. No traces, no evals, no token dashboards, no per-session spend.</>,
      },
      {
        q: 'Is it a codebase knowledge graph?',
        a: (
          <>
            No. It answers one question about one line. Graph tools model the whole
            repository and are a well-populated category; this is deliberately much smaller.
          </>
        ),
      },
      {
        q: 'How is it different from git notes?',
        a: (
          <>
            <C>git notes</C> attaches text to a commit, so the note is anchored to a point
            in history rather than to a span of code, and it does not follow the lines as
            they move. Notes also do not travel by default — they need their own refspec —
            which is the opposite of the property that makes a record useful to whoever
            clones next.
          </>
        ),
      },
    ],
  },

  {
    id: 'project',
    label: 'Project status',
    questions: [
      {
        q: 'How do I install it?',
        a: (
          <>
            <C>go install github.com/Amag1n3/whence@latest</C>, on Go 1.22 or newer.
            Building from a clone works too and is the better path if you intend to
            edit the code.
          </>
        ),
      },
      {
        q: 'What are the dependencies?',
        a: (
          <>
            None beyond the Go standard library. That is deliberate rather than an accident
            of youth: this runs synchronously before every edit an agent makes, and a
            dependency tree is a supply-chain surface in a process that reads your source
            and writes a committed file.
          </>
        ),
      },
      {
        q: 'Is it stable? Will records written today keep working?',
        a: (
          <>
            The store format changed once, from a single JSON array to one record per line,
            and the old shape is still read and never written so that upgrading cannot
            silently empty anyone&rsquo;s decisions. A malformed line is an error with a
            line number rather than a skipped record — dropping a decision quietly is the
            failure this whole tool is about. It is days old; treat it accordingly.
          </>
        ),
      },
      {
        q: 'What is on the roadmap?',
        a: (
          <>
            Nearest is reading enough captured sessions to answer the faithfulness question
            the writing half is blocked on. Beyond that:
            harvesting from more sources than comments, record signing alongside capture,
            and eventually team sync with a dashboard. AST paths and anything in the last
            group wait for a real repository to demand them — several features on this page
            were killed by measuring them first.
          </>
        ),
      },
      {
        q: 'What is the licence?',
        a: (
          <>
            MIT. Lowest friction for individual developers, who adopt a command-line tool
            first.
          </>
        ),
      },
    ],
  },
]
