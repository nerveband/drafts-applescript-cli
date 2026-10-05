package main

import (
	"encoding/json"
	"fmt"
	"github.com/nerveband/drafts-applescript-cli/v4/pkg/drafts"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

var configured bool
var configuredApp drafts.AppInfo

func configureApp() {
	if globalArgs.DataSource == "local" {
		outputError("INVALID_INPUT", "local data source supports list/get only", "")
	}
	if configured {
		return
	}
	var app drafts.AppInfo
	var err error
	if filepath.IsAbs(globalArgs.App) {
		app, err = drafts.InspectApp(globalArgs.App)
	} else {
		var apps []drafts.AppInfo
		apps, err = drafts.DiscoverApps()
		if err == nil {
			app, err = drafts.SelectApp(apps, globalArgs.App, globalArgs.Channel)
		}
	}
	handleDraftsError(err)
	if globalArgs.Channel == "beta" && app.Channel != "beta" {
		outputError("APP_NOT_FOUND", "selected app is not identified as beta", "Use --channel auto with an explicit --app when channel metadata is unknown")
	}
	if globalArgs.Channel == "stable" && app.Channel == "beta" {
		outputError("APP_NOT_FOUND", "selected app is identified as beta", "")
	}
	configuredApp = app
	configured = true
	drafts.Configure(app, commandTimeout)
}
func validateContent(text string) {
	if !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		outputError("INVALID_INPUT", "content must be valid UTF-8 without NUL", "")
	}
}
func validateNames(names []string) {
	for _, name := range names {
		handleDraftsError(drafts.ValidateName(name))
	}
}
func validateFlagType(value *int) {
	if value != nil && (*value < 0 || *value > 6) {
		outputError("INVALID_INPUT", "flag type must be 0 through 6", "")
	}
}
func mutationResult(uuid string) interface{} {
	return map[string]interface{}{"uuid": uuid, "status": "updated"}
}
func mutationPlan(command string, request interface{}, destructive bool) (interface{}, bool) {
	if globalArgs.DataSource == "local" {
		outputError("INVALID_INPUT", "mutations require --data-source live", "")
	}
	// Validate target identifiers even when no app is installed and no events will be sent.
	data, _ := json.Marshal(request)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(data, &fields)
	var uuid string
	_ = json.Unmarshal(fields["uuid"], &uuid)
	if destructive && (command == "replace" || command == "trash") && uuid == "" {
		outputError("INVALID_INPUT", command+" requires an explicit UUID", "Preview and commit the same UUID")
	}
	if uuid != "" {
		handleDraftsError(drafts.ValidateUUID(uuid))
	}
	var action string
	_ = json.Unmarshal(fields["action"], &action)
	if action != "" {
		handleDraftsError(drafts.ValidateName(action))
	}
	if globalArgs.DryRun {
		preconditions := []string{"selected app supports required capabilities", "Drafts is running", "requested targets and actions exist"}
		switch command {
		case "upgrade":
			preconditions = []string{"GitHub is reachable", "release checksums validate", "executable is writable"}
		case "profile-save", "feedback":
			preconditions = []string{"local configuration or state directory is writable"}
		}
		return map[string]interface{}{"dry_run": true, "command": command, "request": request, "requires_commit": destructive, "app": globalArgs.App, "channel": globalArgs.Channel, "availability": "not_checked", "preconditions": preconditions}, true
	}
	if destructive && !globalArgs.Commit {
		outputError("COMMIT_REQUIRED", command+" requires --commit", "Use --dry-run to preview first")
	}
	return nil, false
}
func requireTerminal() {
	if !stdinIsTerminal() {
		outputError("INTERACTIVE_REQUIRED", "this command requires a terminal", "Use get, list, or replace --input for automation")
	}
}
func queryOptions(p *ListCmd) (drafts.QueryOptions, error) {
	if p.Limit < 0 {
		return drafts.QueryOptions{}, fmt.Errorf("limit must be at least 0")
	}
	opt := drafts.QueryOptions{Tags: p.Tag, OmitTags: p.OmitTag, SortDescending: p.Order == "desc", SortFlaggedToTop: p.FlaggedFirst, Limit: p.Limit, Full: p.Full, FlagType: p.FlagType}
	switch p.Sort {
	case "created":
		opt.Sort = drafts.SortCreated
	case "modified":
		opt.Sort = drafts.SortModified
	case "accessed":
		opt.Sort = drafts.SortAccessed
	default:
		return opt, fmt.Errorf("sort must be created, modified, or accessed, got %q", p.Sort)
	}
	if p.Order != "asc" && p.Order != "desc" {
		return opt, fmt.Errorf("order must be asc or desc, got %q", p.Order)
	}
	if p.FlagType != nil && (*p.FlagType < 0 || *p.FlagType > 6) {
		return opt, fmt.Errorf("flag type must be 0 through 6")
	}
	for _, name := range append(append([]string{}, p.Tag...), p.OmitTag...) {
		if err := drafts.ValidateName(name); err != nil {
			return opt, err
		}
	}
	if p.Workspace != "" {
		if err := drafts.ValidateName(p.Workspace); err != nil {
			return opt, err
		}
	}
	for _, bound := range []struct {
		input string
		dest  *string
	}{{p.CreatedAfter, &opt.CreatedAfter}, {p.CreatedBefore, &opt.CreatedBefore}, {p.ModifiedAfter, &opt.ModifiedAfter}, {p.ModifiedBefore, &opt.ModifiedBefore}} {
		if bound.input == "" {
			continue
		}
		date, e := time.Parse(time.RFC3339, bound.input)
		if e != nil {
			date, e = time.Parse("2006-01-02", bound.input)
		}
		if e != nil {
			return opt, fmt.Errorf("date bounds must be RFC3339 or YYYY-MM-DD")
		}
		*bound.dest = date.UTC().Format(time.RFC3339Nano)
	}
	for _, pair := range [][2]string{{opt.CreatedAfter, opt.CreatedBefore}, {opt.ModifiedAfter, opt.ModifiedBefore}} {
		if pair[0] != "" && pair[1] != "" {
			a, _ := time.Parse(time.RFC3339, pair[0])
			b, _ := time.Parse(time.RFC3339, pair[1])
			if !a.Before(b) {
				return opt, fmt.Errorf("after bound must precede before bound")
			}
		}
	}
	return opt, nil
}
func detectedSchema(schema interface{}) interface{} {
	configureApp()
	annotate := func(tool *ToolDefinition) {
		name := strings.TrimPrefix(tool.Name, "drafts_")
		cap := "core"
		switch name {
		case "workspace", "workspaces":
			cap = "workspace"
		case "actions":
			cap = "action"
		case "run":
			cap = "command.perform"
		case "open", "select":
			cap = "open.draft"
		case "tags":
			cap = "tag"
		case "apps", "schema", "version", "info", "upgrade", "profile", "config", "feedback", "jobs", "skills":
			cap = ""
		}
		if props, ok := tool.Parameters["properties"].(map[string]interface{}); ok {
			if !configuredApp.Capabilities["flag_type"] {
				delete(props, "flagType")
			}
			if name == "workspace" && !configuredApp.Capabilities["open.workspace"] {
				delete(props, "open")
			}
		}
		if props, ok := tool.Parameters["properties"].(map[string]interface{}); ok {
			if name == "workspace" && !configuredApp.Capabilities["workspace.rename"] {
				delete(props, "rename")
				delete(props, "to")
			}
		}
		tool.RequiresCapability = cap
		available := cap == "" || configuredApp.Capabilities[cap]
		tool.Available = &available
	}
	switch s := schema.(type) {
	case SchemaDocument:
		for i := range s.Tools {
			annotate(&s.Tools[i])
		}
		schema = s
	case ToolDefinition:
		annotate(&s)
		schema = s
	}
	return map[string]interface{}{"schema": schema, "app": configuredApp, "flag_type_available": configuredApp.Capabilities["flag_type"], "workspace_open_available": configuredApp.Capabilities["open.workspace"]}
}
