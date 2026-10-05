package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nerveband/drafts-applescript-cli/v4/pkg/drafts"
)

func readTextInput(text string) (string, error) {
	if text != "" {
		if globalArgs.Stdin || globalArgs.TextFile != "" {
			return "", fmt.Errorf("cannot combine positional content with --stdin or --text-file")
		}
		if !utf8.ValidString(text) {
			return "", fmt.Errorf("content must be UTF-8")
		}
		return text, nil
	}
	if globalArgs.TextFile != "" {
		data, err := os.ReadFile(globalArgs.TextFile)
		if err != nil {
			return "", err
		}
		if !utf8.Valid(data) {
			return "", fmt.Errorf("content must be UTF-8")
		}
		return string(data), nil
	}
	if !globalArgs.Stdin {
		return "", fmt.Errorf("content is required; use --stdin, --text-file, or --input with explicit content")
	}
	return readStdin()
}
func readStdin() (string, error) {
	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() { data, err := io.ReadAll(io.LimitReader(os.Stdin, 16*1024*1024+1)); ch <- result{data, err} }()
	select {
	case r := <-ch:
		if r.err != nil {
			return "", r.err
		}
		if len(r.data) > 16*1024*1024 {
			return "", fmt.Errorf("input exceeds 16 MiB")
		}
		if !utf8.Valid(r.data) {
			return "", fmt.Errorf("input must be UTF-8")
		}
		return string(r.data), nil
	case <-time.After(commandTimeout):
		return "", &drafts.Error{Code: "TIMEOUT", Message: "stdin timed out before a request was submitted", Hint: "Supply input and close the stream, or increase --timeout", RetrySafe: true}
	}
}

func resolveActiveUUID(uuid string) (string, error) {
	if uuid != "" {
		return uuid, nil
	}
	return drafts.Active()
}

// Run FZF on input, return UUID.
func fzfUUID(input string) (string, error) {
	line, err := fzf(input)
	if err != nil {
		return "", err
	}
	return strings.Split(line, fmt.Sprintf(" %c ", drafts.Separator))[0], nil
}

// Run FZF on input, return line.
func fzf(input string) (string, error) {
	var result strings.Builder
	cmd := exec.Command("fzf", "--delimiter", "\\|", "--with-nth", "2")
	cmd.Stdout = &result
	cmd.Stderr = os.Stderr
	cmd.Stdin = strings.NewReader(input)

	if err := cmd.Start(); err != nil {
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 130 {
			return "", fmt.Errorf("selection cancelled")
		}
		return "", err
	}

	return strings.TrimSpace(result.String()), nil
}

func editor(input string) (string, error) {
	f, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name()) // clean up

	if _, err := f.Write([]byte(input)); err != nil {
		return "", err
	}

	if err := f.Close(); err != nil {
		return "", err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	parts, err := splitEditor(editor)
	if err != nil {
		return "", err
	}
	cmd := exec.Command(parts[0], append(parts[1:], f.Name())...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	data, err := os.ReadFile(f.Name())
	if err != nil {
		return "", err
	}

	if !utf8.Valid(data) {
		return "", fmt.Errorf("editor produced invalid UTF-8")
	}
	return string(data), nil
}

// Parse editor arguments without a shell, expansion, or command substitution.
func splitEditor(value string) ([]string, error) {
	var parts []string
	var b strings.Builder
	var quote rune
	escaped, started := false, false
	for _, r := range value {
		if escaped {
			b.WriteRune(r)
			escaped = false
			started = true
			continue
		}
		if r == '\\' && quote != '\'' {
			escaped = true
			started = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			started = true
			continue
		}
		if r == ' ' || r == '\t' {
			if started {
				parts = append(parts, b.String())
				b.Reset()
				started = false
			}
			continue
		}
		b.WriteRune(r)
		started = true
	}
	if escaped || quote != 0 {
		return nil, fmt.Errorf("EDITOR has unfinished quoting")
	}
	if started {
		parts = append(parts, b.String())
	}
	if len(parts) == 0 || parts[0] == "" {
		return nil, fmt.Errorf("EDITOR is empty")
	}
	return parts, nil
}
