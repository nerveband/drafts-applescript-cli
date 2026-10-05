package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/nerveband/drafts-applescript-cli/v4/pkg/drafts"
)

type DraftView struct {
	DataSource        string        `json:"data_source,omitempty"`
	Snapshot          *SnapshotInfo `json:"snapshot,omitempty"`
	AccessedAt        string        `json:"accessedAt,omitempty"`
	FlagType          *int          `json:"flagType,omitempty"`
	UUID              string        `json:"uuid"`
	Title             string        `json:"title"`
	Tags              []string      `json:"tags"`
	IsFlagged         bool          `json:"isFlagged"`
	IsArchived        bool          `json:"isArchived"`
	IsTrashed         bool          `json:"isTrashed"`
	Folder            string        `json:"folder"`
	CreatedAt         string        `json:"createdAt"`
	ModifiedAt        string        `json:"modifiedAt"`
	Permalink         string        `json:"permalink"`
	Content           *string       `json:"content,omitempty"`
	CreatedLatitude   *float64      `json:"createdLatitude,omitempty"`
	CreatedLongitude  *float64      `json:"createdLongitude,omitempty"`
	ModifiedLatitude  *float64      `json:"modifiedLatitude,omitempty"`
	ModifiedLongitude *float64      `json:"modifiedLongitude,omitempty"`
}

type ListResult struct {
	DataSource string        `json:"data_source"`
	Snapshot   *SnapshotInfo `json:"snapshot,omitempty"`
	Drafts     []DraftView   `json:"drafts"`
	Count      int           `json:"count"`
	Filter     string        `json:"filter"`
	Limit      int           `json:"limit"`
	Full       bool          `json:"full"`
	Search     string        `json:"search,omitempty"`
	Workspace  string        `json:"workspace,omitempty"`
}

type RunResult struct {
	Status       string     `json:"status"`
	Action       string     `json:"action"`
	UUID         string     `json:"uuid"`
	CreatedDraft bool       `json:"createdDraft"`
	Draft        *DraftView `json:"draft,omitempty"`
}

type ActionsResult struct {
	Actions []drafts.NamedResource `json:"actions"`
	Count   int                    `json:"count"`
	Search  string                 `json:"search,omitempty"`
}

type WorkspaceResult struct {
	Current    string   `json:"current,omitempty"`
	Opened     string   `json:"opened,omitempty"`
	Workspaces []string `json:"workspaces,omitempty"`
	Count      int      `json:"count,omitempty"`
}

type CreateRequest struct {
	FlagType *int     `json:"flagType,omitempty" desc:"Capability-gated flag type, 0 through 6"`
	Content  string   `json:"content" desc:"The draft content"`
	Tags     []string `json:"tags,omitempty" desc:"Tags to apply to the draft"`
	Folder   string   `json:"folder,omitempty" enum:"inbox,archive" default:"inbox" desc:"Folder to create draft in"`
	Flagged  bool     `json:"flagged,omitempty" default:"false" desc:"Whether to flag the draft"`
	Action   string   `json:"action,omitempty" desc:"Action name to run after creation"`
}

type ModifyRequest struct {
	Separator *string  `json:"separator,omitempty" desc:"Separator between existing and new content, default newline"`
	UUID      string   `json:"uuid,omitempty" desc:"UUID of the draft (omit for active draft)"`
	Content   string   `json:"content" desc:"Text to append or prepend"`
	Tags      []string `json:"tags,omitempty" desc:"Tags to add"`
	Action    string   `json:"action,omitempty" desc:"Action name to run after modification"`
}

type ReplaceRequest struct {
	UUID    string `json:"uuid" desc:"Explicit UUID of the draft to replace"`
	Content string `json:"content" desc:"New content for the draft"`
}

type GetRequest struct {
	UUID string `json:"uuid,omitempty" desc:"UUID of the draft (omit for active draft)"`
}

