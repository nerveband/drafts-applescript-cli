package drafts

import (
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func ValidateUUID(uuid string) error {
	if !uuidPattern.MatchString(uuid) {
		return &Error{Code: "INVALID_INPUT", Message: "uuid must be a canonical Drafts UUID", RetrySafe: true}
	}
	return nil
}
func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" || strings.ContainsRune(name, 0) || !utf8.ValidString(name) {
		return &Error{Code: "INVALID_INPUT", Message: "name must be nonempty and contain no NUL characters", RetrySafe: true}
	}
	return nil
}
func checkedDraft(uuid string) error {
	if err := ValidateUUID(uuid); err != nil {
		return err
	}
	exists, err := DraftExists(uuid)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: %s", ErrDraftNotFound, uuid)
	}
	return nil
}
func preflightAction(action string) error {
	if action == "" {
		return nil
	}
	if err := ValidateName(action); err != nil {
		return err
	}
	if err := requireCapability("command.perform"); err != nil {
		return err
	}
	exists, err := ActionExists(action)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: %s", ErrActionNotFound, action)
	}
	return nil
}
func partialError(err error, uuid string, steps ...string) error {
	var detail *Error
	e := &Error{Code: "PARTIAL_FAILURE", Message: "a later step failed after changing the draft", UUID: uuid, Completed: steps, Hint: "Inspect this UUID before retrying. Completed steps must not be repeated.", Cause: err}
	if errors.As(err, &detail) {
		e.AppleScriptCode = detail.AppleScriptCode
	}
	if errors.Is(err, ErrActionNotFound) {
		e.Message = "draft changed, but the requested action was not found"
	}
	return e
}

type ActionRunResult struct {
	UUID         string `json:"uuid"`
	CreatedDraft bool   `json:"createdDraft"`
	Status       string `json:"status"`
}

