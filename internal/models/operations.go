package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type OperationalSetting struct {
	Key         string
	Name        string
	Description string
	Value       float64
	Unit        string
	UpdatedAt   time.Time
}

type StaffSummary struct {
	Waiters      int
	Coordinators int
	Leaders      int
	CoLeaders    int
	Total        int
}

type Supplier struct {
	ID          int64
	Name        string
	ContactName string
	Phone       string
	Email       string
	Notes       string
	Active      bool
}

type ChecklistShortage struct {
	ID                    int64
	ChecklistItemID       int64
	EventID               int64
	MissingQuantity       float64
	Reason                string
	ResolutionType        string
	Status                string
	ResponsibleUserID     sql.NullInt64
	ResponsibleName       string
	DueAt                 time.Time
	SupplierID            sql.NullInt64
	SupplierName          string
	EstimatedCostCents    sql.NullInt64
	Notes                 string
	Automatic             bool
	ResolutionDestination string
	ResolvedBy            sql.NullInt64
	ResolvedAt            time.Time
	RowVersion            int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type RecalculationSummary struct {
	ID                  int64
	EventID             int64
	TriggerKey          string
	PreviousVersion     int
	NewVersion          int
	Added               int
	Removed             int
	QuantitiesUpdated   int
	Shortages           int
	ReservationsUpdated int
	CreatedAt           time.Time
}

type EventMenuSnapshotItem struct {
	ID                   int64
	EventMenuSectionID   int64
	SectionName          string
	SectionType          string
	SourceTemplateItemID sql.NullInt64
	SourceMenuItemID     sql.NullInt64
	DisplayName          string
	Description          string
	SortOrder            int
	Selected             bool
	CustomItem           bool
	Customized           bool
	WasRemoved           bool
	Portions             sql.NullFloat64
	ContainerTypeID      sql.NullInt64
	Notes                string
	RowVersion           int
	CanChooseContainer   bool
	Containers           []EventMenuContainer
	Equipment            []EventMenuEquipment
}

type EventMenuContainer struct {
	ID               int64
	SnapshotItemID   int64
	Purpose          string
	ContainerTypeID  sql.NullInt64
	InventoryItemID  sql.NullInt64
	Name             string
	Quantity         sql.NullFloat64
	CapacityPortions sql.NullFloat64
	RequiresLid      bool
	Notes            string
}

type EventMenuEquipment struct {
	ID              int64
	SnapshotItemID  int64
	InventoryItemID int64
	Name            string
	Quantity        float64
	Required        bool
	Notes           string
}

type EventCakeConfiguration struct {
	EventID               int64
	CakeCount             int
	RequiresRefrigeration bool
	Notes                 string
	RowVersion            int
}

type EventMenuSnapshotSection struct {
	ID                int64
	Name              string
	SectionType       string
	SortOrder         int
	AllowEventChanges bool
	Items             []EventMenuSnapshotItem
}

type DecorationProfile struct {
	ID              int64
	EventID         int64
	Style           string
	Description     string
	PrimaryColors   string
	Theme           string
	Notes           string
	ResponsibleName string
	Active          bool
	RowVersion      int
	Compositions    []DecorationComposition
	Photos          []ReferencePhoto
}

type DecorationComposition struct {
	ID               int64
	ProfileID        int64
	Name             string
	CompositionType  string
	Description      string
	AssemblyLocation string
	Notes            string
	SortOrder        int
	RowVersion       int
	Items            []DecorationCompositionItem
	Photos           []ReferencePhoto
}

type DecorationCompositionItem struct {
	ID                 int64
	CompositionID      int64
	DecorationID       sql.NullInt64
	InventoryItemID    sql.NullInt64
	Name               string
	Color              string
	Quantity           float64
	Origin             string
	SupplierID         sql.NullInt64
	SupplierName       string
	EstimatedCostCents sql.NullInt64
	PickupAt           time.Time
	ReturnAt           time.Time
	OrderReference     string
	RentalStatus       string
	Notes              string
	SortOrder          int
	RowVersion         int
	Photos             []ReferencePhoto
}

type ReferencePhoto struct {
	ID                int64
	ClientUploadID    string
	EventID           int64
	CompositionID     sql.NullInt64
	CompositionItemID sql.NullInt64
	StoragePath       string
	OriginalName      string
	MIMEType          string
	FileSize          int64
	Caption           string
	SortOrder         int
	Primary           bool
	CreatedAt         time.Time
}

type SyncOperationRequest struct {
	ClientOperationID string          `json:"client_operation_id"`
	DeviceID          string          `json:"device_id"`
	OperationType     string          `json:"operation_type"`
	EntityType        string          `json:"entity_type"`
	EntityID          int64           `json:"entity_id"`
	BaseVersion       int             `json:"base_version"`
	Payload           json.RawMessage `json:"payload"`
	LocalDate         string          `json:"local_date"`
}

type SyncOperationResult struct {
	ClientOperationID string `json:"client_operation_id"`
	Status            string `json:"status"`
	EntityID          int64  `json:"entity_id,omitempty"`
	Version           int    `json:"version,omitempty"`
	Error             string `json:"error,omitempty"`
	ServerSnapshot    any    `json:"server_snapshot,omitempty"`
}
