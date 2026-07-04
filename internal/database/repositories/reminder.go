package repositories

import (
	"errors"
	"time"

	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SchedulerNotifier interface for notifying the scheduler of reminder changes
type SchedulerNotifier interface {
	NotifyReminderCreated(reminderID uuid.UUID)
	NotifyReminderUpdated(reminderID uuid.UUID)
	NotifyReminderDeleted(reminderID uuid.UUID)
}

// GarbageCollectorNotifier interface for notifying the garbage collector
type GarbageCollectorNotifier interface {
	NotifyReminderUpdated(reminderID uuid.UUID)
}

// reminderRepository implementation
type reminderRepository struct {
	db               *gorm.DB
	scheduler        SchedulerNotifier
	garbageCollector GarbageCollectorNotifier
}

// NewReminderRepository creates a new reminder repository instance
func NewReminderRepository(db *gorm.DB) ReminderRepository {
	return &reminderRepository{db: db}
}

// SetScheduler sets the scheduler notifier for the repository
func (r *reminderRepository) SetScheduler(scheduler SchedulerNotifier) {
	r.scheduler = scheduler
}

// SetGarbageCollector sets the garbage collector notifier for the repository
func (r *reminderRepository) SetGarbageCollector(gc GarbageCollectorNotifier) {
	r.garbageCollector = gc
}

// Reminder Repository Implementation
func (r *reminderRepository) Create(reminder *models.Reminder, notify bool) error {
	err := r.db.Create(reminder).Error
	if err == nil && notify {
		if r.scheduler != nil {
			r.scheduler.NotifyReminderCreated(reminder.ID)
		}
	}
	return err
}

// CountByAccountID returns how many reminders currently belong to an account,
// used to enforce the per-account reminder cap.
func (r *reminderRepository) CountByAccountID(accountID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.Reminder{}).Where("account_id = ?", accountID).Count(&count).Error
	return count, err
}

