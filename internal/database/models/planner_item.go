package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (PlannerItem) TableName() string {
	return "planner_items"
}

// PlannerPeriod represents which half of the day a planner item belongs to
type PlannerPeriod string

const (
	PlannerPeriodMorning   PlannerPeriod = "morning"
	PlannerPeriodAfternoon PlannerPeriod = "afternoon"
)

// IsValid checks if the period is a known value
func (p PlannerPeriod) IsValid() bool {
	return p == PlannerPeriodMorning || p == PlannerPeriodAfternoon
}

// PlannerItem is a single entry in a account's "Day Planner" — a lightweight,
// manually-cleared checklist (unlike Reminders/DFM, it has no schedule or
// notification of its own). It may optionally be linked to a DFMItem so the
// checked state stays in sync between the two.
type PlannerItem struct {
	ID        uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AccountID uuid.UUID     `gorm:"type:uuid;not null;index" json:"account_id"`
	Content   string        `gorm:"not null" json:"content"`
	Checked   bool          `gorm:"not null;default:false" json:"checked"`
	Position  int           `gorm:"not null;default:0" json:"position"`
	Period    PlannerPeriod `gorm:"type:varchar(16);not null;default:'morning'" json:"period"`
	DFMItemID *uuid.UUID    `gorm:"type:uuid;index" json:"dfm_item_id,omitempty"`
	CreatedAt time.Time     `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time     `gorm:"not null;default:now()" json:"updated_at"`

	// Relationships
	Account *Account `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE" json:"account,omitempty"`
	DFMItem *DFMItem `gorm:"foreignKey:DFMItemID;constraint:OnDelete:SET NULL" json:"dfm_item,omitempty"`
}

// BeforeCreate hooks for setting timestamps and UUIDs
func (p *PlannerItem) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

func (p *PlannerItem) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	return nil
}
