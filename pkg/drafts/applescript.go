package drafts

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrDraftNotFound     = errors.New("draft not found")
	ErrActionNotFound    = errors.New("action not found")
	ErrWorkspaceNotFound = errors.New("workspace not found")
)

// runAppleScript executes an AppleScript and returns the output
func runAppleScript(script string) (string, error) {
	if err := requireCapability("core"); err != nil {
		return "", err
	}
	if !runtimeChecked {
		running, err := IsRunning()
		if err != nil {
			return "", err
		}
		if !running {
			return "", &Error{Code: "DRAFTS_NOT_RUNNING", Message: "selected Drafts app must already be running", Hint: "Open the selected app and retry", RetrySafe: true}
		}
		runtimeChecked = true
	}
	// Source goes through stdin, keeping note bodies out of child process arguments.
	script = strings.ReplaceAll(script, `tell application "Drafts"`, applicationTell())
	wrapped := "use framework \"Foundation\"\nuse scripting additions\nif not (application \"" + escapeForAppleScript(selectedApp.Path) + "\" is running) then error number -600\nusing terms from application \"" + escapeForAppleScript(selectedApp.Path) + "\"\n" + foundationHandlers + script + "\nend using terms from"
	return scriptRunner(wrapped)
}

// PreflightMutation performs only read operations before reserving an idempotency key.
func PreflightMutation(action, uuid string, flagType *int) error {
	if err := requireCapability("core"); err != nil {
		return err
	}
	if flagType != nil {
		if err := requireCapability("flag_type"); err != nil {
			return err
		}
	}
	running, err := IsRunning()
	if err != nil {
		return err
	}
	if !running {
		return &Error{Code: "DRAFTS_NOT_RUNNING", Message: "selected Drafts app must already be running", RetrySafe: true}
	}
	// Probe Automation permission using public app metadata, never library content.
	if _, err := runAppleScript(`tell application "Drafts"
 return version
end tell`); err != nil {
		return err
	}
	if err := preflightAction(action); err != nil {
		return err
	}
	if uuid != "" {
		return checkedDraft(uuid)
	}
	return nil
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

func objectExists(objectSpecifier string) (bool, error) {
	script := fmt.Sprintf(`tell application "Drafts"
	return exists %s
end tell`, objectSpecifier)

	output, err := runAppleScript(script)
	if err != nil {
		return false, err
	}
	if output != "true" && output != "false" {
		return false, &Error{Code: "INVALID_RESPONSE", Message: "invalid existence response"}
	}
	return output == "true", nil
}

// DraftExists reports whether a Drafts draft exists for the given UUID.
func DraftExists(uuid string) (bool, error) {
	if err := ValidateUUID(uuid); err != nil {
		return false, err
	}
	return objectExists(fmt.Sprintf(`draft id "%s"`, escapeForAppleScript(uuid)))
}

// ActionExists reports whether a Drafts action exists by name.
func ActionExists(name string) (bool, error) {
	if err := ValidateName(name); err != nil {
		return false, err
	}
	if err := requireCapability("action"); err != nil {
		return false, err
	}
	return objectExists(fmt.Sprintf(`action "%s"`, escapeForAppleScript(name)))
}

// WorkspaceExists reports whether a Drafts workspace exists by name.
func WorkspaceExists(name string) (bool, error) {
	if err := ValidateName(name); err != nil {
		return false, err
	}
	if err := requireCapability("workspace"); err != nil {
		return false, err
	}
	return objectExists(fmt.Sprintf(`workspace "%s"`, escapeForAppleScript(name)))
}

// RunActionOnDraft runs an action on an existing draft.
func RunActionOnDraft(action, uuid string) error {
	if err := ValidateName(action); err != nil {
		return err
	}
	if err := requireCapability("command.perform"); err != nil {
		return err
	}
	if err := preflightAction(action); err != nil {
		return err
	}
	exists, err := DraftExists(uuid)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: %s", ErrDraftNotFound, uuid)
	}

	script := fmt.Sprintf(`tell application "Drafts"
	set d to draft id "%s"
	perform action (action "%s") on draft d
	return "submitted"
end tell`, escapeForAppleScript(uuid), escapeForAppleScript(action))

	result, err := runAppleScript(script)
	if err != nil {
		return err
	}
	if result != "submitted" {
		return &Error{Code: "INVALID_RESPONSE", Message: "action submission acknowledgement was invalid", UUID: uuid, Hint: "The action may have been submitted. Inspect the target before retrying."}
	}
	return nil
}
