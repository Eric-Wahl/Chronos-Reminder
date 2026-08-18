package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ericp/chronos-bot-reminder/internal/config"
	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/database/repositories"
	"github.com/ericp/chronos-bot-reminder/internal/services"
	"github.com/google/uuid"
)

// fallbackPollInterval is the maximum time the scheduler will wait before
// checking for reminders, even if no timer is set. This prevents the scheduler
// from getting stuck after system sleep, container restart, or when the timer
// channel is orphaned.
const fallbackPollInterval = 5 * time.Minute

// QueueEvent represents different types of events that can trigger a reschedule
type QueueEvent struct {
	Type       string    // "created", "updated", "deleted"
	ReminderID uuid.UUID // ID of the affected reminder (for updated/deleted events)
}

// Scheduler manages the timing and dispatching of reminders
type Scheduler struct {
	reminderRepo       repositories.ReminderRepository
	reminderErrorRepo  repositories.ReminderErrorRepository
	dispatcherRegistry *DispatcherRegistry
	garbageCollector   *GarbageCollector
	stopChan           chan struct{}
	updateChan         chan QueueEvent
	running            bool
	currentTimer       *time.Timer
}

// NewScheduler creates a new scheduler instance
func NewScheduler(reminderRepo repositories.ReminderRepository, reminderErrorRepo repositories.ReminderErrorRepository, dispatcherRegistry *DispatcherRegistry, garbageCollector *GarbageCollector) *Scheduler {
	return &Scheduler{
		reminderRepo:       reminderRepo,
		reminderErrorRepo:  reminderErrorRepo,
		dispatcherRegistry: dispatcherRegistry,
		garbageCollector:   garbageCollector,
		stopChan:           make(chan struct{}),
		updateChan:         make(chan QueueEvent, 100), // Buffered channel for updates
		running:            false,
		currentTimer:       nil,
	}
}

// Start begins the scheduler's main loop
func (s *Scheduler) Start(ctx context.Context) {
	if s.running {
		log.Println("[ENGINE] - Scheduler already running")
		return
	}

	s.running = true

	// Start the main scheduling loop
	go s.scheduleLoop(ctx)
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	if !s.running {
		log.Println("[ENGINE] - Scheduler already stopped")
		return
	}
	
	// Stop the current timer if it exists
	if s.currentTimer != nil {
		s.currentTimer.Stop()
		s.currentTimer = nil
	}
	
	close(s.stopChan)
	s.running = false
}

// IsRunning returns whether the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	return s.running
}

// NotifyReminderCreated notifies the scheduler that a new reminder was created
func (s *Scheduler) NotifyReminderCreated(reminderID uuid.UUID) {
	if !s.running {
		return
	}
	
	select {
	case s.updateChan <- QueueEvent{Type: "created"}:
		if config.IsDebugMode() {
			log.Println("[ENGINE] - Notified of reminder creation")
		}
	default:
		log.Println("[ENGINE] - Update channel full, skipping creation notification")
	}
}

// NotifyReminderUpdated notifies the scheduler that a reminder was updated
func (s *Scheduler) NotifyReminderUpdated(reminderID uuid.UUID) {
	if !s.running {
		return
	}
	
	select {
	case s.updateChan <- QueueEvent{Type: "updated", ReminderID: reminderID}:
		if config.IsDebugMode() {
			log.Printf("[ENGINE] - Notified of reminder update: %s", reminderID)
		}
		// Notify garbage collector to cancel any pending deletion
		if s.garbageCollector != nil {
			s.garbageCollector.NotifyReminderUpdated(reminderID)
		}
	default:
		log.Println("[ENGINE] - Update channel full, skipping update notification")
	}
}

// NotifyReminderDeleted notifies the scheduler that a reminder was deleted
func (s *Scheduler) NotifyReminderDeleted(reminderID uuid.UUID) {
	if !s.running {
		return
	}
	
	select {
	case s.updateChan <- QueueEvent{Type: "deleted", ReminderID: reminderID}:
		if config.IsDebugMode() {
			log.Printf("[ENGINE] - Notified of reminder deletion: %s", reminderID)
		}
		// Notify garbage collector to cancel any pending deletion
		if s.garbageCollector != nil {
			s.garbageCollector.NotifyReminderUpdated(reminderID)
		}
	default:
		log.Println("[ENGINE] - Update channel full, skipping deletion notification")
	}
}