func (r *reminderRepository) GetByID(id uuid.UUID) (*models.Reminder, error) {
	var reminder models.Reminder
	err := r.db.First(&reminder, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reminder, nil
}

func (r *reminderRepository) GetByAccountID(accountID uuid.UUID) ([]models.Reminder, error) {
	var reminders []models.Reminder
	err := r.db.Where("account_id = ?", accountID).Find(&reminders).Error
	return reminders, err
}

func (r *reminderRepository) GetByAccountIDWithDestinations(accountID uuid.UUID) ([]models.Reminder, error) {
	var reminders []models.Reminder
	err := r.db.Preload("Destinations").Where("account_id = ?", accountID).Find(&reminders).Error
	return reminders, err
}

func (r *reminderRepository) GetWithDestinations(id uuid.UUID) (*models.Reminder, error) {
	var reminder models.Reminder
	err := r.db.Preload("Destinations").First(&reminder, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reminder, nil
}

func (r *reminderRepository) GetWithAccount(id uuid.UUID) (*models.Reminder, error) {
	var reminder models.Reminder
	err := r.db.Preload("Account").Preload("Account.Timezone").First(&reminder, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reminder, nil
}

func (r *reminderRepository) GetWithAccountAndDestinations(id uuid.UUID) (*models.Reminder, error) {
	var reminder models.Reminder
	err := r.db.Preload("Account").Preload("Account.Timezone").Preload("Destinations").First(&reminder, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reminder, nil
}

func (r *reminderRepository) Update(reminder *models.Reminder, notify bool) error {
	err := r.db.Save(reminder).Error
	if err == nil && notify {
		if r.scheduler != nil {
			r.scheduler.NotifyReminderUpdated(reminder.ID)
		}
	}
	return err
}

func (r *reminderRepository) Delete(id uuid.UUID, notify bool) error {
	err := r.db.Delete(&models.Reminder{}, "id = ?", id).Error
	if err == nil {
		if r.scheduler != nil && notify {
			r.scheduler.NotifyReminderDeleted(id)
		}
	}
	return err
}

func (r *reminderRepository) GetNextReminders() ([]models.Reminder, error) {
	// First, check if the table is empty
	var count int64
	err := r.db.Model(&models.Reminder{}).Count(&count).Error
	if err != nil {
		return nil, err
	}

	if count == 0 {
		// No reminders exist
		return []models.Reminder{}, nil
	}

	// Find the next reminder(s) to process in a single query
	// Priority: past due reminders first (including snoozed), then earliest future reminders
	var reminders []models.Reminder
	now := time.Now().UTC()
	pauseBit := 128 // PauseBit
	
	dbResult := r.db.Preload("Account").
		Preload("Account.Timezone").
		Preload("Destinations").
		Where(`
			(
				next_fire_utc <= ?

				OR
				next_fire_utc = (
					SELECT MIN(next_fire_utc)
					FROM reminders s
					WHERE s.next_fire_utc > ?
					AND (s.recurrence & ?) = 0
				)
			)
			AND (recurrence & ?) = 0
			AND id NOT IN (
				SELECT reminder_id
				FROM reminder_errors
				WHERE fixed = false
			)
		`, now, now, pauseBit, pauseBit).
		Order("next_fire_utc ASC").
		Find(&reminders)


	err = dbResult.Error
	if err != nil {
		return nil, err
	}

	// If we found reminders, prioritize past due ones
	if len(reminders) > 0 {
		// Check if we have past due reminders (considering snooze time)
		for _, reminder := range reminders {
			if reminder.NextFireUTC.Before(now) || reminder.NextFireUTC.Equal(now) {
				// Return only the first past due reminder
				return []models.Reminder{reminder}, nil
			}
		}
		
		// No past due reminders, return all reminders with the earliest future time
		var futureReminders []models.Reminder
		for _, reminder := range reminders {
			effectiveTime := reminder.RemindAtUTC
			if reminder.SnoozedAtUTC != nil {
				effectiveTime = *reminder.SnoozedAtUTC
			}
			if effectiveTime.Equal(*reminders[0].NextFireUTC) {
				futureReminders = append(futureReminders, reminder)
			}
		}
		return futureReminders, nil
	}

	// No reminders found
	return []models.Reminder{}, nil
}

// Return all the reminders that are scheduled for deletion (one-time reminders that have been dispatched)
func (r *reminderRepository) GetNextsRemindersToDelete() ([]models.Reminder, error) {
	var reminders []models.Reminder
	err := r.db.Preload("Account").
		Preload("Account.Timezone").
		Preload("Destinations").
		Where("recurrence = 0 AND snoozed_at_utc IS NULL AND next_fire_utc IS NULL").
		Find(&reminders).Error
	return reminders, err
}

// Reschedule, used for snoozing and recurrence
func (r *reminderRepository) Reschedule(id uuid.UUID, newTime time.Time, notify bool) error {
	// First, get the current reminder to check snoozed_at_utc
	var reminder models.Reminder
	if err := r.db.First(&reminder, "id = ?", id).Error; err != nil {
		return err
	}

	// Calculate next_fire_utc as the minimum between remind_at_utc and snoozed_at_utc
	nextFireUTC := newTime
	if reminder.SnoozedAtUTC != nil && reminder.SnoozedAtUTC.Before(newTime) {
		nextFireUTC = *reminder.SnoozedAtUTC
	}

	// Update both remind_at_utc and next_fire_utc
	updates := map[string]interface{}{
		"remind_at_utc": newTime,
		"next_fire_utc": nextFireUTC,
	}

	err := r.db.Model(&models.Reminder{}).Where("id = ?", id).Updates(updates).Error
	if err == nil && notify {
		if r.scheduler != nil {
			r.scheduler.NotifyReminderUpdated(id)
		}
	}
	return err
}

func (r *reminderRepository) Snooze(id uuid.UUID, snoozeUntil time.Time) error {
	var reminder models.Reminder
	if err := r.db.First(&reminder, "id = ?", id).Error; err != nil {
		return err
	}

	// next_fire_utc must be the snooze wake-up time, not the original remind_at_utc.
	// Using min(remind_at, snooze) would set next_fire to a past time when the
	// reminder has already fired, causing it to either be skipped or re-dispatch
	// immediately without the snooze delay.
	updates := map[string]interface{}{
		"snoozed_at_utc": snoozeUntil,
		"next_fire_utc":  snoozeUntil,
	}

	err := r.db.Model(&models.Reminder{}).Where("id = ?", id).Updates(updates).Error
	if err == nil {
		if r.scheduler != nil {
			r.scheduler.NotifyReminderUpdated(id)
		}
		// Notify garbage collector that reminder was updated
		if r.garbageCollector != nil {
			r.garbageCollector.NotifyReminderUpdated(id)
		}
	}
	return err
}

func (r *reminderRepository) RescheduleReminder(reminder *models.Reminder, newTime time.Time, notify bool) error {
	// Calculate next_fire_utc as the minimum between remind_at_utc and snoozed_at_utc
	nextFireUTC := newTime
	if reminder.SnoozedAtUTC != nil && reminder.SnoozedAtUTC.Before(newTime) {
		nextFireUTC = *reminder.SnoozedAtUTC
	}

	// Update both remind_at_utc and next_fire_utc
	reminder.RemindAtUTC = newTime
	reminder.NextFireUTC = &nextFireUTC

	err := r.db.Save(reminder).Error
	if err == nil && notify {
		if r.scheduler != nil {
			r.scheduler.NotifyReminderUpdated(reminder.ID)
		}
	}
	return err
}

func (r *reminderRepository) SnoozeReminder(reminder *models.Reminder, snoozeUntil time.Time) error {
	// Update snoozed_at_utc and next_fire_utc
	reminder.SnoozedAtUTC = &snoozeUntil
	reminder.NextFireUTC = &snoozeUntil

	err := r.db.Save(reminder).Error
	if err == nil {
		if r.scheduler != nil {
			r.scheduler.NotifyReminderUpdated(reminder.ID)
		}
		// Notify garbage collector that reminder was updated
		if r.garbageCollector != nil {
			r.garbageCollector.NotifyReminderUpdated(reminder.ID)
		}
	}
	return err
}