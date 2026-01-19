package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Response envelope for all JSON output
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

// Global flag for plain output
var plainOutput bool

// Output writes the response as JSON or plain text
func output(data interface{}) {
	if plainOutput {
		fmt.Println(data)
		return
	}
	resp := Response{Success: true, Data: data}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(resp)
}

// OutputError writes an error response
func outputError(code, message, hint string) {
	if plainOutput {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
		if hint != "" {
			fmt.Fprintf(os.Stderr, "Hint: %s\n", hint)
		}
		os.Exit(1)
	}
	resp := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Hint:    hint,
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(resp)
	os.Exit(1)
}
