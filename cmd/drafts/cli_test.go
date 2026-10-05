package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/nerveband/drafts-applescript-cli/v4/pkg/drafts"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("DRAFTS_CLI_TEST_PROCESS") == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type cliResult struct {
	code     int
	out, err string
}

func invokeCLI(t *testing.T, args []string, input io.Reader, extraEnv ...string) cliResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], args...)
	cmd.Env = append(os.Environ(), "DRAFTS_CLI_TEST_PROCESS=1", "DRAFTS_CLI_PROFILE=", "DRAFTS_CLI_APP=", "DRAFTS_CLI_CHANNEL=", "DRAFTS_CLI_TIMEOUT=")
	cmd.Env = append(cmd.Env, extraEnv...)
	if input == nil {
		input = strings.NewReader("")
	}
	cmd.Stdin = input
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	e := cmd.Run()
	code := 0
	if e != nil {
		var ok bool
		if exit, yes := e.(*exec.ExitError); yes {
			code = exit.ExitCode()
			ok = true
		}
		if !ok {
			t.Fatalf("CLI failed: %v %s", e, stderr.String())
		}
	}
	return cliResult{code, out.String(), stderr.String()}
}
func decodedData(t *testing.T, r cliResult) map[string]interface{} {
	t.Helper()
	if r.code != 0 || r.err != "" {
		t.Fatalf("%#v", r)
	}
	var v struct {
		Success bool
		Data    map[string]interface{}
	}
	if e := json.Unmarshal([]byte(r.out), &v); e != nil || !v.Success {
		t.Fatalf("bad JSON %v %s", e, r.out)
	}
	return v.Data
}
func errorCode(t *testing.T, r cliResult) string {
	t.Helper()
	if r.code == 0 || r.out != "" {
		t.Fatalf("expected stderr only: %#v", r)
	}
	var v struct{ Error drafts.Error }
	if e := json.Unmarshal([]byte(r.err), &v); e != nil {
		t.Fatalf("bad error JSON: %s", r.err)
	}
	return v.Error.Code
}
func TestCLIFailFastAndStrictInput(t *testing.T) {
	cases := [][]string{
		{"trash", "--commit"}, {"trash", "--dry-run"}, {"replace", "--input", `{"content":"x"}`, "--commit"}, {"update", "--input", `{"content":"x"}`, "--dry-run"},
		{"unknown-command"}, {"create"}, {"create", "--input", "null", "--dry-run"}, {"create", "--input", "{}", "--dry-run"}, {"replace", "--input", "{}", "--dry-run"},
		{"create", "--input", `{"content":null}`, "--dry-run"}, {"create", "--input", `{"content":"x","content":"y"}`, "--dry-run"}, {"create", "--input", `{"content":"x","unknown":1}`, "--dry-run"},
		{"create", "--input", `{"content":"x"} {}`, "--dry-run"}, {"create", "--input", `{"content":"x","flagType":7}`, "--dry-run"},
		{"run", "Copy", "text", "-u", exampleUUID, "--dry-run"}, {"run", "--input", `{"action":"Copy","uuid":"` + exampleUUID + `","content":""}`, "--dry-run"}, {"run", "--input", `{"action":"Copy"}`, "--dry-run"},
		{"replace", "text", "--commit", "-u", "garbage"}, {"trash", exampleUUID}, {"edit", exampleUUID}, {"upgrade"}, {"list", "--limit", "-1"}, {"list", "--sort", "bad"}, {"list", "--order", "bad"}, {"list", "--created-after", "bad"},
		{"list", "--created-after", "2026-02-01", "--created-before", "2026-01-01"}, {"create", "text", "--stdin", "--dry-run"}, {"create", "--input", `{"content":"x"}`, "--stdin", "--dry-run"},
		{"get", "--timeout", "0s"}, {"version", "--deliver", "webhook:invalid"}, {"version", "--format", "yaml"}, {"get", exampleUUID, "--fields", "bad"}, {"select"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args[:1], " ")+fmt.Sprint(len(args))+fmt.Sprint(args[1:]), func(t *testing.T) {
			r := invokeCLI(t, args, nil)
			code := errorCode(t, r)
			if r.code != 2 {
				t.Fatalf("wanted exit 2: %s %#v", code, r)
			}
		})
	}
}
func TestCLIContentSourcesPreserveBytes(t *testing.T) {
	text := "one\ttwo\n☃\r\n"
	path := filepath.Join(t.TempDir(), "note.txt")
	os.WriteFile(path, []byte(text), 0600)
	for _, args := range [][]string{{"create", "--stdin", "--dry-run"}, {"create", "--text-file", path, "--dry-run"}} {
		r := invokeCLI(t, args, strings.NewReader(text))
		d := decodedData(t, r)
		request := d["request"].(map[string]interface{})
		if request["content"] != text {
			t.Fatalf("content changed: %#v", request)
		}
	}
	payload, _ := json.Marshal(CreateRequest{Content: text})
	file := filepath.Join(t.TempDir(), "input.json")
	os.WriteFile(file, payload, 0600)
	for _, args := range [][]string{{"create", "--input", "-", "--dry-run"}, {"create", "--input", "@" + file, "--dry-run"}} {
		d := decodedData(t, invokeCLI(t, args, bytes.NewReader(payload)))
		if d["request"].(map[string]interface{})["content"] != text {
			t.Fatal("JSON source changed content")
		}
	}
	r := invokeCLI(t, []string{"create", "--input", `{"content":""}`, "--dry-run"}, nil)
	if decodedData(t, r)["request"].(map[string]interface{})["content"] != "" {
		t.Fatal("explicit empty rejected")
	}
	reader, writer, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	defer reader.Close()
	defer writer.Close()
	r = invokeCLI(t, []string{"create", "--stdin", "--timeout", "30ms", "--dry-run"}, reader)
	if errorCode(t, r) != "TIMEOUT" || !strings.Contains(r.err, "timed out") {
		t.Fatal(r)
	}
}
func TestContractHelpAndSafety(t *testing.T) {
	tools := getTools()
	canonical := map[string]ToolDefinition{}
	for _, tool := range tools {
		canonical[strings.TrimPrefix(tool.Name, "drafts_")] = tool
		for _, a := range tool.Aliases {
			canonical[a] = tool
		}
		if len(tool.Description) > 240 {
			t.Fatal("tool description exceeds budget")
		}
		if tool.Parameters["additionalProperties"] != false {
			t.Fatal("schema permits unknown fields")
		}
	}
	checkFlags := func(typ reflect.Type) {
		for i := 0; i < typ.NumField(); i++ {
			tag := typ.Field(i).Tag.Get("arg")
			for _, banned := range []string{"--force", "--json", "--output"} {
				for _, flag := range strings.Split(tag, ",") {
					if flag == banned {
						t.Fatalf("banned flag %s in %s", banned, typ.Name())
					}
				}
			}
		}
	}
	checkFlags(reflect.TypeOf(Args{}))
	typ := reflect.TypeOf(Args{})
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("arg")
		if !strings.HasPrefix(tag, "subcommand:") {
			continue
		}
		checkFlags(typ.Field(i).Type.Elem())
		name := strings.TrimPrefix(tag, "subcommand:")
		if _, ok := canonical[name]; !ok {
			t.Fatalf("command missing schema: %s", name)
		}
		r := invokeCLI(t, []string{name, "--help"}, nil)
		if r.code != 0 || !strings.Contains(r.out, "Example:") || !strings.Contains(r.out, commandExamples[name]) {
			t.Fatalf("missing help example %s: %#v", name, r)
		}
	}
	for _, tool := range tools {
		if tool.Mutates && strings.TrimPrefix(tool.Name, "drafts_") != "select" && !tool.SupportsDryRun {
			t.Fatal("mutation without dry-run", tool.Name)
		}
	}
	for _, name := range []string{"replace", "edit", "trash", "upgrade"} {
		if !canonical[name].RequiresCommit {
			t.Fatal("missing commit metadata", name)
		}
	}
	for _, name := range []string{"ls", "rm", "force"} {
		if _, ok := canonical[name]; ok {
			t.Fatal("banned alias introduced", name)
		}
	}
	schema := decodedData(t, invokeCLI(t, []string{"schema"}, nil))
	if schema["schema_version"] != "1" || schema["scope"] != "local" {
		t.Fatal("unversioned schema")
	}
	for _, doc := range []string{"../../README.md", "../../skills/SKILL.md"} {
		data, e := os.ReadFile(doc)
		if e != nil {
			t.Fatal(e)
		}
		for _, name := range []string{"apps", "schema", "create", "list", "replace", "trash", "run"} {
			if !strings.Contains(string(data), "drafts "+name) {
				t.Fatalf("%s missing %s workflow", doc, name)
			}
		}
	}
}
func TestEditorArguments(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want []string
	}{{`code --wait`, []string{"code", "--wait"}}, {`'/path with spaces/editor' --wait`, []string{"/path with spaces/editor", "--wait"}}, {`editor "literal $(secret)"`, []string{"editor", "literal $(secret)"}}} {
		got, e := splitEditor(tc.raw)
		if e != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%q %#v %v", tc.raw, got, e)
		}
	}
	if _, e := splitEditor(`editor "`); e == nil {
		t.Fatal("unfinished quotes accepted")
	}
}
func TestAtomicArtifactAndProfiles(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "version.json")
	r := invokeCLI(t, []string{"version", "--deliver", "file:" + out}, nil)
	decodedData(t, r)
	data, e := os.ReadFile(out)
	if e != nil || !json.Valid(data) {
		t.Fatal(e)
	}
	if r = invokeCLI(t, []string{"version", "--deliver", "file:" + out}, nil); errorCode(t, r) != "INVALID_INPUT" {
		t.Fatal(r)
	}
	decodedData(t, invokeCLI(t, []string{"version", "--deliver", "file:" + out, "--overwrite"}, nil))
	config := filepath.Join(dir, "profiles.json")
	decodedData(t, invokeCLI(t, []string{"profile", "save", "work", "--input", `{"app":"/Applications/Drafts.app","channel":"beta","timeout":"5s"}`, "--config-file", config}, nil))
	info, e := os.Stat(config)
	if e != nil || info.Mode().Perm() != 0600 {
		t.Fatal("unsafe config mode", e)
	}
	r = invokeCLI(t, []string{"config", "--profile", "work", "--config-file", config, "--channel", "auto"}, nil, "DRAFTS_CLI_TIMEOUT=2s")
	d := decodedData(t, r)
	sources := d["sources"].(map[string]interface{})
	if d["app"] != "/Applications/Drafts.app" || sources["app"] != "profile" || sources["channel"] != "flag" || sources["timeout"] != "env" {
		t.Fatal(d)
	}
}
func mockAppCLI(t *testing.T) (string, []string, string) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("plutil metadata inspection requires macOS")
	}
	dir := t.TempDir()
	app := filepath.Join(dir, "Drafts.app")
	resources := filepath.Join(app, "Contents", "Resources")
	os.MkdirAll(resources, 0700)
	plist := `<?xml version="1.0"?><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>com.agiletortoise.Drafts-Test</string><key>CFBundleShortVersionString</key><string>54.1</string><key>CFBundleVersion</key><string>900</string></dict></plist>`
	os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(plist), 0600)
	dictionary := `<dictionary><suite><class name="draft"><property name="content"/><property name="tag list"/><property name="query tag names"/><property name="creation date"/><property name="modification date"/><property name="flagged" code="DrFl" type="boolean" access="rw"/></class><class name="workspace"/><class name="action"/><class name="tag"/><command name="perform"/><command name="open"><direct-parameter><type type="draft"/><type type="workspace"/></direct-parameter></command></suite></dictionary>`
	os.WriteFile(filepath.Join(resources, "Drafts.sdef"), []byte(dictionary), 0600)
	sample := drafts.Draft{UUID: exampleUUID, Title: "One\ttwo ☃", Content: "One\ttwo\nThree ||| ☃\n", Tags: []string{"a|||b", "c\td"}, Folder: "inbox", CreatedAt: "2026-01-01T00:00:00Z", ModifiedAt: "2026-01-02T00:00:00Z"}
	row, _ := json.Marshal(sample)
	rows, _ := json.Marshal([]drafts.Draft{sample})
	indexRows, _ := json.Marshal([]map[string]interface{}{{"uuid": sample.UUID, "sortDate": sample.CreatedAt}})
	log := filepath.Join(dir, "events")
	script := `#!/bin/sh
script=$(cat)
printf '%s\n' "$script" >> "$EVENT_LOG"
case "$script" in
 *'use framework "AppKit"'*) if [ "$MOCK_MODE" = stopped ]; then printf 'false\n'; else printf 'true\n'; fi;;
 *'return exists action'*'Missing'*) printf 'false\n';;
 *'return exists '*) printf 'true\n';;
 *'make new draft'*) printf '` + exampleUUID + `\n';;
 *'perform action'*) if [ "$MOCK_MODE" = fail ]; then printf 'synthetic private content (-1743)\n' >&2; exit 1; fi; printf 'submitted\n';;
 *'set indexRows'*) printf '%s\n' '` + string(indexRows) + `';;
 *'set selectedIDs'*) printf '%s\n' '` + string(rows) + `';;
 *'jsonText({count'*) printf '[1]\n';;
 *'jsonText(row)'*) printf '%s\n' '` + string(row) + `';;
 *'jsonText(rows)'*) printf '[]\n';;
 *) printf 'true\n';;
esac
`
	os.WriteFile(filepath.Join(dir, "osascript"), []byte(script), 0700)
	env := []string{"PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH"), "EVENT_LOG=" + log, "MOCK_MODE="}
	return app, env, log
}
func TestCLIWithSyntheticDrafts(t *testing.T) {
	app, env, log := mockAppCLI(t)
	invoke := func(args ...string) cliResult { return invokeCLI(t, append(args, "--app", app), nil, env...) }
	decodedData(t, invoke("info"))
	if data, _ := os.ReadFile(log); len(data) != 0 {
		t.Fatal("info accessed Drafts")
	}
	d := decodedData(t, invoke("schema", "--detected"))
	if d["flag_type_available"] != false {
		t.Fatal("stable dictionary enabled beta capability")
	}
	tools := d["schema"].(map[string]interface{})["tools"].([]interface{})
	for _, raw := range tools {
		tool := raw.(map[string]interface{})
		if tool["name"] == "drafts_new" {
			properties := tool["parameters"].(map[string]interface{})["properties"].(map[string]interface{})
			if _, ok := properties["flagType"]; ok {
				t.Fatal("detected fallback still advertises unavailable flagType")
			}
		}
	}

	r := invoke("flag", exampleUUID, "--flag-type", "2")
	if errorCode(t, r) != "UNSUPPORTED_CAPABILITY" || r.code != 5 {
		t.Fatal(r)
	}
	r = invoke("get", exampleUUID, "--format", "raw")
	if r.code != 0 || r.err != "" || r.out != "One\ttwo\nThree ||| ☃\n" {
		t.Fatal(r)
	}
	r = invoke("get", exampleUUID, "--plain")
	if r.code != 0 || strings.Contains(r.out, "0x") || !strings.Contains(r.out, "content:") {
		t.Fatal(r)
	}
	r = invoke("list", "--id-only")
	if r.out != exampleUUID+"\n" || r.code != 0 {
		t.Fatal(r)
	}
	r = invoke("list", "--format", "jsonl", "--fields", "uuid,title")
	var projected map[string]interface{}
	if json.Unmarshal([]byte(r.out), &projected) != nil || len(projected) != 2 {
		t.Fatal(r)
	}
	r = invoke("list", "--count")
	if decodedData(t, r)["count"] != float64(1) {
		t.Fatal(r)
	}
	for _, args := range [][]string{{"create", "--input", `{"content":"synthetic"}`}, {"append", "text", "-u", exampleUUID}, {"prepend", "text", "-u", exampleUUID}, {"replace", "text", "-u", exampleUUID, "--commit"}, {"archive", exampleUUID}, {"inbox", exampleUUID}, {"trash", exampleUUID, "--commit"}, {"tag", exampleUUID, "-t", "work"}, {"remove-tags", exampleUUID, "-t", "work"}, {"flag", exampleUUID}, {"unflag", exampleUUID}, {"open", exampleUUID}, {"workspace", "--open", "Work"}, {"run", "Copy", "-u", exampleUUID}} {
		decodedData(t, invoke(args...))
	}
	data, _ := os.ReadFile(log)
	for _, required := range []string{`«property DrFl»`, `set folder of draft id`, `set tag list of d`, `perform action (action "Copy")`} {
		if !strings.Contains(string(data), required) {
			t.Fatal("missing wired script", required)
		}
	}
	state := t.TempDir()
	args := []string{"create", "--input", `{"content":"synthetic"}`, "--idempotency-key", "create-1", "--state-dir", state}
	decodedData(t, invoke(args...))
	before, _ := os.ReadFile(log)
	replay := decodedData(t, invoke(args...))
	after, _ := os.ReadFile(log)
	if replay["status"] != "replayed" || len(before) != len(after) {
		t.Fatal("duplicate submission", replay)
	}
	conflicting := append([]string{}, args...)
	conflicting[2] = `{"content":"different"}`
	if r = invoke(conflicting...); errorCode(t, r) != "CONFLICT" {
		t.Fatal(r)
	}
	r = invoke("run", "Copy", "-u", exampleUUID, "--idempotency-key", "run-1", "--state-dir", state)
	if decodedData(t, r)["status"] != "submitted" {
		t.Fatal(r)
	}
}
func TestPreflightFailuresDoNotReserveKeys(t *testing.T) {
	app, env, _ := mockAppCLI(t)
	state := t.TempDir()
	args := []string{"create", "--input", `{"content":"synthetic"}`, "--idempotency-key", "preflight", "--state-dir", state, "--app", app}
	r := invokeCLI(t, args, nil, append(env, "MOCK_MODE=stopped")...)
	if errorCode(t, r) != "DRAFTS_NOT_RUNNING" {
		t.Fatal(r)
	}
	jobs := decodedData(t, invokeCLI(t, []string{"jobs", "preflight", "--state-dir", state}, nil))
	if len(jobs["jobs"].([]interface{})) != 0 {
		t.Fatal(jobs)
	}
	decodedData(t, invokeCLI(t, args, nil, env...))
	for _, request := range []string{`{"content":"synthetic","action":"Missing"}`, `{"content":"synthetic","flagType":2}`} {
		args[2] = request
		args[4] = "preflight-optional"
		r = invokeCLI(t, args, nil, env...)
		if r.code == 0 {
			t.Fatal(r)
		}
		jobs = decodedData(t, invokeCLI(t, []string{"jobs", "preflight-optional", "--state-dir", state}, nil))
		if len(jobs["jobs"].([]interface{})) != 0 {
			t.Fatal(jobs)
		}
	}
}

