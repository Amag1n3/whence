---
description: Check that whence is installed, wired up, and has a store to read — and fix whatever is missing.
allowed-tools: Bash(command -v whence), Bash(whence:*), Bash(ls:*), Bash(go install:*), Read, Glob
---

A silent hook and a broken hook look identical from the outside, which is the
cost of the fail-open rule and the reason this command exists. Work through the
checks in order and stop at the first one that fails — each later check assumes
the earlier ones passed.

## 1. Is the binary installed?

```
command -v whence || ls -la "${GOBIN:-${GOPATH:-$HOME/go}/bin}/whence"
```

The plugin ships the hook configuration; it cannot ship a Go binary. If nothing
is found, this is the whole problem — the hook has been exiting silently, which
is by design and indistinguishable from having nothing to say.

Give the user this and let them run it:

```
go install github.com/Amag1n3/whence@latest
```

Requires Go 1.22+. If they would rather not put it on `$PATH`, `WHENCE_BIN`
pointing at any executable copy is honoured by the hook ahead of every other
location.

**If `command -v whence` printed a path but it is not the Go one, say so.** A
hand-placed binary in `~/.local/bin` shadowing a newer `go install` build has
cost a full day on this project before. The hook prefers the Go path precisely
so reinstalling is what decides which binary runs, but an interactive shell will
still reach the stale one.

## 2. Is there a store to read?

```
whence log
```

`no .whence/records.jsonl found above <dir>` means the repository has no store
yet, and a store with no records is a tool that does nothing. Do not create one
by hand — harvest what the codebase has already written down:

```
whence backfill
```

That reads `HACK:` `WORKAROUND:` `XXX:` `GOTCHA:` `ponytail:` comments always,
and `NOTE:` `TODO:` `FIXME:` `WARNING:` `CAVEAT:` only where the note gives a
reason. It is a dry run by default — it shows what it found and writes nothing,
because the store is committed and shared. Show the user the list, and only when
they agree does it get written:

```
whence backfill --yes
```

Anything matching a credential shape is refused outright, so a key sitting in a
comment is never copied into the store. A bad record still costs more than a
missing one.

## 3. Is the hook actually firing?

The honest check is the log, not a guess:

```
ls -la .whence/surfaced.jsonl
```

That file gains a line every time records are put in front of an agent. If it
does not exist after an edit to a file that has records, the hook is not
reaching the binary. Have the user restart Claude Code — hook configuration is
read at startup — and confirm the plugin is enabled.

To test the path directly without waiting for an edit:

```
echo '{"cwd":"'"$PWD"'","tool_input":{"file_path":"'"$PWD"'/<a file with records>"}}' | whence hook pre
```

JSON on stdout means the whole chain works. Empty output means no records match
that file, which is not a failure.

## 4. Optional: gate CI on it

Only raise this if the first three passed and the repository has CI. `whence
check` compares a diff against the store and exits 1 **only** for records the
change damaged — eroded, orphaned, or evidence deleted. Records that merely
cover the changed lines print and pass, because a gate that fires on proximity
fails on `gofmt` and gets switched off.

`fetch-depth: 0` is required: `check` reads the base revision of the store and
of every cited file.

## Reporting back

Say plainly which checks passed and which did not. If everything passed and the
store is empty, say that too — it means the tool is working and has nothing to
say yet, which is the one state users read as breakage.
