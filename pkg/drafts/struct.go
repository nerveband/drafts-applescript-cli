package drafts

const Separator = '|'

type Draft struct {
	AccessedAt        string   `json:"accessedAt,omitempty"`
	FlagType          *int     `json:"flagType,omitempty"`
	UUID              string   `json:"uuid"`
	Content           string   `json:"content"`
	Title             string   `json:"title"`
	Tags              []string `json:"tags"`
	IsFlagged         bool     `json:"isFlagged"`
	IsArchived        bool     `json:"isArchived"`
	IsTrashed         bool     `json:"isTrashed"`
	Folder            string   `json:"folder"`
	CreatedAt         string   `json:"createdAt"`
	ModifiedAt        string   `json:"modifiedAt"`
	CreatedLatitude   float64  `json:"createdLatitude"`
	CreatedLongitude  float64  `json:"createdLongitude"`
	ModifiedLatitude  float64  `json:"modifiedLatitude"`
	ModifiedLongitude float64  `json:"modifiedLongitude"`
	Permalink         string   `json:"permalink"`
}

// ---- Enums ------------------------------------------------------------------

type Folder int

const (
	FolderInbox Folder = iota
	FolderArchive
)

func (f Folder) String() string {
	return [...]string{"inbox", "archive"}[f]
}

type Filter int

const (
	FilterInbox Filter = iota
	FilterFlagged
	FilterArchive
	FilterTrash
	FilterAll
)

func (f Filter) String() string {
	return [...]string{"inbox", "flagged", "archive", "trash", "all"}[f]
}

type Sort int

const (
	SortCreated Sort = iota
	SortModified
	SortAccessed
)

func (s Sort) String() string {
	return [...]string{"created", "modified", "accessed"}[s]
}
