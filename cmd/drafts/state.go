package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nerveband/drafts-applescript-cli/v4/pkg/drafts"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Profile struct {
	App     string `json:"app,omitempty"`
	Channel string `json:"channel,omitempty"`
	Timeout string `json:"timeout,omitempty"`
}
type profileFile struct {
	Profiles map[string]Profile `json:"profiles"`
}

var configSources = map[string]string{}

func configPath() string {
	if globalArgs.ConfigFile != "" {
		return globalArgs.ConfigFile
	}
	dir, e := os.UserConfigDir()
	handleDraftsError(e)
	return filepath.Join(dir, "drafts-cli", "profiles.json")
}
func statePath() string {
	if globalArgs.StateDir != "" {
		return globalArgs.StateDir
	}
	return filepath.Join(filepath.Dir(configPath()), "state")
}
func loadProfiles() (profileFile, error) {
	f := profileFile{Profiles: map[string]Profile{}}
	data, e := os.ReadFile(configPath())
	if errors.Is(e, os.ErrNotExist) {
		return f, nil
	}
	if e != nil {
		return f, e
	}
	e = json.Unmarshal(data, &f)
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	return f, e
}
func explicitFlag(name string) bool {
	for _, v := range os.Args[1:] {
		if v == "--" {
			break
		}
		if v == "--"+name || strings.HasPrefix(v, "--"+name+"=") {
			return true
		}
	}
	return false
}
func applyConfiguration() {
	if globalArgs.Profile == "" {
		globalArgs.Profile = os.Getenv("DRAFTS_CLI_PROFILE")
	}
	profile := Profile{}
	if globalArgs.Profile != "" {
		f, e := loadProfiles()
		handleDraftsError(e)
		var ok bool
		profile, ok = f.Profiles[globalArgs.Profile]
		if !ok {
			outputError("INVALID_INPUT", "unknown profile: "+globalArgs.Profile, "Use drafts profile list")
		}
	}
	for _, setting := range []struct {
		name, env, profile string
		value              *string
	}{{"app", "DRAFTS_CLI_APP", profile.App, &globalArgs.App}, {"channel", "DRAFTS_CLI_CHANNEL", profile.Channel, &globalArgs.Channel}, {"timeout", "DRAFTS_CLI_TIMEOUT", profile.Timeout, &globalArgs.Timeout}} {
		configSources[setting.name] = "default"
		if explicitFlag(setting.name) {
			configSources[setting.name] = "flag"
			continue
		}
		if v := os.Getenv(setting.env); v != "" {
			*setting.value = v
			configSources[setting.name] = "env"
		} else if setting.profile != "" {
			*setting.value = setting.profile
			configSources[setting.name] = "profile"
		}
	}
}
func configView() interface{} {
	return map[string]interface{}{"app": globalArgs.App, "channel": globalArgs.Channel, "timeout": globalArgs.Timeout, "profile": globalArgs.Profile, "sources": configSources, "config_file": configPath(), "state_dir": statePath(), "credentials": "macOS Automation permission, no API secrets", "scope": "local"}
}

var safeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func runProfile(p *ProfileCmd) interface{} {
	if p.Operation != "list" && p.Operation != "show" && p.Operation != "save" {
		outputError("INVALID_INPUT", "profile operation must be list, show, or save", "")
	}
	f, e := loadProfiles()
	handleDraftsError(e)
	if p.Operation == "list" {
		return f
	}
	if !safeName.MatchString(p.Name) {
		outputError("INVALID_INPUT", "profile name must contain 1 to 128 letters, numbers, '.', '_' or '-'", "")
	}
	if p.Operation == "show" {
		r, ok := f.Profiles[p.Name]
		if !ok {
			outputError("INVALID_INPUT", "unknown profile: "+p.Name, "")
		}
		return r
	}
	if p.Input == "" {
		outputError("INVALID_INPUT", "profile save requires --input", "")
	}
	var r Profile
	handleDraftsError(decodeJSONInput(p.Input, &r))
	if r.Channel != "" && r.Channel != "auto" && r.Channel != "stable" && r.Channel != "beta" {
		outputError("INVALID_INPUT", "channel must be auto, stable, or beta", "")
	}
	if r.Timeout != "" {
		d, e := time.ParseDuration(r.Timeout)
		if e != nil || d <= 0 || d > 10*time.Minute {
			outputError("INVALID_INPUT", "timeout must be greater than zero and at most 10m", "")
		}
	}
	if plan, ok := mutationPlan("profile-save", map[string]interface{}{"name": p.Name, "settings": r}, false); ok {
		return plan
	}
	f.Profiles[p.Name] = r
	data, e := json.MarshalIndent(f, "", "  ")
	handleDraftsError(e)
	handleDraftsError(os.MkdirAll(filepath.Dir(configPath()), 0700))
	handleDraftsError(atomicWrite(configPath(), data, true))
	return map[string]interface{}{"name": p.Name, "settings": r, "scope": "local"}
}
func atomicWrite(path string, data []byte, overwrite bool) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".drafts-output-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(data); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if overwrite {
		return os.Rename(f.Name(), path)
	}
	return os.Link(f.Name(), path)
}
func runFeedback(p *FeedbackCmd) interface{} {
	if strings.TrimSpace(p.Message) == "" {
		outputError("INVALID_INPUT", "feedback requires a message", "")
	}
	validateContent(p.Message)
	if plan, ok := mutationPlan("feedback", map[string]interface{}{"message": p.Message}, false); ok {
		return plan
	}
	dir := filepath.Join(statePath(), "feedback")
	handleDraftsError(os.MkdirAll(dir, 0700))
	f, e := os.CreateTemp(dir, "feedback-*.json")
	handleDraftsError(e)
	record := map[string]interface{}{"timestamp": time.Now().UTC().Format(time.RFC3339Nano), "version": version, "message": p.Message, "scope": "local"}
	e = json.NewEncoder(f).Encode(record)
	closeErr := f.Close()
	handleDraftsError(e)
	handleDraftsError(closeErr)
	return map[string]interface{}{"recorded": true, "path": f.Name(), "upstream": issuesURL}
}

type ledgerRecord struct {
	Key       string `json:"key"`
	Command   string `json:"command"`
	Hash      string `json:"hash"`
	Status    string `json:"status"`
	UUID      string `json:"uuid,omitempty"`
	CreatedAt string `json:"created_at"`
	ErrorCode string `json:"error_code,omitempty"`
}

