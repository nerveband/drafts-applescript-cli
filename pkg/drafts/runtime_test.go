package drafts

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testUUID = "12345678-1234-1234-1234-123456789abc"

func testApp() AppInfo {
	return AppInfo{Path: "/fixture/Drafts.app", Capabilities: map[string]bool{"core": true, "draft.query tag names": true, "application.current draft": true, "application.current workspace": true, "draft.access date": true, "flag_type": true, "action": true, "workspace": true, "tag": true, "command.perform": true, "open.draft": true, "open.workspace": true, "workspace.rename": true}}
}
func mockScripts(t *testing.T, runner func(string) (string, error)) {
	t.Helper()
	oldRunner, oldApp, oldTimeout, oldChecked := scriptRunner, selectedApp, operationTimeout, runtimeChecked
	Configure(testApp(), time.Second)
	runtimeChecked = true
	scriptRunner = runner
	t.Cleanup(func() {
		scriptRunner = oldRunner
		selectedApp = oldApp
		operationTimeout = oldTimeout
		runtimeChecked = oldChecked
	})
}
func syntheticDraft(id, content string) Draft {
	return Draft{UUID: id, Content: content, Title: "First\tline ☃", Tags: []string{"x|||y", "tab\ttag"}, Folder: "inbox", CreatedAt: "2026-01-01T00:00:00Z", ModifiedAt: "2026-01-02T00:00:00Z", AccessedAt: "2026-01-03T00:00:00Z"}
}
func TestStructuredResponseRoundTrip(t *testing.T) {
	d := syntheticDraft(testUUID, "First\tline\nSecond\r\n☃ |||\n")
	raw, _ := json.Marshal(d)
	got, err := parseDraftFromAppleScript(string(raw))
	if err != nil || got.Content != d.Content || strings.Join(got.Tags, "|") != strings.Join(d.Tags, "|") {
		t.Fatalf("round trip: %#v %v", got, err)
	}
	for _, bad := range []string{"not json", `{"uuid":"line two"}`, `null`} {
		if _, err := parseDraftFromAppleScript(bad); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	d.CreatedAt = "locale date"
	raw, _ = json.Marshal(d)
	if _, err := parseDraftFromAppleScript(string(raw)); err == nil {
		t.Fatal("accepted invalid date")
	}
}
func TestPreflightStopsCreateAndAppend(t *testing.T) {
	scripts := []string{}
	mockScripts(t, func(script string) (string, error) { scripts = append(scripts, script); return "false", nil })
	if _, err := Create("body", CreateOptions{Action: "Missing"}); !errors.Is(err, ErrActionNotFound) {
		t.Fatal(err)
	}
	if err := Append(testUUID, "body", ModifyOptions{Action: "Missing"}); !errors.Is(err, ErrActionNotFound) {
		t.Fatal(err)
	}
	for _, s := range scripts {
		if strings.Contains(s, "make new draft") || strings.Contains(s, "set content of d") {
			t.Fatal("mutation before preflight")
		}
	}
}
func TestPartialFailureReceipt(t *testing.T) {
	mockScripts(t, func(script string) (string, error) {
		if strings.Contains(script, "make new draft") {
			return testUUID, nil
		}
		if strings.Contains(script, "perform action") {
			return "", &Error{Code: "PERMISSION_DENIED", AppleScriptCode: -1743}
		}
		return "true", nil
	})
	uuid, err := Create("body", CreateOptions{Action: "Copy"})
	var detail *Error
	if uuid != testUUID || !errors.As(err, &detail) || detail.Code != "PARTIAL_FAILURE" || detail.UUID != testUUID || len(detail.Completed) != 1 || detail.Completed[0] != "create" || detail.RetrySafe {
		t.Fatalf("%s %#v", uuid, err)
	}
}
func TestListLimitsBeforeBodyFetch(t *testing.T) {
	a := syntheticDraft(testUUID, "hidden")
	b := syntheticDraft("abcdefab-1234-1234-1234-123456789abc", "other")
	b.CreatedAt = "2026-01-02T00:00:00Z"
	b.ModifiedAt = "2026-01-01T00:00:00Z"
	reads := 0
	mockScripts(t, func(script string) (string, error) {
		if strings.Contains(script, "set indexRows") {
			if strings.Contains(script, "rows's addObject:(my jsonRow(d,") {
				t.Fatal("index fetches full rows before limiting")
			}
			out, _ := json.Marshal([]map[string]interface{}{{"uuid": a.UUID, "sortDate": a.ModifiedAt}, {"uuid": b.UUID, "sortDate": b.ModifiedAt}})
			return string(out), nil
		}
		if strings.Contains(script, "jsonRow(d, true,") {
			if strings.Contains(script, b.UUID) {
				t.Fatal("fetched unselected row")
			}
			reads++
			out, _ := json.Marshal([]Draft{a})
			return string(out), nil
		}
		return "true", nil
	})
	result, err := Query("", FilterAll, QueryOptions{Full: true, Limit: 1, Sort: SortModified, SortDescending: true})
	if err != nil || len(result) != 1 || result[0].UUID != a.UUID || reads != 1 {
		t.Fatalf("%#v %v reads=%d", result, err, reads)
	}
}
func TestSortingAbsoluteDatesAndTies(t *testing.T) {
	a := syntheticDraft(testUUID, "")
	b := syntheticDraft("abcdefab-1234-1234-1234-123456789abc", "")
	a.CreatedAt = "2026-01-01T03:00:00+04:00"
	b.CreatedAt = "2026-01-01T00:00:00Z"
	got := applyQuerySorting([]Draft{b, a}, QueryOptions{})
	if got[0].UUID != a.UUID {
		t.Fatal("not chronological")
	}
	for i := range got {
		got[i].IsFlagged = got[i].UUID == b.UUID
	}
	got = applyQuerySorting(got, QueryOptions{SortFlaggedToTop: true})
	if got[0].UUID != b.UUID {
		t.Fatal("flagged first ignored")
	}
}
func TestCapabilitiesGateBeforeEvents(t *testing.T) {
	calls := 0
	mockScripts(t, func(string) (string, error) { calls++; return "true", nil })
	selectedApp.Capabilities["flag_type"] = false
	value := 2
	if _, err := Create("text", CreateOptions{FlagType: &value}); err == nil {
		t.Fatal("unsupported create succeeded")
	}
	if err := SetFlagType(testUUID, 2); err == nil {
		t.Fatal("unsupported flag succeeded")
	}
	if calls != 0 {
		t.Fatal("unsupported option sent events")
	}
}
func TestUUIDAndQueryValidationBeforeEvents(t *testing.T) {
	mockScripts(t, func(string) (string, error) { t.Fatal("invalid input sent event"); return "", nil })
	if _, err := Get("line two"); err == nil {
		t.Fatal("invalid UUID")
	}
	if _, err := Query("", Filter(99), QueryOptions{}); err == nil {
		t.Fatal("invalid filter")
	}
	if _, err := Count("", FilterAll, QueryOptions{CreatedAfter: "bad"}, ""); err == nil {
		t.Fatal("invalid date")
	}
}
func TestEmptyWorkspaceSafe(t *testing.T) {
	mockScripts(t, func(string) (string, error) { t.Fatal("unexpected event"); return "", nil })
	ds, err := QueryWorkspace("", "", FilterAll, QueryOptions{})
	if err != nil || len(ds) != 0 {
		t.Fatal(err)
	}
}
func TestConflictGuardAndEscaping(t *testing.T) {
	scriptText := ""
	mockScripts(t, func(script string) (string, error) {
		if strings.Contains(script, "set content of d") {
			scriptText = script
		}
		return "true", nil
	})
	if err := ReplaceIfUnchanged(testUUID, "new\n", "old\"\\\n"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(scriptText, base64.StdEncoding.EncodeToString([]byte("old\"\\\n"))) || !strings.Contains(scriptText, "dataUsingEncoding:4") || !strings.Contains(scriptText, "number -2701") {
		t.Fatal(scriptText)
	}
	if strings.Contains(scriptText, `tell application "Drafts"`) {
		t.Fatal("did not select exact app")
	}
}
func TestRuntimeDoesNotLaunchStoppedApp(t *testing.T) {
	mockScripts(t, func(script string) (string, error) {
		if strings.Contains(script, "AppKit") {
			return "false", nil
		}
		t.Fatal("sent Drafts event")
		return "", nil
	})
	runtimeChecked = false
	_, err := Get(testUUID)
	var detail *Error
	if !errors.As(err, &detail) || detail.Code != "DRAFTS_NOT_RUNNING" {
		t.Fatal(err)
	}
}
func TestExecutorStdinTimeoutAndRedaction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "osascript")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SCRIPT_CAPTURE", filepath.Join(dir, "captured"))
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0700); err != nil {
			t.Fatal(err)
		}
	}
	old := operationTimeout
	operationTimeout = time.Second
	t.Cleanup(func() { operationTimeout = old })
	write("test \"$#\" = 1 && test \"$1\" = - || exit 10\ncat > \"$SCRIPT_CAPTURE\"\nprintf 'true\\n'\n")
	if _, err := executeScript("synthetic secret body"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(os.Getenv("SCRIPT_CAPTURE"))
	if string(data) != "synthetic secret body" {
		t.Fatal("body not on stdin")
	}
	write("printf 'synthetic secret body failed (-1743)\\n' >&2\nexit 1\n")
	_, err := executeScript("synthetic secret body")
	var detail *Error
	if !errors.As(err, &detail) || detail.Code != "PERMISSION_DENIED" || strings.Contains(detail.Message, "secret") {
		t.Fatal(err)
	}
	operationTimeout = 20 * time.Millisecond
	write("exec sleep 2\n")
	_, err = executeScript("safe")
	if !errors.As(err, &detail) || detail.Code != "TIMEOUT" || detail.RetrySafe {
		t.Fatal(err)
	}
}
func TestDictionaryAndAppSelection(t *testing.T) {
	data := []byte(`<dictionary><suite><class name="draft"><property name="content"/><property name="tag list"/><property name="creation date"/><property name="modification date"/><property name="flagged" code="DrFl" type="boolean" access="rw"/><property name="flagged" code="DrFt" type="integer" access="rw"/></class><command name="open"><direct-parameter><type type="draft"/><type type="workspace"/></direct-parameter></command></suite></dictionary>`)
	cap, warnings, err := InspectDictionary(data)
	if err != nil || !cap["core"] || !cap["flag_type"] || !cap["open.workspace"] || len(warnings) != 1 {
		t.Fatalf("%v %v %v", cap, warnings, err)
	}
	beta := AppInfo{Path: "/Beta.app", Channel: "beta"}
	stable := AppInfo{Path: "/Drafts.app", Channel: "unknown"}
	app, err := SelectApp([]AppInfo{beta, stable}, "", "auto")
	if err != nil || app.Path != stable.Path {
		t.Fatal(err)
	}
	app, err = SelectApp([]AppInfo{beta, stable}, "", "beta")
	if err != nil || app.Path != beta.Path {
		t.Fatal(err)
	}
	if _, err := SelectApp([]AppInfo{stable, {Path: "/Other.app", Channel: "unknown"}}, "", "auto"); err == nil {
		t.Fatal("ambiguous default")
	}
	if _, err := SelectApp([]AppInfo{stable}, "", "beta"); err == nil {
		t.Fatal("guessed beta")
	}
}

