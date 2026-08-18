package repositories

import (
	"errors"

	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// plannerItemRepository implementation
type plannerItemRepository struct {
	db *gorm.DB
}

// NewPlannerItemRepository creates a new planner item repository instance
func NewPlannerItemRepository(db *gorm.DB) PlannerItemRepository {
	return &plannerItemRepository{db: db}
}

// CountByAccountID returns how many planner items currently belong to an
// account, used to enforce the per-account item cap.
func (r *plannerItemRepository) CountByAccountID(accountID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.PlannerItem{}).Where("account_id = ?", accountID).Count(&count).Error
	return count, err
}

func (r *plannerItemRepository) Create(item *models.PlannerItem) error {
	// Append at the end of its period by default
	if item.Position == 0 {
		var maxPosition int
		r.db.Model(&models.PlannerItem{}).
			Where("account_id = ? AND period = ?", item.AccountID, item.Period).
			Select("COALESCE(MAX(position), 0)").
			Scan(&maxPosition)
		item.Position = maxPosition + 1
	}
	return r.db.Create(item).Error
}

func (r *plannerItemRepository) GetByID(id uuid.UUID) (*models.PlannerItem, error) {
	var item models.PlannerItem
	err := r.db.First(&item, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

// GetByAccountID returns all of the account's planner items, ordered by
// period then position so the caller can render the two columns directly.
func (r *plannerItemRepository) GetByAccountID(accountID uuid.UUID) ([]models.PlannerItem, error) {
	var items []models.PlannerItem
	err := r.db.Where("account_id = ?", accountID).
		Order("period ASC, position ASC, created_at ASC").
		Find(&items).Error
	return items, err
}

// GetByDFMItemID returns all planner items linked to a given DFM item, used
// to keep the checked state in sync when the DFM item changes.
func (r *plannerItemRepository) GetByDFMItemID(dfmItemID uuid.UUID) ([]models.PlannerItem, error) {
	var items []models.PlannerItem
	err := r.db.Where("dfm_item_id = ?", dfmItemID).Find(&items).Error
	return items, err
}

func (r *plannerItemRepository) Update(item *models.PlannerItem) error {
	return r.db.Save(item).Error
}

func (r *plannerItemRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.PlannerItem{}, "id = ?", id).Error
}

// DeleteByAccountID removes every planner item for the account, used to
// "erase everything" and start a fresh plan.
func (r *plannerItemRepository) DeleteByAccountID(accountID uuid.UUID) error {
	return r.db.Delete(&models.PlannerItem{}, "account_id = ?", accountID).Error
}

// ReorderInput describes a single item's new position within the reorder transaction
type ReorderInput struct {
	ID       uuid.UUID
	Position int
	Period   models.PlannerPeriod
}

// Reorder atomically updates the position/period of multiple items, e.g.
// after a drag-and-drop reorder or a move between Morning/Afternoon.
func (r *plannerItemRepository) Reorder(accountID uuid.UUID, updates []ReorderInput) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, u := range updates {
			res := tx.Model(&models.PlannerItem{}).
				Where("id = ? AND account_id = ?", u.ID, accountID).
				Updates(map[string]interface{}{
					"position": u.Position,
					"period":   u.Period,
				})
			if res.Error != nil {
				return res.Error
			}
		}
		return nil
	})
}
