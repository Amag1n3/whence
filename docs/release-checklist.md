# Cutting a GitHub Release

Do this **after** the v0.5.0 prep PR is merged to `main`. Do not tag from the
PR branch. This repo has Git tags (`v0.4.0`, …) but no GitHub Release objects;
`gh release list` is empty until the first `gh release create`.

`@latest` follows the newest tag. Tagging is what makes the install line in
the README true.

## 1. Tag from main

```bash
git checkout main
git pull origin main
git tag -a v0.5.0 -m "v0.5.0"
git push origin v0.5.0
```

## 2. Create the Release

Paste the notes below. `--notes` is the body people see; `--title` is the
heading.

```bash
gh release create v0.5.0 --title "v0.5.0" --notes-file - <<'EOF'
Codex and OpenCode surfacing, symlink-safe file reads, site/llms/privacy, install docs that match how `@latest` actually works.

`go install github.com/Amag1n3/whence@latest` now resolves here. `whence --version` prints the version Go stamped at build time — that flag shipped in v0.4.0; this tag is the first with a GitHub Release object.

## Surface

- **Codex CLI:** copy `.codex/hooks.json`, restart, `/hooks`, trust the hook. Matcher includes `apply_patch`. Until trusted, Codex skips it.
- **OpenCode:** drop `.opencode/plugins/whence.js` and add the AGENTS.md line. Lookup tool, not automatic. Does not write `.whence/surfaced.jsonl`.
- Claude Code plugin is unchanged.

## Anchoring

- `fileLinesWithin` resolves symlinks before reading, matching the outside-root guard. A symlink inside the repo cannot hash a file the record never concerned.

## Site and docs

- whence.fyi: `/llms.txt`, privacy page, no-JS homepage copy, 404, sitemap.
- README: `@latest` is the newest Git tag, not `main`.
- `example/demo.sh`: lookup a curated record, mutate the anchored lines, `whence check` exits 1.

## Not in this tag

- Capture still does not write records. `whence capture` is read-only.
- `--version` is not new here; it is in v0.4.0.
- CodeHype footer badge add/revert is not a product change.

Full log: CHANGELOG.md
EOF
```

## 3. Check the install lie is gone

```bash
go install github.com/Amag1n3/whence@v0.5.0
# then, once the tag is the newest:
# go install github.com/Amag1n3/whence@latest
whence --version
```

Expect `whence v0.5.0`. A binary from `go build` on an untagged commit will
print a pseudo-version, not `v0.5.0`.
