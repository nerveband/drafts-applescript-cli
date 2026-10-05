//go:build integration

package drafts

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/nerveband/drafts-applescript-cli/v4/internal/assert"
)

func TestCreateDefault(t *testing.T) {
	requireIntegration(t)
	text := rand()
	uuid, err := Create(text, CreateOptions{})
	requireNoError(t, err)
	defer func() {
		requireNoError(t, Trash(uuid))
	}()
	draft, err := Get(uuid)
	requireNoError(t, err)
	assert.Equal(t, uuid, draft.UUID)
	assert.Equal(t, text, draft.Content)
	assert.EqualSlice(t, []string{}, draft.Tags)
	assert.Equal(t, false, draft.IsFlagged)
	assert.Equal(t, false, draft.IsArchived)
	assert.Equal(t, false, draft.IsTrashed)
}

func TestCreateFlagged(t *testing.T) {
	requireIntegration(t)
	text := rand()
	uuid, err := Create(text, CreateOptions{Flagged: true})
	requireNoError(t, err)
	defer func() {
		requireNoError(t, Trash(uuid))
	}()
	draft, err := Get(uuid)
	requireNoError(t, err)
	assert.Equal(t, uuid, draft.UUID)
	assert.Equal(t, text, draft.Content)
	assert.EqualSlice(t, []string{}, draft.Tags)
	assert.Equal(t, true, draft.IsFlagged)
}

func TestCreateArchived(t *testing.T) {
	requireIntegration(t)
	text := rand()
	uuid, err := Create(text, CreateOptions{Folder: FolderArchive})
	requireNoError(t, err)
	defer func() {
		requireNoError(t, Trash(uuid))
	}()
	draft, err := Get(uuid)
	requireNoError(t, err)
	assert.Equal(t, uuid, draft.UUID)
	assert.Equal(t, text, draft.Content)
	assert.EqualSlice(t, []string{}, draft.Tags)
	assert.Equal(t, true, draft.IsArchived)
}

func TestCreateTags(t *testing.T) {
	requireIntegration(t)
	text := rand()
	tag := rand()
	uuid, err := Create(text, CreateOptions{Tags: []string{tag}})
	requireNoError(t, err)
	defer func() {
		requireNoError(t, Trash(uuid))
	}()
	draft, err := Get(uuid)
	requireNoError(t, err)
	assert.Equal(t, uuid, draft.UUID)
	assert.Equal(t, text, draft.Content)
	assert.EqualSlice(t, []string{tag}, draft.Tags)
}

func TestPrepend(t *testing.T) {
	requireIntegration(t)
	text := rand()
	prefix := rand()
	uuid, err := Create(text, CreateOptions{})
	requireNoError(t, err)
	defer func() {
		requireNoError(t, Trash(uuid))
	}()
	requireNoError(t, Prepend(uuid, prefix, ModifyOptions{}))
	draft, err := Get(uuid)
	requireNoError(t, err)
	content := draft.Content
	assert.Equal(t, prefix+"\n"+text, content)
}

func TestAppend(t *testing.T) {
	requireIntegration(t)
	text := rand()
	suffix := rand()
	uuid, err := Create(text, CreateOptions{})
	requireNoError(t, err)
	defer func() {
		requireNoError(t, Trash(uuid))

	}()
	requireNoError(t, Append(uuid, suffix, ModifyOptions{}))
	draft, err := Get(uuid)
	requireNoError(t, err)
	content := draft.Content
	assert.Equal(t, text+"\n"+suffix, content)
}

func TestReplace(t *testing.T) {
	requireIntegration(t)
	text := rand()
	replacement := rand()
	uuid, err := Create(text, CreateOptions{})
	requireNoError(t, err)
	defer func() {
		requireNoError(t, Trash(uuid))
	}()
	requireNoError(t, Replace(uuid, replacement))
	draft, err := Get(uuid)
	requireNoError(t, err)
	content := draft.Content
	assert.Equal(t, replacement, content)
}

func TestTrash(t *testing.T) {
	requireIntegration(t)
	text := rand()
	uuid, err := Create(text, CreateOptions{})
	requireNoError(t, err)
	requireNoError(t, Trash(uuid))
	draft, err := Get(uuid)
	requireNoError(t, err)
	assert.Equal(t, uuid, draft.UUID)
	assert.Equal(t, text, draft.Content)
	assert.EqualSlice(t, []string{}, draft.Tags)
	assert.Equal(t, true, draft.IsTrashed)
}

