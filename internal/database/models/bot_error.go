package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (BotError) TableName() string {
	return "bot_errors"
}

// BotError is a general-purpose error log, distinct from ReminderError (which
// is scoped to reminder dispatch failures). It captures any error surfaced by
// the Discord bot — either shown to a user via SendError/SendErrorDetailed, or
// caught by the top-level interaction handler when nothing was ever shown to
// the user at all — so failures are always visible somewhere other than
// server stdout logs.
type BotError struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt     time.Time  `gorm:"not null;default:now();index" json:"created_at"`
	Source        string     `gorm:"not null;index" json:"source"` // e.g. "command:remindus", "component:show_reminder"
	AccountID     *uuid.UUID `gorm:"type:uuid;index" json:"account_id,omitempty"`
	DiscordUserID *string    `json:"discord_user_id,omitempty"`
	Title         string     `gorm:"not null" json:"title"`
	Message       string     `gorm:"type:text;not null" json:"message"`
	Resolved      bool       `gorm:"not null;default:false;index" json:"resolved"`

	// Relationships
	Account *Account `gorm:"foreignKey:AccountID;constraint:OnDelete:SET NULL" json:"account,omitempty"`
}

// BeforeCreate hooks for setting timestamps and UUIDs
func (be *BotError) BeforeCreate(tx *gorm.DB) error {
	if be.ID == uuid.Nil {
		be.ID = uuid.New()
	}
	if be.CreatedAt.IsZero() {
		be.CreatedAt = time.Now()
	}
	return nil
}
