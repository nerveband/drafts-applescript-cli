package drafts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type Error struct {
	Code            string   `json:"code"`
	Message         string   `json:"message"`
	Hint            string   `json:"hint,omitempty"`
	UUID            string   `json:"uuid,omitempty"`
	Completed       []string `json:"completed,omitempty"`
	RetrySafe       bool     `json:"retry_safe"`
	AppleScriptCode int      `json:"applescript_code,omitempty"`
	Cause           error    `json:"-"`
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Cause }

var runtimeChecked bool
var selectedApp AppInfo
var operationTimeout = 30 * time.Second

// scriptRunner is replaced only by package tests with synthetic data.
var scriptRunner = executeScript

func Configure(app AppInfo, timeout time.Duration) {
	selectedApp = app
	runtimeChecked = false
	operationTimeout = timeout
}

func requireCapability(name string) error {
	if selectedApp.Path == "" {
		apps, err := DiscoverApps()
		if err != nil {
			return err
		}
		app, err := SelectApp(apps, "", "auto")
		if err != nil {
			return err
		}
		selectedApp = app
	}
	if !selectedApp.Capabilities[name] {
		return &Error{Code: "UNSUPPORTED_CAPABILITY", Message: "selected Drafts app does not support " + name, Hint: "Use apps or schema --detected to inspect the selected installation", RetrySafe: true}
	}
	return nil
}

func executeScript(script string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "osascript", "-")
	cmd.Stdin = strings.NewReader(script)
	data, err := cmd.Output()
	if err == nil {
		return strings.TrimSuffix(string(data), "\n"), nil
	}
	if ctx.Err() != nil {
		return "", &Error{Code: "TIMEOUT", Message: "Drafts operation exceeded its timeout", Hint: "The operation may have completed. Inspect the target before retrying a mutation.", Cause: ctx.Err()}
	}
	e := &Error{Code: "APPLESCRIPT_ERROR", Message: "AppleScript execution failed", Hint: "Check info --test-permissions, Drafts Pro, and the selected application capabilities", Cause: err}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		// Preserve the numeric diagnostic without echoing source or draft content.
		matches := regexp.MustCompile(`\((-?\d+)\)\s*$`).FindStringSubmatch(strings.TrimSpace(string(exit.Stderr)))
		if len(matches) > 1 {
			fmt.Sscanf(matches[1], "%d", &e.AppleScriptCode)
		}
	}
	switch e.AppleScriptCode {
	case -1743, -10004:
		e.Code = "PERMISSION_DENIED"
		e.Message = "macOS denied automation access"
		e.Hint = "Allow your terminal to control the selected Drafts app in Privacy & Security > Automation"
		e.RetrySafe = true
	case -600:
		e.Code = "DRAFTS_NOT_RUNNING"
		e.Message = "selected Drafts app is not running"
		e.RetrySafe = true
	case -1712:
		e.Code = "TIMEOUT"
		e.Message = "Drafts timed out responding to AppleScript"
	case -2701:
		e.Code = "CONFLICT"
		e.Message = "Draft changed while editing"
		e.Hint = "Read the latest draft before applying your edits"
		e.RetrySafe = true
	case -1708:
		e.Code = "UNSUPPORTED_CAPABILITY"
		e.Message = "Drafts does not handle the requested Apple event"
		e.RetrySafe = true
	}
	return "", e
}

func applicationTell() string {
	return `tell application "` + escapeForAppleScript(selectedApp.Path) + `"`
}

func IsRunning() (bool, error) {
	if selectedApp.Path == "" {
		return false, &Error{Code: "APP_NOT_FOUND", Message: "no selected application"}
	}
	script := `use framework "AppKit"
use framework "Foundation"
set runningApps to current application's NSWorkspace's sharedWorkspace()'s runningApplications()
repeat with a in runningApps
	set appURL to a's bundleURL()
	if appURL is not missing value then
		if (appURL's |path|() as text) is "` + escapeForAppleScript(selectedApp.Path) + `" then return "true"
	end if
end repeat
return "false"`
	out, err := scriptRunner(script)
	return out == "true", err
}