type ListRequest struct {
	OmitTags       []string `json:"omitTags,omitempty" desc:"Exclude these tags"`
	Sort           string   `json:"sort,omitempty" enum:"created,modified,accessed" default:"created"`
	Order          string   `json:"order,omitempty" enum:"asc,desc" default:"desc"`
	FlaggedFirst   bool     `json:"flaggedFirst,omitempty"`
	FlagType       *int     `json:"flagType,omitempty" desc:"Capability-gated type, 0 through 6"`
	CreatedAfter   string   `json:"createdAfter,omitempty" desc:"Exclusive RFC3339 or date bound"`
	CreatedBefore  string   `json:"createdBefore,omitempty"`
	ModifiedAfter  string   `json:"modifiedAfter,omitempty"`
	ModifiedBefore string   `json:"modifiedBefore,omitempty"`
	Count          bool     `json:"count,omitempty" desc:"Count without fetching draft bodies"`

	Filter    string   `json:"filter,omitempty" enum:"inbox,flagged,archive,trash,all" default:"inbox" desc:"Filter drafts by folder"`
	Tags      []string `json:"tags,omitempty" desc:"Filter by tags"`
	Search    string   `json:"search,omitempty" desc:"Search draft content"`
	Workspace string   `json:"workspace,omitempty" desc:"Filter by workspace name"`
	Limit     int      `json:"limit,omitempty" default:"20" desc:"Maximum drafts to return (0 for all)"`
	Full      bool     `json:"full,omitempty" default:"false" desc:"Include full draft content and location fields"`
}

type RunRequest struct {
	Action  string `json:"action" desc:"Name of the action to run"`
	Content string `json:"content,omitempty" desc:"Text to process when not targeting an existing draft"`
	UUID    string `json:"uuid,omitempty" desc:"UUID of the draft to run the action on"`
}

type UUIDRequest struct {
	UUID string `json:"uuid,omitempty" desc:"UUID of the draft (omit for active draft)"`
}

type WorkspaceRequest struct {
	Rename string `json:"rename,omitempty" desc:"Old workspace name, requires to and commit"`
	To     string `json:"to,omitempty" desc:"New workspace name"`
	List   bool   `json:"list,omitempty" default:"false" desc:"List all workspaces instead of showing current"`
	Open   string `json:"open,omitempty" desc:"Open a workspace by name"`
}

type ActionsRequest struct {
	Search string `json:"search,omitempty" desc:"Filter action names by substring"`
}

type InfoRequest struct {
	Counts          bool `json:"counts,omitempty" desc:"Explicitly count each folder"`
	Verbose         bool `json:"verbose,omitempty" default:"false" desc:"Show full lists of actions, tags, and workspaces"`
	TestPermissions bool `json:"test_permissions,omitempty" default:"false" desc:"Read-only automation count probe"`
}

type SchemaRequest struct {
	Detected bool   `json:"detected,omitempty" desc:"Include selected app availability"`
	Command  string `json:"command,omitempty" desc:"Command name or alias (omit for the full schema)"`
}

func toDraftView(d drafts.Draft, full bool) DraftView {
	view := DraftView{
		UUID:       d.UUID,
		AccessedAt: d.AccessedAt, FlagType: d.FlagType,
		Title:      d.Title,
		Tags:       d.Tags,
		IsFlagged:  d.IsFlagged,
		IsArchived: d.IsArchived,
		IsTrashed:  d.IsTrashed,
		Folder:     d.Folder,
		CreatedAt:  d.CreatedAt,
		ModifiedAt: d.ModifiedAt,
		Permalink:  d.Permalink,
	}

	if full {
		content := d.Content
		createdLatitude := d.CreatedLatitude
		createdLongitude := d.CreatedLongitude
		modifiedLatitude := d.ModifiedLatitude
		modifiedLongitude := d.ModifiedLongitude
		view.Content = &content
		view.CreatedLatitude = &createdLatitude
		view.CreatedLongitude = &createdLongitude
		view.ModifiedLatitude = &modifiedLatitude
		view.ModifiedLongitude = &modifiedLongitude
	}

	return view
}

