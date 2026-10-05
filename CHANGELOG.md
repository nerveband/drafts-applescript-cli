# Changelog

## 4.0.0

- Discover stable and beta app bundles, and enable optional features from their actual AppleScript dictionaries.
- Serialize draft responses as structured JSON, preserving Unicode, newlines, tags, flags, and UTC dates.
- Add folders, tag removal, flag types, date filters, sorting, counts, workspace rename, and scoped snapshots.
- Add strict JSON input, explicit stdin/file content, field selection, raw/JSONL output, and atomic file delivery.
- Add profiles, durable idempotency receipts, local feedback, and embedded agent guidance.
- Require explicit commit flags for destructive operations and self-update. Preview mutations without Apple events.
- Remove automatic network update checks. Verify checksums for explicitly requested upgrades.
- Add synthetic CLI/runtime tests, dictionary compilation, fixture-only live test tooling, CI, and both macOS release architectures.
- Include the MIT license with maintainer-confirmed redistribution rights.

### Migration

- Go installs and imports use the major-version module path: `github.com/nerveband/drafts-applescript-cli/v4`.
- Automation requires Drafts running. Select an app explicitly when discovery is ambiguous.
- Pipe content only with `--stdin` or `--input -`; omitted content fails immediately.
- Use `--commit` for replace, edit, trash/delete, workspace rename, and upgrade.
- Lists return at most 20 summaries by default. Use `--full` for bodies and `--limit 0` for all matching summaries.
- Ordinary mutations return receipts. Actions report submission, not completion.
- Keep JSON errors on stderr and check exit codes. Use `--format plain`, `--format raw`, or `--format jsonl` when needed.
