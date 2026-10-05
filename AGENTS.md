# Repository contract

This repository builds a macOS Drafts AppleScript CLI. Follow inherited user instructions, preserve unrelated work and secrets, and do not use em dashes.

- Current source of truth: Go command/parser types, request structs, runtime schema, README.md, and skills/SKILL.md. Contract tests check command coverage, examples, safety gates, and documentation workflows.
- Build with Go 1.24.11 or newer. Keep dependency changes separate unless needed for the authorized fix. Use `-mod=readonly` for verification.
- Default `make verify` tests are synthetic. Do not access private Drafts contents during code review or routine verification.
- Optional dictionary QA compiles scripts without running Drafts events. Foundation smoke tests use synthetic values only.
- Live integration tests require the integration build tag, DRAFTS_INTEGRATION=1, DRAFTS_DISPOSABLE_LIBRARY=1, and an absolute DRAFTS_TEST_APP path. All four gates and explicit user authorization for a disposable library are required.
- Separately, scripts/live-fixture-test.py supports explicitly authorized fixture-only tests in an existing library with DRAFTS_LIVE_FIXTURES=1 and an absolute DRAFTS_TEST_APP. It reads only created UUIDs or a unique fixture tag, previews destructive operations, and moves its fixture drafts to Trash. Never set the disposable-library gate for an existing personal library.
- Never read Drafts' private database, launch an app implicitly, install the obsolete helper action, or claim JavaScript-only APIs are externally available.
- Always preview destructive work with `--dry-run`; replace, edit, trash, and self-upgrade require `--commit`. Prefer explicit UUIDs and `--fields`/`--limit` for reads.
- Capability decisions come from the selected app dictionary. Channel and version labels alone must not enable features. Test both optional-capability-present and absent branches.
- Keep draft text out of process arguments and diagnostic errors. Temporary output/state files use mode 600, state directories mode 700. No API secrets belong in profiles or feedback.
- Actions report submission only. Do not fabricate completion, retry uncertain mutations, or bypass blocked idempotency receipts.
- Do not run release.sh, commit, push, publish, install dependencies, or change the user's app/library without current explicit authorization.
- Historical docs/plans and demo assets are not the maintained command contract. Preserve unrelated demo work.