func TestCLIPartialActionError(t *testing.T) {
	app, env, _ := mockAppCLI(t)
	env = append(env, "MOCK_MODE=fail")
	r := invokeCLI(t, []string{"create", "--input", `{"content":"synthetic","action":"Copy"}`, "--app", app}, nil, env...)
	if errorCode(t, r) != "PARTIAL_FAILURE" || r.code != 8 || !strings.Contains(r.err, exampleUUID) || strings.Contains(r.err, "private content") {
		t.Fatal(r)
	}
}

func TestSnapshotsAndUncertainReceipts(t *testing.T) {
	app, env, log := mockAppCLI(t)
	state := t.TempDir()
	invoke := func(args ...string) cliResult {
		return invokeCLI(t, append(args, "--app", app, "--state-dir", state), nil, env...)
	}
	info := decodedData(t, invoke("sync", "--filter", "all", "--full", "--limit", "5"))
	if info["data_source"] != "live" {
		t.Fatal(info)
	}
	provenance := info["snapshot"].(map[string]interface{})
	options := provenance["query_options"].(map[string]interface{})
	if options["Limit"] != float64(5) || options["Full"] != true {
		t.Fatal("snapshot lost query provenance", provenance)
	}
	before, _ := os.ReadFile(log)
	d := decodedData(t, invoke("list", "--data-source", "local", "--search", "Three", "--full"))
	after, _ := os.ReadFile(log)
	if d["data_source"] != "local" || d["count"] != float64(1) || len(before) != len(after) {
		t.Fatal("local query contacted Drafts", d)
	}
	raw := invoke("get", exampleUUID, "--data-source", "local", "--format", "raw")
	if raw.out != "One\ttwo\nThree ||| ☃\n" || raw.code != 0 {
		t.Fatal(raw)
	}
	if c := decodedData(t, invoke("list", "--data-source", "local", "--count")); c["count"] != float64(1) {
		t.Fatal(c)
	}
	filtered := decodedData(t, invoke("list", "--data-source", "local", "--created-after", "2026-01-02"))
	if filtered["count"] != float64(0) {
		t.Fatal(filtered)
	}
	if r := invoke("archive", exampleUUID, "--data-source", "local", "--dry-run"); errorCode(t, r) != "INVALID_INPUT" {
		t.Fatal(r)
	}
	snapshotData, e := os.ReadFile(filepath.Join(state, "snapshot.json"))
	if e != nil || !json.Valid(snapshotData) {
		t.Fatal(e)
	}
	scoped := decodedData(t, invoke("sync", "--filter", "inbox", "--full", "--search", "Three", "--tag", "fixture", "--created-after", "2025-01-01"))["snapshot"].(map[string]interface{})
	if scoped["search"] != "Three" || scoped["query_options"].(map[string]interface{})["CreatedAfter"] == "" {
		t.Fatal(scoped)
	}
	if r := invoke("list", "--data-source", "local", "--filter", "archive"); errorCode(t, r) != "UNSUPPORTED_CAPABILITY" {
		t.Fatal(r)
	}
	decodedData(t, invoke("sync", "--filter", "all"))
	if r := invoke("list", "--data-source", "local", "--search", "Three"); errorCode(t, r) != "UNSUPPORTED_CAPABILITY" {
		t.Fatal(r)
	}
	env = append(env, "MOCK_MODE=fail")
	args := []string{"create", "--input", `{"content":"synthetic","action":"Copy"}`, "--idempotency-key", "uncertain-1"}
	r := invoke(args...)
	if errorCode(t, r) != "PARTIAL_FAILURE" {
		t.Fatal(r)
	}
	before, _ = os.ReadFile(log)
	r = invoke(args...)
	after, _ = os.ReadFile(log)
	if errorCode(t, r) != "CONFLICT" || len(before) != len(after) {
		t.Fatal("uncertain request replayed", r)
	}
	jobs := decodedData(t, invoke("jobs", "uncertain-1"))
	record := jobs["jobs"].([]interface{})[0].(map[string]interface{})
	if record["status"] != "failed_or_uncertain" || record["uuid"] != exampleUUID {
		t.Fatal(jobs)
	}
}
func TestNoEventsForEveryMutationPreview(t *testing.T) {
	app, env, log := mockAppCLI(t)
	cases := [][]string{
		{"create", "--input", `{"content":"synthetic"}`}, {"append", "text", "-u", exampleUUID}, {"prepend", "text", "-u", exampleUUID}, {"replace", "text", "-u", exampleUUID}, {"edit", exampleUUID}, {"select"}, {"flag", exampleUUID}, {"unflag", exampleUUID}, {"archive", exampleUUID}, {"inbox", exampleUUID}, {"trash", exampleUUID}, {"open", exampleUUID}, {"tag", exampleUUID, "-t", "x"}, {"remove-tags", exampleUUID, "-t", "x"}, {"workspace", "--open", "Work"}, {"workspace", "--rename", "Work", "--to", "New Work"}, {"run", "Copy", "-u", exampleUUID}, {"sync"}, {"profile", "save", "work", "--input", `{"timeout":"5s"}`}, {"feedback", "synthetic feedback"}, {"upgrade"},
	}
	for _, args := range cases {
		args = append(args, "--dry-run", "--app", app, "--state-dir", t.TempDir())
		d := decodedData(t, invokeCLI(t, args, nil, env...))
		if d["dry_run"] != true {
			t.Fatal(args, d)
		}
		if args[0] == "upgrade" || args[0] == "profile" || args[0] == "feedback" {
			if strings.Contains(fmt.Sprint(d["preconditions"]), "Drafts is running") {
				t.Fatal("local or network operation claims a Drafts dependency", d)
			}
		}
	}
	if data, _ := os.ReadFile(log); len(data) != 0 {
		t.Fatal("preview sent events")
	}
}
func TestOutputOptionsFailBeforeMutation(t *testing.T) {
	app, env, log := mockAppCLI(t)
	cases := [][]string{{"create", "text", "--format", "raw"}, {"create", "text", "--fields", "uuid"}, {"feedback", "text", "--id-only"}, {"profile", "save", "work", "--input", `{"timeout":"5s"}`, "--id-only"}}
	for _, args := range cases {
		args = append(args, "--app", app)
		r := invokeCLI(t, args, nil, env...)
		if errorCode(t, r) != "INVALID_INPUT" {
			t.Fatal(r)
		}
	}
	if data, _ := os.ReadFile(log); len(data) != 0 {
		t.Fatal("invalid output sent events")
	}
}
func TestEditorPreservesContentAndKeepsStdoutClean(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "editor with spaces")
	os.WriteFile(exe, []byte("#!/bin/sh\n[ \"$1\" = --wait ] || exit 2\nprintf 'synthetic editor message\\n'\nprintf 'changed\\n\\n' > \"$2\"\n"), 0700)
	t.Setenv("EDITOR", "'"+exe+"' --wait")
	old := os.Stdout
	reader, writer, _ := os.Pipe()
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = old; reader.Close(); writer.Close() })
	content, e := editor("before\n")
	writer.Close()
	data, _ := io.ReadAll(reader)
	if e != nil || content != "changed\n\n" || len(data) != 0 {
		t.Fatalf("editor result %q stdout %q err %v", content, data, e)
	}
}

