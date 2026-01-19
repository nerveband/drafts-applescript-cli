# Drafts CLI

Command line interface for [Drafts](https://getdrafts.com). Requires Drafts Pro and macOS.

## Install

```bash
go install github.com/nerveband/drafts/cmd/drafts@latest
```

Or build from source:

```bash
git clone https://github.com/nerveband/drafts
cd drafts
go build ./cmd/drafts
```

**No additional dependencies required!** The CLI communicates directly with Drafts via AppleScript.

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

See further: `drafts <command> --help`

## LLM Integration

This CLI is designed for LLM tool use. Get the schema:

```bash
drafts schema
```

Output is JSON by default for easy parsing. Use `--plain` for human-readable output.

### Example: Create a draft

```bash
# JSON output (default)
drafts create "My new draft" -t tag1 -t tag2

# Plain text output
drafts create "My new draft" --plain
```

### Example: List drafts

```bash
# List inbox drafts
drafts list

# List archived drafts
drafts list -f archive

# List drafts with specific tag
drafts list -t mytag
```

### Example: Run an action

```bash
# Run action on text
drafts run "Copy" "Text to copy"

# Run action on existing draft
drafts run "Copy" -u <uuid>
```

## Implementation

The CLI communicates directly with Drafts via AppleScript (`osascript`). No helper apps or Drafts actions required.

## Development

```bash
go build ./cmd/drafts    # Build
go test ./...            # Test
```