func Create(text string, opt CreateOptions) (string, error) {
	if strings.ContainsRune(text, 0) || !utf8.ValidString(text) {
		return "", &Error{Code: "INVALID_INPUT", Message: "content cannot contain NUL characters", RetrySafe: true}
	}
	if err := validateNames(opt.Tags); err != nil {
		return "", err
	}
	if opt.Folder != FolderInbox && opt.Folder != FolderArchive {
		return "", &Error{Code: "INVALID_INPUT", Message: "folder must be inbox or archive"}
	}
	if opt.FlagType != nil && (*opt.FlagType < 0 || *opt.FlagType > 6) {
		return "", &Error{Code: "INVALID_INPUT", Message: "flag type must be 0 through 6"}
	}
	if err := preflightAction(opt.Action); err != nil {
		return "", err
	}
	folder := "inbox"
	if opt.Folder == FolderArchive {
		folder = "archive"
	}
	flagType := ""
	if opt.FlagType != nil {
		opt.Flagged = true
		if err := requireCapability("flag_type"); err != nil {
			return "", err
		}
		if *opt.FlagType < 0 || *opt.FlagType > 6 {
			return "", &Error{Code: "INVALID_INPUT", Message: "flag type must be 0 through 6"}
		}
		flagType = fmt.Sprintf(", «property DrFt»:%d", *opt.FlagType)
	}
	script := fmt.Sprintf(`tell application "Drafts"
 set d to make new draft with properties {content:"%s", «property DrFl»:%t, tag list:%s, folder:%s%s}
 return id of d
end tell`, escapeForAppleScript(text), opt.Flagged, tagsToAppleScript(opt.Tags), folder, flagType)
	uuid, err := runAppleScript(script)
	if err != nil {
		return "", err
	}
	if err := ValidateUUID(uuid); err != nil {
		return "", partialError(&Error{Code: "INVALID_RESPONSE", Message: "invalid created UUID"}, "", "create")
	}
	if opt.Action != "" {
		if err := RunActionOnDraft(opt.Action, uuid); err != nil {
			return uuid, partialError(err, uuid, "create")
		}
	}
	return uuid, nil
}
func Prepend(uuid, text string, opt ModifyOptions) error { return modify(uuid, text, opt, true) }
func Append(uuid, text string, opt ModifyOptions) error  { return modify(uuid, text, opt, false) }
func modify(uuid, text string, opt ModifyOptions, prepend bool) error {
	if strings.ContainsRune(text, 0) || !utf8.ValidString(text) {
		return &Error{Code: "INVALID_INPUT", Message: "content cannot contain NUL characters"}
	}
	if err := validateNames(opt.Tags); err != nil {
		return err
	}
	if err := preflightAction(opt.Action); err != nil {
		return err
	}
	if err := checkedDraft(uuid); err != nil {
		return err
	}
	separator := "\n"
	if opt.Separator != nil {
		separator = *opt.Separator
	}
	expr := fmt.Sprintf(`(content of d) & "%s" & "%s"`, escapeForAppleScript(separator), escapeForAppleScript(text))
	if prepend {
		expr = fmt.Sprintf(`"%s" & "%s" & (content of d)`, escapeForAppleScript(text), escapeForAppleScript(separator))
	}
	script := fmt.Sprintf(`tell application "Drafts"
 set d to draft id "%s"
 set content of d to %s
end tell`, escapeForAppleScript(uuid), expr)
	if _, err := runAppleScript(script); err != nil {
		return err
	}
	steps := []string{"content"}
	if len(opt.Tags) > 0 {
		if err := Tag(uuid, opt.Tags...); err != nil {
			return partialError(err, uuid, steps...)
		}
		steps = append(steps, "tags")
	}
	if opt.Action != "" {
		if err := RunActionOnDraft(opt.Action, uuid); err != nil {
			return partialError(err, uuid, steps...)
		}
	}
	return nil
}
func Replace(uuid, text string) error                      { return replace(uuid, text, nil) }
func ReplaceIfUnchanged(uuid, text, original string) error { return replace(uuid, text, &original) }
func replace(uuid, text string, original *string) error {
	if strings.ContainsRune(text, 0) || !utf8.ValidString(text) {
		return &Error{Code: "INVALID_INPUT", Message: "content cannot contain NUL characters"}
	}
	if err := checkedDraft(uuid); err != nil {
		return err
	}
	guard := ""
	if original != nil {
		guard = fmt.Sprintf(`set originalBytes to current application's NSString's stringWithString:"%s"
 set currentText to current application's NSString's stringWithString:(content of d as text)
 set currentBytes to currentText's dataUsingEncoding:4
 set encodedBytes to currentBytes's base64EncodedStringWithOptions:0
 if not (originalBytes's isEqualToString:encodedBytes) then error "Draft changed while editing" number -2701`, base64.StdEncoding.EncodeToString([]byte(*original)))
	}
	_, err := runAppleScript(fmt.Sprintf(`tell application "Drafts"
 set d to draft id "%s"
 %s
 set content of d to "%s"
end tell`, escapeForAppleScript(uuid), guard, escapeForAppleScript(text)))
	return err
}
func Move(uuid, folder string) error {
	if folder != "inbox" && folder != "archive" && folder != "trash" {
		return &Error{Code: "INVALID_INPUT", Message: "folder must be inbox, archive, or trash"}
	}
	if err := checkedDraft(uuid); err != nil {
		return err
	}
	_, err := runAppleScript(fmt.Sprintf(`tell application "Drafts"
 set folder of draft id "%s" to %s
end tell`, escapeForAppleScript(uuid), folder))
	return err
}
func Trash(uuid string) error   { return Move(uuid, "trash") }
func Archive(uuid string) error { return Move(uuid, "archive") }
func Tag(uuid string, tags ...string) error {
	if err := validateNames(tags); err != nil {
		return err
	}
	if len(tags) == 0 {
		return nil
	}
	if err := checkedDraft(uuid); err != nil {
		return err
	}
	_, err := runAppleScript(fmt.Sprintf(`tell application "Drafts"
 set d to draft id "%s"
 set existingTags to tag list of d
 considering case
 repeat with t in %s
  if (contents of t) is not in existingTags then set end of existingTags to (contents of t as text)
 end repeat
 end considering
 set tag list of d to existingTags
end tell`, escapeForAppleScript(uuid), tagsToAppleScript(tags)))
	return err
}
func RemoveTags(uuid string, tags ...string) error {
	if err := validateNames(tags); err != nil {
		return err
	}
	if err := checkedDraft(uuid); err != nil {
		return err
	}
	_, err := runAppleScript(fmt.Sprintf(`tell application "Drafts"
 set d to draft id "%s"
 set removeTags to %s
 set keptTags to {}
 set existingTags to tag list of d
 considering case
 repeat with t in existingTags
  if (contents of t as text) is not in removeTags then set end of keptTags to (contents of t as text)
 end repeat
 end considering
 set tag list of d to keptTags
end tell`, escapeForAppleScript(uuid), tagsToAppleScript(tags)))
	return err
}
func SetFlagged(uuid string, flagged bool) error {
	if err := checkedDraft(uuid); err != nil {
		return err
	}
	_, err := runAppleScript(fmt.Sprintf(`tell application "Drafts"
 set «property DrFl» of draft id "%s" to %t
end tell`, escapeForAppleScript(uuid), flagged))
	return err
}
func SetFlagType(uuid string, flagType int) error {
	if flagType < 0 || flagType > 6 {
		return &Error{Code: "INVALID_INPUT", Message: "flag type must be 0 through 6"}
	}
	if err := requireCapability("flag_type"); err != nil {
		return err
	}
	if err := checkedDraft(uuid); err != nil {
		return err
	}
	_, err := runAppleScript(fmt.Sprintf(`tell application "Drafts"
 set d to draft id "%s"
 set «property DrFl» of d to true
 set «property DrFt» of d to %d
end tell`, escapeForAppleScript(uuid), flagType))
	return err
}
func Get(uuid string) (Draft, error) {
	if err := checkedDraft(uuid); err != nil {
		return Draft{}, err
	}
	out, err := runAppleScript(fmt.Sprintf(`tell application "Drafts"
 set d to draft id "%s"
 set row to my jsonRow(d, true, %t, %t)
 return my jsonText(row)
end tell`, escapeForAppleScript(uuid), selectedApp.Capabilities["draft.access date"], selectedApp.Capabilities["flag_type"]))
	if err != nil {
		return Draft{}, err
	}
	d, err := parseDraftFromAppleScript(out)
	if err != nil {
		return Draft{}, err
	}
	if !strings.EqualFold(d.UUID, uuid) {
		return Draft{}, &Error{Code: "INVALID_RESPONSE", Message: "Drafts returned a different UUID than requested"}
	}
	return d, nil
}
func parseDraftFromAppleScript(output string) (Draft, error) {
	var d Draft
	if err := decodeResult(output, &d); err != nil {
		return d, err
	}
	if err := validateDraft(d); err != nil {
		return Draft{}, err
	}
	return normalizeDraft(d), nil
}
func validateDraft(d Draft) error {
	invalid := func() error {
		return &Error{Code: "INVALID_RESPONSE", Message: "Drafts response has invalid identifiers, dates, or folder", Hint: "Do not use this response to target a mutation"}
	}
	if ValidateUUID(d.UUID) != nil {
		return invalid()
	}
	if d.Folder != "inbox" && d.Folder != "archive" && d.Folder != "trash" {
		return invalid()
	}
	for _, v := range []string{d.CreatedAt, d.ModifiedAt} {
		if _, err := time.Parse(time.RFC3339Nano, v); err != nil {
			return invalid()
		}
	}
	if d.AccessedAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, d.AccessedAt); err != nil {
			return invalid()
		}
	}
	if d.FlagType != nil && (*d.FlagType < 0 || *d.FlagType > 6) {
		return invalid()
	}
	if d.CreatedLatitude < -90 || d.CreatedLatitude > 90 || d.ModifiedLatitude < -90 || d.ModifiedLatitude > 90 || d.CreatedLongitude < -180 || d.CreatedLongitude > 180 || d.ModifiedLongitude < -180 || d.ModifiedLongitude > 180 {
		return invalid()
	}
	return nil
}
func normalizeDraft(d Draft) Draft {
	d.IsArchived = d.Folder == "archive"
	d.IsTrashed = d.Folder == "trash"
	if d.Tags == nil {
		d.Tags = []string{}
	}
	return d
}
func Query(query string, filter Filter, opt QueryOptions) ([]Draft, error) {
	return queryDrafts("every draft", query, filter, opt)
}
func QueryWorkspace(workspace, query string, filter Filter, opt QueryOptions) ([]Draft, error) {
	if workspace == "" {
		return []Draft{}, nil
	}
	if err := ValidateName(workspace); err != nil {
		return nil, err
	}
	exists, err := WorkspaceExists(workspace)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrWorkspaceNotFound, workspace)
	}
	return queryDrafts(fmt.Sprintf(`every draft of workspace "%s"`, escapeForAppleScript(workspace)), query, filter, opt)
}
func queryDrafts(scope, query string, filter Filter, opt QueryOptions) ([]Draft, error) {
	if err := validateQuery(filter, query, opt); err != nil {
		return nil, err
	}
	if err := requireCapability("core"); err != nil {
		return nil, err
	}
	if opt.Sort == SortAccessed && !selectedApp.Capabilities["draft.access date"] {
		return nil, &Error{Code: "UNSUPPORTED_CAPABILITY", Message: "selected app has no access date property"}
	}
	if opt.FlagType != nil {
		if err := requireCapability("flag_type"); err != nil {
			return nil, err
		}
	}
	dateProperty := "creation date"
	if opt.Sort == SortModified {
		dateProperty = "modification date"
	}
	if opt.Sort == SortAccessed {
		dateProperty = "access date"
	}
	flagScript := "set draftFlags to {}"
	selection := scope + buildWhereClause(filter, query, opt)
	if opt.SortFlaggedToTop {
		flagScript = "set draftFlags to get «property DrFl» of (" + selection + ")"
	}
	out, err := runAppleScript(fmt.Sprintf(`tell application "Drafts"
 set draftIDs to get id of (%s)
 set draftDates to get %s of (%s)
 %s
 set indexRows to current application's NSMutableArray's array()
 repeat with i from 1 to count of draftIDs
  set row to current application's NSMutableDictionary's dictionary()
  row's setObject:(item i of draftIDs as text) forKey:"uuid"
  row's setObject:(my isoDate(item i of draftDates)) forKey:"sortDate"
  set flaggedValue to false
  if %t then set flaggedValue to item i of draftFlags
  row's setObject:(current application's NSNumber's numberWithBool:flaggedValue) forKey:"isFlagged"
  indexRows's addObject:row
 end repeat
 return my jsonText(indexRows)
end tell`, selection, dateProperty, selection, flagScript, opt.SortFlaggedToTop))
	if err != nil {
		return nil, err
	}
	var index []struct {
		UUID      string `json:"uuid"`
		SortDate  string `json:"sortDate"`
		IsFlagged bool   `json:"isFlagged"`
	}
	if err := decodeResult(out, &index); err != nil {
		return nil, err
	}
	items := make([]Draft, len(index))
	for i, row := range index {
		if ValidateUUID(row.UUID) != nil {
			return nil, &Error{Code: "INVALID_RESPONSE", Message: "invalid query index UUID"}
		}
		if _, err := time.Parse(time.RFC3339Nano, row.SortDate); err != nil {
			return nil, &Error{Code: "INVALID_RESPONSE", Message: "invalid query index date"}
		}
		items[i] = Draft{UUID: row.UUID, CreatedAt: row.SortDate, ModifiedAt: row.SortDate, AccessedAt: row.SortDate, IsFlagged: row.IsFlagged}
	}
	items = applyQuerySorting(items, opt)
	if opt.Limit > 0 && len(items) > opt.Limit {
		items = items[:opt.Limit]
	}
	if len(items) == 0 {
		return []Draft{}, nil
	}
	ids := make([]string, len(items))
	for i := range items {
		ids[i] = items[i].UUID
	}
	out, err = runAppleScript(fmt.Sprintf(`tell application "Drafts"
 set selectedIDs to %s
 set rows to current application's NSMutableArray's array()
 repeat with selectedID in selectedIDs
  set d to draft id (contents of selectedID as text)
  rows's addObject:(my jsonRow(d, %t, %t, %t))
 end repeat
 return my jsonText(rows)
end tell`, tagsToAppleScript(ids), opt.Full, selectedApp.Capabilities["draft.access date"], selectedApp.Capabilities["flag_type"]))
	if err != nil {
		return nil, err
	}
	var result []Draft
	if err := decodeResult(out, &result); err != nil {
		return nil, err
	}
	if len(result) != len(ids) {
		return nil, &Error{Code: "INVALID_RESPONSE", Message: "query returned an incomplete selection"}
	}
	for i := range result {
		if err := validateDraft(result[i]); err != nil {
			return nil, err
		}
		if !strings.EqualFold(result[i].UUID, ids[i]) {
			return nil, &Error{Code: "INVALID_RESPONSE", Message: "query returned an unexpected UUID"}
		}
		result[i] = normalizeDraft(result[i])
	}
	return result, nil
}
func Count(query string, filter Filter, opt QueryOptions, workspace string) (int, error) {
	if err := validateQuery(filter, query, opt); err != nil {
		return 0, err
	}
	scope := "every draft"
	if workspace != "" {
		exists, err := WorkspaceExists(workspace)
		if err != nil {
			return 0, err
		}
		if !exists {
			return 0, fmt.Errorf("%w: %s", ErrWorkspaceNotFound, workspace)
		}
		scope = fmt.Sprintf(`every draft of workspace "%s"`, escapeForAppleScript(workspace))
	}
	if opt.FlagType != nil {
		if err := requireCapability("flag_type"); err != nil {
			return 0, err
		}
	}
	out, err := runAppleScript(fmt.Sprintf(`tell application "Drafts"
 return my jsonText({count of (%s%s)})
end tell`, scope, buildWhereClause(filter, query, opt)))
	if err != nil {
		return 0, err
	}
	var counts []int
	if err := decodeResult(out, &counts); err != nil {
		return 0, err
	}
	if len(counts) != 1 || counts[0] < 0 {
		return 0, &Error{Code: "INVALID_RESPONSE", Message: "invalid count response"}
	}
	return counts[0], nil
}
func buildWhereClause(filter Filter, query string, opt QueryOptions) string {
	var clauses []string
	switch filter {
	case FilterArchive:
		clauses = append(clauses, "folder is archive")
	case FilterTrash:
		clauses = append(clauses, "folder is trash")
	case FilterFlagged:
		clauses = append(clauses, "folder is inbox", "«property DrFl» is true")
	case FilterAll:
	default:
		clauses = append(clauses, "folder is inbox")
	}
	if query != "" {
		clauses = append(clauses, fmt.Sprintf(`content contains "%s"`, escapeForAppleScript(query)))
	}
	for _, tag := range opt.Tags {
		clauses = append(clauses, fmt.Sprintf(`query tag names contains "#%s#"`, escapeForAppleScript(tag)))
	}
	for _, tag := range opt.OmitTags {
		clauses = append(clauses, fmt.Sprintf(`query tag names does not contain "#%s#"`, escapeForAppleScript(tag)))
	}
	if opt.FlagType != nil {
		clauses = append(clauses, fmt.Sprintf("«property DrFl» is true and «property DrFt» is %d", *opt.FlagType))
	}
	// Dates are parsed in Go, emitted in UTC, and converted by Foundation.
	for _, bound := range []struct{ field, op, value string }{{"creation date", ">", opt.CreatedAfter}, {"creation date", "<", opt.CreatedBefore}, {"modification date", ">", opt.ModifiedAfter}, {"modification date", "<", opt.ModifiedBefore}} {
		if bound.value != "" {
			clauses = append(clauses, fmt.Sprintf(`%s %s (my dateValue("%s"))`, bound.field, bound.op, escapeForAppleScript(bound.value)))
		}
	}
	if len(clauses) == 0 {
		return ""
	}
	return " whose " + strings.Join(clauses, " and ")
}
func applyQuerySorting(items []Draft, opt QueryOptions) []Draft {
	key := func(d Draft) string {
		switch opt.Sort {
		case SortModified:
			return d.ModifiedAt
		case SortAccessed:
			return d.AccessedAt
		default:
			return d.CreatedAt
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if opt.SortFlaggedToTop && items[i].IsFlagged != items[j].IsFlagged {
			return items[i].IsFlagged
		}
		a, _ := time.Parse(time.RFC3339Nano, key(items[i]))
		b, _ := time.Parse(time.RFC3339Nano, key(items[j]))
		if a.Equal(b) {
			return items[i].UUID < items[j].UUID
		}
		if opt.SortDescending {
			return a.After(b)
		}
		return a.Before(b)
	})
	return items
}
func Select(uuid string) error {
	if err := requireCapability("open.draft"); err != nil {
		return err
	}
	if err := checkedDraft(uuid); err != nil {
		return err
	}
	_, err := runAppleScript(fmt.Sprintf(`tell application "Drafts"
 open draft id "%s"
end tell`, escapeForAppleScript(uuid)))
	return err
}
func Active() (string, error) {
	if err := requireCapability("application.current draft"); err != nil {
		return "", err
	}
	out, err := runAppleScript(`tell application "Drafts"
 return id of current draft
end tell`)
	if err != nil {
		return "", err
	}
	if ValidateUUID(out) != nil {
		return "", fmt.Errorf("%w: active draft", ErrDraftNotFound)
	}
	return out, nil
}