func decodeJSONInput(raw string, dest interface{}) error {
	var data []byte
	var err error

	if strings.HasPrefix(raw, "@") {
		data, err = os.ReadFile(strings.TrimPrefix(raw, "@"))
		if err != nil {
			return err
		}
	} else if raw == "-" {
		var text string
		text, err = readStdin()
		data = []byte(text)
		if err != nil {
			return err
		}
	} else {
		data = []byte(raw)
	}

	if !utf8.Valid(data) {
		return fmt.Errorf("input must be valid UTF-8")
	}
	fields := map[string]json.RawMessage{}
	scanner := json.NewDecoder(bytes.NewReader(data))
	token, err := scanner.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('{') {
		return fmt.Errorf("input must be a JSON object")
	}
	for scanner.More() {
		token, err := scanner.Token()
		if err != nil {
			return err
		}
		name := token.(string)
		if _, ok := fields[name]; ok {
			return fmt.Errorf("duplicate field: %s", name)
		}
		var value json.RawMessage
		if err := scanner.Decode(&value); err != nil {
			return err
		}
		fields[name] = value
	}
	if _, err := scanner.Token(); err != nil {
		return err
	}
	if _, err := scanner.Token(); err != io.EOF {
		return fmt.Errorf("input must contain a single JSON object")
	}
	if len(data) > 16*1024*1024 {
		return fmt.Errorf("input exceeds 16 MiB")
	}
	if _, ok := dest.(*RunRequest); ok {
		_, hasUUID := fields["uuid"]
		_, hasContent := fields["content"]
		if hasUUID == hasContent {
			return fmt.Errorf("run requires exactly one of uuid or content")
		}
	}
	if rawUUID, ok := fields["uuid"]; ok {
		var uuid string
		if err := json.Unmarshal(rawUUID, &uuid); err != nil {
			return err
		}
		if uuid == "" {
			return fmt.Errorf("uuid cannot be empty; omit it to target the active draft")
		}
	}

	if fields == nil {
		return fmt.Errorf("input must be a JSON object")
	}
	for name, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("%s cannot be null", name)
		}
	}
	t := reflect.TypeOf(dest).Elem()
	for i := 0; i < t.NumField(); i++ {
		name, optional := parseJSONField(t.Field(i).Tag.Get("json"))
		if name != "" && !optional {
			if _, ok := fields[name]; !ok {
				return fmt.Errorf("missing required field: %s", name)
			}
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return err
	}

	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("input must contain a single JSON object")
	}

	return nil
}

func resolveCreateRequest(param *NewCmd) (CreateRequest, error) {
	if param.Input != "" {
		if globalArgs.Stdin || globalArgs.TextFile != "" {
			return CreateRequest{}, fmt.Errorf("cannot combine --input with --stdin or --text-file")
		}
		if param.Message != "" || len(param.Tag) > 0 || param.Archive || param.Flagged || param.Action != "" || param.FlagType != nil {
			return CreateRequest{}, fmt.Errorf("cannot combine --input with positional or flag arguments")
		}
		var request CreateRequest
		if err := decodeJSONInput(param.Input, &request); err != nil {
			return CreateRequest{}, err
		}
		if request.Folder == "" {
			request.Folder = "inbox"
		}
		if request.Folder != "inbox" && request.Folder != "archive" {
			return CreateRequest{}, fmt.Errorf("folder must be inbox or archive, got %q", request.Folder)
		}
		return request, nil
	}

	content, err := readTextInput(param.Message)
	if err != nil {
		return CreateRequest{}, err
	}

	request := CreateRequest{
		Content:  content,
		Tags:     param.Tag,
		Folder:   "inbox",
		Flagged:  param.Flagged,
		Action:   param.Action,
		FlagType: param.FlagType,
	}
	if param.Archive {
		request.Folder = "archive"
	}
	return request, nil
}

