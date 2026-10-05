package drafts

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// AppInfo describes the installed application, never its user library.
type AppInfo struct {
	Path         string          `json:"path"`
	BundleID     string          `json:"bundle_id"`
	Version      string          `json:"version"`
	Build        string          `json:"build"`
	Channel      string          `json:"channel"`
	Capabilities map[string]bool `json:"capabilities"`
	Warnings     []string        `json:"warnings,omitempty"`
}

type dictionary struct {
	Suites []suite `xml:"suite"`
}
type suite struct {
	Classes  []class `xml:"class"`
	Commands []struct {
		Name       string `xml:"name,attr"`
		Parameters []struct {
			Name string `xml:"name,attr"`
		} `xml:"parameter"`
		Direct struct {
			Types []struct {
				Type string `xml:"type,attr"`
			} `xml:"type"`
		} `xml:"direct-parameter"`
	} `xml:"command"`
}
type class struct {
	Name       string `xml:"name,attr"`
	Properties []struct {
		Name   string `xml:"name,attr"`
		Code   string `xml:"code,attr"`
		Type   string `xml:"type,attr"`
		Access string `xml:"access,attr"`
	} `xml:"property"`
}

// InspectApp reads only application metadata and the scripting dictionary.
func InspectApp(path string) (AppInfo, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return AppInfo{}, err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return AppInfo{}, &Error{Code: "APP_NOT_FOUND", Message: "selected app path is unavailable", Hint: "Use an existing absolute Drafts app path", Cause: err, RetrySafe: true}
	}
	path = resolved
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	data, err := exec.CommandContext(ctx, "plutil", "-convert", "json", "-o", "-", filepath.Join(path, "Contents", "Info.plist")).Output()
	if err != nil {
		return AppInfo{}, fmt.Errorf("cannot read application metadata: %w", err)
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(data, &meta); err != nil {
		return AppInfo{}, err
	}
	get := func(k string) string { v, _ := meta[k].(string); return v }
	app := AppInfo{Path: path, BundleID: get("CFBundleIdentifier"), Version: get("CFBundleShortVersionString"), Build: get("CFBundleVersion"), Channel: "unknown"}
	if !strings.HasPrefix(app.BundleID, "com.agiletortoise.Drafts") {
		return AppInfo{}, fmt.Errorf("selected bundle is not Drafts")
	}
	label := strings.ToLower(filepath.Base(path) + " " + app.BundleID + " " + get("CFBundleName") + " " + app.Version)
	if strings.Contains(label, "beta") || strings.Contains(label, "testflight") {
		app.Channel = "beta"
	}
	if _, err := os.Stat(filepath.Join(path, "Contents", "_MASReceipt", "sandboxReceipt")); err == nil {
		app.Channel = "beta"
	}
	if app.Channel == "unknown" {
		if _, err := os.Stat(filepath.Join(path, "Contents", "_MASReceipt", "receipt")); err == nil {
			app.Channel = "stable"
		}
	}
	if b, ok := meta["Beta"].(bool); ok && b {
		app.Channel = "beta"
	}
	dictPath := get("OSAScriptingDefinition")
	if dictPath == "" {
		dictPath = "Drafts.sdef"
	}
	data, err = os.ReadFile(filepath.Join(path, "Contents", "Resources", filepath.Base(dictPath)))
	if err != nil {
		return AppInfo{}, fmt.Errorf("cannot read scripting dictionary: %w", err)
	}
	app.Capabilities, app.Warnings, err = InspectDictionary(data)
	return app, err
}

// InspectDictionary uses Apple event codes, including builds with duplicate names.
func InspectDictionary(data []byte) (map[string]bool, []string, error) {
	var d dictionary
	if err := xml.Unmarshal(data, &d); err != nil {
		return nil, nil, err
	}
	cap := map[string]bool{}
	warnings := []string{}
	for _, s := range d.Suites {
		for _, c := range s.Classes {
			cap[c.Name] = true
			names := map[string]bool{}
			for _, p := range c.Properties {
				cap[c.Name+"."+p.Name] = true
				if c.Name == "workspace" && p.Name == "name" && p.Access == "rw" {
					cap["workspace.rename"] = true
				}
				if names[p.Name] {
					warnings = append(warnings, "duplicate "+c.Name+" property: "+p.Name+"; Apple event codes are used")
				}
				names[p.Name] = true
				if c.Name == "draft" && p.Code == "DrFl" && p.Type == "boolean" {
					cap["boolean_flag"] = true
				}
				if c.Name == "draft" && p.Code == "DrFt" && p.Type == "integer" && p.Access == "rw" {
					cap["flag_type"] = true
				}
			}
		}
		for _, c := range s.Commands {
			cap["command."+c.Name] = true
			if c.Name == "open" {
				for _, t := range c.Direct.Types {
					cap["open."+t.Type] = true
				}
			}
		}
	}
	cap["core"] = cap["draft.content"] && cap["draft.tag list"] && cap["draft.creation date"] && cap["draft.modification date"] && cap["boolean_flag"]
	return cap, warnings, nil
}

func DiscoverApps() ([]AppInfo, error) {
	if runtime.GOOS != "darwin" {
		return nil, &Error{Code: "UNSUPPORTED_PLATFORM", Message: "Drafts automation requires macOS"}
	}
	roots := []string{"/Applications", "/System/Applications"}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, "Applications"))
	}
	apps := []AppInfo{}
	seen := map[string]bool{}
	for _, root := range roots {
		paths, _ := filepath.Glob(filepath.Join(root, "*Drafts*.app"))
		for _, path := range paths {
			app, err := InspectApp(path)
			if err == nil && !seen[app.Path] {
				apps = append(apps, app)
				seen[app.Path] = true
			}
		}
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Path < apps[j].Path })
	return apps, nil
}

func SelectApp(apps []AppInfo, selector, channel string) (AppInfo, error) {
	if channel != "auto" && channel != "stable" && channel != "beta" {
		return AppInfo{}, &Error{Code: "INVALID_INPUT", Message: "channel must be auto, stable, or beta"}
	}
	if selector != "" {
		matches := []AppInfo{}
		for _, app := range apps {
			if app.Path == selector || app.BundleID == selector {
				matches = append(matches, app)
			}
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
		if len(matches) > 1 {
			return AppInfo{}, &Error{Code: "AMBIGUOUS_APP", Message: "bundle identifier matches multiple installations", Hint: "Select an absolute --app path"}
		}
		return AppInfo{}, &Error{Code: "APP_NOT_FOUND", Message: "selected Drafts application was not found"}
	}
	eligible := []AppInfo{}
	for _, app := range apps {
		if channel == "auto" || (channel == "beta" && app.Channel == "beta") || (channel == "stable" && app.Channel != "beta") {
			eligible = append(eligible, app)
		}
	}
	if len(eligible) == 0 {
		return AppInfo{}, &Error{Code: "APP_NOT_FOUND", Message: "no Drafts installation matches the selected channel", Hint: "Use apps to inspect installations, or select an absolute --app path"}
	}
	if len(eligible) == 1 {
		return eligible[0], nil
	}
	if channel == "auto" {
		nonBeta := []AppInfo{}
		for _, app := range eligible {
			if app.Channel != "beta" {
				nonBeta = append(nonBeta, app)
			}
		}
		if len(nonBeta) == 1 {
			return nonBeta[0], nil
		}
	}
	return AppInfo{}, &Error{Code: "AMBIGUOUS_APP", Message: "multiple Drafts installations match", Hint: "Select the exact application with --app /Applications/Drafts.app"}
}
