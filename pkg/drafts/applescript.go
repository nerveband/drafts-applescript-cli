package drafts

import (
	"fmt"
	"os/exec"
	"strings"
)

// runAppleScript executes an AppleScript and returns the output
func runAppleScript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("applescript error: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// escapeForAppleScript escapes a string for use in AppleScript
func escapeForAppleScript(s string) string {
	// Escape backslashes first, then quotes, then newlines
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

// tagsToAppleScript converts a slice of tags to AppleScript list format
func tagsToAppleScript(tags []string) string {
	if len(tags) == 0 {
		return "{}"
	}
	escaped := make([]string, len(tags))
	for i, t := range tags {
		escaped[i] = fmt.Sprintf("\"%s\"", escapeForAppleScript(t))
	}
	return "{" + strings.Join(escaped, ", ") + "}"
}

// RunActionOnDraft runs an action on an existing draft.
func RunActionOnDraft(action, uuid string) error {
	script := fmt.Sprintf(`tell application "Drafts"
	set d to draft id "%s"
	set actionToRun to missing value
	repeat with a in (every action)
		if name of a is "%s" then
			set actionToRun to a
			exit repeat
		end if
	end repeat
	if actionToRun is not missing value then
		perform action actionToRun on draft d
		return "success"
	else
		return "action not found"
	end if
end tell`, escapeForAppleScript(uuid), escapeForAppleScript(action))

	result, err := runAppleScript(script)
	if err != nil {
		return err
	}
	if result == "action not found" {
		return fmt.Errorf("action not found: %s", action)
	}
	return nil
}
