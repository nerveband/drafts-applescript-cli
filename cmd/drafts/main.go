package main

import (
	"errors"
	"fmt"
	arg "github.com/alexflint/go-arg"
	"github.com/nerveband/drafts-applescript-cli/v4/pkg/drafts"
	"os"
	"strings"
	"time"
)

const repoURL = "https://github.com/nerveband/drafts-applescript-cli"
const issuesURL = repoURL + "/issues"
const docsURL = "https://docs.getdrafts.com"

type NewCmd struct {
	Message  string   `arg:"positional" help:"content, or use --stdin, --text-file, --input"`
	Tag      []string `arg:"-t,separate" help:"tag"`
	Archive  bool     `arg:"-a" help:"create in archive"`
	Flagged  bool     `arg:"-f" help:"flag draft"`
	FlagType *int     `arg:"--flag-type" help:"flag type 0 through 6, if supported"`
	Action   string   `arg:"--action" help:"submit action after creation"`
	Input    string   `arg:"--input" help:"JSON object, @file, or '-' for stdin"`
}
type CreateCmd = NewCmd
type PrependCmd struct {
	Message   string   `arg:"positional" help:"content"`
	UUID      string   `arg:"-u" help:"UUID, default active draft"`
	Tag       []string `arg:"-t,separate" help:"tag to add"`
	Action    string   `arg:"--action" help:"submit action after modification"`
	Separator *string  `arg:"--separator" help:"separator, default newline"`
	Input     string   `arg:"--input" help:"JSON object, @file, or '-'"`
}
type AppendCmd = PrependCmd
type ReplaceCmd struct {
	Message string `arg:"positional" help:"replacement content"`
	UUID    string `arg:"-u" help:"UUID, default active draft"`
	Input   string `arg:"--input" help:"JSON object, @file, or '-'"`
}
type EditCmd struct {
	UUID string `arg:"positional" help:"UUID, default active draft"`
}
type GetCmd = EditCmd
type SelectCmd struct{}
type FlagCmd struct {
	UUID  string `arg:"positional" help:"UUID, default active draft"`
	Input string `arg:"--input" help:"JSON object, @file, or '-'"`
	Type  *int   `arg:"--flag-type" help:"flag type 0 through 6, if supported"`
}
type UnflagCmd = FlagCmd
type WorkspaceCmd struct {
	Rename string `arg:"--rename" help:"existing workspace name, requires --to and --commit"`
	To     string `arg:"--to" help:"new workspace name"`
	List   bool   `arg:"-l,--list" help:"list workspace names"`
	Open   string `arg:"-o,--open" help:"open workspace by name"`
}
type ActionsCmd struct {
	Search string `arg:"-s,--search" help:"filter names by substring"`
}
type ListCmd struct {
	Filter         string   `arg:"-f" default:"inbox" help:"inbox|flagged|archive|trash|all"`
	Tag            []string `arg:"-t,separate" help:"include tag"`
	OmitTag        []string `arg:"--omit-tag,separate" help:"exclude tag"`
	Search         string   `arg:"-s" help:"search content"`
	Workspace      string   `arg:"-w" help:"workspace name"`
	Limit          int      `arg:"--limit" default:"20" help:"maximum returned drafts, 0 for all"`
	Full           bool     `arg:"--full" help:"fetch bodies only for returned drafts"`
	Sort           string   `arg:"--sort" default:"created" help:"created|modified|accessed"`
	Order          string   `arg:"--order" default:"desc" help:"asc|desc"`
	FlaggedFirst   bool     `arg:"--flagged-first" help:"flagged drafts first"`
	FlagType       *int     `arg:"--flag-type" help:"supported flag type"`
	CreatedAfter   string   `arg:"--created-after" help:"exclusive bound, RFC3339 or YYYY-MM-DD"`
	CreatedBefore  string   `arg:"--created-before" help:"exclusive bound"`
	ModifiedAfter  string   `arg:"--modified-after" help:"exclusive bound"`
	ModifiedBefore string   `arg:"--modified-before" help:"exclusive bound"`
	Count          bool     `arg:"--count" help:"count matches without bodies"`
}
type RunCmd struct {
	Action string `arg:"positional" help:"action name"`
	Text   string `arg:"positional" help:"content for a new persistent draft"`
	UUID   string `arg:"-u" help:"target existing draft, incompatible with content"`
	Input  string `arg:"--input" help:"JSON object, @file, or '-'"`
}
type SchemaCmd struct {
	Command  string `arg:"positional" help:"command or alias"`
	Detected bool   `arg:"--detected" help:"include selected app capability availability"`
}
type InfoCmd struct {
	Verbose         bool `arg:"-v,--verbose" help:"explicitly read action, tag, and workspace resources"`
	Counts          bool `arg:"--counts" help:"explicitly count drafts in each folder"`
	TestPermissions bool `arg:"--test-permissions" help:"read-only automation probe"`
}
type UpgradeCmd struct{}
type VersionCmd struct{}
type AppsCmd struct{}
type MoveCmd struct {
	UUID  string `arg:"positional" help:"UUID, default active draft"`
	Input string `arg:"--input" help:"JSON UUID object"`
}
type TagCmd struct {
	UUID  string   `arg:"positional" help:"draft UUID"`
	Tag   []string `arg:"-t,separate" help:"tag to add or remove"`
	Input string   `arg:"--input" help:"JSON uuid/tags object"`
}
type ProfileCmd struct {
	Operation string `arg:"positional" default:"list" help:"list|show|save"`
	Name      string `arg:"positional" help:"profile name"`
	Input     string `arg:"--input" help:"profile JSON object for save"`
}
type FeedbackCmd struct {
	Message string `arg:"positional" help:"local feedback, no private draft content"`
}
type JobsCmd struct {
	Key string `arg:"positional" help:"idempotency key, omit to list receipts"`
}
type Args struct {
	DataSource     string       `arg:"--data-source" default:"live" help:"live|local, local reads an explicit CLI snapshot"`
	Sync           *ListCmd     `arg:"subcommand:sync" help:"save a bounded CLI-owned snapshot, no Drafts database access"`
	Quiet          bool         `arg:"-q,--quiet" help:"data-only output, already the default"`
	FormatError    string       `arg:"--format-error" default:"json" help:"json|plain, independent error stream"`
	Deliver        string       `arg:"--deliver" default:"stdout" help:"stdout|file:<path>"`
	Overwrite      bool         `arg:"--overwrite" help:"allow replacing an output artifact"`
	Profile        string       `arg:"--profile" help:"named non-secret app/channel/timeout defaults"`
	ConfigFile     string       `arg:"--config-file" help:"profile configuration file"`
	StateDir       string       `arg:"--state-dir" help:"private CLI-owned ledger and feedback directory"`
	IdempotencyKey string       `arg:"--idempotency-key" help:"durable duplicate prevention for create and run"`
	Profiles       *ProfileCmd  `arg:"subcommand:profile" help:"list, show, or save profiles"`
	Config         *AppsCmd     `arg:"subcommand:config" help:"inspect effective settings and their sources"`
	Feedback       *FeedbackCmd `arg:"subcommand:feedback" help:"record local structured feedback"`
	Jobs           *JobsCmd     `arg:"subcommand:jobs" help:"inspect idempotency records, UUID positional is a key"`
	Skills         *AppsCmd     `arg:"subcommand:skills" help:"emit bundled skill guidance"`
	AgentContext   *SchemaCmd   `arg:"subcommand:agent-context" help:"alias for schema"`
	Update         *ReplaceCmd  `arg:"subcommand:update" help:"alias for replace"`
	Delete         *MoveCmd     `arg:"subcommand:delete" help:"alias for trash, requires --commit"`

	App            string        `arg:"--app" help:"absolute app path or installed bundle identifier"`
	Channel        string        `arg:"--channel" default:"auto" help:"auto|stable|beta"`
	Timeout        string        `arg:"--timeout" default:"30s" help:"operation and stdin timeout, at most 10m"`
	DryRun         bool          `arg:"--dry-run" help:"preview mutations without Apple events"`
	Commit         bool          `arg:"--commit" help:"authorize replace, edit, trash, or upgrade"`
	Stdin          bool          `arg:"--stdin" help:"explicitly read content from stdin"`
	TextFile       string        `arg:"--text-file" help:"read exact content bytes from file"`
	Format         string        `arg:"--format" default:"json" help:"json|plain|raw|jsonl"`
	Plain          bool          `arg:"--plain" help:"alias for --format plain"`
	Fields         string        `arg:"--fields" help:"comma-separated draft fields to return"`
	IDOnly         bool          `arg:"--id-only" help:"UUIDs, one per line"`
	Apps           *AppsCmd      `arg:"subcommand:apps" help:"inspect installed app dictionaries"`
	New            *NewCmd       `arg:"subcommand:new" help:"create draft"`
	Create         *CreateCmd    `arg:"subcommand:create" help:"alias for new"`
	Prepend        *PrependCmd   `arg:"subcommand:prepend" help:"prepend content"`
	Append         *AppendCmd    `arg:"subcommand:append" help:"append content"`
	Replace        *ReplaceCmd   `arg:"subcommand:replace" help:"replace content, requires --commit"`
	Edit           *EditCmd      `arg:"subcommand:edit" help:"edit in $EDITOR with conflict check, requires --commit"`
	Get            *GetCmd       `arg:"subcommand:get" help:"get full draft"`
	Select         *SelectCmd    `arg:"subcommand:select" help:"interactive selection with fzf"`
	List           *ListCmd      `arg:"subcommand:list" help:"query draft summaries"`
	Flag           *FlagCmd      `arg:"subcommand:flag" help:"flag draft"`
	Unflag         *UnflagCmd    `arg:"subcommand:unflag" help:"unflag draft"`
	Archive        *MoveCmd      `arg:"subcommand:archive" help:"move to archive"`
	Inbox          *MoveCmd      `arg:"subcommand:inbox" help:"restore to inbox"`
	Trash          *MoveCmd      `arg:"subcommand:trash" help:"move to trash, requires --commit"`
	Open           *MoveCmd      `arg:"subcommand:open" help:"open draft"`
	Tag            *TagCmd       `arg:"subcommand:tag" help:"add tags"`
	RemoveTags     *TagCmd       `arg:"subcommand:remove-tags" help:"remove tags"`
	Tags           *AppsCmd      `arg:"subcommand:tags" help:"list tag resources"`
	Workspaces     *AppsCmd      `arg:"subcommand:workspaces" help:"list workspace identifiers and permalinks"`
	Workspace      *WorkspaceCmd `arg:"subcommand:workspace" help:"show, list, or open workspace"`
	Actions        *ActionsCmd   `arg:"subcommand:actions" help:"list action identifiers and names"`
	Run            *RunCmd       `arg:"subcommand:run" help:"submit action on draft or content"`
	Info           *InfoCmd      `arg:"subcommand:info" help:"inspect environment without draft reads by default"`
	Schema         *SchemaCmd    `arg:"subcommand:schema" help:"machine-readable command contract"`
	Upgrade        *UpgradeCmd   `arg:"subcommand:upgrade" help:"checksum-verified self-update, requires --commit"`
	VersionCommand *VersionCmd   `arg:"subcommand:version" help:"offline version information"`
}

