---
name: gh-pr-comments
description: List and create GitHub pull request inline review comments with JSON output for agents
---

# gh-pr-comments

A focused GitHub CLI extension for pull request inline review comments.

Supported operations:
- `gh pr-comments list`
- `gh pr-comments create`

## When to Use

Use this skill when you need to:
- Read inline review threads/comments for a PR as JSON
- Post a new inline comment thread at a specific file/line
- Build agent workflows around PR comment state

Use built-in `gh pr` commands for broader review lifecycle actions (pending reviews, approve/request changes, etc.).

## Installation

```sh
gh extension install <publisher>/gh-pr-comments
```

## Commands

### 1. List Inline Review Threads

```sh
gh pr-comments list
```

Returns:
- `pull_request`: resolved PR identity metadata
- `threads`: inline review threads with comments

Thread fields:
- `id`
- `path`
- `line` (optional)
- `start_line` (optional)
- `is_resolved`
- `is_outdated`
- `comments[]` with `id`, `body`, `author`, `created_at`, `url`

### 2. Create Inline Review Comment

```sh
gh pr-comments create \
  --path <file> \
  --line <line-number> \
  --body "<comment>" \
  [--side LEFT|RIGHT] \
  [--start-line <line>] [--start-side LEFT|RIGHT]
```

Required flags:
- `--path`
- `--line`
- `--body`

Defaults:
- `--side RIGHT`

Returns:
- `pull_request`: resolved PR identity
- `comment`:
  - `thread_id`
  - `comment_id`
  - `path`
  - `line` (optional)
  - `start_line` (optional)
  - `author`
  - `body`
  - `created_at`
  - `url`
  - `is_resolved`
  - `is_outdated`
  - `requested_side`

## PR Resolution Rules

Commands are intended to run from a checked-out branch that maps to an open pull request.

PR selection follows the same branch-context inference as `gh pr view --json url`.

## Validation and Behavior Notes

- `--line` must be greater than `0`
- `--start-line` must be greater than `0` when provided
- `--side` and `--start-side` must be `LEFT` or `RIGHT`
- Empty/whitespace `--path` or `--body` is rejected
- List fetches up to `100` threads and up to `100` comments per thread

## Common Agent Workflows

### Get inline threads for the current PR

```sh
gh pr-comments list
```

### Create a single-line inline comment

```sh
gh pr-comments create \
  --path internal/comments/service.go \
  --line 57 \
  --body "Please add a unit test for this branch."
```

### Create a multi-line inline comment

```sh
gh pr-comments create \
  --path cmd/create.go \
  --line 54 \
  --start-line 48 \
  --side RIGHT \
  --start-side RIGHT \
  --body "Consider splitting validation and API call setup into separate helpers."
```
