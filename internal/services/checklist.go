package services

import (
	"context"
	"database/sql"
	"fmt"

	"buffetflow/internal/models"
	"buffetflow/internal/repositories"
)

type ChecklistService struct {
	store *repositories.Store
}

func NewChecklistService(store *repositories.Store) *ChecklistService {
	return &ChecklistService{store: store}
}

func (s *ChecklistService) Generate(ctx context.Context, eventID int64) (models.Checklist, error) {
	return s.GenerateTracked(ctx, eventID, "automatic", 0)
}

func (s *ChecklistService) GenerateTracked(ctx context.Context, eventID int64, trigger string, userID int64) (models.Checklist, error) {
	before, _ := s.store.GetChecklistByEvent(ctx, eventID)
	after, err := s.generate(ctx, eventID)
	if err != nil {
		return after, err
	}
	if err := s.store.RecordEventRecalculation(ctx, eventID, trigger, userID, before, after); err != nil {
		return models.Checklist{}, fmt.Errorf("record checklist recalculation: %w", err)
	}
	return after, nil
}

func (s *ChecklistService) generate(ctx context.Context, eventID int64) (models.Checklist, error) {
	event, err := s.store.GetEvent(ctx, eventID)
	if err != nil {
		return models.Checklist{}, fmt.Errorf("load event: %w", err)
	}
	containedInventoryIDs, err := s.store.EventKitchenCookContainedInventoryIDs(ctx, event.ID)
	if err != nil {
		return models.Checklist{}, fmt.Errorf("load kitchen box contents: %w", err)
	}
	rules, err := s.store.ListRules(ctx, false)
	if err != nil {
		return models.Checklist{}, err
	}
	hasDadinhoTapioca, err := s.store.EventHasDadinhoTapioca(ctx, event.ID)
	if err != nil {
		return models.Checklist{}, fmt.Errorf("check dadinho de tapioca in menu: %w", err)
	}
	settings, err := s.store.OperationalSettings(ctx)
	if err != nil {
		return models.Checklist{}, fmt.Errorf("load operational settings: %w", err)
	}
	additionalMargin := settings["additional_staff_margin"].Value
	if event.AdditionalGuestMarginOverride.Valid {
		additionalMargin = event.AdditionalGuestMarginOverride.Float64
	}
	coffeeKitDivisor := settings["people_per_coffee_spoon_kit"].Value
	hasMainBuffet, err := s.store.EventHasMenuCategory(ctx, event.ID, "main_courses")
	if err != nil {
		return models.Checklist{}, fmt.Errorf("check main buffet: %w", err)
	}
	requiresDessertSpoon, err := s.store.EventRequiresDessertSpoon(ctx, event.ID)
	if err != nil {
		return models.Checklist{}, fmt.Errorf("check dessert utensil: %w", err)
	}
	input := CalculationInput{
		GuestCount:            event.GuestCount,
		HasWelcomeDrinks:      event.HasWelcomeDrinks,
		HasDadinhoTapioca:     hasDadinhoTapioca,
		HasDecoration:         event.HasDecoration,
		SafetyMarginPercent:   event.SafetyMarginPercent,
		AdditionalGuestMargin: additionalMargin,
		CoffeeKitDivisor:      coffeeKitDivisor,
		HasMainBuffet:         hasMainBuffet,
		UsesGlassware:         event.UsesGlassware,
		HasCoffeeTable:        event.HasCoffeeTable,
		RequiresDessertSpoon:  requiresDessertSpoon,
	}
	if event.WaiterOverride.Valid {
		value := int(event.WaiterOverride.Int64)
		input.WaiterOverride = &value
	}
	if event.CoordinatorOverride.Valid {
		value := int(event.CoordinatorOverride.Int64)
		input.CoordinatorOverride = &value
	}
	if event.LeaderOverride.Valid {
		value := int(event.LeaderOverride.Int64)
		input.LeaderOverride = &value
	}
	if event.CoLeaderOverride.Valid {
		value := int(event.CoLeaderOverride.Int64)
		input.CoLeaderOverride = &value
	}
	results, err := CalculateRules(input, rules)
	if err != nil {
		return models.Checklist{}, err
	}
	items := make([]models.ChecklistItem, 0, len(results))
	for _, result := range results {
		if containedInventoryIDs[result.Rule.ResultInventoryItemID] {
			continue
		}
		inventory, reserved, err := s.store.InventorySnapshot(ctx, event.ID, result.Rule.ResultInventoryItemID, event.StartsAt, event.EndsAt)
		if err != nil {
			return models.Checklist{}, fmt.Errorf("load inventory for rule %s: %w", result.Rule.RuleKey, err)
		}
		available := AvailableQuantity(inventory.StockQuantity, reserved, inventory.DamagedQuantity)
		missing := result.Quantity - available
		if missing < 0 {
			missing = 0
		}
		items = append(items, models.ChecklistItem{
			InventoryItemID:           sql.NullInt64{Int64: inventory.ID, Valid: true},
			CategoryID:                result.Rule.CategoryID,
			SourceRuleID:              sql.NullInt64{Int64: result.Rule.ID, Valid: true},
			SourceKey:                 fmt.Sprintf("rule:%d", result.Rule.ID),
			Name:                      inventory.Name,
			Unit:                      inventory.Unit,
			CalculatedQuantity:        result.Quantity,
			RequiredQuantity:          result.Quantity,
			AvailableQuantity:         available,
			ReservedElsewhereQuantity: reserved,
			MissingQuantity:           missing,
			CalculationOrigin:         result.Origin,
			ItemKind:                  inventory.ItemKind,
			LocationSnapshot:          inventory.LocationName,
		})
	}
	requirements, err := s.store.EventMenuRequirements(ctx, event.ID)
	if err != nil {
		return models.Checklist{}, fmt.Errorf("load menu requirements: %w", err)
	}
	for _, requirement := range requirements {
		if containedInventoryIDs[requirement.InventoryItemID] {
			continue
		}
		inventory, reserved, err := s.store.InventorySnapshot(ctx, event.ID, requirement.InventoryItemID, event.StartsAt, event.EndsAt)
		if err != nil {
			return models.Checklist{}, fmt.Errorf("load menu inventory %d: %w", requirement.InventoryItemID, err)
		}
		available := AvailableQuantity(inventory.StockQuantity, reserved, inventory.DamagedQuantity)
		missing := requirement.Quantity - available
		if missing < 0 {
			missing = 0
		}
		items = append(items, models.ChecklistItem{InventoryItemID: sql.NullInt64{Int64: inventory.ID, Valid: true}, CategoryID: inventory.CategoryID, SourceKey: requirement.SourceKey, Name: inventory.Name, Unit: inventory.Unit, CalculatedQuantity: requirement.Quantity, RequiredQuantity: requirement.Quantity, AvailableQuantity: available, ReservedElsewhereQuantity: reserved, MissingQuantity: missing, CalculationOrigin: requirement.Origin, ItemKind: inventory.ItemKind, LocationSnapshot: inventory.LocationName})
	}
	recipeRequirements, err := s.store.EventMenuRecipeRequirements(ctx, event.ID)
	if err != nil {
		return models.Checklist{}, fmt.Errorf("load menu recipe requirements: %w", err)
	}
	for _, requirement := range recipeRequirements {
		if containedInventoryIDs[requirement.InventoryItemID] {
			continue
		}
		inventory, reserved, err := s.store.InventorySnapshot(ctx, event.ID, requirement.InventoryItemID, event.StartsAt, event.EndsAt)
		if err != nil {
			return models.Checklist{}, fmt.Errorf("load recipe ingredient %d: %w", requirement.InventoryItemID, err)
		}
		available := AvailableQuantity(inventory.StockQuantity, reserved, inventory.DamagedQuantity)
		missing := requirement.Quantity - available
		if missing < 0 {
			missing = 0
		}
		items = append(items, models.ChecklistItem{InventoryItemID: sql.NullInt64{Int64: inventory.ID, Valid: true}, CategoryID: inventory.CategoryID, SourceKey: requirement.SourceKey, Name: inventory.Name, Unit: inventory.Unit, CalculatedQuantity: requirement.Quantity, RequiredQuantity: requirement.Quantity, AvailableQuantity: available, ReservedElsewhereQuantity: reserved, MissingQuantity: missing, CalculationOrigin: requirement.Origin, ItemKind: inventory.ItemKind, LocationSnapshot: inventory.LocationName})
	}
	decorationRequirements, err := s.store.EventDecorationRequirements(ctx, event.ID, event.HasDecoration)
	if err != nil {
		return models.Checklist{}, fmt.Errorf("load decoration requirements: %w", err)
	}
	for _, requirement := range decorationRequirements {
		if containedInventoryIDs[requirement.InventoryItemID] {
			continue
		}
		inventory, reserved, err := s.store.InventorySnapshot(ctx, event.ID, requirement.InventoryItemID, event.StartsAt, event.EndsAt)
		if err != nil {
			return models.Checklist{}, err
		}
		available := AvailableQuantity(inventory.StockQuantity, reserved, inventory.DamagedQuantity)
		missing := requirement.Quantity - available
		if missing < 0 {
			missing = 0
		}
		items = append(items, models.ChecklistItem{InventoryItemID: sql.NullInt64{Int64: inventory.ID, Valid: true}, CategoryID: inventory.CategoryID, SourceKey: requirement.SourceKey, Name: inventory.Name, Unit: inventory.Unit, CalculatedQuantity: requirement.Quantity, RequiredQuantity: requirement.Quantity, AvailableQuantity: available, ReservedElsewhereQuantity: reserved, MissingQuantity: missing, CalculationOrigin: requirement.Origin, ItemKind: inventory.ItemKind, LocationSnapshot: inventory.LocationName})
	}
	serviceRequirements, err := s.store.EventServiceRequirements(ctx, event.ID, event.GuestCount)
	if err != nil {
		return models.Checklist{}, fmt.Errorf("load service requirements: %w", err)
	}
	for _, requirement := range serviceRequirements {
		if containedInventoryIDs[requirement.InventoryItemID] {
			continue
		}
		inventory, reserved, err := s.store.InventorySnapshot(ctx, event.ID, requirement.InventoryItemID, event.StartsAt, event.EndsAt)
		if err != nil {
			return models.Checklist{}, err
		}
		available := AvailableQuantity(inventory.StockQuantity, reserved, inventory.DamagedQuantity)
		missing := requirement.Quantity - available
		if missing < 0 {
			missing = 0
		}
		items = append(items, models.ChecklistItem{InventoryItemID: sql.NullInt64{Int64: inventory.ID, Valid: true}, CategoryID: inventory.CategoryID, SourceKey: requirement.SourceKey, Name: inventory.Name, Unit: inventory.Unit, CalculatedQuantity: requirement.Quantity, RequiredQuantity: requirement.Quantity, AvailableQuantity: available, ReservedElsewhereQuantity: reserved, MissingQuantity: missing, CalculationOrigin: requirement.Origin, ItemKind: inventory.ItemKind, LocationSnapshot: inventory.LocationName})
	}
	cookRequirements, err := s.store.EventKitchenCookRequirements(ctx, event.ID)
	if err != nil {
		return models.Checklist{}, fmt.Errorf("load kitchen cook requirements: %w", err)
	}
	for _, requirement := range cookRequirements {
		inventory, reserved, err := s.store.InventorySnapshot(ctx, event.ID, requirement.InventoryItemID, event.StartsAt, event.EndsAt)
		if err != nil {
			return models.Checklist{}, err
		}
		available := AvailableQuantity(inventory.StockQuantity, reserved, inventory.DamagedQuantity)
		missing := requirement.Quantity - available
		if missing < 0 {
			missing = 0
		}
		items = append(items, models.ChecklistItem{InventoryItemID: sql.NullInt64{Int64: inventory.ID, Valid: true}, CategoryID: inventory.CategoryID, SourceKey: requirement.SourceKey, Name: inventory.Name, Unit: inventory.Unit, CalculatedQuantity: requirement.Quantity, RequiredQuantity: requirement.Quantity, AvailableQuantity: available, ReservedElsewhereQuantity: reserved, MissingQuantity: missing, CalculationOrigin: requirement.Origin, ItemKind: inventory.ItemKind, LocationSnapshot: inventory.LocationName})
	}
	if _, err := s.store.RegenerateChecklist(ctx, eventID, items); err != nil {
		return models.Checklist{}, err
	}
	if err := s.store.SyncDecorationRentalChecklist(ctx, eventID); err != nil {
		return models.Checklist{}, fmt.Errorf("sync decoration rentals: %w", err)
	}
	if err := s.store.EnsureCalculatedShortages(ctx, eventID); err != nil {
		return models.Checklist{}, fmt.Errorf("sync calculated shortages: %w", err)
	}
	if event.Status != "planning" && event.Status != "cancelled" && event.Status != "completed" {
		if err := s.store.SyncEventReservations(ctx, eventID); err != nil {
			return models.Checklist{}, fmt.Errorf("sync event reservations: %w", err)
		}
	}
	return s.store.GetChecklistByEvent(ctx, eventID)
}

