package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID         int64
	Name       string
	Email      string
	Role       string
	AccessRole string
	RowVersion int
	Active     bool
	Password   string
}

type Event struct {
	ID                            int64
	TemplateID                    sql.NullInt64
	ClientName                    string
	Name                          string
	Venue                         string
	StartsAt                      time.Time
	EndsAt                        time.Time
	GuestCount                    int
	HasDecoration                 bool
	HasWelcomeDrinks              bool
	HasCoffeeTable                bool
	StartersNotes                 string
	MainCoursesNotes              string
	SidesNotes                    string
	BeveragesNotes                string
	CoffeeTableNotes              string
	CakeNotes                     string
	SweetsNotes                   string
	DessertsNotes                 string
	Notes                         string
	SafetyMarginPercent           float64
	WaiterOverride                sql.NullInt64
	CoordinatorOverride           sql.NullInt64
	LeaderOverride                sql.NullInt64
	CoLeaderOverride              sql.NullInt64
	AdditionalGuestMarginOverride sql.NullFloat64
	UsesGlassware                 bool
	KitchenCookID                 sql.NullInt64
	KitchenCookName               string
	Status                        string
	Active                        bool
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
	RowVersion                    int
	ChecklistProgress             float64
	SeparationProgress            float64
	LoadingProgress               float64
	MissingItems                  int
	PendingPurchases              int
	PendingRentals                int
}

type KitchenCook struct {
	ID     int64
	Slug   string
	Name   string
	Active bool
}

type KitchenCookBox struct {
	ID                int64
	KitchenCookID     int64
	KitchenCookName   string
	InventoryItemID   int64
	InventoryItemName string
	BoxType           string
	Active            bool
	ItemCount         int
	TotalQuantity     float64
	Items             []KitchenCookBoxItem
}

type KitchenCookBoxItem struct {
	ID              int64
	BoxID           int64
	InventoryItemID int64
	InventoryName   string
	CategoryName    string
	Unit            string
	Quantity        float64
	Notes           string
	Active          bool
}

type InventoryItem struct {
	ID                    int64
	Name                  string
	Description           string
	CategoryID            int64
	CategoryName          string
	Subcategory           string
	Unit                  string
	StockQuantity         float64
	MinimumStock          float64
	DamagedQuantity       float64
	ReservedQuantity      float64
	AvailableQuantity     float64
	LocationID            sql.NullInt64
	LocationName          string
	InternalCode          string
	Barcode               string
	PhotoURL              string
	ItemKind              string
	Ownership             string
	RequiresReturn        bool
	ReplacementValueCents int64
	Notes                 string
	Active                bool
}

type Category struct {
	ID        int64
	Name      string
	SortOrder int
}

type Location struct {
	ID   int64
	Name string
}

type RuleCondition struct {
	WelcomeDrinks            *bool   `json:"welcome_drinks,omitempty"`
	DadinhoTapioca           *bool   `json:"dadinho_tapioca,omitempty"`
	Decoration               *bool   `json:"decoration,omitempty"`
	DistributionGroup        string  `json:"distribution_group,omitempty"`
	Percentage               float64 `json:"percentage,omitempty"`
	UseEventSafetyMargin     bool    `json:"use_event_safety_margin,omitempty"`
	UseAdditionalGuestMargin bool    `json:"use_additional_guest_margin,omitempty"`
	UseCoffeeKitDivisor      bool    `json:"use_coffee_kit_divisor,omitempty"`
	HasMainBuffet            *bool   `json:"has_main_buffet,omitempty"`
	UsesGlassware            *bool   `json:"uses_glassware,omitempty"`
	CoffeeTable              *bool   `json:"coffee_table,omitempty"`
	RequiresDessertSpoon     *bool   `json:"requires_dessert_spoon,omitempty"`
	Formula                  string  `json:"formula,omitempty"`
}

type CalculationRule struct {
	ID                    int64
	RuleKey               string
	Name                  string
	Description           string
	CategoryID            int64
	CategoryName          string
	TriggerEvent          string
	CalculationType       string
	BaseValue             float64
	Divisor               float64
	Multiplier            float64
	MinimumQuantity       sql.NullFloat64
	MaximumQuantity       sql.NullFloat64
	SafetyPercent         float64
	ConditionJSON         string
	Condition             RuleCondition
	ResultInventoryItemID int64
	ResultItemName        string
	ResultUnit            string
	Priority              int
	Active                bool
}

type Checklist struct {
	ID          int64
	EventID     int64
	Version     int
	GeneratedAt time.Time
	UpdatedAt   time.Time
	Items       []ChecklistItem
	Progress    ChecklistProgress
}

type ChecklistItem struct {
	ID                        int64
	ChecklistID               int64
	InventoryItemID           sql.NullInt64
	CategoryID                int64
	CategoryName              string
	CategorySortOrder         int
	SourceRuleID              sql.NullInt64
	SourceKey                 string
	Name                      string
	Unit                      string
	CalculatedQuantity        float64
	RequiredQuantity          float64
	AvailableQuantity         float64
	ReservedElsewhereQuantity float64
	MissingQuantity           float64
	CalculationOrigin         string
	Notes                     string
	Status                    string
	ItemKind                  string
	LocationSnapshot          string
	ManualItem                bool
	ManualOverride            bool
	OverrideReason            string
	OverrideBy                sql.NullInt64
	OverrideAt                time.Time
	SeparatedQuantity         float64
	SeparatedBy               sql.NullInt64
	SeparatedByName           string
	SeparatedAt               time.Time
	LoadedQuantity            float64
	LoadedBy                  sql.NullInt64
	LoadedByName              string
	LoadedAt                  time.Time
	LoadingDecision           string
	LoadingMissingQuantity    float64
	ReturnedQuantity          float64
	DamagedQuantity           float64
	LostQuantity              float64
	Active                    bool
	RowVersion                int
	Shortage                  *ChecklistShortage
}

