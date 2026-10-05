# Independent release review

Opus 5.5 reviewed the implementation read-only on 2026-10-05. Provider status confirmed effective `xhigh` reasoning. The reviewer used source, the public dictionary, synthetic AppleScript probes, and vet. It accessed no Drafts library data and launched no additional agents.

| Finding | Resolution and evidence |
|---|---|
| Destructive active-draft fallback | Replace/update and trash/delete require explicit UUIDs in policy and schema. Fail-fast tests cover preview and commit. |
| Listing serialized every matching row | Bulk UUID/sort-key index is sorted and limited before selected rows are fetched. Date formatter/calendar are reused per script. Synthetic tests reject unselected row fetches; scoped live query verified selection. |
| RemoveTags remote-reference coercion | Copy the tag list locally and dereference loop items. Live removal passed. |
| Case-insensitive edit conflict guard | Foundation compares UTF-8 bytes encoded as ASCII. Live checks cover case-only and Unicode-plus-case changes and allow unchanged decomposed Unicode editing. Drafts elides canonical-equivalent content setters, retaining the original bytes. |
| Tag mutations ignored case | Explicit `considering case` comparisons preserve native distinct tag names. Native fixture creation confirmed case-distinct tags. Tag queries retain the application's case-insensitive predicate semantics, mirrored by local snapshots. |
| Preflight failures blocked idempotency keys | Running, permission, capability, action and target checks precede reserving new keys. Replays send no events. Regression tests verify stopped-app, missing-action and unsupported-flag failures leave keys available. |
| Later events could relaunch a quit app | Every wrapped script checks `application is running` before its operations. Native compilation validates the guard. A very small process-exit race remains inherent in AppleScript targeting by app path. |
| Snapshot queries exceeded recorded scope | Search and all query options are recorded; local queries outside that scope fail explicitly. Regression test covers incompatible archive queries. |
| Detected schema overstated capabilities | Open/select require open.draft and run requires command.perform. Runtime gates cover tag predicates and current draft/workspace. |
| Documentation put content in arguments | Real-content workflow examples use @payload.json and text files. Inline transport remains supported for explicitly chosen synthetic or nonsensitive inputs. |
| Unknown jobs key returned IO_ERROR | Unknown keys return an empty job list. Regression test uses an unreserved key. |

Remaining platform limits: no server-side pagination, no atomic edit transaction, and no action completion endpoint. A large disposable-library performance benchmark was unavailable. Bulk indexes still scale with matching UUIDs and sort keys, while detailed-row work scales with the selected limit. See [verification](verification.md) for final check results.
