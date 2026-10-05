# Drafts API coverage

Updated 2026-10-05. This table distinguishes the externally callable AppleScript dictionary from JavaScript executed inside Drafts. The selected installed dictionary is authoritative; stable/beta documentation can lag the installed app.

| Capability | CLI coverage | Boundary |
|---|---|---|
| Create/read/update content | create/get/append/prepend/replace/edit | Persistent drafts; explicit destructive commitment |
| Folder moves | archive/inbox/trash, delete alias | Trash is a move, not permanent deletion |
| Boolean flag | create/flag/unflag/list | Raw boolean Apple event code avoids duplicate names |
| Flag type | create --flag-type, flag --flag-type, list --flag-type | Enabled only when writable DrFt integer exists, stable or beta |
| Include/exclude tags | list --tag/--omit-tag | Dictionary query tag names |
| Tag mutation | tag/remove-tags | Deduplicated addition and exact-name removal |
| Global tag rename | Not exposed | Installed tag name property is read-only |
| Tag resources | tags, info --verbose | Requires top-level tag class; no sampled 50-draft approximation |
| Workspaces | workspace/workspaces/list --workspace | IDs and available permalinks; open requires dictionary direct workspace type; rename requires writable name capability and commitment |
| Actions | actions/run/create --action/append --action | Names/IDs; preflight before content changes; submission only |
| Date filters and ordering | list date bounds/sort/order/flagged-first | Exclusive UTC bounds; AppleScript date resolution |
| Native counts | list --count, info --counts | Includes true all-folder count; no body retrieval |
| Local query snapshot | sync; list/get --data-source local | Explicit bounded CLI-owned JSON, never private Drafts SQLite |
| Selected/active draft | get without UUID, open/select | Interactive select requires terminal; UUID preferred for agents |
| Syntax/language grammar | Not exposed | Internal JavaScript API, absent from inspected external dictionary |
| Task objects and task advancement | Not exposed | Internal JavaScript API, absent from inspected external dictionary |
| Version history/action logs | Not exposed | No external dictionary endpoint |
| AI/Location script objects | Not exposed | Internal scripting objects, not an external AppleScript API |
| Completion polling/--wait | Not exposed | Dictionary submits action, exposes no durable completion handle |
| API-native query language | Literal containment only | External AppleScript content containment differs from full Drafts search |
| Incremental sync and pagination | Not exposed | Dictionary has no server-side cursor/change feed; snapshots rebuild explicitly |
| Permanent draft deletion | Not exposed | Dictionary folder moves only; trash preserves recoverability |

## Comparison with the official project

The [official Drafts MCP/CLI project](https://github.com/agiletortoise/drafts-mcp-server) uses AppleScript and also exposes draft CRUD, folder moves, flags, tags, actions, workspaces, and date filters. This fork now covers the relevant external dictionary operations and adds capability-based app selection, explicit previews/commit gates, strict payload validation, structured errors, exact output, profiles, receipts, and snapshots. No benchmark or claim of superior live compatibility is made; actual Drafts library operations were not executed during this implementation.

Both projects may install a command named `drafts`. Verify `drafts version` and its repository field. Do not treat this CLI's schema or flags as the official project's contract.

## Sources

- [External AppleScript guide](https://docs.getdrafts.com/docs/automation/applescript)
- [Stable release notes](https://docs.getdrafts.com/changelog/)
- [Beta program](https://docs.getdrafts.com/beta/) and [Mac beta notes](https://docs.getdrafts.com/beta/changelog-mac)
- [Stable JavaScript reference](https://scripting.getdrafts.com/) and [beta JavaScript reference](https://beta-scripting.getdrafts.com/)
- Locally inspected `/Applications/Drafts.app/Contents/Resources/Drafts.sdef`, version 54.1 build 924, channel unknown from available metadata
- [Official backend source](https://github.com/agiletortoise/drafts-mcp-server/blob/main/src/drafts.ts)

The public release notes currently lead with 54.0.5 and contain a 54.1 build-902 entry. The beta reference is not consistently newer than the stable reference. Neither is used to guess installed capabilities.