type ChecklistProgress struct {
	Total                int
	Completed            int
	Pending              int
	Missing              int
	Percentage           int
	SeparationPercentage int
	LoadingPercentage    int
}

type ChecklistGroup struct {
	Category string
	Items    []ChecklistItem
}

type Dashboard struct {
	UpcomingEvents      []Event
	EventsWithShortages int
	AwaitingSeparation  int
	AwaitingReturn      int
	BelowMinimumStock   int
	DamagedItems        int
	RentalsDue          int
	PendingPurchases    int
	PendingRentals      int
}

type ReservationWindow struct {
	EventID  int64
	StartsAt time.Time
	EndsAt   time.Time
	Quantity float64
}

type ReturnResult struct {
	NewStock          float64
	NewDamaged        float64
	Unaccounted       float64
	ReturnedAvailable float64
}

type MenuCategory struct {
	ID        int64
	Name      string
	Slug      string
	SortOrder int
}

type ContainerType struct {
	ID                  int64
	Name                string
	CapacityPortions    sql.NullFloat64
	Disposable          bool
	RequiresLid         bool
	IsDefault           bool
	TransportNotes      string
	InventoryItemID     sql.NullInt64
	InventoryItemName   string
	QuantityMode        string
	RequiredUtensilType string
	CustomUtensilName   string
	FixedQuantity       sql.NullFloat64
	Active              bool
}

type MenuTemplate struct {
	ID               int64
	Name             string
	Description      string
	HasDecoration    bool
	HasWelcomeDrinks bool
	HasCoffeeTable   bool
	ItemIDs          []int64
	ItemIDsCSV       string
	ItemCount        int
	Active           bool
}

type MenuItem struct {
	ID                         int64
	CategoryID                 int64
	CategoryName               string
	Name                       string
	Description                string
	ContainerTypeID            sql.NullInt64
	ContainerTypeName          string
	ContainerCapacity          sql.NullFloat64
	PanInventoryItemID         sql.NullInt64
	PanInventoryItemName       string
	TransportInventoryItemID   sql.NullInt64
	TransportInventoryItemName string
	ResultInventoryItemID      sql.NullInt64
	ResultInventoryItemName    string
	CalculationType            string
	CalculationGroup           string
	CalculationDivisor         float64
	CalculationMultiplier      float64
	CalculationWeight          float64
	TemplateOwnerID            sql.NullInt64
	SourceMenuItemID           sql.NullInt64
	Active                     bool
	Equipment                  []EquipmentLink
	Ingredients                []MenuItemIngredient
}

type MenuItemIngredient struct {
	ID                int64
	MenuItemID        int64
	InventoryItemID   int64
	InventoryItemName string
	Unit              string
	CalculationType   string
	Quantity          float64
	PeopleDivisor     float64
	Notes             string
	Active            bool
}

type EquipmentLink struct {
	EquipmentID     int64
	InventoryItemID int64
	Name            string
	Quantity        float64
	Required        bool
	Selected        bool
}

type EventMenuItem struct {
	ID               int64
	EventID          int64
	MenuItemID       int64
	MenuItemName     string
	CategoryName     string
	TemplateOwnerID  sql.NullInt64
	SourceMenuItemID sql.NullInt64
	Portions         int
	Selected         bool
}

type AutomaticRequirement struct {
	SourceKey       string
	InventoryItemID int64
	Quantity        float64
	Origin          string
}

type EventOperation struct {
	ID              int64
	EventID         int64
	Stage           string
	ResponsibleName string
	Vehicle         string
	Notes           string
	PhotoURL        string
	OccurredAt      time.Time
}

type ReturnInspection struct {
	ChecklistItemID     int64
	Name                string
	Unit                string
	LoadedQuantity      float64
	ReturnedQuantity    float64
	DamagedQuantity     float64
	LostQuantity        float64
	LaundryQuantity     float64
	MaintenanceQuantity float64
	Notes               string
}

type InventoryMovement struct {
	ID            int64
	MovementType  string
	Quantity      float64
	PreviousStock float64
	NewStock      float64
	Reason        string
	EventName     string
	CreatedAt     time.Time
}

type EventDecoration struct {
	DecorationID        int64
	Name                string
	UsageLocation       string
	Color               string
	Model               string
	Ownership           string
	RentalCompany       string
	PhotoURL            string
	Notes               string
	InventoryItemID     sql.NullInt64
	Unit                string
	StockQuantity       float64
	DamagedQuantity     float64
	ReservedQuantity    float64
	AvailableQuantity   float64
	AvailabilityTracked bool
	Selectable          bool
	Quantity            float64
	Selected            bool
	Active              bool
}
