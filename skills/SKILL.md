---
name: drafts-applescript-cli
description: Use this macOS AppleScript CLI to create, query, modify, organize, and submit actions on Drafts, with capability detection and explicit destructive commitment.
---

# Drafts AppleScript CLI

Use the ordinary `drafts` commands. Confirm `drafts version` identifies this repository because the official Drafts MCP project also ships a command named `drafts`.

## Setup

Requires macOS, Drafts Pro, and a running selected Drafts application for library commands. macOS Automation permission belongs to the calling terminal. `$EDITOR` and `fzf` are optional, used only by interactive commands. No helper app/action or API key is needed.

```sh
drafts apps
drafts schema --detected
drafts info
```

Select the exact app using `--app /Applications/Drafts.app`. `--channel auto|stable|beta` uses bundle evidence; unknown channel is never guessed from a version. Features are gated by dictionary capabilities, including optional flag types and workspace opening. Stable builds can support the same capabilities as beta builds.

## Safe workflow

```sh
drafts create --input @payload.json --dry-run
drafts create --input @payload.json --idempotency-key meeting-2026-10-05
drafts list --limit 5 --sort modified --fields uuid,title
drafts get 12345678-1234-1234-1234-123456789abc --format raw
drafts replace -u 12345678-1234-1234-1234-123456789abc --text-file note.md --dry-run
# Only after the user authorizes replacement:
drafts replace -u 12345678-1234-1234-1234-123456789abc --text-file note.md --commit
drafts trash 12345678-1234-1234-1234-123456789abc --dry-run
drafts run --input '{"action":"Copy","uuid":"12345678-1234-1234-1234-123456789abc"}' --dry-run
```

- Replace/update and trash/delete require explicit UUIDs. Other supported commands can omit the UUID to target the active draft. Validate returned IDs before using them.
- Preview mutations with `--dry-run`. Previews send no Apple events and do not prove target/action availability.
- Replace/update, edit, trash/delete, and self-upgrade require `--commit`. Permission to inspect a library does not authorize its mutation or action execution.
- `run` accepts exactly one of `uuid` or `content`. Content creates a persistent draft. Status `submitted` never promises completion. Actions may perform network or destructive work.
- Create/action preflight checks action existence before changing content. Partial failures expose UUID and completed steps. Do not repeat completed steps.
- Use `--idempotency-key` on create/run to prevent duplicates. `drafts jobs <key>` inspects durable receipts. An uncertain or interrupted request is blocked. Do not bypass it by changing the key without inspecting the target.
- Never infer private JavaScript API access from the external AppleScript dictionary. Syntax, tasks, AI APIs, version history, and action-completion polling are unsupported.
- Never read Drafts' private database directly or install the obsolete eval-based helper action.

## Composition and bounded output

JSON is the default. Success goes to stdout; structured errors go to stderr. Use `--format-error plain` only when requested. Exit codes: 2 validation/commit, 3 not found/not running, 4 permission, 5 unsupported, 6 conflict, 7 timeout, 8 partial failure, 9 execution/output/update error. Timeouts can follow a completed mutation; inspect before retrying.

Use `drafts schema <command>` and `<command> --help` for supported flags and payloads. `list` defaults to 20 summaries. `--full` fetches returned bodies after limiting. `--count`, `--id-only`, and `--fields` reduce tokens. `--sort created|modified|accessed`, date bounds, omitted tags, and optional flag types are supported. Dates are UTC RFC3339 with second resolution. Search is literal containment, not Drafts' full query language.

Use `--input @payload.json` or `--input -` for raw JSON. Required content must be present; explicit empty content is allowed. Null, unknown, duplicate, and conflicting inputs fail. Use `--text-file` or explicit `--stdin` for exact UTF-8 content, including final newlines. Do not combine sources. Stdin has a 16 MiB cap and timeout. No shell expansion is applied to `$EDITOR` arguments.

`--deliver file:<path>` writes mode-600 output atomically, requiring `--overwrite` for existing paths. `--data-source local` on list/get reads an explicitly created CLI snapshot; it never refreshes automatically. `sync --full` persists private content only when explicitly invoked. Counts/search are bounded to cached rows and include provenance.

Profiles store app/channel/timeout only: explicit flag > environment > selected profile > defaults. `drafts config` reports sources. Feedback stays local with `drafts feedback`; do not record private content or secrets.

## Repository work

Follow `AGENTS.md`. Run `make verify` for safe synthetic tests and contract checks. Never run live integration tests without explicit authorization for a disposable library. Optional dictionary compilation uses `DRAFTS_DICTIONARY_APP` and sends no Drafts events. Do not commit, push, release, self-upgrade, or publish without explicit user authorization.
