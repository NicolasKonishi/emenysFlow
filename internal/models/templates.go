package models

import (
	"database/sql"
	"time"
)

type MenuModel struct {
	ID                 int64
	Slug               string
	Name               string
	Description        string
	MenuType           string
	ImageURL           sql.NullString
	Active             bool
	CurrentVersion     int
	SourceName         string
	SourceUpdatedMonth string
	SectionCount       int
	ItemCount          int
	ChoiceGroupCount   int
	ItemIDsCSV         string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          sql.NullTime
}

type MenuModelVersion struct {
	ID            int64
	MenuModelID   int64
	Version       int
	ChangeSummary string
	SnapshotJSON  string
	CreatedBy     sql.NullInt64
	CreatedAt     time.Time
}

type MenuSectionDefinition struct {
	ID          int64
	Slug        string
	Name        string
	SectionType string
	Active      bool
}

type MenuModelSection struct {
	ID                int64
	MenuModelID       int64
	SectionID         int64
	Name              string
	SectionType       string
	SortOrder         int
	Required          bool
	SelectionMin      int
	SelectionMax      sql.NullInt64
	AllowEventChanges bool
	Notes             string
	Items             []MenuModelItem
	ChoiceGroups      []MenuChoiceGroup
}

type MenuModelItem struct {
	ID              int64
	SectionID       int64
	MenuItemID      sql.NullInt64
	Slug            string
	SourceLabel     string
	NormalizedName  string
	Description     string
	SortOrder       int
	Included        bool
	Optional        bool
	Configurable    bool
	Notes           string
	Active          bool
	Selected        bool
	InChoiceGroup   bool
	Portions        sql.NullFloat64
	ContainerTypeID sql.NullInt64
}

type MenuChoiceGroup struct {
	ID              int64
	SectionID       int64
	Slug            string
	Name            string
	SelectionMin    int
	SelectionMax    sql.NullInt64
	Required        bool
	AllowExtraItems bool
	AllowCustomItem bool
	Configurable    bool
	Items           []MenuModelItem
}

type ServiceModel struct {
	ID                 int64
	Slug               string
	Name               string
	Description        string
	Category           string
	DurationMinutes    sql.NullInt64
	BillingUnit        string
	PriceCents         sql.NullInt64
	CostCents          sql.NullInt64
	CommissionCents    sql.NullInt64
	Supplier           sql.NullString
	ConfigurationJSON  string
	Active             bool
	CurrentVersion     int
	SourceName         string
	SourceUpdatedMonth string
	ComponentCount     int
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          sql.NullTime
}

type EventModelSelection struct {
	MenuModelID     int64
	SelectedItemIDs []int64
	ServiceModelIDs []int64
}

type EventMenuItemConfiguration struct {
	TemplateItemID  int64
	Portions        sql.NullFloat64
	ContainerTypeID sql.NullInt64
}

type ModelDifference struct {
	Kind     string
	ItemName string
	Detail   string
}

type ServiceModelVersion struct {
	ID             int64
	ServiceModelID int64
	Version        int
	ChangeSummary  string
	SnapshotJSON   string
	CreatedBy      sql.NullInt64
	CreatedAt      time.Time
}

type ServiceComponent struct {
	TemplateComponentID int64
	ID                  int64
	Slug                string
	Name                string
	Description         string
	Category            string
	SourceLabel         string
	NormalizedName      string
	SearchAliases       string
	Configurable        bool
	Active              bool
	Included            bool
	Optional            bool
	ConfigurationJSON   string
	Notes               string
}

type EventMenuSnapshot struct {
	ID                int64
	EventID           int64
	SourceMenuModelID sql.NullInt64
	SourceVersion     int
	SnapshotName      string
	SnapshotJSON      string
	AppliedBy         sql.NullInt64
	AppliedAt         time.Time
	UpdatedAt         time.Time
}

type EventServiceSnapshot struct {
	ID                   int64
	EventID              int64
	SourceServiceModelID sql.NullInt64
	SourceVersion        int
	SnapshotName         string
	SnapshotJSON         string
	DurationMinutes      sql.NullInt64
	Supplier             sql.NullString
	Status               string
	AppliedBy            sql.NullInt64
	AppliedAt            time.Time
	UpdatedAt            time.Time
}
