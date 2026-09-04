# Changelog

User-visible changes. Internal-only site noise (badge add/revert) is omitted.

`go install github.com/Amag1n3/whence@latest` installs the newest **Git tag**,
not `main`. A merge is invisible to `@latest` until it is tagged.

## Unreleased — v0.5.0

Not tagged yet. Until `git tag -a v0.5.0` and `gh release create` run (see
[docs/release-checklist.md](docs/release-checklist.md)), `@latest` still
installs v0.4.0.

### Surface

- Codex CLI: ship `.codex/hooks.json`. The hook matcher includes `apply_patch`;
  `whence hook pre` reads paths from `*** Update File:` / `*** Add File:`
  lines. Until you `/hooks` and trust it, Codex skips it.
- OpenCode: ship `.opencode/plugins/whence.js`. It is a lookup tool the agent
  has to call, not a PreToolUse hook. It does not write `.whence/surfaced.jsonl`.

### Anchoring

- `fileLinesWithin` resolves symlinks before the outside-root guard, so a
  symlink inside the repo cannot turn a record into a hash oracle on a file
  the record never concerned.

### Site and docs

- whence.fyi: static homepage prose for no-JS crawlers, `/llms.txt`, a privacy
  page (the site and CLI collect nothing), a real 404, sitemap.
- README install note no longer pins v0.4.0 as “current”.
- `example/` is a runnable lookup-then-`check` walkthrough (curated records,
  not capture).

## v0.4.0 — 2026-08-17

Already tagged. Includes `whence --version` / `-v`: it prints the version Go
stamped at build time (`debug.ReadBuildInfo`), not a hand-maintained constant.
A stale `go install` reports the tag it came from. There was no GitHub Release
object for this tag — tags existed, Releases API was empty.
