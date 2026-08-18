package repositories

import (
	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// botErrorRepository implementation
type botErrorRepository struct {
	db *gorm.DB
}

// NewBotErrorRepository creates a new bot error repository instance
func NewBotErrorRepository(db *gorm.DB) BotErrorRepository {
	return &botErrorRepository{db: db}
}

// Create creates a new bot error record
func (r *botErrorRepository) Create(botError *models.BotError) error {
	return r.db.Create(botError).Error
}

// GetByID retrieves a bot error by ID
func (r *botErrorRepository) GetByID(id uuid.UUID) (*models.BotError, error) {
	var botError models.BotError
	err := r.db.First(&botError, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &botError, nil
}

// GetRecent retrieves the most recent bot errors, newest first
func (r *botErrorRepository) GetRecent(limit int) ([]models.BotError, error) {
	var botErrors []models.BotError
	err := r.db.Order("created_at DESC").Limit(limit).Find(&botErrors).Error
	return botErrors, err
}

// GetUnresolved retrieves all unresolved bot errors, newest first
func (r *botErrorRepository) GetUnresolved() ([]models.BotError, error) {
	var botErrors []models.BotError
	err := r.db.Where("resolved = false").Order("created_at DESC").Find(&botErrors).Error
	return botErrors, err
}

// GetByAccountID retrieves all bot errors for a specific account
func (r *botErrorRepository) GetByAccountID(accountID uuid.UUID) ([]models.BotError, error) {
	var botErrors []models.BotError
	err := r.db.Where("account_id = ?", accountID).Order("created_at DESC").Find(&botErrors).Error
	return botErrors, err
}

// MarkResolved marks a bot error as resolved
func (r *botErrorRepository) MarkResolved(id uuid.UUID) error {
	return r.db.Model(&models.BotError{}).Where("id = ?", id).Update("resolved", true).Error
}