func (Args) Version() string { return version }
func (Args) Description() string {
	return "Drafts CLI for macOS. Automation commands require Drafts running and Drafts Pro."
}
func (Args) Epilogue() string {
	return "Examples:\n  drafts apps\n  drafts schema --detected\n  drafts create --input '{\"content\":\"Hello\"}' --dry-run\n  drafts list --limit 5 --sort modified\n  drafts replace -u <uuid> --text-file note.md --commit\nDocumentation: " + repoURL
}

var globalArgs Args
var commandTimeout = 30 * time.Second

func main() {
	var args Args
	p, err := arg.NewParser(arg.Config{Program: "drafts"}, &args)
	if err != nil {
		outputError("INTERNAL_ERROR", err.Error(), "")
	}
	err = p.Parse(os.Args[1:])
	globalArgs = args
	if err == arg.ErrHelp {
		p.WriteHelp(os.Stdout)
		printCommandExample(args)
		return
	}
	if err == arg.ErrVersion {
		args.VersionCommand = &VersionCmd{}
		err = nil
	}
	if err != nil {
		outputError("INVALID_INPUT", err.Error(), "Use drafts --help")
	}
	globalArgs = args
	plainOutput = args.Plain
	applyConfiguration()
	args = globalArgs
	if args.Plain {
		globalArgs.Format = "plain"
	}
	commandTimeout, err = time.ParseDuration(args.Timeout)
	if err != nil || commandTimeout <= 0 || commandTimeout > 10*time.Minute {
		outputError("INVALID_INPUT", "timeout must be greater than zero and at most 10m", "")
	}
	if !validFormat(globalArgs.Format) {
		outputError("INVALID_INPUT", fmt.Sprintf("format must be json, plain, raw, or jsonl, got %q", globalArgs.Format), "")
	}
	if args.Stdin && args.TextFile != "" {
		outputError("INVALID_INPUT", "cannot combine --stdin and --text-file", "")
	}
	if args.Channel != "auto" && args.Channel != "stable" && args.Channel != "beta" {
		outputError("INVALID_INPUT", fmt.Sprintf("channel must be auto, stable, or beta, got %q", args.Channel), "")
	}
	validateGlobalOptions(args)
	if p.Subcommand() == nil && args.VersionCommand == nil {
		p.WriteHelp(os.Stdout)
		return
	}
	switch {
	case args.Sync != nil:
		output(runSync(args.Sync))
	case args.Profiles != nil:
		output(runProfile(args.Profiles))
	case args.Config != nil:
		output(configView())
	case args.Feedback != nil:
		output(runFeedback(args.Feedback))
	case args.Jobs != nil:
		output(runJobs(args.Jobs.Key))
	case args.Skills != nil:
		output(map[string]interface{}{"skill": bundledSkill})
	case args.AgentContext != nil:
		r := getSchema(args.AgentContext.Command)
		if args.AgentContext.Detected {
			r = detectedSchema(r)
		}
		output(r)
	case args.Update != nil:
		output(runReplace(args.Update))
	case args.Delete != nil:
		output(runMove(args.Delete, "trash"))
	case args.Apps != nil:
		a, e := drafts.DiscoverApps()
		handleDraftsError(e)
		output(a)
	case args.New != nil:
		output(runNew(args.New))
	case args.Create != nil:
		output(runNew(args.Create))
	case args.Prepend != nil:
		output(runModify(args.Prepend, true))
	case args.Append != nil:
		output(runModify(args.Append, false))
	case args.Replace != nil:
		output(runReplace(args.Replace))
	case args.Edit != nil:
		output(runEdit(args.Edit))
	case args.Get != nil:
		if args.DataSource == "local" {
			d, e := localGet(args.Get.UUID)
			handleDraftsError(e)
			view := toDraftView(d, true)
			view.DataSource = "local"
			snapshot, e := loadSnapshot()
			handleDraftsError(e)
			view.Snapshot = &snapshot.Info
			output(view)
			return
		}
		u, e := resolveCommandUUID(args.Get.UUID)
		handleDraftsError(e)
		d, e := drafts.Get(u)
		handleDraftsError(e)
		view := toDraftView(d, true)
		view.DataSource = "live"
		output(view)
	case args.Select != nil:
		output(runSelect())
	case args.List != nil:
		output(runList(args.List))
	case args.Flag != nil:
		output(runFlag(args.Flag, true))
	case args.Unflag != nil:
		output(runFlag(args.Unflag, false))
	case args.Archive != nil:
		output(runMove(args.Archive, "archive"))
	case args.Inbox != nil:
		output(runMove(args.Inbox, "inbox"))
	case args.Trash != nil:
		output(runMove(args.Trash, "trash"))
	case args.Open != nil:
		output(runMove(args.Open, "open"))
	case args.Tag != nil:
		output(runTag(args.Tag, false))
	case args.RemoveTags != nil:
		output(runTag(args.RemoveTags, true))
	case args.Tags != nil:
		configureApp()
		r, e := drafts.Tags()
		handleDraftsError(e)
		output(r)
	case args.Workspaces != nil:
		configureApp()
		r, e := drafts.Resources("workspace")
		handleDraftsError(e)
		output(r)
	case args.Workspace != nil:
		output(runWorkspace(args.Workspace))
	case args.Actions != nil:
		output(runActions(args.Actions))
	case args.Run != nil:
		output(runAction(args.Run))
	case args.Info != nil:
		output(runInfo(args.Info))
	case args.Schema != nil:
		r := getSchema(args.Schema.Command)
		if args.Schema.Detected {
			r = detectedSchema(r)
		}
		output(r)
	case args.Upgrade != nil:
		if plan, ok := mutationPlan("upgrade", struct{}{}, true); ok {
			output(plan)
		} else {
			output(runUpgrade())
		}
	case args.VersionCommand != nil:
		output(runVersion())
	}
}
func runNew(p *NewCmd) interface{} {
	r, e := resolveCreateRequest(p)
	handleDraftsError(e)
	validateContent(r.Content)
	validateNames(r.Tags)
	validateFlagType(r.FlagType)
	if plan, ok := mutationPlan("create", r, false); ok {
		return plan
	}
	configureApp()
	opt := drafts.CreateOptions{Tags: r.Tags, Flagged: r.Flagged, FlagType: r.FlagType, Action: r.Action}
	if r.Folder == "archive" {
		opt.Folder = drafts.FolderArchive
	}
	finish, replayed := beginIdempotency("create", r, func() error { return drafts.PreflightMutation(r.Action, "", r.FlagType) })
	if replayed != nil {
		return replayed
	}
	u, e := drafts.Create(r.Content, opt)
	finish(u, e)
	handleDraftsError(e)
	return mutationResult(u)
}
func runModify(p *PrependCmd, prepend bool) interface{} {
	r, e := resolveModifyRequest(p.Input, p.Message, p.Tag, p.Action, p.UUID)
	handleDraftsError(e)
	validateContent(r.Content)
	validateNames(r.Tags)
	if p.Input != "" && p.Separator != nil {
		outputError("INVALID_INPUT", "cannot combine --input with --separator", "")
	}
	sep := r.Separator
	if p.Input == "" {
		sep = p.Separator
	}
	separator := "\n"
	if sep != nil {
		separator = *sep
	}
	validateContent(separator)
	name := "append"
	if prepend {
		name = "prepend"
	}
	r.Separator = &separator
	if plan, ok := mutationPlan(name, r, false); ok {
		return plan
	}
	u, e := resolveCommandUUID(r.UUID)
	handleDraftsError(e)
	opt := drafts.ModifyOptions{Tags: r.Tags, Action: r.Action, Separator: &separator}
	if prepend {
		e = drafts.Prepend(u, r.Content, opt)
	} else {
		e = drafts.Append(u, r.Content, opt)
	}
	handleDraftsError(e)
	return mutationResult(u)
}
func runReplace(p *ReplaceCmd) interface{} {
	r, e := resolveReplaceRequest(p)
	handleDraftsError(e)
	validateContent(r.Content)
	if plan, ok := mutationPlan("replace", r, true); ok {
		return plan
	}
	u, e := resolveCommandUUID(r.UUID)
	handleDraftsError(e)
	handleDraftsError(drafts.Replace(u, r.Content))
	return mutationResult(u)
}
func runEdit(p *EditCmd) interface{} {
	if plan, ok := mutationPlan("edit", UUIDRequest{UUID: p.UUID}, true); ok {
		return plan
	}
	requireTerminal()
	u, e := resolveCommandUUID(p.UUID)
	handleDraftsError(e)
	d, e := drafts.Get(u)
	handleDraftsError(e)
	text, e := editor(d.Content)
	handleDraftsError(e)
	validateContent(text)
	if text == d.Content {
		return map[string]interface{}{"uuid": u, "status": "unchanged"}
	}
	handleDraftsError(drafts.ReplaceIfUnchanged(u, text, d.Content))
	return mutationResult(u)
}
func runSelect() interface{} {
	if plan, ok := mutationPlan("select", struct{}{}, false); ok {
		return plan
	}
	requireTerminal()
	configureApp()
	ds, e := drafts.Query("", drafts.FilterInbox, drafts.QueryOptions{SortDescending: true})
	handleDraftsError(e)
	var b strings.Builder
	for _, d := range ds {
		fmt.Fprintf(&b, "%s %c %s\n", d.UUID, drafts.Separator, strings.Join(strings.Fields(d.Title), " "))
	}
	u, e := fzfUUID(b.String())
	handleDraftsError(e)
	handleDraftsError(drafts.Select(u))
	return mutationResult(u)
}
func runFlag(p *FlagCmd, flagged bool) interface{} {
	r := FlagRequest{UUID: p.UUID, FlagType: p.Type}
	if p.Input != "" {
		if p.UUID != "" || p.Type != nil {
			outputError("INVALID_INPUT", "cannot combine --input and arguments", "")
		}
		if flagged {
			handleDraftsError(decodeJSONInput(p.Input, &r))
		} else {
			u, e := resolveUUIDRequest(p.Input, "")
			handleDraftsError(e)
			r.UUID = u.UUID
		}
	}
	p.Type = r.FlagType
	validateFlagType(p.Type)
	if !flagged && p.Type != nil {
		outputError("INVALID_INPUT", "unflag does not accept --flag-type", "")
	}

	name := "unflag"
	if flagged {
		name = "flag"
	}
	payload := map[string]interface{}{"uuid": r.UUID, "flagged": flagged}
	if p.Type != nil {
		payload["flagType"] = *p.Type
	}
	if plan, ok := mutationPlan(name, payload, false); ok {
		return plan
	}
	u, e := resolveCommandUUID(r.UUID)
	handleDraftsError(e)
	if p.Type != nil {
		e = drafts.SetFlagType(u, *p.Type)
	} else {
		e = drafts.SetFlagged(u, flagged)
	}
	handleDraftsError(e)
	return mutationResult(u)
}
func runMove(p *MoveCmd, folder string) interface{} {
	r, e := resolveUUIDRequest(p.Input, p.UUID)
	handleDraftsError(e)
	if plan, ok := mutationPlan(folder, r, folder == "trash"); ok {
		return plan
	}
	u, e := resolveCommandUUID(r.UUID)
	handleDraftsError(e)
	if folder == "open" {
		e = drafts.Select(u)
	} else {
		e = drafts.Move(u, folder)
	}
	handleDraftsError(e)
	return mutationResult(u)
}
func runTag(p *TagCmd, remove bool) interface{} {
	r := TagRequest{UUID: p.UUID, Tags: p.Tag}
	if p.Input != "" {
		if p.UUID != "" || len(p.Tag) > 0 {
			outputError("INVALID_INPUT", "cannot combine --input and arguments", "")
		}
		handleDraftsError(decodeJSONInput(p.Input, &r))
	}
	if len(r.Tags) == 0 {
		outputError("INVALID_INPUT", "at least one tag is required", "")
	}
	validateNames(r.Tags)
	name := "tag"
	if remove {
		name = "remove-tags"
	}
	if plan, ok := mutationPlan(name, r, false); ok {
		return plan
	}
	u, e := resolveCommandUUID(r.UUID)
	handleDraftsError(e)
	if remove {
		e = drafts.RemoveTags(u, r.Tags...)
	} else {
		e = drafts.Tag(u, r.Tags...)
	}
	handleDraftsError(e)
	return mutationResult(u)
}
func runWorkspace(p *WorkspaceCmd) interface{} {
	if p.Rename != "" || p.To != "" {
		if p.Rename == "" || p.To == "" || p.List || p.Open != "" {
			outputError("INVALID_INPUT", "rename requires --rename and --to, without --list or --open", "")
		}
		handleDraftsError(drafts.ValidateName(p.Rename))
		handleDraftsError(drafts.ValidateName(p.To))
		if plan, ok := mutationPlan("rename-workspace", WorkspaceRequest{Rename: p.Rename, To: p.To}, true); ok {
			return plan
		}
		configureApp()
		handleDraftsError(drafts.RenameWorkspace(p.Rename, p.To))
		return map[string]interface{}{"renamed": p.Rename, "name": p.To}
	}

	if p.List && p.Open != "" {
		outputError("INVALID_INPUT", "cannot combine --list and --open", "")
	}
	if p.Open != "" {
		handleDraftsError(drafts.ValidateName(p.Open))
		if plan, ok := mutationPlan("open-workspace", WorkspaceRequest{Open: p.Open}, false); ok {
			return plan
		}
		configureApp()
		handleDraftsError(drafts.OpenWorkspace(p.Open))
		return WorkspaceResult{Opened: p.Open}
	}
	configureApp()
	if p.List {
		r, e := drafts.Workspaces()
		handleDraftsError(e)
		return WorkspaceResult{Workspaces: r, Count: len(r)}
	}
	r, e := drafts.CurrentWorkspace()
	handleDraftsError(e)
	return WorkspaceResult{Current: r}
}
func runActions(p *ActionsCmd) interface{} {
	configureApp()
	resources, e := drafts.Actions()
	handleDraftsError(e)
	r := []drafts.NamedResource{}
	for _, a := range resources {
		if strings.Contains(strings.ToLower(a.Name), strings.ToLower(p.Search)) {
			r = append(r, a)
		}
	}
	return map[string]interface{}{"actions": r, "count": len(r)}
}
func parseFilter(value string) (drafts.Filter, error) {
	switch value {
	case "inbox":
		return drafts.FilterInbox, nil
	case "archive":
		return drafts.FilterArchive, nil
	case "trash":
		return drafts.FilterTrash, nil
	case "flagged":
		return drafts.FilterFlagged, nil
	case "all":
		return drafts.FilterAll, nil
	}
	return drafts.FilterInbox, fmt.Errorf("filter must be inbox, archive, trash, flagged, or all, got %q", value)
}
func runList(p *ListCmd) interface{} {
	filter, e := parseFilter(p.Filter)
	handleDraftsError(e)
	opt, e := queryOptions(p)
	handleDraftsError(e)
	if globalArgs.DataSource == "local" {
		return localList(p, filter, opt)
	}
	configureApp()
	if p.Count {
		count, e := drafts.Count(p.Search, filter, opt, p.Workspace)
		handleDraftsError(e)
		return map[string]interface{}{"count": count, "filter": p.Filter, "data_source": "live"}
	}
	var ds []drafts.Draft
	if p.Workspace != "" {
		ds, e = drafts.QueryWorkspace(p.Workspace, p.Search, filter, opt)
	} else {
		ds, e = drafts.Query(p.Search, filter, opt)
	}
	handleDraftsError(e)
	views := make([]DraftView, len(ds))
	for i, d := range ds {
		views[i] = toDraftView(d, p.Full)
	}
	return ListResult{DataSource: "live", Drafts: views, Count: len(views), Filter: p.Filter, Limit: p.Limit, Full: p.Full, Search: p.Search, Workspace: p.Workspace}
}
func runAction(p *RunCmd) interface{} {
	r, e := resolveRunRequest(p)
	handleDraftsError(e)
	handleDraftsError(drafts.ValidateName(r.Action))
	validateContent(r.Content)
	if r.UUID != "" && r.Content != "" {
		outputError("INVALID_INPUT", "run accepts uuid or content, not both", "")
	}
	if plan, ok := mutationPlan("run", r, false); ok {
		return plan
	}
	configureApp()
	finish, replayed := beginIdempotency("run", r, func() error { return drafts.PreflightMutation(r.Action, r.UUID, nil) })
	if replayed != nil {
		return replayed
	}
	if r.UUID != "" {
		handleDraftsError(drafts.ValidateUUID(r.UUID))
		e := drafts.RunActionOnDraft(r.Action, r.UUID)
		finish(r.UUID, e)
		handleDraftsError(e)
		return RunResult{Action: r.Action, UUID: r.UUID, Status: "submitted"}
	}
	r2, e := drafts.RunAction(r.Action, r.Content)
	finish(r2.UUID, e)
	handleDraftsError(e)
	return RunResult{Action: r.Action, UUID: r2.UUID, CreatedDraft: true, Status: "submitted"}
}
func resolveCommandUUID(uuid string) (string, error) {
	if uuid != "" {
		if e := drafts.ValidateUUID(uuid); e != nil {
			return "", e
		}
	}
	configureApp()
	if uuid != "" {
		return uuid, nil
	}
	return drafts.Active()
}
func handleDraftsError(err error) {
	if err == nil {
		return
	}
	var e *drafts.Error
	if errors.As(err, &e) {
		outputDetailedError(e)
	}
	switch {
	case errors.Is(err, drafts.ErrDraftNotFound):
		outputError("DRAFT_NOT_FOUND", err.Error(), "Use drafts list")
	case errors.Is(err, drafts.ErrActionNotFound):
		outputError("ACTION_NOT_FOUND", err.Error(), "Use drafts actions")
	case errors.Is(err, drafts.ErrWorkspaceNotFound):
		outputError("WORKSPACE_NOT_FOUND", err.Error(), "Use drafts workspaces")
	default:
		var pathError *os.PathError
		if errors.As(err, &pathError) {
			outputError("IO_ERROR", err.Error(), "Check the supplied path and filesystem permissions")
		}
		outputError("INVALID_INPUT", err.Error(), "")
	}
}
