package drafts

type CreateOptions struct {
	Tags     []string
	Folder   Folder
	Flagged  bool
	Action   string
	FlagType *int
}

type QueryOptions struct {
	Tags             []string
	OmitTags         []string
	Sort             Sort
	SortDescending   bool
	SortFlaggedToTop bool
	Limit            int
	Full             bool
	FlagType         *int
	CreatedAfter     string
	CreatedBefore    string
	ModifiedAfter    string
	ModifiedBefore   string
}

type ModifyOptions struct {
	Separator *string
	Tags      []string
	Action    string
}