// Optional native QA compiles scripts without executing Drafts events. Generic Foundation
// date/JSON helpers execute against synthetic values only.
func TestNativeAppleScript(t *testing.T) {
	path := os.Getenv("DRAFTS_DICTIONARY_APP")
	if path == "" {
		t.Skip("set DRAFTS_DICTIONARY_APP for dictionary-only native QA")
	}
	app, err := InspectApp(path)
	if err != nil {
		t.Fatal(err)
	}
	mockScripts(t, func(script string) (string, error) {
		file := filepath.Join(t.TempDir(), "script.applescript")
		if err := os.WriteFile(file, []byte(script), 0600); err != nil {
			t.Fatal(err)
		}
		out, err := exec.Command("osacompile", "-o", filepath.Join(t.TempDir(), "script.scpt"), file).CombinedOutput()
		if err != nil {
			t.Fatalf("compile: %s\n%s", out, script)
		}
		if strings.Contains(script, `return exists workspace "New Work"`) {
			return "false", nil
		}
		if strings.Contains(script, "perform action") {
			return "submitted", nil
		}
		if strings.Contains(script, "set indexRows") {
			return `[{"uuid":"` + testUUID + `","sortDate":"2026-01-01T00:00:00Z"}]`, nil
		}
		if strings.Contains(script, "set selectedIDs") {
			data, _ := json.Marshal([]Draft{syntheticDraft(testUUID, "synthetic")})
			return string(data), nil
		}
		if strings.Contains(script, "jsonText(rows)") {
			return "[]", nil
		}
		if strings.Contains(script, "jsonText({count") {
			return "[0]", nil
		}
		if strings.Contains(script, "make new draft") || strings.Contains(script, "id of current draft") {
			return testUUID, nil
		}
		if strings.Contains(script, "jsonText(row)") {
			data, _ := json.Marshal(syntheticDraft(testUUID, "synthetic"))
			return string(data), nil
		}
		return "true", nil
	})
	selectedApp = app
	cases := []func() error{
		func() error {
			_, e := Create("tab\tline\n☃", CreateOptions{Folder: FolderArchive, Tags: []string{"x"}})
			return e
		},
		func() error {
			return Append(testUUID, "test", ModifyOptions{Tags: []string{"x"}, Separator: stringPointer("\n"), Action: "Copy"})
		},
		func() error { return ReplaceIfUnchanged(testUUID, "new", "old") },
		func() error { return Move(testUUID, "trash") }, func() error { return RemoveTags(testUUID, "x") }, func() error { return SetFlagged(testUUID, true) },
		func() error { _, e := Get(testUUID); return e }, func() error {
			_, e := Query("Hello", FilterFlagged, QueryOptions{Tags: []string{"x"}, OmitTags: []string{"y"}, CreatedAfter: "2026-01-01T00:00:00Z", ModifiedBefore: "2027-01-01T00:00:00Z"})
			return e
		},
		func() error { _, e := QueryWorkspace("Work", "", FilterInbox, QueryOptions{}); return e }, func() error { _, e := Count("", FilterAll, QueryOptions{}, ""); return e }, func() error { return Select(testUUID) }, func() error { return OpenWorkspace("Work") }, func() error { return RenameWorkspace("Work", "New Work") }, func() error { _, e := Resources("workspace"); return e }, func() error { _, e := Resources("action"); return e }, func() error { _, e := Resources("tag"); return e },
	}
	for i, f := range cases {
		if err := f(); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
	}
	if app.Capabilities["flag_type"] {
		value := 3
		if _, e := Create("text", CreateOptions{FlagType: &value}); e != nil {
			t.Fatal(e)
		}
		if e := SetFlagType(testUUID, 3); e != nil {
			t.Fatal(e)
		}
		if _, e := Query("", FilterAll, QueryOptions{FlagType: &value}); e != nil {
			t.Fatal(e)
		}
	}
	// A synthetic record exercises the actual JSON handler without an application tell.
	script := "use framework \"Foundation\"\nuse scripting additions\nusing terms from application \"" + escapeForAppleScript(app.Path) + "\"\n" + foundationHandlers + `
set syntheticDate to my dateValue("2026-10-05T12:00:01Z")
set syntheticDraft to {id:"12345678-1234-1234-1234-123456789abc", title:"One\tTwo ☃", content:"One\tTwo\nThree ||| ☃\n", tag list:{"x|||y", "z\ttag"}, folder:inbox, «property DrFl»:true, «property DrFt»:3, creation date:syntheticDate, modification date:syntheticDate, «property DrAC»:syntheticDate, permalink:"drafts://open?uuid=12345678-1234-1234-1234-123456789abc", creation latitude:0, creation longitude:0, modification latitude:0, modification longitude:0}
return my jsonText(my jsonRow(syntheticDraft, true, true, true))
end using terms from
`
	file := filepath.Join(t.TempDir(), "row.applescript")
	if e := os.WriteFile(file, []byte(script), 0600); e != nil {
		t.Fatal(e)
	}
	out, e := exec.Command("osascript", file).CombinedOutput()
	if e != nil {
		t.Fatalf("synthetic row %s %v", out, e)
	}
	row, e := parseDraftFromAppleScript(string(out))
	if e != nil || row.Content != "One\tTwo\nThree ||| ☃\n" || row.CreatedAt != "2026-10-05T12:00:01Z" || !row.IsFlagged || row.FlagType == nil || *row.FlagType != 3 {
		t.Fatalf("synthetic row %#v %v", row, e)
	}

	helpers := strings.Split(foundationHandlers, "on jsonRow(")[0]
	for _, value := range []string{"2026-01-05T12:00:01Z", "2026-10-05T12:00:01Z"} {
		script := "use framework \"Foundation\"\nuse scripting additions\n" + helpers + "return my isoDate(my dateValue(\"" + value + "\"))"
		file := filepath.Join(t.TempDir(), "date.applescript")
		os.WriteFile(file, []byte(script), 0600)
		out, e := exec.Command("osascript", file).CombinedOutput()
		if e != nil || strings.TrimSpace(string(out)) != value {
			t.Fatalf("date bridge %s: %s %v", value, out, e)
		}
	}
}

