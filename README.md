# Drafts AppleScript CLI

Command line interface for [Drafts](https://getdrafts.com) on macOS.

## Requirements

> **IMPORTANT: This CLI only works on macOS with Drafts running.**

| Requirement | Details |
|-------------|---------|
| **Operating System** | macOS only (uses AppleScript) |
| **Drafts App** | Must be installed AND running |
| **Drafts Pro** | Required for automation features |
| **Go** | 1.21+ (for installation) |

**This CLI will NOT work if:**
- You're on Linux or Windows
- Drafts app is not installed
- Drafts app is not running (it must be open)
- You don't have Drafts Pro subscription

## How It Works

The CLI communicates with Drafts via AppleScript (`osascript`). This means:
- Drafts must be running on your Mac for any command to work
- Commands execute in the context of the running Drafts app
- All data stays local on your Mac

## Install

### Option 1: Go Install

```bash
go install github.com/nerveband/drafts/cmd/drafts@latest
```

### Option 2: Build from Source

```bash
git clone https://github.com/nerveband/drafts
cd drafts
go build ./cmd/drafts

# Optionally move to PATH
mv drafts /usr/local/bin/
```

### Option 3: Download Binary

Download from [Releases](https://github.com/nerveband/drafts/releases) (macOS only).

## Quick Start

```bash
# Make sure Drafts is running first!
open -a Drafts

# Create a draft
drafts create "Hello from the CLI"

# List your drafts
drafts list

# Get a specific draft
drafts get <uuid>
```

## Usage

```
$ drafts --help
Usage: drafts [--plain] <command> [<args>]

Options:
  --plain              output plain text instead of JSON
  --help, -h           display this help and exit

Commands:
  new, create          create new draft
  prepend              prepend to draft
  append               append to draft
  replace              replace content of draft
  edit                 edit draft in $EDITOR
  get                  get content of draft
  select               select active draft using fzf
  list                 list drafts
  run                  run a Drafts action
  schema               output tool-use schema for LLM integration
```

## Commands

### create / new

Create a new draft.

```bash
drafts create "Content here" [options]

Options:
  -t, --tag TAG        Add tag (can be used multiple times)
  -a, --archive        Create in archive folder
  -f, --flagged        Create as flagged
  --action ACTION      Run action after creation
```

**Examples:**
```bash
drafts create "Meeting notes"
drafts create "Shopping list" -t groceries -t todo
drafts create "Important!" -f
```

### get

Get a draft by UUID.

```bash
drafts get [UUID]      # Omit UUID to get active draft
```

### list

List drafts with optional filtering.

```bash
drafts list [options]

Options:
  -f, --filter FILTER  Filter: inbox|archive|trash|all (default: inbox)
  -t, --tag TAG        Filter by tag (can be used multiple times)
```

**Examples:**
```bash
drafts list                    # List inbox
drafts list -f archive         # List archived
drafts list -t work            # Filter by tag
```

### prepend / append

Add content to an existing draft.

```bash
drafts prepend "Text" -u UUID [options]
drafts append "Text" -u UUID [options]

Options:
  -u, --uuid UUID      Target draft UUID (omit to use active draft)
  -t, --tag TAG        Add tag
  --action ACTION      Run action after modification
```

### replace

Replace entire content of a draft.

```bash
drafts replace "New content" -u UUID
```

### edit

Open draft in your $EDITOR.

```bash
drafts edit [UUID]     # Omit UUID to edit active draft
```

### run

Run a Drafts action.

```bash
drafts run "Action Name" "Text to process"
drafts run "Action Name" -u UUID    # Run on existing draft
```

### schema

Output tool-use schema for LLM integration.

```bash
drafts schema          # Full schema
drafts schema create   # Schema for specific command
```

## Output Formats

**JSON (default)** - Structured output for programmatic use:
```json
{
  "success": true,
  "data": {
    "uuid": "ABC-123",
    "content": "Note content",
    "title": "Note title",
    "tags": ["tag1"],
    "folder": "inbox"
  }
}
```

**Plain text** - Human-readable output:
```bash
drafts list --plain
```

## LLM Integration

This CLI is designed for LLM tool use:

- **JSON output by default** - Easy to parse
- **Structured errors** - Error code, message, and recovery hints
- **Tool-use schema** - Get schema with `drafts schema`
- **Full metadata** - All draft properties returned

### ClawdBot Skill

A ClawdBot skill is available for this CLI. Install to `~/.clawdbot/skills/drafts/SKILL.md`.

## Troubleshooting

### "AppleScript error" or no response

1. **Is Drafts running?** The app must be open: `open -a Drafts`
2. **Is Drafts Pro active?** Automation requires Pro subscription
3. **Permissions granted?** Go to System Settings > Privacy & Security > Automation and ensure Terminal (or your app) can control Drafts

### "command not found: drafts"

Add to your PATH:
```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Commands hang or timeout

Drafts may be showing a dialog. Check the Drafts app window.

## Architecture

```
┌─────────────┐      AppleScript      ┌─────────────┐
│  drafts CLI │ ──────────────────▶   │  Drafts.app │
└─────────────┘      (osascript)      └─────────────┘
```

- No network requests
- No helper apps
- No Drafts actions to install
- Pure local AppleScript communication

## Development

```bash
go build ./cmd/drafts    # Build
go test ./...            # Run tests
go vet ./...             # Lint
```

## License

MIT

## Credits

Forked from [ernstwi/drafts](https://github.com/ernstwi/drafts). Refactored to use AppleScript backend (no helper app required).