// RunAction creates a persistent draft. Action execution is submitted, not guaranteed completed.
func RunAction(action, text string) (ActionRunResult, error) {
	if err := ValidateName(action); err != nil {
		return ActionRunResult{}, err
	}
	uuid, err := Create(text, CreateOptions{Action: action})
	return ActionRunResult{UUID: uuid, CreatedDraft: true, Status: "submitted"}, err
}
func CurrentWorkspace() (string, error) {
	if err := requireCapability("application.current workspace"); err != nil {
		return "", err
	}
	return runAppleScript(`tell application "Drafts"
 return name of current workspace
end tell`)
}
func OpenWorkspace(name string) error {
	if err := requireCapability("open.workspace"); err != nil {
		return err
	}
	exists, err := WorkspaceExists(name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: %s", ErrWorkspaceNotFound, name)
	}
	_, err = runAppleScript(fmt.Sprintf(`tell application "Drafts"
 open workspace "%s"
end tell`, escapeForAppleScript(name)))
	return err
}

type NamedResource struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Permalink string `json:"permalink,omitempty"`
}

func Resources(kind string) ([]NamedResource, error) {
	if kind != "workspace" && kind != "action" && kind != "tag" {
		return nil, &Error{Code: "INVALID_INPUT", Message: "resource must be workspace, action, or tag"}
	}
	if err := requireCapability(kind); err != nil {
		return nil, err
	}
	extra := ""
	if kind != "tag" {
		extra = `row's setObject:(id of d as text) forKey:"id"`
	}
	if kind == "workspace" && selectedApp.Capabilities["workspace.permalink"] {
		extra += `\nrow's setObject:(permalink of d as text) forKey:"permalink"`
		extra = strings.ReplaceAll(extra, `\n`, "\n")
	}
	out, err := runAppleScript(fmt.Sprintf(`tell application "Drafts"
 set rows to current application's NSMutableArray's array()
 repeat with d in (every %s)
  set row to current application's NSMutableDictionary's dictionary()
  row's setObject:(name of d as text) forKey:"name"
  %s
  rows's addObject:row
 end repeat
 return my jsonText(rows)
end tell`, kind, extra))
	if err != nil {
		return nil, err
	}
	items := []NamedResource{}
	if err := decodeResult(out, &items); err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}
