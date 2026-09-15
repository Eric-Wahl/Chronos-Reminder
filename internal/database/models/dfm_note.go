package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (DFMNote) TableName() string {
	return "dfm_notes"
}

// DFMNote represents the dfm_notes table.
// Each account owns at most one "Don't Forget Me" note that holds a private
// todo list. A recurring reminder can be attached to the whole note: when it
// fires, the full note content is sent to the user privately.
type DFMNote struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AccountID   uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex" json:"account_id"`
	RemindAtUTC *time.Time `gorm:"default:null" json:"remind_at_utc,omitempty"`
	NextFireUTC *time.Time `gorm:"default:null;index" json:"next_fire_utc,omitempty"`
	Recurrence  int16      `gorm:"not null;default:0" json:"recurrence"`
	// Private delivery channels; both can be enabled at the same time
	SendDiscordDM bool      `gorm:"not null;default:true" json:"send_discord_dm"`
	SendEmail     bool      `gorm:"not null;default:false" json:"send_email"`
	LastSentAt    *time.Time `gorm:"default:null" json:"last_sent_at,omitempty"`
	// Set to the time of the first consecutive delivery failure on that
	// channel, cleared on the next success. Used to detect "zombie" channels
	// that have been failing uninterrupted for a long time (see
	// ZombieCleaner), without touching the note's actual content.
	DiscordFailingSince *time.Time `gorm:"default:null" json:"discord_failing_since,omitempty"`
	EmailFailingSince   *time.Time `gorm:"default:null" json:"email_failing_since,omitempty"`
	CreatedAt     time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Relationships
	Account *Account  `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE" json:"account,omitempty"`
	Items   []DFMItem `gorm:"foreignKey:NoteID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

// BeforeCreate hooks for setting timestamps and UUIDs
func (n *DFMNote) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	now := time.Now()
	n.CreatedAt = now
	n.UpdatedAt = now
	return nil
}

func (n *DFMNote) BeforeUpdate(tx *gorm.DB) error {
	n.UpdatedAt = time.Now()
	return nil
}

// HasReminder returns true when a reminder is configured on the note
func (n *DFMNote) HasReminder() bool {
	return n.RemindAtUTC != nil
}

// IsValidDFMDestination checks that the destination is private (DFM notes
// never go to Discord server channels or webhooks)
func IsValidDFMDestination(destination DestinationType) bool {
	return destination == DestinationDiscordDM || destination == DestinationEmail
}

// Destinations returns the enabled delivery channels of the note
func (n *DFMNote) Destinations() []DestinationType {
	var destinations []DestinationType
	if n.SendDiscordDM {
		destinations = append(destinations, DestinationDiscordDM)
	}
	if n.SendEmail {
		destinations = append(destinations, DestinationEmail)
	}
	return destinations
}
