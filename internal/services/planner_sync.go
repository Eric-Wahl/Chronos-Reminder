package services

import (
	"github.com/ericp/chronos-bot-reminder/internal/database"
	"github.com/google/uuid"
)

// SyncPlannerItemsFromDFMItem updates the checked state of any Day Planner
// items linked to the given DFM item, keeping the two in sync whenever the
// DFM item is checked/unchecked (from the web/mobile DFM UI or a bot command).
func SyncPlannerItemsFromDFMItem(dfmItemID uuid.UUID, checked bool) error {
	repo := database.GetRepositories()

	items, err := repo.PlannerItem.GetByDFMItemID(dfmItemID)
	if err != nil {
		return err
	}

	for i := range items {
		if items[i].Checked == checked {
			continue
		}
		items[i].Checked = checked
		if err := repo.PlannerItem.Update(&items[i]); err != nil {
			return err
		}
	}
	return nil
}