// scheduleLoop is the main loop that waits for the next reminder or updates
func (s *Scheduler) scheduleLoop(ctx context.Context) {
	// Initial schedule setup
	s.scheduleNext()

	for {
		select {
		case <-ctx.Done():
			log.Println("[ENGINE] - Context cancelled, stopping scheduler")
			s.running = false
			return
		case <-s.stopChan:
			s.running = false
			return
		case <-s.updateChan:
			if config.IsDebugMode() {
				log.Println("[ENGINE] - Received update event, rescheduling...")
			}
			s.scheduleNext()
		case <-s.getTimerChan():
			// Timer fired, process due reminders
			s.checkAndProcessReminders()
			// Schedule the next batch
			s.scheduleNext()
		}
	}
}

// getTimerChan returns the timer channel or a nil channel if no timer is set
func (s *Scheduler) getTimerChan() <-chan time.Time {
	if s.currentTimer != nil {
		return s.currentTimer.C
	}
	// Return a channel that will never fire
	return make(<-chan time.Time)
}

// scheduleNext sets up the timer for the next reminder(s)
func (s *Scheduler) scheduleNext() {
	// Stop existing timer
	if s.currentTimer != nil {
		s.currentTimer.Stop()
		s.currentTimer = nil
	}

	// Get the next reminders
	nextReminders, err := s.reminderRepo.GetNextReminders()
	if err != nil {
		log.Printf("[ENGINE] - Error fetching next reminders: %v", err)
		services.LogBotError("engine:scheduler", "Failed to fetch next reminders", err.Error(), nil)
		// Set fallback timer to retry later
		log.Printf("[ENGINE] - Setting fallback poll timer (%v) due to error", fallbackPollInterval)
		s.currentTimer = time.NewTimer(fallbackPollInterval)
		return
	}

	if len(nextReminders) == 0 {
		if config.IsDebugMode() {
			log.Println("[ENGINE] - No upcoming reminders, waiting for updates...")
		}
		// Set fallback timer to periodically check for new reminders
		// This handles cases where reminders are added externally or notifications are missed
		s.currentTimer = time.NewTimer(fallbackPollInterval)
		return
	}

	// Calculate time until next reminder
	nextTime := nextReminders[0].NextFireUTC
	now := time.Now().UTC()
	duration := nextTime.Sub(now)

	if duration <= 0 {
		// Reminder is already due, set a very short timer to process it
		// We use a timer instead of a goroutine to ensure currentTimer is always set
		// and the main loop handles all processing consistently
		log.Printf("[ENGINE] - Reminder is already due, processing shortly")
		s.currentTimer = time.NewTimer(100 * time.Millisecond)
		return
	}

	// Cap the duration to fallbackPollInterval to handle system sleep/restart scenarios
	// where a long timer might get orphaned
	if duration > fallbackPollInterval {
		log.Printf("[ENGINE] - Next reminder at %v (in %v), using fallback poll interval (%v)", nextTime, duration, fallbackPollInterval)
		s.currentTimer = time.NewTimer(fallbackPollInterval)
	} else {
		log.Printf("[ENGINE] - Next reminder at %v (in %v)", nextTime, duration)
		s.currentTimer = time.NewTimer(duration)
	}
}

// checkAndProcessReminders fetches and processes all due reminders
func (s *Scheduler) checkAndProcessReminders() {
	// Get the next reminders (ones with the closest time)
	nextReminders, err := s.reminderRepo.GetNextReminders()
	if err != nil {
		log.Printf("[ENGINE] - Error fetching next reminders: %v", err)
		services.LogBotError("engine:scheduler", "Failed to fetch next reminders", err.Error(), nil)
		return
	}

	if len(nextReminders) == 0 {
		log.Println("[ENGINE] - No upcoming reminders found")
		return
	}

	log.Printf("[ENGINE] - Next reminder: %s", nextReminders[0].Message)

	// Check if any of the next reminders are due (they should be, since timer fired)
	now := time.Now().UTC()
	tolerance := time.Minute // Allow 1 minute tolerance

	var dueReminders []models.Reminder
	for _, reminder := range nextReminders {
		if reminder.NextFireUTC.Before(now.Add(tolerance)) {
			dueReminders = append(dueReminders, reminder)
		}
	}

	if len(dueReminders) == 0 {
		log.Println("[ENGINE] - Timer fired but no reminders are actually due")
		return
	}

	if (config.IsDebugMode()) {
		log.Printf("[ENGINE] - Found %d due reminders", len(dueReminders))
	}

	// Process each due reminder
	for _, reminder := range dueReminders {
		s.processReminder(&reminder)
	}
}

