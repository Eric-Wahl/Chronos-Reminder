package repositories

import (
	"errors"
	"time"

	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// dfmNoteRepository implementation
type dfmNoteRepository struct {
	db *gorm.DB
}

// NewDFMNoteRepository creates a new DFM note repository instance
func NewDFMNoteRepository(db *gorm.DB) DFMNoteRepository {
	return &dfmNoteRepository{db: db}
}

// GetOrCreateByAccountID returns the account's note, creating it on first use
func (r *dfmNoteRepository) GetOrCreateByAccountID(accountID uuid.UUID) (*models.DFMNote, error) {
	note, err := r.GetByAccountID(accountID)
	if err != nil {
		return nil, err
	}
	if note != nil {
		return note, nil
	}

	note = &models.DFMNote{AccountID: accountID}
	if err := r.db.Create(note).Error; err != nil {
		return nil, err
	}
	return note, nil
}

func (r *dfmNoteRepository) GetByAccountID(accountID uuid.UUID) (*models.DFMNote, error) {
	var note models.DFMNote
	err := r.db.First(&note, "account_id = ?", accountID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &note, nil
}

func (r *dfmNoteRepository) GetByID(id uuid.UUID) (*models.DFMNote, error) {
	var note models.DFMNote
	err := r.db.First(&note, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &note, nil
}

// GetWithItems returns the account's note with its items ordered by position
func (r *dfmNoteRepository) GetWithItems(accountID uuid.UUID) (*models.DFMNote, error) {
	var note models.DFMNote
	err := r.db.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("position ASC, created_at ASC")
	}).First(&note, "account_id = ?", accountID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &note, nil
}

func (r *dfmNoteRepository) Update(note *models.DFMNote) error {
	return r.db.Save(note).Error
}

func (r *dfmNoteRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.DFMNote{}, "id = ?", id).Error
}

// GetDueNotes returns all notes whose reminder is due, with account, timezone
// and items preloaded so they can be dispatched directly
func (r *dfmNoteRepository) GetDueNotes(now time.Time) ([]models.DFMNote, error) {
	pauseBit := 128
	var notes []models.DFMNote
	err := r.db.Preload("Account").
		Preload("Account.Timezone").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC, created_at ASC")
		}).
		Where("next_fire_utc IS NOT NULL AND next_fire_utc <= ? AND (recurrence & ?) = 0", now, pauseBit).
		Find(&notes).Error
	return notes, err
}

// dfmItemRepository implementation
type dfmItemRepository struct {
	db *gorm.DB
}

// NewDFMItemRepository creates a new DFM item repository instance
func NewDFMItemRepository(db *gorm.DB) DFMItemRepository {
	return &dfmItemRepository{db: db}
}

// CountByNoteID returns how many items currently belong to a note, used to
// enforce the per-note item cap.
func (r *dfmItemRepository) CountByNoteID(noteID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.DFMItem{}).Where("note_id = ?", noteID).Count(&count).Error
	return count, err
}

func (r *dfmItemRepository) Create(item *models.DFMItem) error {
	// Append at the end of the list by default
	if item.Position == 0 {
		var maxPosition int
		r.db.Model(&models.DFMItem{}).
			Where("note_id = ?", item.NoteID).
			Select("COALESCE(MAX(position), 0)").
			Scan(&maxPosition)
		item.Position = maxPosition + 1
	}
	return r.db.Create(item).Error
}

func (r *dfmItemRepository) GetByID(id uuid.UUID) (*models.DFMItem, error) {
	var item models.DFMItem
	err := r.db.First(&item, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *dfmItemRepository) GetByNoteID(noteID uuid.UUID) ([]models.DFMItem, error) {
	var items []models.DFMItem
	err := r.db.Where("note_id = ?", noteID).
		Order("position ASC, created_at ASC").
		Find(&items).Error
	return items, err
}

func (r *dfmItemRepository) Update(item *models.DFMItem) error {
	return r.db.Save(item).Error
}

func (r *dfmItemRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.DFMItem{}, "id = ?", id).Error
}

func (r *dfmItemRepository) DeleteByNoteID(noteID uuid.UUID) error {
	return r.db.Delete(&models.DFMItem{}, "note_id = ?", noteID).Error
}