func (s *ChecklistService) EnsureDemoChecklist(ctx context.Context) error {
	_, err := s.store.GetChecklistByEvent(ctx, 1)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}
	_, err = s.Generate(ctx, 1)
	return err
}

func (s *ChecklistService) StaffSummary(ctx context.Context, eventID int64) (models.StaffSummary, error) {
	event, err := s.store.GetEvent(ctx, eventID)
	if err != nil {
		return models.StaffSummary{}, err
	}
	rules, err := s.store.ListRules(ctx, false)
	if err != nil {
		return models.StaffSummary{}, err
	}
	input := CalculationInput{GuestCount: event.GuestCount, HasWelcomeDrinks: event.HasWelcomeDrinks, HasDecoration: event.HasDecoration, HasCoffeeTable: event.HasCoffeeTable, UsesGlassware: event.UsesGlassware}
	if event.WaiterOverride.Valid {
		value := int(event.WaiterOverride.Int64)
		input.WaiterOverride = &value
	}
	if event.CoordinatorOverride.Valid {
		value := int(event.CoordinatorOverride.Int64)
		input.CoordinatorOverride = &value
	}
	if event.LeaderOverride.Valid {
		value := int(event.LeaderOverride.Int64)
		input.LeaderOverride = &value
	}
	if event.CoLeaderOverride.Valid {
		value := int(event.CoLeaderOverride.Int64)
		input.CoLeaderOverride = &value
	}
	results, err := CalculateRules(input, rules)
	if err != nil {
		return models.StaffSummary{}, err
	}
	return StaffSummaryFromResults(results), nil
}