func decodeResult(out string, dest interface{}) error {
	if strings.TrimSpace(out) == "null" {
		return &Error{Code: "INVALID_RESPONSE", Message: "Drafts returned null instead of structured data"}
	}
	if err := json.Unmarshal([]byte(out), dest); err != nil {
		return &Error{Code: "INVALID_RESPONSE", Message: "Drafts returned a malformed structured response", Hint: "Do not use this response to target a mutation", Cause: err}
	}
	return nil
}

// Foundation provides lossless JSON and timezone-aware, locale-independent dates.
const foundationHandlers = `property cachedDateFormatter : missing value
property cachedCalendar : missing value
on jsonText(value)
	set jsonData to current application's NSJSONSerialization's dataWithJSONObject:value options:0 |error|:(missing value)
	if jsonData is missing value then error "Cannot serialize response" number -2700
	return (current application's NSString's alloc()'s initWithData:jsonData encoding:4) as text
end jsonText
on isoDate(value)
	if my cachedDateFormatter is missing value then
	 set my cachedDateFormatter to current application's NSDateFormatter's alloc()'s init()
	 (my cachedDateFormatter)'s setLocale:(current application's NSLocale's localeWithLocaleIdentifier:"en_US_POSIX")
	 (my cachedDateFormatter)'s setTimeZone:(current application's NSTimeZone's timeZoneForSecondsFromGMT:0)
	 (my cachedDateFormatter)'s setDateFormat:"yyyy-MM-dd'T'HH:mm:ss'Z'"
	end if
	set formatter to my cachedDateFormatter
	set components to current application's NSDateComponents's alloc()'s init()
 components's setYear:(year of value as integer)
 components's setMonth:(month of value as integer)
 components's setDay:(day of value as integer)
 components's setHour:(hours of value as integer)
 components's setMinute:(minutes of value as integer)
 components's setSecond:(seconds of value as integer)
 if my cachedCalendar is missing value then
  set my cachedCalendar to current application's NSCalendar's alloc()'s initWithCalendarIdentifier:(current application's NSCalendarIdentifierGregorian)
  (my cachedCalendar)'s setTimeZone:(current application's NSTimeZone's localTimeZone())
 end if
 set calendar to my cachedCalendar
 set nativeDate to calendar's dateFromComponents:components
 return (formatter's stringFromDate:nativeDate) as text
end isoDate
on dateValue(value)
 set formatter to current application's NSISO8601DateFormatter's alloc()'s init()
 if value contains "." then formatter's setFormatOptions:3955
 set parsedDate to formatter's dateFromString:value
 if parsedDate is missing value then error "Invalid date" number -2700
 return parsedDate as date
end dateValue
on jsonRow(d, includeBody, includeAccess, includeFlagType)
	set row to current application's NSMutableDictionary's dictionary()
	tell d
		row's setObject:(id as text) forKey:"uuid"
		row's setObject:(title as text) forKey:"title"
		row's setObject:(tag list) forKey:"tags"
		row's setObject:(folder as text) forKey:"folder"
		row's setObject:(current application's NSNumber's numberWithBool:(«property DrFl»)) forKey:"isFlagged"
		row's setObject:(my isoDate(creation date)) forKey:"createdAt"
		row's setObject:(my isoDate(modification date)) forKey:"modifiedAt"
		row's setObject:(permalink as text) forKey:"permalink"
		if includeAccess then row's setObject:(my isoDate(«property DrAC»)) forKey:"accessedAt"
		if includeFlagType then row's setObject:(current application's NSNumber's numberWithInteger:(«property DrFt»)) forKey:"flagType"
		if includeBody then
			row's setObject:(content as text) forKey:"content"
			row's setObject:(current application's NSNumber's numberWithDouble:(creation latitude)) forKey:"createdLatitude"
			row's setObject:(current application's NSNumber's numberWithDouble:(creation longitude)) forKey:"createdLongitude"
			row's setObject:(current application's NSNumber's numberWithDouble:(modification latitude)) forKey:"modifiedLatitude"
			row's setObject:(current application's NSNumber's numberWithDouble:(modification longitude)) forKey:"modifiedLongitude"
		end if
	end tell
	return row
end jsonRow
`
