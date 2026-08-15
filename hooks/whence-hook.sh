#!/bin/sh
# Locate the whence binary and hand it the PreToolUse payload on stdin.
#
# This script exists for one reason: a plugin can ship hook configuration but
# cannot install a Go binary, so the hook has to find one that may not be there.
# A bare `whence hook pre` in hooks.json would have been shorter and would fail
# on the most common setup — hooks do not reliably inherit $PATH from a shell
# profile, which is why the README has always told people to write an absolute
# path. A plugin cannot know that path in advance, so it searches.
#
# FAIL OPEN, ALWAYS, and here that means twice over. The binary itself exits 0
# printing nothing on every error path; this script has to hold the same line
# for the cases the binary cannot cover — absent, unreadable, or not the program
# we meant. Someone who installs the plugin and never installs the tool must get
# silence, not a broken hook on every edit in every session. Silence is also what
# a working install with nothing to say looks like, which is the intended cost.
#
# Deliberately POSIX sh and dependency-free. whence has no runtime dependencies
# and the plugin does not get to add one to a script that runs before every edit.

# run executes a candidate and ALWAYS exits 0, whatever the binary did.
#
# Not `exec`, and that is the one deliberate cost here — exec would replace this
# shell and save a process on a path that fires before every edit, but it also
# forwards the binary's exit code, and a non-zero code from a PreToolUse hook is
# read by Claude Code as the hook objecting to the edit. That turns any crash,
# any older build, any unrelated program that happens to be called `whence` into
# a tool that blocks edits — the exact failure the fail-open rule exists to
# prevent, reintroduced by the wrapper meant to serve it. One extra process is
# cheap; blocking every edit in a session is not.
#
# stdout still flows straight through to Claude Code, which is what carries the
# injected context.
# Which half of the hook this is: `pre` surfaces records before an edit, `post`
# records the reason after one. hooks.json passes it.
MODE=$1

# run executes a candidate and ALWAYS exits 0, whatever the binary did.
#
# $1 inside the function is the binary; MODE came from the script's own arguments
# above, before any function was called.
run() {
	"$1" hook "$MODE"
	exit 0
}

# An explicit override wins, for anyone with a non-standard install or testing a
# build. Checked first so it can also point at a specific binary on purpose.
if [ -n "$WHENCE_BIN" ] && [ -f "$WHENCE_BIN" ] && [ -x "$WHENCE_BIN" ]; then
	run "$WHENCE_BIN"
fi

# Then the usual places, before $PATH. `go install` lands in the first, honouring
# GOBIN and GOPATH when set, and a hand-placed binary usually lands in the second
# — which on this machine sits EARLIER on $PATH than ~/go/bin, so a stale copy
# there silently shadowed every reinstall for a day. Preferring the Go path makes
# `go install` the thing that decides which binary runs.
for dir in \
	"${GOBIN:-${GOPATH:-$HOME/go}/bin}" \
	"$HOME/.local/bin" \
	/usr/local/bin \
	/opt/homebrew/bin
do
	if [ -f "$dir/whence" ] && [ -x "$dir/whence" ]; then
		run "$dir/whence"
	fi
done

# $PATH last. `command -v` rather than `which` because it is POSIX, but its
# answer is only trusted when it names a real executable file: ksh has a `whence`
# BUILTIN, and a builtin resolves here to the bare word rather than to a path.
# Handing the payload to that would be handing it to the wrong program.
resolved=$(command -v whence 2>/dev/null) || resolved=""
if [ -n "$resolved" ] && [ -f "$resolved" ] && [ -x "$resolved" ]; then
	run "$resolved"
fi

# Not installed. Say nothing, cost nothing.
exit 0