func TestArchive(t *testing.T) {
	requireIntegration(t)
	text := rand()
	uuid, err := Create(text, CreateOptions{})
	requireNoError(t, err)
	defer func() { requireNoError(t, Trash(uuid)) }()
	requireNoError(t, Archive(uuid))
	draft, err := Get(uuid)
	requireNoError(t, err)
	assert.Equal(t, uuid, draft.UUID)
	assert.Equal(t, text, draft.Content)
	assert.EqualSlice(t, []string{}, draft.Tags)
	assert.Equal(t, true, draft.IsArchived)
}

func TestQuery(t *testing.T) {
	requireIntegration(t)
	a, err := Create("A", CreateOptions{Tags: []string{"test", "a"}})
	requireNoError(t, err)
	b, err := Create("B", CreateOptions{Tags: []string{"test", "b"}, Flagged: true})
	requireNoError(t, err)
	c, err := Create("C", CreateOptions{Tags: []string{"test", "c"}})
	requireNoError(t, err)

	defer func() {
		for _, uuid := range []string{a, b, c} {
			requireNoError(t, Trash(uuid))
		}
	}()

	results, err := Query("", FilterInbox, QueryOptions{Tags: []string{"test"}})
	requireNoError(t, err)
	uuids := getUUIDs(results)
	assertSameUUIDs(t, []string{a, b, c}, uuids)

	results, err = Query("", FilterInbox, QueryOptions{Tags: []string{"test", "a"}})
	requireNoError(t, err)
	uuids = getUUIDs(results)
	assert.EqualSlice(t, []string{a}, uuids)

	results, err = Query("", FilterInbox, QueryOptions{
		Tags:     []string{"test"},
		OmitTags: []string{"a"},
	})
	requireNoError(t, err)
	uuids = getUUIDs(results)
	assertSameUUIDs(t, []string{b, c}, uuids)
}

func TestSelect(t *testing.T) {
	requireIntegration(t)
	a, err := Create("a", CreateOptions{})
	requireNoError(t, err)
	b, err := Create("b", CreateOptions{})
	requireNoError(t, err)
	defer func() {
		requireNoError(t, Trash(a))
		requireNoError(t, Trash(b))
	}()
	requireNoError(t, Select(b))
	activeUUID, err := Active()
	requireNoError(t, err)
	activeDraft, err := Get(activeUUID)
	requireNoError(t, err)
	requireNoError(t, Select(a))
	activeUUID, err = Active()
	requireNoError(t, err)
	selectedDraft, err := Get(activeUUID)
	requireNoError(t, err)
	b_ := activeDraft.Content
	a_ := selectedDraft.Content
	assert.Equal(t, "a", a_)
	assert.Equal(t, "b", b_)
}

func TestGetSpecialChars(t *testing.T) {
	requireIntegration(t)
	// https://en.wikipedia.org/wiki/URL_encoding#Percent-encoding_reserved_characters
	chars := []string{"␣", "!", "\"", "#", "$", "%", "&", "'", "(", ")", "*", "+", ",", "/", ":", ";", "=", "?", "@", "[", "]"}
	for _, c := range chars {
		uuid, err := Create(c, CreateOptions{})
		requireNoError(t, err)
		defer func() {
			requireNoError(t, Trash(uuid))
		}()
		draft, err := Get(uuid)
		requireNoError(t, err)
		content := draft.Content
		assert.Equal(t, c, content)
	}
}

func TestTag(t *testing.T) {
	requireIntegration(t)
	text := rand()
	tag := rand()
	uuid, err := Create(text, CreateOptions{})
	requireNoError(t, err)
	defer func() {
		requireNoError(t, Trash(uuid))
	}()
	requireNoError(t, Tag(uuid, tag))
	draft, err := Get(uuid)
	requireNoError(t, err)
	assert.Equal(t, uuid, draft.UUID)
	assert.Equal(t, text, draft.Content)
	assert.EqualSlice(t, []string{tag}, draft.Tags)
}

func TestGetReturnsLocationFields(t *testing.T) {
	requireIntegration(t)
	text := rand()
	uuid, err := Create(text, CreateOptions{})
	requireNoError(t, err)
	defer func() {
		requireNoError(t, Trash(uuid))
	}()
	draft, err := Get(uuid)
	requireNoError(t, err)
	// Location fields are float64; verify they parsed successfully.
	// Values may be zero or actual coordinates depending on device location services.
	if draft.CreatedLatitude < -90 || draft.CreatedLatitude > 90 {
		t.Errorf("CreatedLatitude out of range: %f", draft.CreatedLatitude)
	}
	if draft.CreatedLongitude < -180 || draft.CreatedLongitude > 180 {
		t.Errorf("CreatedLongitude out of range: %f", draft.CreatedLongitude)
	}
	if draft.ModifiedLatitude < -90 || draft.ModifiedLatitude > 90 {
		t.Errorf("ModifiedLatitude out of range: %f", draft.ModifiedLatitude)
	}
	if draft.ModifiedLongitude < -180 || draft.ModifiedLongitude > 180 {
		t.Errorf("ModifiedLongitude out of range: %f", draft.ModifiedLongitude)
	}
}

