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
// TODO: Full implementation in Task 7
func RunActionOnDraft(action, uuid string) error {
	script := fmt.Sprintf(`tell application "Drafts"
	set d to draft id "%s"
	execute (action named "%s") with draft d
end tell`, escapeForAppleScript(uuid), escapeForAppleScript(action))

	_, err := runAppleScript(script)
	return err
}
