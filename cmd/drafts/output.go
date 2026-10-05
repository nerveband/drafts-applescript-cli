package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/nerveband/drafts-applescript-cli/v4/pkg/drafts"
	"os"
	"sort"
	"strings"
)

type Response struct {
	Success bool          `json:"success"`
	Data    interface{}   `json:"data,omitempty"`
	Error   *drafts.Error `json:"error,omitempty"`
}

var plainOutput bool

func validFormat(format string) bool {
	return format == "json" || format == "plain" || format == "raw" || format == "jsonl"
}
func output(data interface{}) {
	encoded, err := json.Marshal(data)
	if err != nil {
		outputError("OUTPUT_ERROR", "cannot encode result", "")
	}
	var value interface{}
	if err = json.Unmarshal(encoded, &value); err != nil {
		outputError("OUTPUT_ERROR", "cannot decode result", "")
	}
	if globalArgs.Fields != "" {
		value = projectFields(value, strings.Split(globalArgs.Fields, ","))
	}
	var b bytes.Buffer
	if globalArgs.IDOnly {
		for _, r := range outputRows(value) {
			m, ok := r.(map[string]interface{})
			if !ok || m["uuid"] == nil {
				outputError("INVALID_INPUT", "--id-only requires draft results", "")
			}
			fmt.Fprintln(&b, m["uuid"])
		}
	} else if globalArgs.Format == "raw" {
		m, ok := value.(map[string]interface{})
		if !ok {
			outputError("INVALID_INPUT", "raw output requires get", "")
		}
		content, ok := m["content"].(string)
		if !ok {
			outputError("INVALID_INPUT", "raw output requires a draft content field", "")
		}
		b.WriteString(content)
	} else if globalArgs.Format == "plain" || plainOutput {
		renderPlain(&b, value, "")
	} else if globalArgs.Format == "jsonl" {
		enc := json.NewEncoder(&b)
		for _, row := range outputRows(value) {
			if err = enc.Encode(row); err != nil {
				outputError("OUTPUT_ERROR", "cannot encode result", "")
			}
		}
	} else {
		enc := json.NewEncoder(&b)
		enc.SetIndent("", "  ")
		if err = enc.Encode(Response{Success: true, Data: value}); err != nil {
			outputError("OUTPUT_ERROR", "cannot encode result", "")
		}
	}
	if strings.HasPrefix(globalArgs.Deliver, "file:") {
		path := strings.TrimPrefix(globalArgs.Deliver, "file:")
		if err := atomicWrite(path, b.Bytes(), globalArgs.Overwrite); err != nil {
			outputError("OUTPUT_ERROR", err.Error(), "Choose a new path or use --overwrite")
		}
		_ = json.NewEncoder(os.Stdout).Encode(Response{Success: true, Data: map[string]interface{}{"path": path, "bytes": b.Len(), "scope": "local"}})
		return
	}
	if _, err = os.Stdout.Write(b.Bytes()); err != nil {
		outputError("OUTPUT_ERROR", "cannot write stdout", "")
	}
}
func outputRows(value interface{}) []interface{} {
	if a, ok := value.([]interface{}); ok {
		return a
	}
	if m, ok := value.(map[string]interface{}); ok {
		for _, key := range []string{"drafts", "actions", "workspaces", "tags"} {
			if a, ok := m[key].([]interface{}); ok {
				return a
			}
		}
	}
	return []interface{}{value}
}
func projectFields(value interface{}, fields []string) interface{} {
	for _, f := range fields {
		if !validDraftField(f) {
			outputError("INVALID_INPUT", "unknown draft field: "+f, "")
		}
	}
	if a, ok := value.([]interface{}); ok {
		for i := range a {
			a[i] = projectFields(a[i], fields)
		}
		return a
	}
	if m, ok := value.(map[string]interface{}); ok {
		if ds, ok := m["drafts"]; ok {
			m["drafts"] = projectFields(ds, fields)
			return m
		}
		if _, ok := m["uuid"]; ok {
			out := map[string]interface{}{}
			for _, f := range fields {
				if v, ok := m[f]; ok {
					out[f] = v
				}
			}
			return out
		}
	}
	outputError("INVALID_INPUT", "--fields requires draft results", "")
	return nil
}
func validDraftField(name string) bool {
	switch name {
	case "data_source", "snapshot", "uuid", "title", "tags", "isFlagged", "isArchived", "isTrashed", "folder", "createdAt", "modifiedAt", "accessedAt", "flagType", "permalink", "content", "createdLatitude", "createdLongitude", "modifiedLatitude", "modifiedLongitude":
		return true
	}
	return false
}
func renderPlain(b *bytes.Buffer, value interface{}, prefix string) {
	switch v := value.(type) {
	case map[string]interface{}:
		keys := []string{}
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			renderPlain(b, v[k], prefix+k+": ")
		}
	case []interface{}:
		for _, item := range v {
			renderPlain(b, item, prefix)
		}
	case nil:
	default:
		fmt.Fprintf(b, "%s%v\n", prefix, v)
	}
}
func outputError(code, message, hint string) {
	outputDetailedError(&drafts.Error{Code: code, Message: message, Hint: hint, RetrySafe: code == "INVALID_INPUT" || code == "COMMIT_REQUIRED"})
}
func exitCode(code string) int {
	switch code {
	case "INVALID_INPUT", "INVALID_FILTER", "UNKNOWN_COMMAND", "COMMIT_REQUIRED", "INTERACTIVE_REQUIRED":
		return 2
	case "DRAFT_NOT_FOUND", "ACTION_NOT_FOUND", "WORKSPACE_NOT_FOUND", "APP_NOT_FOUND", "DRAFTS_NOT_RUNNING":
		return 3
	case "PERMISSION_DENIED", "PRO_REQUIRED":
		return 4
	case "UNSUPPORTED_CAPABILITY":
		return 5
	case "CONFLICT":
		return 6
	case "TIMEOUT":
		return 7
	case "PARTIAL_FAILURE":
		return 8
	default:
		return 9
	}
}
func outputDetailedError(err *drafts.Error) {
	if globalArgs.FormatError == "plain" {
		fmt.Fprintf(os.Stderr, "%s: %s\n", err.Code, err.Message)
		if err.Hint != "" {
			fmt.Fprintln(os.Stderr, err.Hint)
		}
	} else {
		_ = json.NewEncoder(os.Stderr).Encode(Response{Success: false, Error: err})
	}
	os.Exit(exitCode(err.Code))
}
