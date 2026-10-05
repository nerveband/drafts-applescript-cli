package main

import "github.com/nerveband/drafts-applescript-cli/v4/pkg/drafts"

// Default diagnostics read installed app metadata only. Resource and count reads are opt-in.
func runInfo(p *InfoCmd) interface{} {
	configureApp()
	result := map[string]interface{}{"cli": runVersion(), "app": configuredApp, "automation_tested": false}
	if p.TestPermissions {
		_, e := drafts.Count("", drafts.FilterInbox, drafts.QueryOptions{}, "")
		handleDraftsError(e)
		result["automation_tested"] = true
		result["probe"] = "read-only count"
	}
	if p.Counts {
		counts := map[string]int{}
		for _, folder := range []string{"inbox", "archive", "trash", "all"} {
			filter, _ := parseFilter(folder)
			count, e := drafts.Count("", filter, drafts.QueryOptions{}, "")
			handleDraftsError(e)
			counts[folder] = count
		}
		result["counts"] = counts
	}
	if p.Verbose {
		for _, kind := range []string{"workspace", "action", "tag"} {
			if !configuredApp.Capabilities[kind] {
				result[kind+"s"] = map[string]interface{}{"available": false}
				continue
			}
			r, e := drafts.Resources(kind)
			handleDraftsError(e)
			result[kind+"s"] = r
		}
	}
	return result
}
