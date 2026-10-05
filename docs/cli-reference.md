# CLI reference

Detailed options and workflows for Drafts CLI. Start with the [README](../README.md) for setup and common tasks. Use `drafts schema` for the machine-readable command contract.

## Stable and beta selection

```sh
drafts apps
drafts schema --detected
drafts --app /Applications/Drafts.app info
drafts --channel beta --app '/Applications/Drafts Beta.app' schema --detected
```

`apps` reads bundle metadata and the public `.sdef` dictionary, never the Drafts library. `--channel auto` prefers a single stable or unknown-channel installation over identified betas. Multiple matching installations require an exact `--app` path. A bundle identifier is accepted only when it identifies one installation.

A beta/TestFlight label or sandbox receipt identifies beta; a normal App Store receipt identifies stable. Otherwise the channel is `unknown`. A version number alone is insufficient evidence. Use an explicit path with `--channel auto` when the channel cannot be established.

**Features are enabled by dictionary capabilities, not by channel labels.** A stable installation with flag types gets flag types. An older installation without them keeps boolean flags, and an unsupported optional request fails with exit 5 before sending that mutation. `schema --detected` exposes `flag_type_available` and `workspace_open_available`. Duplicate `flagged` property names are disambiguated using Apple event codes `DrFl` (boolean) and `DrFt` (integer).