// processReminder handles the dispatching of a single reminder
func (s *Scheduler) processReminder(reminder *models.Reminder) {
	// Check for unfixed errors before dispatching
	unfixedErrors, err := s.reminderErrorRepo.GetUnfixedByReminderID(reminder.ID)
	if err != nil {
		log.Printf("[ENGINE] - Error checking unfixed errors for reminder %s: %v", reminder.ID, err)
		// Continue with dispatch attempt despite error checking failure
	} else if len(unfixedErrors) > 0 {
		if config.IsDebugMode() {
			log.Printf("[ENGINE] - Skipping reminder %s due to %d unfixed errors", reminder.ID, len(unfixedErrors))
		}
		return
	}

	// Dispatch the reminder to all its destinations
	err = s.dispatcherRegistry.DispatchReminder(reminder)
	if err != nil {
		log.Printf("[ENGINE] - Error dispatching reminder %s: %v", reminder.ID, err)
		return
	}

	// isFromSnooze returns if the reminder was sent due to snooze expiration (so a snooze time earlier than the original remind time)
	isFromSnooze := reminder.SnoozedAtUTC != nil && reminder.NextFireUTC != nil && reminder.SnoozedAtUTC.Equal(*reminder.NextFireUTC)
	log.Printf("[ENGINE] - Reminder %s dispatched (from snooze: %v)", reminder.ID, isFromSnooze)

	// If it's from a snooze we don't want to touch the reminder more than necessary
	if isFromSnooze {
		reminder.SnoozedAtUTC = nil
		
		// For recurring reminders from snooze, set next_fire_utc back to remind_at_utc
		if reminder.Recurrence != 0 {
			reminder.NextFireUTC = &reminder.RemindAtUTC
		} else {
			// For one-time reminders, clear next_fire_utc
			reminder.NextFireUTC = nil
		}
		
		err = s.reminderRepo.Update(reminder, false)
		if err != nil {
			log.Printf("[ENGINE] - Error updating reminder %s after snooze dispatch: %v", reminder.ID, err)
			services.LogBotError("engine:scheduler", "Failed to update reminder after snooze dispatch", fmt.Sprintf("reminder %s: %v", reminder.ID, err), nil)
		}

		// If it's a one-time reminder from snooze, add to garbage collector
		if reminder.Recurrence == 0 && s.garbageCollector != nil {
			s.garbageCollector.NotifyReminderDispatched(reminder.ID)
		}

		return
	}

	if reminder.Recurrence != 0 {
		// Handle recurrence only if not from snooze
		s.handleRecurrence(reminder)
	} else {
		// One-time reminder dispatched, add to garbage collector queue
		reminder.NextFireUTC = nil
		if err := s.reminderRepo.Update(reminder, false); err != nil {
			log.Printf("[ENGINE] - Error clearing next_fire_utc for reminder %s: %v", reminder.ID, err)
			services.LogBotError("engine:scheduler", "Failed to clear next_fire_utc after dispatch", fmt.Sprintf("reminder %s: %v", reminder.ID, err), nil)
		}

		if s.garbageCollector != nil {
			s.garbageCollector.NotifyReminderDispatched(reminder.ID)
		}
	}
}

// handleRecurrence manages recurring reminders
func (s *Scheduler) handleRecurrence(reminder *models.Reminder) {
	if reminder.Recurrence == 0 {
		return // No recurrence
	}

	// Get the user's timezone for proper DST-aware calculation
	ianaLocation := "UTC" // Default to UTC
	if reminder.Account != nil && reminder.Account.Timezone != nil {
		ianaLocation = reminder.Account.Timezone.IANALocation
	}

	newTime, err := services.GetNextOccurrence(reminder.RemindAtUTC, int(reminder.Recurrence), ianaLocation)
	if err != nil {
		log.Printf("[ENGINE] - Error getting next occurrence for reminder %s: %v", reminder.ID, err)
		// Critical: if this fails, the recurring reminder never gets
		// rescheduled and silently stops firing forever with no other trace.
		services.LogBotError("engine:scheduler", "Failed to compute next occurrence — recurring reminder will stop firing", fmt.Sprintf("reminder %s: %v", reminder.ID, err), nil)
		return
	}

	// Update the reminder with the new time
	err = s.reminderRepo.RescheduleReminder(reminder, newTime, false)
	if err != nil {
		log.Printf("[ENGINE] - Error rescheduling recurring reminder %s: %v", reminder.ID, err)
		// Critical: same as above — the reminder will not fire again.
		services.LogBotError("engine:scheduler", "Failed to reschedule recurring reminder — it will stop firing", fmt.Sprintf("reminder %s: %v", reminder.ID, err), nil)
	}
}

