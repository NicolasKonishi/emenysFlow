package handlers

import "buffetflow/internal/models"

// splitEventChecklists derives the two working lists from the checklist that
// is persisted for an event. Keeping one source of truth means an item can be
// checked, loaded and reported without having to synchronize duplicate rows.
func splitEventChecklists(checklist models.Checklist) (models.Checklist, models.Checklist) {
	operational := checklist
	decoration := checklist
	operational.Items = make([]models.ChecklistItem, 0, len(checklist.Items))
	decoration.Items = make([]models.ChecklistItem, 0, len(checklist.Items))

	for _, item := range checklist.Items {
		if checklistIsStaffPerson(item) {
			continue
		}
		if checklistIsDecoration(item) {
			decoration.Items = append(decoration.Items, item)
			continue
		}
		operational.Items = append(operational.Items, item)
	}
	operational.Progress = checklistProgressForView(operational.Items)
	decoration.Progress = checklistProgressForView(decoration.Items)
	return operational, decoration
}

func splitEventShortages(shortages []models.ChecklistShortage, operationalChecklist, decoration models.Checklist) ([]models.ChecklistShortage, []models.ChecklistShortage) {
	operationalIDs := make(map[int64]bool, len(operationalChecklist.Items))
	for _, item := range operationalChecklist.Items {
		operationalIDs[item.ID] = true
	}
	decorationIDs := make(map[int64]bool, len(decoration.Items))
	for _, item := range decoration.Items {
		decorationIDs[item.ID] = true
	}
	operationalShortages := make([]models.ChecklistShortage, 0, len(shortages))
	decorationShortages := make([]models.ChecklistShortage, 0, len(shortages))
	for _, shortage := range shortages {
		if decorationIDs[shortage.ChecklistItemID] {
			decorationShortages = append(decorationShortages, shortage)
			continue
		}
		if operationalIDs[shortage.ChecklistItemID] {
			operationalShortages = append(operationalShortages, shortage)
		}
	}
	return operationalShortages, decorationShortages
}

func checklistProgressForView(items []models.ChecklistItem) models.ChecklistProgress {
	progress := models.ChecklistProgress{Total: len(items)}
	for _, item := range items {
		if item.MissingQuantity > 0 {
			progress.Missing++
		}
		switch item.Status {
		case "separated", "checked", "loaded", "at_event", "returned", "not_applicable":
			progress.Completed++
		default:
			progress.Pending++
		}
	}
	if progress.Total == 0 {
		return progress
	}
	separated, loaded := 0.0, 0.0
	for _, item := range items {
		if item.RequiredQuantity <= 0 {
			continue
		}
		if item.Status == "not_applicable" {
			separated++
			loaded++
			continue
		}
		separated += minChecklistProgress(1, item.SeparatedQuantity/item.RequiredQuantity)
		loaded += minChecklistProgress(1, item.LoadedQuantity/item.RequiredQuantity)
	}
	progress.Percentage = int(float64(progress.Completed)*100/float64(progress.Total) + .5)
	progress.SeparationPercentage = int(separated*100/float64(progress.Total) + .5)
	progress.LoadingPercentage = int(loaded*100/float64(progress.Total) + .5)
	return progress
}

func minChecklistProgress(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
