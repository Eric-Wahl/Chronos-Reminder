package repositories

import "gorm.io/gorm"

// Repositories contains all repository instances
type Repositories struct {
	Timezone            TimezoneRepository
	Account             AccountRepository
	Identity            IdentityRepository
	Reminder            ReminderRepository
	ReminderDestination ReminderDestinationRepository
	ReminderError       ReminderErrorRepository
	BotError            BotErrorRepository
	EmailVerification   EmailVerificationRepository
	PasswordReset       PasswordResetRepository
	DFMNote             DFMNoteRepository
	DFMItem             DFMItemRepository
	PlannerItem         PlannerItemRepository
	FcmToken            FcmTokenRepository
}

// NewRepositories creates new repository instances
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Timezone:            NewTimezoneRepository(db),
		Account:             NewAccountRepository(db),
		Identity:            NewIdentityRepository(db),
		Reminder:            NewReminderRepository(db),
		ReminderDestination: NewReminderDestinationRepository(db),
		ReminderError:       NewReminderErrorRepository(db),
		BotError:            NewBotErrorRepository(db),
		EmailVerification:   NewEmailVerificationRepository(db),
		PasswordReset:       NewPasswordResetRepository(db),
		DFMNote:             NewDFMNoteRepository(db),
		DFMItem:             NewDFMItemRepository(db),
		PlannerItem:         NewPlannerItemRepository(db),
		FcmToken:            NewFcmTokenRepository(db),
	}
}