func stringPointer(value string) *string { return &value }

func TestBetaAndStableBundleMetadata(t *testing.T) {
	if _, err := exec.LookPath("plutil"); err != nil {
		t.Skip("plutil unavailable")
	}
	dir := t.TempDir()
	app := filepath.Join(dir, "Drafts.app")
	resources := filepath.Join(app, "Contents", "Resources")
	os.MkdirAll(resources, 0700)
	meta := `<?xml version="1.0"?><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>com.agiletortoise.Drafts-OSX</string><key>CFBundleShortVersionString</key><string>54.1</string><key>CFBundleVersion</key><string>924</string></dict></plist>`
	os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(meta), 0600)
	dictionary := `<dictionary><suite><class name="draft"><property name="content"/><property name="tag list"/><property name="creation date"/><property name="modification date"/><property name="flagged" code="DrFl" type="boolean" access="rw"/></class></suite></dictionary>`
	dictPath := filepath.Join(resources, "Drafts.sdef")
	os.WriteFile(dictPath, []byte(dictionary), 0600)
	info, e := InspectApp(app)
	if e != nil || info.Channel != "unknown" || info.Capabilities["flag_type"] {
		t.Fatalf("unknown mislabeled %#v %v", info, e)
	}
	receipts := filepath.Join(app, "Contents", "_MASReceipt")
	os.MkdirAll(receipts, 0700)
	os.WriteFile(filepath.Join(receipts, "receipt"), []byte("synthetic"), 0600)
	info, e = InspectApp(app)
	if e != nil || info.Channel != "stable" {
		t.Fatalf("stable %#v %v", info, e)
	}
	os.WriteFile(filepath.Join(receipts, "sandboxReceipt"), []byte("synthetic"), 0600)
	dictionary = strings.Replace(dictionary, "</class>", `<property name="flagged" code="DrFt" type="integer" access="rw"/></class>`, 1)
	os.WriteFile(dictPath, []byte(dictionary), 0600)
	info, e = InspectApp(app)
	if e != nil || info.Channel != "beta" || !info.Capabilities["flag_type"] {
		t.Fatalf("beta %#v %v", info, e)
	}
	sameID := []AppInfo{info, {Path: "/other/Drafts.app", BundleID: info.BundleID, Channel: "stable"}}
	if _, e := SelectApp(sameID, info.BundleID, "auto"); e == nil {
		t.Fatal("ambiguous bundle accepted")
	}
}
