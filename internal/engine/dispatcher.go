package engine

import (
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/ericp/chronos-bot-reminder/internal/config"
	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/database/repositories"
	"github.com/ericp/chronos-bot-reminder/internal/dispatchers"
	"github.com/google/uuid"
)

// Dispatcher interface defines how reminders are sent to different destinations
type Dispatcher interface {
	Dispatch(reminder *models.Reminder, destination *models.ReminderDestination, account *models.Account) error
	GetSupportedType() models.DestinationType
}

// DispatcherRegistry manages all available dispatchers
type DispatcherRegistry struct {
	dispatchers       map[models.DestinationType]Dispatcher
	reminderErrorRepo repositories.ReminderErrorRepository
}

// NewDispatcherRegistry creates a new dispatcher registry
func NewDispatcherRegistry(reminderErrorRepo repositories.ReminderErrorRepository) *DispatcherRegistry {
	return &DispatcherRegistry{
		dispatchers:       make(map[models.DestinationType]Dispatcher),
		reminderErrorRepo: reminderErrorRepo,
	}
}

// RegisterDispatcher registers a new dispatcher for a specific destination type
func (dr *DispatcherRegistry) RegisterDispatcher(dispatcher Dispatcher) {
	dr.dispatchers[dispatcher.GetSupportedType()] = dispatcher
}

// DispatchReminder dispatches a reminder to all its destinations
func (dr *DispatcherRegistry) DispatchReminder(reminder *models.Reminder) error {
	if len(reminder.Destinations) == 0 {
		return fmt.Errorf("reminder %s has no destinations", reminder.ID)
	}

	var errors []error
	for _, destination := range reminder.Destinations {
		dispatcher, exists := dr.dispatchers[destination.Type]
		if !exists {
			log.Printf("[DISPATCHER] - No dispatcher found for type %s, skipping", destination.Type)
			
			// Create error record for missing dispatcher
			dr.createErrorRecord(reminder.ID, destination.ID, fmt.Sprintf("No dispatcher found for type %s", destination.Type))
			continue
		}

		if err := dispatcher.Dispatch(reminder, &destination, reminder.Account); err != nil {
			log.Printf("[DISPATCHER] - Error dispatching to %s: %v", destination.Type, err)
			errors = append(errors, err)

			// Known, permanent Discord delivery failures (DMs closed, no
			// mutual guild, missing access, ...) aren't bugs — a Go stack
			// trace adds nothing beyond what the Discord error already
			// says, so skip it and just record the message. Anything else
			// is unexpected and keeps the full trace for debugging.
			var detail string
			if dispatchers.IsExpectedDiscordDeliveryError(err) {
				detail = fmt.Sprintf("Dispatch error: %v", err)
			} else {
				detail = fmt.Sprintf("Dispatch error: %v\nStack trace:\n%s", err, string(debug.Stack()))
			}
			dr.createErrorRecord(reminder.ID, destination.ID, detail)

			continue
		}

		if config.IsDebugMode() {
			log.Printf("[DISPATCHER] - [DEBUG] Dispatched reminder %s to %s", reminder.ID, destination.Type)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to dispatch to %d destinations", len(errors))
	}

	return nil
}

// GetDispatcher returns a dispatcher for a specific type
func (dr *DispatcherRegistry) GetDispatcher(destinationType models.DestinationType) (Dispatcher, bool) {
	dispatcher, exists := dr.dispatchers[destinationType]
	return dispatcher, exists
}

// createErrorRecord creates a reminder error record
func (dr *DispatcherRegistry) createErrorRecord(reminderID, destinationID uuid.UUID, stacktrace string) {
	if dr.reminderErrorRepo == nil {
		log.Printf("[DISPATCHER] - Warning: Cannot create error record - reminder error repository is nil")
		return
	}

	reminderError := &models.ReminderError{
		ReminderID:            reminderID,
		ReminderDestinationID: destinationID,
		Timestamp:             time.Now(),
		Stacktrace:            stacktrace,
		Fixed:                 false,
	}

	if err := dr.reminderErrorRepo.Create(reminderError); err != nil {
		log.Printf("[DISPATCHER] - Error creating reminder error record: %v", err)
	} else if config.IsDebugMode() {
		log.Printf("[DISPATCHER] - [DEBUG] Created error record for reminder %s, destination %s", reminderID, destinationID)
	}
}