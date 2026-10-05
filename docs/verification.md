# Verification and provenance

Implementation version: `4.0.0`. Assessment date: 2026-10-05.

The user authorized implementation, live tests using uniquely tagged fixtures in the current library, an independent Opus 5.5 review at xhigh, GitHub release, and deployment to /Users/nerveband/bin/drafts with a backup. The maintainer confirmed MIT redistribution rights. No dependencies were installed and no existing private draft bodies were read. Demo assets and historical plans were preserved. This record covers candidate validation; publication and deployment results are reported in the release/thread.

## Checks performed

| Check | Result |
|---|---|
| Offline synthetic `go test -mod=readonly ./... -count=1` | Passed, CLI subprocesses use fake apps and fake osascript |
| `go vet -mod=readonly ./...` | Passed |
| Darwin arm64 and amd64 builds | Passed, binaries written to temporary paths |
| Native dictionary compilation | Passed for generated create, modify, folder, tag, flag, get, query, count, open, workspace rename, action, and resource scripts |
| Foundation JSON/date smoke tests | Passed with synthetic records, tabs, newlines, Unicode, delimiter-like tags, boolean/type flags, winter and summer UTC dates |
| Integration gate check with no opt-in environment | Live test skipped with explicit gate message |
| `bash -n build.sh release.sh` | Passed |
| `goreleaser check .goreleaser.yaml` | Passed after replacing deprecated archives.builds with ids |
| `git diff --check` | Passed |
| Default Go cache | Initially blocked by sandbox; rerun successfully with a temporary cache |
| govulncheck | Unavailable locally; not installed or run |
| Self-update network/download/checksum/replacement | Passed on a temporary binary built as 3.0.1, updated to the existing published 3.0.2; installed CLI unchanged |
| Real Drafts library operations | Passed 39 fixture lifecycle/safety checks plus 3 targeted receipt/preflight checks after Automation permission was allowed. All created fixtures moved to Trash, including the recovered initial timed-out create. |
| Editor and action behavior | Live pseudo-terminal editor tests passed case-only and Unicode-plus-case conflicts, concurrent-content preservation, and unchanged decomposed Unicode editing. Missing-action preflight passed. No existing user action was submitted; completion is unavailable. |
| GitHub-hosted CI | Required before publication; see the [Verify workflow](https://github.com/nerveband/drafts-applescript-cli/actions/workflows/verify.yml) for the final commit's result |
| Snapshot release archives | Both macOS architectures packaged, checksums matched, archive contents inspected, arm64 version command executed |
| GitHub account/repository | Authenticated as nerveband, repository ADMIN access, remote main matches local base, latest release v3.0.2 |
| Independent reviewer | Opus 5.5 read-only review completed with verified effective xhigh. Findings addressed and regression-checked; see review.md. |

Tests use the existing cached dependency set with `GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local` and `GOCACHE=/private/tmp/drafts-review-t7_8lkre/cache`. No package download was required. `make verify` is the repeatable project gate. Optional native QA uses `DRAFTS_DICTIONARY_APP`; it never executes generated Drafts library commands.

## CLI best-practices assessment

Source: [nerveband/cli-best-practices](https://github.com/nerveband/cli-best-practices/blob/main/scorecards/agent-cli-audit.md). The source advertises 85 checks, but its numbered checklist contains 97. This assessment uses all 97 and distinguishes unsupported or unexecuted checks. See [per-check evidence](cli-audit.json).

Result: **86 pass, 4 fail, 6 not applicable, 1 unverified**. Passes include scoped/static evidence where identified; they are not proof of live app behavior.

| Category | Pass | Fail | N/A | Unverified |
|---|---:|---:|---:|---:|
| Discoverability | 7 | 0 | 0 | 0 |
| Structured output | 6 | 0 | 0 | 0 |
| Input flexibility | 2 | 1 | 2 | 0 |
| Safety rails | 6 | 0 | 0 | 0 |
| Error handling | 6 | 0 | 0 | 1 |
| Context window discipline | 4 | 0 | 1 | 0 |
| Predictability | 6 | 1 | 0 | 0 |
| Agent knowledge | 7 | 0 | 0 | 0 |
| Resilience | 6 | 0 | 1 | 0 |
| Distribution and lifecycle | 4 | 0 | 1 | 0 |
| Three-layer introspection | 5 | 0 | 0 | 0 |
| Persistent identity and configuration | 5 | 0 | 0 | 0 |
| Two-way I/O and artifacts | 5 | 0 | 0 | 0 |
| Contract and generation discipline | 3 | 1 | 1 | 0 |
| Unix composability and agent restraint | 5 | 0 | 0 | 0 |
| API-native payload ergonomics | 4 | 1 | 0 | 0 |
| Domain depth and proof gates | 5 | 0 | 0 | 0 |

The remaining failed checks are full non-interactive/flag-only coverage for intentionally interactive commands, a uniform resource/verb hierarchy, complete generation from one contract, and API-shaped resource hierarchy. They are explicit product/contract limits. The implementation supplies non-interactive replace/open alternatives and CI coverage, while preserving existing Drafts vocabulary.

## Remaining limits and release decisions

- Broad disposable-library integration tests were not run. Live proof uses only new, uniquely tagged fixtures. Workspace rename, workspace switching, and existing user action submission were not exercised live, preserving unrelated library state. Native compilation and synthetic tests cover those paths. Large disposable-library performance benchmarking was unavailable.
- Drafts provides no action completion endpoint, server-side pagination, or incremental change feed. Receipts track submission. Bulk UUID/sort-key indexes are fetched before sorting/limiting; detailed rows are fetched only for selected UUIDs. Date formatters/calendars are reused per script. The matching index still scales with library size.
- AppleScript dates have second resolution and local daylight-saving ambiguity. Foundation tests cover ordinary winter/summer conversion, not every ambiguous transition.
- Content comparison before edit replacement is a best-effort guard, not an app transaction or lock. Timeout and crash outcomes can be uncertain; receipts block unsafe retries.
- Snapshots replace the previous bounded cache and may contain private text only when the user explicitly requests sync. No automatic refresh, background job, or private database access exists.
- The original README claimed MIT, but lacked a license file. The maintainer confirmed MIT and inherited-code redistribution rights on 2026-10-05; LICENSE and upstream attribution are now included in release archives.
- Download and binary replacement were verified against the existing published 3.0.2 release on a temporary executable. New-release downloads, signing/notarization, and current vulnerability advisories remain unverified. Dependencies remain at existing versions; x/sys is now correctly classified as direct for terminal detection.

## API and implementation sources

- [Drafts external AppleScript guide](https://docs.getdrafts.com/docs/automation/applescript)
- [Drafts release notes](https://docs.getdrafts.com/changelog/), [beta program](https://docs.getdrafts.com/beta/), and [Mac beta notes](https://docs.getdrafts.com/beta/changelog-mac)
- [Stable JavaScript reference](https://scripting.getdrafts.com/) and [beta reference](https://beta-scripting.getdrafts.com/), used to distinguish internal-only APIs
- Installed public dictionary: `/Applications/Drafts.app/Contents/Resources/Drafts.sdef`, version 54.1 build 924. Latest metadata discovery identifies channel as stable; DrFl/DrFt are distinct despite duplicate flagged names.
- [Official Drafts MCP/CLI backend](https://github.com/agiletortoise/drafts-mcp-server/blob/main/src/drafts.ts), comparison of external operations and binary-name collision
- [Apple NSJSONSerialization](https://developer.apple.com/documentation/foundation/nsjsonserialization) and [NSDateFormatter](https://developer.apple.com/documentation/foundation/nsdateformatter), native structured serialization/date formatting
- [GoReleaser deprecations](https://goreleaser.com/resources/deprecations/#archivesbuilds), validated migration to archive ids
- [Original ernstwi/drafts repository](https://github.com/ernstwi/drafts), project attribution and observed absence of license file

Historical documents under docs/plans and the obsolete root drafts-cli-helper.js are retained for provenance. The helper is not executed, imported, required, or distributed in release archives. Current schema, README, skill, and code define the maintained interface.

- [Go module major-version rules](https://go.dev/ref/mod#major-version-suffixes): the v4 release uses the matching /v4 module path, including source installation instructions.
