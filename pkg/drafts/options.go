package drafts

type CreateOptions struct {
	Tags    []string
	Folder  Folder
	Flagged bool
	Action  string
}

type QueryOptions struct {
	Tags             []string
	OmitTags         []string
	Sort             Sort
	SortDescending   bool
	SortFlaggedToTop bool
}

type ModifyOptions struct {
	Tags   []string
	Action string
}
