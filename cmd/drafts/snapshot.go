package main

import (
	"encoding/json"
	"fmt"
	"github.com/nerveband/drafts-applescript-cli/v4/pkg/drafts"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SnapshotInfo struct {
	FetchedAt string              `json:"fetched_at"`
	AppPath   string              `json:"app_path"`
	Filter    string              `json:"filter"`
	Limit     int                 `json:"limit"`
	Full      bool                `json:"full"`
	Workspace string              `json:"workspace,omitempty"`
	Count     int                 `json:"count"`
	Coverage  string              `json:"coverage"`
	Search    string              `json:"search,omitempty"`
	Options   drafts.QueryOptions `json:"query_options"`
}
type snapshot struct {
	Info   SnapshotInfo   `json:"info"`
	Drafts []drafts.Draft `json:"drafts"`
}

func snapshotPath() string { return filepath.Join(statePath(), "snapshot.json") }
func runSync(p *ListCmd) interface{} {
	filter, e := parseFilter(p.Filter)
	handleDraftsError(e)
	opt, e := queryOptions(p)
	handleDraftsError(e)
	if p.Count {
		outputError("INVALID_INPUT", "sync cannot use --count", "")
	}
	if plan, ok := mutationPlan("sync", map[string]interface{}{"filter": p.Filter, "limit": p.Limit, "full": p.Full, "search": p.Search, "workspace": p.Workspace}, false); ok {
		return plan
	}
	configureApp()
	var ds []drafts.Draft
	if p.Workspace != "" {
		ds, e = drafts.QueryWorkspace(p.Workspace, p.Search, filter, opt)
	} else {
		ds, e = drafts.Query(p.Search, filter, opt)
	}
	handleDraftsError(e)
	s := snapshot{Info: SnapshotInfo{FetchedAt: time.Now().UTC().Format(time.RFC3339), AppPath: configuredApp.Path, Filter: p.Filter, Limit: p.Limit, Full: p.Full, Workspace: p.Workspace, Search: p.Search, Options: opt, Count: len(ds), Coverage: "bounded snapshot, counts and search reflect cached rows only"}, Drafts: ds}
	data, e := json.Marshal(s)
	handleDraftsError(e)
	handleDraftsError(os.MkdirAll(statePath(), 0700))
	handleDraftsError(atomicWrite(snapshotPath(), data, true))
	return map[string]interface{}{"snapshot": s.Info, "path": snapshotPath(), "data_source": "live", "scope": "local"}
}
func loadSnapshot() (snapshot, error) {
	var s snapshot
	data, e := os.ReadFile(snapshotPath())
	if e != nil {
		return s, &drafts.Error{Code: "SNAPSHOT_NOT_FOUND", Message: "local snapshot is unavailable", Hint: "Run sync with explicit filters and --full if content search is needed", Cause: e, RetrySafe: true}
	}
	if e = json.Unmarshal(data, &s); e != nil {
		return s, &drafts.Error{Code: "INVALID_RESPONSE", Message: "local snapshot is invalid", Cause: e}
	}
	if _, err := time.Parse(time.RFC3339, s.Info.FetchedAt); err != nil || s.Info.AppPath == "" || s.Info.Count != len(s.Drafts) {
		return s, &drafts.Error{Code: "INVALID_RESPONSE", Message: "snapshot provenance is invalid"}
	}
	for _, d := range s.Drafts {
		if err := drafts.ValidateDraft(d); err != nil {
			return s, err
		}
	}
	return s, nil
}
func localGet(uuid string) (drafts.Draft, error) {
	if e := drafts.ValidateUUID(uuid); e != nil {
		return drafts.Draft{}, e
	}
	s, e := loadSnapshot()
	if e != nil {
		return drafts.Draft{}, e
	}
	if !s.Info.Full {
		return drafts.Draft{}, &drafts.Error{Code: "UNSUPPORTED_CAPABILITY", Message: "snapshot has summaries only", Hint: "Run sync --full explicitly, or use --data-source live"}
	}
	for _, d := range s.Drafts {
		if strings.EqualFold(d.UUID, uuid) {
			return d, nil
		}
	}
	return drafts.Draft{}, fmt.Errorf("%w: %s", drafts.ErrDraftNotFound, uuid)
}
func localList(p *ListCmd, filter drafts.Filter, opt drafts.QueryOptions) interface{} {
	s, e := loadSnapshot()
	handleDraftsError(e)
	if (p.Full || p.Search != "") && !s.Info.Full {
		outputError("UNSUPPORTED_CAPABILITY", "snapshot has summaries only", "Run sync --full explicitly before local content search")
	}
	if p.Workspace != "" && p.Workspace != s.Info.Workspace {
		outputError("UNSUPPORTED_CAPABILITY", "snapshot is not scoped to the requested workspace", "Sync that workspace first")
	}
	base := s.Info.Options
	compatible := s.Info.Filter == "all" || s.Info.Filter == p.Filter
	if s.Info.Workspace != "" && p.Workspace != s.Info.Workspace {
		compatible = false
	}
	if s.Info.Search != "" && !strings.EqualFold(s.Info.Search, p.Search) {
		compatible = false
	}
	for _, restriction := range []struct{ base, requested string }{{base.CreatedAfter, opt.CreatedAfter}, {base.CreatedBefore, opt.CreatedBefore}, {base.ModifiedAfter, opt.ModifiedAfter}, {base.ModifiedBefore, opt.ModifiedBefore}} {
		if restriction.base != "" && restriction.base != restriction.requested {
			compatible = false
		}
	}
	if base.FlagType != nil && (opt.FlagType == nil || *base.FlagType != *opt.FlagType) {
		compatible = false
	}
	containsAll := func(required, requested []string) bool {
		for _, tag := range required {
			found := false
			for _, candidate := range requested {
				if strings.EqualFold(tag, candidate) {
					found = true
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	if !containsAll(base.Tags, opt.Tags) || !containsAll(base.OmitTags, opt.OmitTags) {
		compatible = false
	}
	if !compatible {
		outputError("UNSUPPORTED_CAPABILITY", "query exceeds the snapshot's recorded scope", "Sync the requested scope first, or use live data")
	}
	items := []drafts.Draft{}
	for _, d := range s.Drafts {
		match := true
		switch filter {
		case drafts.FilterInbox:
			match = d.Folder == "inbox"
		case drafts.FilterArchive:
			match = d.Folder == "archive"
		case drafts.FilterTrash:
			match = d.Folder == "trash"
		case drafts.FilterFlagged:
			match = d.Folder == "inbox" && d.IsFlagged
		}
		if !match {
			continue
		}
		if p.Search != "" && !strings.Contains(strings.ToLower(d.Content), strings.ToLower(p.Search)) {
			continue
		}
		if opt.FlagType != nil && (!d.IsFlagged || d.FlagType == nil || *d.FlagType != *opt.FlagType) {
			continue
		}
		tags := map[string]bool{}
		for _, tag := range d.Tags {
			tags[strings.ToLower(tag)] = true
		}
		for _, tag := range opt.Tags {
			if !tags[strings.ToLower(tag)] {
				match = false
			}
		}
		for _, tag := range opt.OmitTags {
			if tags[strings.ToLower(tag)] {
				match = false
			}
		}
		for _, bound := range []struct{ value, after, before string }{{d.CreatedAt, opt.CreatedAfter, opt.CreatedBefore}, {d.ModifiedAt, opt.ModifiedAfter, opt.ModifiedBefore}} {
			v, err := time.Parse(time.RFC3339Nano, bound.value)
			if err != nil {
				outputError("INVALID_RESPONSE", "snapshot has invalid dates", "")
			}
			if bound.after != "" {
				a, _ := time.Parse(time.RFC3339Nano, bound.after)
				if !v.After(a) {
					match = false
				}
			}
			if bound.before != "" {
				b, _ := time.Parse(time.RFC3339Nano, bound.before)
				if !v.Before(b) {
					match = false
				}
			}
		}
		if match {
			items = append(items, d)
		}
	}
	if opt.Sort == drafts.SortAccessed {
		for _, d := range items {
			if d.AccessedAt == "" {
				outputError("UNSUPPORTED_CAPABILITY", "snapshot does not have access dates", "")
			}
		}
	}
	key := func(d drafts.Draft) time.Time {
		v := d.CreatedAt
		if opt.Sort == drafts.SortModified {
			v = d.ModifiedAt
		}
		if opt.Sort == drafts.SortAccessed {
			v = d.AccessedAt
		}
		parsed, _ := time.Parse(time.RFC3339Nano, v)
		return parsed
	}
	sort.SliceStable(items, func(i, j int) bool {
		if opt.SortFlaggedToTop && items[i].IsFlagged != items[j].IsFlagged {
			return items[i].IsFlagged
		}
		a, b := key(items[i]), key(items[j])
		if a.Equal(b) {
			return items[i].UUID < items[j].UUID
		}
		if opt.SortDescending {
			return a.After(b)
		}
		return a.Before(b)
	})
	if p.Count {
		return map[string]interface{}{"count": len(items), "data_source": "local", "snapshot": s.Info}
	}
	if p.Limit > 0 && len(items) > p.Limit {
		items = items[:p.Limit]
	}
	views := make([]DraftView, len(items))
	for i, d := range items {
		views[i] = toDraftView(d, p.Full)
	}
	return ListResult{DataSource: "local", Snapshot: &s.Info, Drafts: views, Count: len(views), Filter: p.Filter, Limit: p.Limit, Full: p.Full, Search: p.Search, Workspace: p.Workspace}
}
