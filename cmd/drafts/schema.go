package main

import (
	"github.com/nerveband/drafts-applescript-cli/v4/pkg/drafts"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type SchemaDocument struct {
	SchemaVersion string                 `json:"schema_version"`
	Scope         string                 `json:"scope"`
	GlobalOptions map[string]interface{} `json:"global_options"`
	ExitCodes     map[string]int         `json:"exit_codes"`
	ProfileNames  []string               `json:"profile_names"`
	FeedbackURL   string                 `json:"feedback_url"`

	Name    string           `json:"name"`
	Version string           `json:"version"`
	Tools   []ToolDefinition `json:"tools"`
}

type ToolDefinition struct {
	CommitConditions []string               `json:"commit_conditions,omitempty"`
	Examples         []string               `json:"examples"`
	Scope            string                 `json:"scope"`
	Response         map[string]interface{} `json:"response"`
	Idempotency      string                 `json:"idempotency"`

	Available          *bool                  `json:"available,omitempty"`
	RequiresCapability string                 `json:"requires_capability,omitempty"`
	Mutates            bool                   `json:"mutates"`
	RequiresCommit     bool                   `json:"requires_commit"`
	SupportsDryRun     bool                   `json:"supports_dry_run"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	Parameters         map[string]interface{} `json:"parameters"`
	Aliases            []string               `json:"aliases,omitempty"`
	RawInputFlag       string                 `json:"raw_input_flag,omitempty"`
	Interactive        bool                   `json:"interactive,omitempty"`
}

// Schema returns the tool-use formatted schema for all commands.
func getSchema(command string) interface{} {
	tools := getTools()
	schema := SchemaDocument{
		Name:          "drafts",
		SchemaVersion: "1", Scope: "local", GlobalOptions: globalOptionsSchema(), ExitCodes: errorCodeManifest(), ProfileNames: profileNames(), FeedbackURL: issuesURL,
		Version: version,
		Tools:   tools,
	}

	if command == "" {
		return schema
	}

	command = strings.TrimSpace(command)
	command = strings.TrimPrefix(command, "drafts_")

	for _, tool := range tools {
		if strings.TrimPrefix(tool.Name, "drafts_") == command {
			return tool
		}
		for _, alias := range tool.Aliases {
			if alias == command {
				return tool
			}
		}
	}

	outputError("UNKNOWN_COMMAND",
		"Unknown command: "+command,
		"Use 'drafts schema' to see all available commands")
	return nil
}

func getTools() []ToolDefinition {
	tools := []ToolDefinition{
		newTool("drafts_new", "Create a new draft in Drafts.app", CreateRequest{}, []string{"create"}, "--input", false),
		newTool("drafts_get", "Get a draft by UUID, returns full draft metadata", GetRequest{}, nil, "", false),
		newTool("drafts_list", "List drafts with filtering, workspace scoping, and token-aware summaries by default", ListRequest{}, nil, "", false),
		newTool("drafts_append", "Append text to an existing draft", ModifyRequest{}, nil, "--input", false),
		newTool("drafts_prepend", "Prepend text to an existing draft", ModifyRequest{}, nil, "--input", false),
		newTool("drafts_replace", "Replace the content of an existing draft", ReplaceRequest{}, []string{"update"}, "--input", false),
		newTool("drafts_edit", "Open a draft in $EDITOR and replace its content with the edited result", GetRequest{}, nil, "", true),
		newTool("drafts_select", "Interactively select the active draft with fzf", struct{}{}, nil, "", true),
		newTool("drafts_flag", "Flag a draft", FlagRequest{}, nil, "--input", false),
		newTool("drafts_unflag", "Unflag a draft", UUIDRequest{}, nil, "--input", false),
		newTool("drafts_workspace", "Show current workspace, list all workspaces, or open a workspace by name", WorkspaceRequest{}, nil, "", false),
		newTool("drafts_actions", "List available Drafts actions, optionally filtered by substring", ActionsRequest{}, nil, "", false),
		newTool("drafts_run", "Run a Drafts action on text or an existing draft", RunRequest{}, nil, "--input", false),
		newTool("drafts_info", "Get environment information and diagnostics. Use verbose mode for actions, tags, and workspaces; use test_permissions to verify automation access.", InfoRequest{}, nil, "", false),
		newTool("drafts_schema", "Return the machine-readable command contract for the full CLI or a single command alias.", SchemaRequest{}, nil, "", false),
		newTool("drafts_upgrade", "Upgrade to the latest version from GitHub releases.", struct{}{}, nil, "", false),
		newTool("drafts_version", "Show current CLI version information including OS and architecture.", struct{}{}, nil, "", false),
	}
	for _, name := range []string{"archive", "inbox", "trash", "open"} {
		tools = append(tools, newTool("drafts_"+name, "Move or open a draft", UUIDRequest{}, nil, "--input", false))
	}
	for _, name := range []string{"tag", "remove-tags"} {
		tools = append(tools, newTool("drafts_"+name, "Modify draft tags", TagRequest{}, nil, "--input", false))
	}
	for _, name := range []string{"apps", "tags", "workspaces"} {
		tools = append(tools, newTool("drafts_"+name, "List structured installation or resource metadata", struct{}{}, nil, "", false))
	}
	tools = append(tools, newTool("drafts_sync", "Save a bounded CLI-owned snapshot", ListRequest{}, nil, "", false), newTool("drafts_profile", "Manage non-secret profiles", ProfileRequest{}, nil, "", false), newTool("drafts_config", "Inspect setting sources", struct{}{}, nil, "", false), newTool("drafts_feedback", "Record local feedback", FeedbackRequest{}, nil, "", false), newTool("drafts_jobs", "Inspect duplicate-prevention receipts", JobsRequest{}, nil, "", false), newTool("drafts_skills", "Emit bundled task guidance", struct{}{}, nil, "", false))
	for i := range tools {
		name := strings.TrimPrefix(tools[i].Name, "drafts_")
		switch name {
		case "workspace", "new", "append", "prepend", "replace", "edit", "select", "flag", "unflag", "run", "archive", "inbox", "trash", "open", "tag", "remove-tags", "upgrade":
			tools[i].Mutates = true
			tools[i].SupportsDryRun = true
		}
		switch name {
		case "replace", "edit", "trash", "upgrade":
			tools[i].RequiresCommit = true
		}
		if name == "trash" {
			tools[i].Parameters["required"] = []string{"uuid"}
			tools[i].Parameters["properties"].(map[string]interface{})["uuid"].(map[string]interface{})["description"] = "Explicit UUID of the draft to move to Trash"
		}
	}
	for i := range tools {
		name := strings.TrimPrefix(tools[i].Name, "drafts_")
		tools[i].Scope = "local"
		tools[i].Examples = []string{commandExamples[name]}
		tools[i].Response = responseSchema(name)
		tools[i].Idempotency = "read_only"
		if tools[i].Mutates {
			tools[i].Idempotency = "inspect_before_retry"
		}
		switch name {
		case "new", "run":
			tools[i].Idempotency = "optional_durable_key"
		case "flag", "unflag", "archive", "inbox", "trash", "tag", "remove-tags", "replace":
			tools[i].Idempotency = "idempotent_state_assignment"
		}
		if name == "trash" {
			tools[i].Aliases = []string{"delete"}
		}
		if name == "workspace" {
			tools[i].CommitConditions = []string{"rename"}
		}
		if name == "schema" {
			tools[i].Aliases = []string{"agent-context"}
		}
		if name == "profile" || name == "feedback" || name == "sync" {
			tools[i].Mutates = true
			tools[i].SupportsDryRun = true
		}
	}
	return tools
}

func newTool(name, description string, request interface{}, aliases []string, rawInputFlag string, interactive bool) ToolDefinition {
	return ToolDefinition{
		Name:         name,
		Description:  description,
		Parameters:   requestSchema(request),
		Aliases:      aliases,
		RawInputFlag: rawInputFlag,
		Interactive:  interactive,
	}
}

func buildParametersSchema(request interface{}) map[string]interface{} {
	t := reflect.TypeOf(request)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	properties := map[string]interface{}{}
	required := []string{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name, omitempty := parseJSONField(field.Tag.Get("json"))
		if name == "" {
			continue
		}

		property := typeSchema(field.Type)

		if field.Type.Kind() == reflect.Slice {
			property["items"] = typeSchema(field.Type.Elem())
		}

		if desc := field.Tag.Get("desc"); desc != "" {
			property["description"] = desc
		}

		if enumTag := field.Tag.Get("enum"); enumTag != "" {
			property["enum"] = strings.Split(enumTag, ",")
		}

		if defaultTag := field.Tag.Get("default"); defaultTag != "" {
			property["default"] = parseDefaultValue(field.Type, defaultTag)
		}

		if name == "uuid" {
			property["pattern"] = `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`
		}
		if name == "flagType" {
			property["minimum"] = 0
			property["maximum"] = 6
		}
		properties[name] = property
		if !omitempty {
			required = append(required, name)
		}
	}

	return map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
		"required":             required,
	}
}

func parseJSONField(tag string) (string, bool) {
	if tag == "" || tag == "-" {
		return "", false
	}

	parts := strings.Split(tag, ",")
	name := parts[0]
	omitempty := false
	for _, option := range parts[1:] {
		if option == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty
}

func fieldJSONType(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		return fieldJSONType(t.Elem())
	}
	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice:
		return "array"
	case reflect.Struct, reflect.Map:
		return "object"
	default:
		return "string"
	}
}

func parseDefaultValue(t reflect.Type, value string) interface{} {
	switch t.Kind() {
	case reflect.Bool:
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return value
}

func globalOptionsSchema() map[string]interface{} {
	result := map[string]interface{}{}
	t := reflect.TypeOf(Args{})
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("arg")
		if strings.HasPrefix(tag, "subcommand:") {
			continue
		}
		parts := strings.Split(tag, ",")
		name := parts[len(parts)-1]
		name = strings.TrimPrefix(name, "--")
		property := map[string]interface{}{"type": fieldJSONType(f.Type), "description": f.Tag.Get("help")}
		if value, ok := f.Tag.Lookup("default"); ok {
			property["default"] = parseDefaultValue(f.Type, value)
		}
		result[name] = property
	}
	return result
}
func errorCodeManifest() map[string]int {
	result := map[string]int{}
	for _, code := range []string{"INVALID_INPUT", "COMMIT_REQUIRED", "INTERACTIVE_REQUIRED", "DRAFT_NOT_FOUND", "ACTION_NOT_FOUND", "WORKSPACE_NOT_FOUND", "APP_NOT_FOUND", "DRAFTS_NOT_RUNNING", "PERMISSION_DENIED", "UNSUPPORTED_CAPABILITY", "CONFLICT", "TIMEOUT", "PARTIAL_FAILURE", "INVALID_RESPONSE", "OUTPUT_ERROR", "IO_ERROR", "UPDATE_ERROR", "SNAPSHOT_NOT_FOUND", "AMBIGUOUS_APP", "UNSUPPORTED_PLATFORM", "APPLESCRIPT_ERROR"} {
		result[code] = exitCode(code)
	}
	return result
}
func profileNames() []string {
	f, err := loadProfiles()
	names := []string{}
	if err == nil {
		for n := range f.Profiles {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}
func responseSchema(name string) map[string]interface{} {
	data := map[string]interface{}{"type": "object"}
	switch name {
	case "get":
		data = typeSchema(reflect.TypeOf(DraftView{}))
	case "list":
		data = map[string]interface{}{"anyOf": []interface{}{typeSchema(reflect.TypeOf(ListResult{})), typeSchema(reflect.TypeOf(CountResult{}))}}
	case "run":
		data = map[string]interface{}{"anyOf": []interface{}{typeSchema(reflect.TypeOf(RunResult{})), typeSchema(reflect.TypeOf(MutationReceipt{}))}}
	case "apps":
		data = typeSchema(reflect.TypeOf([]drafts.AppInfo{}))
	case "tags", "workspaces":
		data = typeSchema(reflect.TypeOf([]drafts.NamedResource{}))
	case "actions":
		data = typeSchema(reflect.TypeOf(ActionsResult{}))
	case "workspace":
		data = map[string]interface{}{"anyOf": []interface{}{typeSchema(reflect.TypeOf(WorkspaceResult{})), map[string]interface{}{"type": "object", "required": []string{"renamed", "name"}, "properties": map[string]interface{}{"renamed": map[string]interface{}{"type": "string"}, "name": map[string]interface{}{"type": "string"}}}}}
	case "new", "append", "prepend", "replace", "edit", "select", "flag", "unflag", "archive", "inbox", "trash", "open", "tag", "remove-tags":
		data = typeSchema(reflect.TypeOf(MutationReceipt{}))
	}
	// A dry-run returns a plan rather than the command's ordinary result.
	data = map[string]interface{}{"anyOf": []interface{}{data, map[string]interface{}{"type": "object", "required": []string{"dry_run", "command", "request"}, "properties": map[string]interface{}{"dry_run": map[string]interface{}{"const": true}, "command": map[string]interface{}{"type": "string"}, "request": map[string]interface{}{"type": "object"}}}}}
	return map[string]interface{}{"type": "object", "required": []string{"success", "data"}, "properties": map[string]interface{}{"success": map[string]interface{}{"const": true}, "data": data}}
}

type MutationReceipt struct {
	UUID            string `json:"uuid"`
	Status          string `json:"status"`
	OriginalCommand string `json:"original_command,omitempty"`
}
type CountResult struct {
	Count      int           `json:"count"`
	Filter     string        `json:"filter,omitempty"`
	DataSource string        `json:"data_source"`
	Snapshot   *SnapshotInfo `json:"snapshot,omitempty"`
}

type ProfileRequest struct {
	Operation string  `json:"operation,omitempty" enum:"list,show,save"`
	Name      string  `json:"name,omitempty"`
	Settings  Profile `json:"settings,omitempty"`
}
type FeedbackRequest struct {
	Message string `json:"message"`
}
type JobsRequest struct {
	Key string `json:"key,omitempty"`
}

func typeSchema(t reflect.Type) map[string]interface{} {
	if t.Kind() == reflect.Pointer {
		return typeSchema(t.Elem())
	}
	if t.Kind() == reflect.Map {
		return map[string]interface{}{"type": "object", "additionalProperties": typeSchema(t.Elem())}
	}
	if t.Kind() == reflect.Interface {
		return map[string]interface{}{}
	}
	if t.Kind() == reflect.Struct {
		return buildParametersSchema(reflect.New(t).Elem().Interface())
	}
	if t.Kind() == reflect.Slice {
		return map[string]interface{}{"type": "array", "items": typeSchema(t.Elem())}
	}
	return map[string]interface{}{"type": fieldJSONType(t)}
}

func requestSchema(request interface{}) map[string]interface{} {
	schema := buildParametersSchema(request)
	if _, ok := request.(RunRequest); ok {
		schema["oneOf"] = []interface{}{map[string]interface{}{"required": []string{"uuid"}, "not": map[string]interface{}{"required": []string{"content"}}}, map[string]interface{}{"required": []string{"content"}, "not": map[string]interface{}{"required": []string{"uuid"}}}}
	}
	return schema
}