func Workspaces() ([]string, error) {
	items, err := Resources("workspace")
	names := []string{}
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names, err
}
func Tags() ([]NamedResource, error)    { return Resources("tag") }
func Actions() ([]NamedResource, error) { return Resources("action") }

func validateNames(names []string) error {
	for _, name := range names {
		if err := ValidateName(name); err != nil {
			return err
		}
	}
	return nil
}
func validateQuery(filter Filter, query string, opt QueryOptions) error {
	if len(opt.Tags) != 0 || len(opt.OmitTags) != 0 {
		if err := requireCapability("draft.query tag names"); err != nil {
			return err
		}
	}
	invalid := func(message string) error { return &Error{Code: "INVALID_INPUT", Message: message, RetrySafe: true} }
	if filter < FilterInbox || filter > FilterAll {
		return invalid("filter must be inbox, flagged, archive, trash, or all")
	}
	if opt.Sort < SortCreated || opt.Sort > SortAccessed {
		return invalid("sort must be created, modified, or accessed")
	}
	if opt.Limit < 0 {
		return invalid("limit must be at least 0")
	}
	if opt.FlagType != nil && (*opt.FlagType < 0 || *opt.FlagType > 6) {
		return invalid("flag type must be 0 through 6")
	}
	if strings.ContainsRune(query, 0) {
		return invalid("search must not contain NUL")
	}
	if err := validateNames(append(append([]string{}, opt.Tags...), opt.OmitTags...)); err != nil {
		return err
	}
	for _, pair := range [][2]string{{opt.CreatedAfter, opt.CreatedBefore}, {opt.ModifiedAfter, opt.ModifiedBefore}} {
		for _, v := range pair {
			if v != "" {
				if _, err := time.Parse(time.RFC3339Nano, v); err != nil {
					return invalid("date bounds must be RFC3339")
				}
			}
		}
		if pair[0] != "" && pair[1] != "" {
			a, _ := time.Parse(time.RFC3339Nano, pair[0])
			b, _ := time.Parse(time.RFC3339Nano, pair[1])
			if !a.Before(b) {
				return invalid("after bound must precede before bound")
			}
		}
	}
	return nil
}

// ValidateDraft verifies untrusted structured data without accessing an application.
func ValidateDraft(d Draft) error { return validateDraft(d) }

// RenameWorkspace changes a dictionary-supported workspace name without inventing
// workspace creation or configuration APIs that the dictionary does not expose.
func RenameWorkspace(name, newName string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if err := ValidateName(newName); err != nil {
		return err
	}
	if err := requireCapability("workspace.rename"); err != nil {
		return err
	}
	exists, err := WorkspaceExists(name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: %s", ErrWorkspaceNotFound, name)
	}
	if name == newName {
		return nil
	}
	exists, err = WorkspaceExists(newName)
	if err != nil {
		return err
	}
	if exists {
		return &Error{Code: "CONFLICT", Message: "destination workspace name already exists", Hint: "Choose a distinct workspace name", RetrySafe: true}
	}
	_, err = runAppleScript(fmt.Sprintf(`tell application "Drafts"
 set name of workspace "%s" to "%s"
end tell`, escapeForAppleScript(name), escapeForAppleScript(newName)))
	return err
}