The [AppleScript guide](https://docs.getdrafts.com/docs/automation/applescript) describes external automation. The [stable](https://scripting.getdrafts.com/) and [beta](https://beta-scripting.getdrafts.com/) JavaScript references describe scripts executed inside Drafts; they do not imply external AppleScript support. Syntax, tasks, version history, AI objects, and action completion monitoring are not available through this CLI's supported dictionary. See [feature coverage](../FEATURE_GAP_ANALYSIS.md).

## Commands

| Command | Purpose |
|---|---|
| `create` (`new`) | Create persistent content with tags, folder, flag, optional action |
| `get` | Full content and metadata by UUID, default active draft |
| `list` | Bounded summaries, folder/tag/search/workspace/date filtering, sorting, counts |
| `append`, `prepend` | Add content with an explicit optional separator, tags, and action |
| `replace` (`update`) | Replace content, requires `--commit` |
| `edit` | Edit in `$EDITOR`, detects content changes before saving, requires `--commit` |
| `select` | Select/open using `fzf`, interactive only |
| `flag`, `unflag` | Boolean flag, optional `flag --flag-type 0..6` when supported |
| `archive`, `inbox`, `trash` (`delete`) | Move between folders; trash requires `--commit` |
| `open` | Open an existing draft |
| `tag`, `remove-tags` | Add/remove tag names on a draft |
| `tags`, `actions`, `workspaces` | Structured resource lists with available IDs/permalinks |
| `workspace` | Current workspace, `--list`, capability-gated `--open`, or `--rename <old> --to <new> --commit` |
| `run` | Submit action on existing UUID or new persistent content |
| `sync` | Save an explicit, bounded CLI-owned snapshot |
| `apps`, `info` | Installation metadata and opt-in read diagnostics |
| `schema` (`agent-context`) | Versioned contract, safety metadata, examples, optional detection |
| `profile`, `config` | Named non-secret defaults and configuration source inspection |
| `jobs` | Durable duplicate-prevention receipts, optionally by key |
| `feedback`, `skills` | Local structured feedback and embedded agent skill |
| `version`, `upgrade` | Offline version; explicit checksum-verified self-update |

Each command supports `--help` with an example. `drafts schema <command>` describes one command. Global flag types, safety, response descriptions, examples, and exit codes are in `drafts schema`. Raw JSON is available for create, append, prepend, replace, flag/unflag, folder moves, open, tag changes, and run. Other commands accept their documented flags.

## Safe workflows

```sh
# Inspect capability availability first, without reading drafts.
drafts apps
drafts schema --detected

# Preview, then create using a durable request key.
drafts create --input @payload.json --dry-run
drafts create --input @payload.json --idempotency-key notes-001

# Select a target explicitly before destructive changes.
drafts list --limit 5 --sort modified --fields uuid,title,modifiedAt
drafts get 12345678-1234-1234-1234-123456789abc --format raw
drafts replace -u 12345678-1234-1234-1234-123456789abc --text-file note.md --dry-run
drafts replace -u 12345678-1234-1234-1234-123456789abc --text-file note.md --commit
drafts trash 12345678-1234-1234-1234-123456789abc --dry-run

# An action can change, move, or delete its target. Status means submitted.
drafts run --input '{"action":"Copy","uuid":"12345678-1234-1234-1234-123456789abc"}' --dry-run
```

`--dry-run` validates local input and describes the request without Apple events or ledger writes. It reports capability/target availability as `not_checked`, so a preview does not guarantee that a target or action exists. Active-draft targeting is represented by an omitted UUID. Explicit UUIDs are safer in automation.

`--commit` is mandatory for replace/update, edit, trash/delete, upgrade, and workspace rename. Replace/update and trash/delete also require an explicit UUID. Other mutations require their explicit command. Actions can perform their own destructive or network operations. `run` with content creates a persistent draft; it is not a transient scratch buffer. Mutations return a UUID receipt rather than fetching potentially changed content after success. Use `get` explicitly when needed.

## Input and output contract

- JSON success goes to stdout as `{"success":true,"data":...}`. Errors go to stderr as `{"success":false,"error":{"code":...,"message":...,"retry_safe":...}}`.
- `--format json` is the default; `plain` renders readable values; `jsonl` emits list rows; `raw` is exact `get` content. `--plain` is an alias. `--format-error json|plain` is independent. `--quiet` preserves data-only output, already the default.
- `list --fields uuid,title`, `--id-only`, and `--count` reduce output. Default limit is 20, default sort is created descending. `--limit 0` removes the returned-row limit.
- `--full` fetches bodies only for selected UUIDs. Bulk UUID/sort-key reads precede sorting and limiting; detailed rows are fetched only for the selection. AppleScript provides no server-side pagination, so the matching index is still loaded. Search is literal content containment, not the full in-app query language.
- `--sort created|modified|accessed`, `--order asc|desc`, `--flagged-first`, `--omit-tag`, and `--flag-type` are supported. Accessed sorting and flag types require dictionary support.
- Date bounds are exclusive: `--created-after`, `--created-before`, `--modified-after`, `--modified-before`. Use RFC3339 or `YYYY-MM-DD` (UTC midnight). AppleScript dates have second resolution. Output dates are locale-independent UTC RFC3339.
- `--input '{...}'`, `--input @payload.json`, or `--input -` accept one UTF-8 JSON object. Unknown/duplicate fields, null fields, missing required content, malformed UUIDs, and trailing objects are rejected. Explicit `{"content":""}` is allowed.
- `--stdin` explicitly reads content, preserving trailing newlines. `--text-file note.md` reads exact UTF-8 text. A literal starting with `@` remains literal positional content; only `--input @...` expands a file. Binary content is unsupported.
- Omitted content fails immediately. Explicit stdin is bounded by `--timeout` and 16 MiB. Do not combine content sources or mix `--input` with command arguments.
- Append/prepend default to a newline separator; `--separator ''` joins directly. Raw JSON uses `separator`.
- `run` requires exactly one of `uuid` or `content`. An empty string still counts as explicit content.
- Interactive tools reject non-terminal stdin. `$EDITOR` accepts quoted arguments such as `code --wait`, without shell expansion. Editor output goes to stderr; edited content is not trimmed.

### Artifacts

```sh
drafts get 12345678-1234-1234-1234-123456789abc --format raw --deliver file:note.md
drafts version --deliver file:version.json --overwrite
```

`--deliver stdout|file:<path>` routes the result. The directory must exist. Files are written atomically with mode 600. Existing paths require `--overwrite`; stdout then contains a delivery receipt. Output option errors are validated before Drafts mutations.

### Profiles and receipts

```sh
drafts profile save work --input '{"app":"/Applications/Drafts.app","channel":"auto","timeout":"30s"}'
drafts profile list
drafts --profile work config
drafts jobs notes-001
drafts feedback 'A command example was unclear'
drafts skills
```

Precedence: explicit flag > `DRAFTS_CLI_APP`, `DRAFTS_CLI_CHANNEL`, `DRAFTS_CLI_TIMEOUT` > selected profile > default. Select a profile using `--profile` or `DRAFTS_CLI_PROFILE`. `config` reports sources without credentials. macOS Automation permission is external to profiles; no API secrets are stored.

Profiles default to `os.UserConfigDir()/drafts-cli/profiles.json`; CLI state lives beside it under `state`. Override with `--config-file` and `--state-dir`. Profiles, receipts, snapshots, and feedback files use private permissions. Feedback is local only; upstream issues are linked but never submitted automatically. Do not put secrets or private content in feedback.

`--idempotency-key` on create/new or run records a request hash, selected app path, UUID, and status without storing content. Completed requests replay their receipt. Reuse with different inputs fails. An interrupted, failed, or uncertain request is blocked from resubmission. `jobs` inspects receipts. There is no unsafe automatic retry or promise of exactly-once delivery across a crash. Action receipts track submission, not completion; there is no fabricated `--wait` endpoint.

### Explicit local snapshots

```sh
drafts sync --filter all --limit 100 --full --dry-run
drafts sync --filter all --limit 100 --full
drafts list --data-source local --search meeting --limit 5
drafts get 12345678-1234-1234-1234-123456789abc --data-source local --format raw
```

`sync` uses AppleScript and stores a separate mode-600 JSON snapshot owned by this CLI. It never opens Drafts' private SQLite database. Sync replaces the previous snapshot and is bounded by its filter, workspace, search, and limit. Local counts/search cover cached rows only and report snapshot provenance. Content search/get require a `--full` snapshot. Local search is a case-insensitive substring search. No automatic refresh occurs; mutations require the live source. Snapshots may contain private draft text, so choose a protected state directory and remove snapshots yourself when no longer needed.

Queries outside the snapshot's recorded filter/search/tag/date/flag/workspace scope fail explicitly. Tag addition and removal preserve case, matching distinct native tag names. Tag filters follow Drafts' case-insensitive AppleScript query matching in both live and local reads.

## Diagnostics and exit codes

`drafts info` reads metadata only. `info --counts`, `--verbose`, and `--test-permissions` opt into read-only Drafts queries. Permission testing counts inbox drafts and creates nothing. Allow the terminal to control the selected app in System Settings > Privacy & Security > Automation. There is no System Events or Accessibility requirement for app detection.

| Exit | Meaning |
|---|---|
| 0 | Success |
| 2 | Invalid input, interactive terminal required, or missing `--commit` |
| 3 | Target/application not found or app not running |
| 4 | Automation permission denied |
| 5 | Unsupported capability |
| 6 | Edit conflict or blocked idempotency request |
| 7 | Timeout, AppleScript mutation outcome may be uncertain |
| 8 | Partial failure after a completed mutation step |
| 9 | Other execution, response, output, snapshot, or update failure |

`PARTIAL_FAILURE` includes UUID and completed steps where known. Inspect that UUID before retrying. Timeouts have `retry_safe:false`; a write may have completed. Edit compares original content immediately before replacement, but AppleScript is not a transactional compare-and-swap API, so a very small concurrent-write window remains. Date round trips during ambiguous daylight-saving transitions inherit AppleScript's local-date limitations.

Ordinary commands perform no CLI network calls or automatic update checks. `upgrade --commit` explicitly contacts GitHub, verifies release checksums, and replaces this executable under a timeout. `upgrade --dry-run` does neither. Drafts actions may themselves use network services.

## Architecture and development

`cmd/drafts` owns parsing, strict request validation, safety gates, profiles/receipts/snapshots, schema/help, formatting, and self-update. `pkg/drafts` owns dictionary detection, app selection, request validation, AppleScript generation, typed failures, and structured response parsing. `assets.go` embeds `skills/SKILL.md` in the binary. Note bodies travel over `osascript` stdin, not process arguments. Foundation serializes JSON, avoiding tab/newline delimiters and locale-formatted dates.

Direct dependencies include `go-arg` 1.4.3, `go-selfupdate` 1.5.2, `go-version` 1.8.0, and the already pinned `x/sys` 0.39.0 for reliable terminal detection, with existing pinned transitive modules. See `go.mod` for the full dependency list.

```sh
make verify                       # safe synthetic tests, vet, read-only module build
make build                        # local ./drafts binary
DRAFTS_DICTIONARY_APP=/Applications/Drafts.app go test ./pkg/drafts -run '^TestNativeAppleScript$'
```

Default tests use synthetic responses, fake app metadata, and temporary files. The optional dictionary test compiles generated AppleScripts without running Drafts events, and executes Foundation helpers against synthetic values only. CI verifies both macOS architectures and schema/help/docs contract coverage.

Live integration tests are excluded by default. They require `-tags integration`, `DRAFTS_INTEGRATION=1`, `DRAFTS_DISPOSABLE_LIBRARY=1`, and an absolute `DRAFTS_TEST_APP` path. Run them only against a disposable app/library with explicit authorization. They create, edit, read, open, and trash drafts.

For separately authorized fixture-only testing in an existing library, explicitly start the selected app, then run `DRAFTS_LIVE_FIXTURES=1 DRAFTS_TEST_APP=/Applications/Drafts.app python3 scripts/live-fixture-test.py`. This harness targets only new UUIDs and a unique tag, previews destructive changes, and moves its drafts to Trash. It preserves a private recovery journal on failure. Never retry an uncertain create with a new idempotency key.

GoReleaser snapshot packaging is available with `./build.sh`. Publication requires explicitly invoking `./release.sh --publish`; builds/tests do not publish. Historical design documents under `docs/plans/` and demo assets describe earlier contracts. Current code, schema, README, and embedded skill are the maintained contract.

The release script requires a pushed tag at HEAD. It tests, packages both architectures with GoReleaser, verifies checksums, and publishes using the authenticated GitHub CLI session. It does not extract authentication tokens into scripts or arguments.

See [verification and provenance](verification.md) for the CLI best-practices assessment, API evidence, and remaining platform limitations.