func ledgerPath(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(statePath(), "jobs", hex.EncodeToString(sum[:])+".json")
}
func beginIdempotency(command string, request interface{}, preflight func() error) (func(string, error), interface{}) {
	noop := func(string, error) {}
	key := globalArgs.IdempotencyKey
	if key == "" {
		return noop, nil
	}
	if !safeName.MatchString(key) {
		outputError("INVALID_INPUT", "idempotency key must be 1 to 128 safe name characters", "")
	}
	payload, _ := json.Marshal(map[string]interface{}{"command": command, "app": configuredApp.Path, "request": request})
	sum := sha256.Sum256(payload)
	hash := hex.EncodeToString(sum[:])
	path := ledgerPath(key)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		handleDraftsError(preflight())
	}
	handleDraftsError(os.MkdirAll(filepath.Dir(path), 0700))
	record := ledgerRecord{Key: key, Command: command, Hash: hash, Status: "in_progress", CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	data, _ := json.Marshal(record)
	e := atomicWrite(path, data, false)
	if errors.Is(e, os.ErrExist) {
		existing, e := os.ReadFile(path)
		handleDraftsError(e)
		handleDraftsError(json.Unmarshal(existing, &record))
		if record.Hash != hash {
			outputError("CONFLICT", "idempotency key belongs to a different request", "Use a new key only for a new operation")
		}
		if record.Status != "completed" {
			outputDetailedError(&drafts.Error{Code: "CONFLICT", Message: "previous operation is incomplete or uncertain", UUID: record.UUID, Hint: "Inspect jobs and the target before retrying. Do not change the key to bypass duplicate protection."})
		}
		return noop, map[string]interface{}{"uuid": record.UUID, "status": "replayed", "original_command": record.Command}
	}
	handleDraftsError(e)
	return func(uuid string, err error) {
		record.UUID = uuid
		record.Status = "completed"
		if err != nil {
			record.Status = "failed_or_uncertain"
			var detail *drafts.Error
			if errors.As(err, &detail) {
				record.ErrorCode = detail.Code
			}
		}
		data, e := json.Marshal(record)
		if e == nil {
			e = atomicWrite(path, data, true)
		}
		if e != nil {
			outputDetailedError(&drafts.Error{Code: "PARTIAL_FAILURE", Message: "operation finished but its durable receipt could not be saved", UUID: uuid, Hint: "Inspect the target and ledger before retrying", Completed: nil})
		}
	}, nil
}
func runJobs(key string) interface{} {
	records := []ledgerRecord{}
	paths := []string{}
	if key != "" {
		paths = append(paths, ledgerPath(key))
	} else {
		paths, _ = filepath.Glob(filepath.Join(statePath(), "jobs", "*.json"))
	}
	for _, path := range paths {
		data, e := os.ReadFile(path)
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		handleDraftsError(e)
		var r ledgerRecord
		handleDraftsError(json.Unmarshal(data, &r))
		records = append(records, r)
	}
	return map[string]interface{}{"jobs": records, "scope": "local", "completion": "action receipts track submission, not action completion"}
}
func validateGlobalOptions(args Args) {
	if args.DataSource != "live" && args.DataSource != "local" {
		outputError("INVALID_INPUT", fmt.Sprintf("data-source must be live or local, got %q", args.DataSource), "")
	}
	if args.DataSource == "local" && args.List == nil && args.Get == nil {
		outputError("INVALID_INPUT", "local data source supports list/get only", "")
	}
	if args.Format == "raw" && args.Get == nil {
		outputError("INVALID_INPUT", "raw output requires get", "")
	}
	if args.Fields != "" && args.Get == nil && args.List == nil {
		outputError("INVALID_INPUT", "fields applies to get or list", "")
	}
	if args.IDOnly && args.Get == nil && args.List == nil {
		outputError("INVALID_INPUT", "id-only applies to get or list", "")
	}
	if args.DryRun && (args.Fields != "" || args.IDOnly) {
		outputError("INVALID_INPUT", "dry-run cannot be combined with draft projections", "")
	}

	if args.FormatError != "json" && args.FormatError != "plain" {
		outputError("INVALID_INPUT", fmt.Sprintf("format-error must be json or plain, got %q", args.FormatError), "")
	}
	if args.Deliver != "stdout" && !strings.HasPrefix(args.Deliver, "file:") {
		outputError("INVALID_INPUT", fmt.Sprintf("deliver must be stdout or file:<path>, got %q", args.Deliver), "")
	}
	if strings.HasPrefix(args.Deliver, "file:") {
		path := strings.TrimPrefix(args.Deliver, "file:")
		if path == "" {
			outputError("INVALID_INPUT", "file delivery requires a path", "")
		}
		info, e := os.Stat(filepath.Dir(path))
		if e != nil || !info.IsDir() {
			outputError("INVALID_INPUT", "output directory must already exist", "")
		}
		if _, e := os.Lstat(path); e == nil && !args.Overwrite {
			outputError("INVALID_INPUT", "output file already exists", "Use --overwrite or a new path")
		}
	}
	if args.IdempotencyKey != "" {
		if !safeName.MatchString(args.IdempotencyKey) {
			outputError("INVALID_INPUT", "invalid idempotency key", "")
		}
		if args.New == nil && args.Create == nil && args.Run == nil {
			outputError("INVALID_INPUT", "idempotency-key applies to create/new and run", "")
		}
	}
	if args.Stdin || args.TextFile != "" {
		supported := args.New != nil || args.Create != nil || args.Append != nil || args.Prepend != nil || args.Replace != nil || args.Update != nil || args.Run != nil
		if !supported {
			outputError("INVALID_INPUT", "content input options require create, append, prepend, replace/update, or run", "")
		}
	}
	if args.Fields != "" {
		for _, f := range strings.Split(args.Fields, ",") {
			if !validDraftField(f) {
				outputError("INVALID_INPUT", "unknown draft field: "+f, "")
			}
		}
		if args.IDOnly && !strings.Contains(","+args.Fields+",", ",uuid,") {
			outputError("INVALID_INPUT", "id-only requires uuid in selected fields", "")
		}
	}
}