func TestCLIDetectedBetaEnablesOptionalFlags(t *testing.T) {
	app, env, log := mockAppCLI(t)
	invoke := func(args ...string) cliResult { return invokeCLI(t, append(args, "--app", app), nil, env...) }
	r := invoke("flag", exampleUUID, "--flag-type", "2", "--channel", "beta")
	if errorCode(t, r) != "APP_NOT_FOUND" {
		t.Fatal(r)
	}
	resources := filepath.Join(app, "Contents", "Resources", "Drafts.sdef")
	data, e := os.ReadFile(resources)
	if e != nil {
		t.Fatal(e)
	}
	dictionary := strings.Replace(string(data), "</class>", `<property name="flagged" code="DrFt" type="integer" access="rw"/></class>`, 1)
	os.WriteFile(resources, []byte(dictionary), 0600)
	receipts := filepath.Join(app, "Contents", "_MASReceipt")
	os.MkdirAll(receipts, 0700)
	os.WriteFile(filepath.Join(receipts, "sandboxReceipt"), []byte("synthetic receipt"), 0600)
	d := decodedData(t, invoke("schema", "--detected", "--channel", "beta"))
	if d["flag_type_available"] != true || d["app"].(map[string]interface{})["channel"] != "beta" {
		t.Fatal(d)
	}
	decodedData(t, invoke("flag", exampleUUID, "--flag-type", "2", "--channel", "beta"))
	decodedData(t, invoke("flag", "--input", `{"uuid":"`+exampleUUID+`","flagType":3}`, "--channel", "beta"))
	decodedData(t, invoke("create", "--input", `{"content":"synthetic","flagType":4}`, "--channel", "beta"))
	events, _ := os.ReadFile(log)
	for _, fragment := range []string{`set «property DrFt» of d to 2`, `set «property DrFt» of d to 3`, `«property DrFt»:4`} {
		if !strings.Contains(string(events), fragment) {
			t.Fatal("optional flag was not wired", fragment)
		}
	}
}