func TestSetFlagged(t *testing.T) {
	requireIntegration(t)
	text := rand()
	uuid, err := Create(text, CreateOptions{})
	requireNoError(t, err)
	defer func() {
		requireNoError(t, Trash(uuid))
	}()

	// Verify starts unflagged
	draft, err := Get(uuid)
	requireNoError(t, err)
	assert.Equal(t, false, draft.IsFlagged)

	// Flag it
	requireNoError(t, SetFlagged(uuid, true))
	draft, err = Get(uuid)
	requireNoError(t, err)
	assert.Equal(t, true, draft.IsFlagged)

	// Unflag it
	requireNoError(t, SetFlagged(uuid, false))
	draft, err = Get(uuid)
	requireNoError(t, err)
	assert.Equal(t, false, draft.IsFlagged)
}

func TestQueryContentSearch(t *testing.T) {
	requireIntegration(t)
	tag := rand()
	needle := "UNIQUE_SEARCH_" + rand()

	a, err := Create("has the needle "+needle+" inside", CreateOptions{Tags: []string{tag}})
	requireNoError(t, err)
	b, err := Create("no match here", CreateOptions{Tags: []string{tag}})
	requireNoError(t, err)
	defer func() {
		requireNoError(t, Trash(a))
		requireNoError(t, Trash(b))
	}()

	// Search with content filter, should only return draft "a"
	results, err := Query(needle, FilterInbox, QueryOptions{Tags: []string{tag}})
	requireNoError(t, err)
	uuids := getUUIDs(results)
	assert.EqualSlice(t, []string{a}, uuids)
}

func TestQueryFlagged(t *testing.T) {
	requireIntegration(t)
	tag := rand()

	a, err := Create("flagged draft", CreateOptions{Tags: []string{tag}, Flagged: true})
	requireNoError(t, err)
	b, err := Create("unflagged draft", CreateOptions{Tags: []string{tag}})
	requireNoError(t, err)
	defer func() {
		requireNoError(t, Trash(a))
		requireNoError(t, Trash(b))
	}()

	// FilterFlagged should only return the flagged draft
	results, err := Query("", FilterFlagged, QueryOptions{Tags: []string{tag}})
	requireNoError(t, err)
	uuids := getUUIDs(results)
	assert.EqualSlice(t, []string{a}, uuids)
}

// ---- Workspace tests --------------------------------------------------------

func TestQueryWorkspace(t *testing.T) {
	requireIntegration(t)
	results, err := QueryWorkspace("", "", FilterAll, QueryOptions{})
	requireNoError(t, err)
	// Empty workspace name should return empty slice, not nil
	if results == nil {
		t.Errorf("expected non-nil result from QueryWorkspace with empty name")
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty workspace name, got %d", len(results))
	}
}

func TestCurrentWorkspace(t *testing.T) {
	requireIntegration(t)
	ws, err := CurrentWorkspace()
	requireNoError(t, err)
	// Should return some string (Drafts always has a workspace)
	if ws == "" {
		t.Errorf("expected non-empty current workspace name")
	}
}

func TestWorkspaces(t *testing.T) {
	requireIntegration(t)
	workspaces, err := Workspaces()
	requireNoError(t, err)
	if workspaces == nil {
		t.Errorf("expected non-nil workspaces list")
	}
	if len(workspaces) == 0 {
		t.Errorf("expected at least one workspace")
	}
}

// ---- Helpers ----------------------------------------------------------------

// Return a random string.
func rand() string {
	return fmt.Sprint(time.Now().UnixNano())
}

// Extract UUIDs from a slice of Drafts
func getUUIDs(ds []Draft) []string {
	uuids := make([]string, len(ds))
	for i := range ds {
		uuids[i] = ds[i].UUID
	}
	return uuids
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func assertSameUUIDs(t *testing.T, want, got []string) {
	t.Helper()
	wantCopy := append([]string(nil), want...)
	gotCopy := append([]string(nil), got...)
	sort.Strings(wantCopy)
	sort.Strings(gotCopy)
	assert.EqualSlice(t, wantCopy, gotCopy)
}

func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("DRAFTS_INTEGRATION") != "1" || os.Getenv("DRAFTS_DISPOSABLE_LIBRARY") != "1" || !filepath.IsAbs(os.Getenv("DRAFTS_TEST_APP")) {
		t.Skip("live tests require explicit integration/disposable-library gates and DRAFTS_TEST_APP")
	}
	app, err := InspectApp(os.Getenv("DRAFTS_TEST_APP"))
	if err != nil {
		t.Fatal(err)
	}
	Configure(app, 30*time.Second)
}
