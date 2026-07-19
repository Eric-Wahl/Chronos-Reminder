package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (Account) TableName() string {
	return "accounts"
}

// Account represents the accounts table
type Account struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TimezoneID    *uint     `gorm:"index" json:"timezone_id"`
	Email         *string   `gorm:"uniqueIndex" json:"email"` // login email, nullable (Discord-only accounts have none)
	Username      *string   `json:"username"`                 // display name, nullable
	PasswordHash  *string   `json:"-"`                        // login password, nullable; hidden in JSON
	EmailVerified bool      `gorm:"type:boolean;default:false" json:"email_verified"`
	Preferences   JSONB     `gorm:"type:jsonb" json:"preferences,omitempty"` // free-form user preferences, nullable
	CreatedAt     time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Relationships
	Timezone   *Timezone  `gorm:"foreignKey:TimezoneID" json:"timezone,omitempty"`
	Identities []Identity `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE" json:"identities,omitempty"`
	Reminders  []Reminder `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE" json:"reminders,omitempty"`
}

// Preference keys stored in Account.Preferences
const (
	PrefDiscordSendImage    = "discord_send_image"
	PrefDiscordEnableSnooze = "discord_enable_snooze"
)

// DiscordSendImage reports whether the account wants Discord reminders
// (remindme/remindus) to include a generated image, in addition to the text
// embed. Defaults to true when the preference hasn't been set.
func (a *Account) DiscordSendImage() bool {
	return a.boolPreference(PrefDiscordSendImage, true)
}

// DiscordSnoozeEnabled reports whether the account wants a Snooze button on
// Discord reminders. Defaults to true when the preference hasn't been set.
func (a *Account) DiscordSnoozeEnabled() bool {
	return a.boolPreference(PrefDiscordEnableSnooze, true)
}

// boolPreference reads a boolean preference, falling back to defaultValue
// when it hasn't been set.
func (a *Account) boolPreference(key string, defaultValue bool) bool {
	if a.Preferences == nil {
		return defaultValue
	}
	if v, ok := a.Preferences[key].(bool); ok {
		return v
	}
	return defaultValue
}

// HasPassword reports whether the account has email/password credentials set.
// This must be used instead of checking Email for non-nil: a Discord-linked
// account can have an Email populated (e.g. copied over from a legacy
// migration) without ever having a usable password.
func (a *Account) HasPassword() bool {
	return a.PasswordHash != nil
}

// MarshalJSON exposes computed, read-only fields (like has_password)
// alongside the account's regular fields, without adding real DB columns.
func (a Account) MarshalJSON() ([]byte, error) {
	type Alias Account
	return json.Marshal(struct {
		Alias
		HasPassword bool `json:"has_password"`
	}{
		Alias:       Alias(a),
		HasPassword: a.HasPassword(),
	})
}

// BeforeCreate hooks for setting timestamps and UUIDs
func (a *Account) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	return nil
}

func (a *Account) BeforeUpdate(tx *gorm.DB) error {
	a.UpdatedAt = time.Now()
	return nil
}
