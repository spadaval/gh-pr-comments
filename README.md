# gh-pr-comments

`gh-pr-comments` is a focused GitHub CLI extension that adds two missing capabilities for pull request inline review comments:

- `gh pr-comments list`
- `gh pr-comments create`

It is intentionally minimal and does not reimplement broader `gh pr` functionality.

## Commands

### List inline comments

```bash
gh pr-comments list [<number> | <url>] [-R <owner/repo>] [--pr <number>]
```

Outputs PR metadata and inline review threads/comments as JSON.

### Create an inline comment

```bash
gh pr-comments create [<number> | <url>] \
  --path <file> \
  --line <line-number> \
  --body "<comment>" \
  [--side LEFT|RIGHT] \
  [--start-line <line>] [--start-side LEFT|RIGHT] \
  [-R <owner/repo>] [--pr <number>]
```

Creates a new inline review thread comment and outputs created comment details as JSON.

## PR/Repo Inference

PR resolution is delegated to `gh pr view --json url` so behavior matches normal `gh pr` semantics as closely as possible:

- If you pass a selector (`<number>` or `<url>`), it uses that PR.
- If you pass `--pr`, that PR number is used.
- If you pass neither, it infers the PR from the current branch context (same behavior as `gh pr view`).
- If `-R/--repo` is provided, it is forwarded to `gh`.

## Installation

```bash
gh extension install <owner>/gh-pr-comments
```

This extension is installed as a source extension and runs via `go run`, so Go must be installed and available in `PATH`.