func resolveModifyRequest(input string, content string, tags []string, action string, uuid string) (ModifyRequest, error) {
	if input != "" {
		if globalArgs.Stdin || globalArgs.TextFile != "" {
			return ModifyRequest{}, fmt.Errorf("cannot combine --input with --stdin or --text-file")
		}
		if content != "" || len(tags) > 0 || action != "" || uuid != "" {
			return ModifyRequest{}, fmt.Errorf("cannot combine --input with positional or flag arguments")
		}
		var request ModifyRequest
		if err := decodeJSONInput(input, &request); err != nil {
			return ModifyRequest{}, err
		}
		return request, nil
	}

	text, err := readTextInput(content)
	if err != nil {
		return ModifyRequest{}, err
	}

	return ModifyRequest{
		UUID:    uuid,
		Content: text,
		Tags:    tags,
		Action:  action,
	}, nil
}

func resolveReplaceRequest(param *ReplaceCmd) (ReplaceRequest, error) {
	if param.Input != "" {
		if globalArgs.Stdin || globalArgs.TextFile != "" {
			return ReplaceRequest{}, fmt.Errorf("cannot combine --input with --stdin or --text-file")
		}
		if param.Message != "" || param.UUID != "" {
			return ReplaceRequest{}, fmt.Errorf("cannot combine --input with positional or flag arguments")
		}
		var request ReplaceRequest
		if err := decodeJSONInput(param.Input, &request); err != nil {
			return ReplaceRequest{}, err
		}
		return request, nil
	}

	content, err := readTextInput(param.Message)
	if err != nil {
		return ReplaceRequest{}, err
	}

	return ReplaceRequest{
		UUID:    param.UUID,
		Content: content,
	}, nil
}

func resolveRunRequest(param *RunCmd) (RunRequest, error) {
	if param.Input != "" {
		if globalArgs.Stdin || globalArgs.TextFile != "" {
			return RunRequest{}, fmt.Errorf("cannot combine --input with --stdin or --text-file")
		}
		if param.Action != "" || param.Text != "" || param.UUID != "" {
			return RunRequest{}, fmt.Errorf("cannot combine --input with positional or flag arguments")
		}
		var request RunRequest
		if err := decodeJSONInput(param.Input, &request); err != nil {
			return RunRequest{}, err
		}
		return request, nil
	}

	request := RunRequest{
		Action: param.Action,
		UUID:   param.UUID,
	}
	if param.UUID != "" && (param.Text != "" || globalArgs.Stdin || globalArgs.TextFile != "") {
		return RunRequest{}, fmt.Errorf("run accepts uuid or content, not both")
	}
	if param.UUID == "" {
		text, err := readTextInput(param.Text)
		if err != nil {
			return RunRequest{}, err
		}
		request.Content = text
	}
	return request, nil
}

func resolveUUIDRequest(input, uuid string) (UUIDRequest, error) {
	if input != "" {
		if uuid != "" {
			return UUIDRequest{}, fmt.Errorf("cannot combine --input with positional or flag arguments")
		}
		var request UUIDRequest
		if err := decodeJSONInput(input, &request); err != nil {
			return UUIDRequest{}, err
		}
		return request, nil
	}

	return UUIDRequest{UUID: uuid}, nil
}

func filterActions(actions []string, query string) []string {
	if query == "" {
		return actions
	}

	filtered := make([]string, 0, len(actions))
	needle := strings.ToLower(query)
	for _, action := range actions {
		if strings.Contains(strings.ToLower(action), needle) {
			filtered = append(filtered, action)
		}
	}
	return filtered
}

type TagRequest struct {
	UUID string   `json:"uuid" desc:"Draft UUID"`
	Tags []string `json:"tags" desc:"Tags to add or remove"`
}

type FlagRequest struct {
	UUID     string `json:"uuid,omitempty"`
	FlagType *int   `json:"flagType,omitempty"`
}
