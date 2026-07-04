package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/database/repositories"
	"github.com/ericp/chronos-bot-reminder/internal/services"
	"github.com/google/uuid"
)

// ReminderHandler handles reminder-related HTTP requests
type ReminderHandler struct {
	reminderRepo      repositories.ReminderRepository
	destinationRepo   repositories.ReminderDestinationRepository
	reminderErrorRepo repositories.ReminderErrorRepository
	accountRepo       repositories.AccountRepository
	timezoneRepo      repositories.TimezoneRepository
}

// NewReminderHandler creates a new reminder handler
func NewReminderHandler(
	reminderRepo repositories.ReminderRepository,
	destinationRepo repositories.ReminderDestinationRepository,
	reminderErrorRepo repositories.ReminderErrorRepository,
) *ReminderHandler {
	return &ReminderHandler{
		reminderRepo:      reminderRepo,
		destinationRepo:   destinationRepo,
		reminderErrorRepo: reminderErrorRepo,
	}
}

// SetAccountRepository sets the account repository
func (h *ReminderHandler) SetAccountRepository(repo repositories.AccountRepository) {
	h.accountRepo = repo
}

// SetTimezoneRepository sets the timezone repository
func (h *ReminderHandler) SetTimezoneRepository(repo repositories.TimezoneRepository) {
	h.timezoneRepo = repo
}

// GetReminder retrieves a single reminder by ID
func (h *ReminderHandler) GetReminder(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)
	reminderID := r.PathValue("id")

	id, err := uuid.Parse(reminderID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid reminder ID")
		return
	}

	reminder, err := h.reminderRepo.GetWithAccountAndDestinations(id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch reminder")
		return
	}

	if reminder == nil || reminder.AccountID != accountID {
		WriteError(w, http.StatusNotFound, "Reminder not found")
		return
	}

	WriteJSON(w, http.StatusOK, ToReminderResponse(reminder))
}

// UpdateReminder updates an existing reminder
func (h *ReminderHandler) UpdateReminder(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)
	reminderID := r.PathValue("id")

	id, err := uuid.Parse(reminderID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid reminder ID")
		return
	}

	// Fetch existing reminder
	reminder, err := h.reminderRepo.GetWithAccountAndDestinations(id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch reminder")
		return
	}

	if reminder == nil || reminder.AccountID != accountID {
		WriteError(w, http.StatusNotFound, "Reminder not found")
		return
	}

	// Fetch account with timezone for date/time conversion
	account, err := h.accountRepo.GetWithTimezone(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve account")
		return
	}

	if account == nil {
		WriteError(w, http.StatusNotFound, "Account not found")
		return
	}

	if account.Timezone == nil {
		WriteError(w, http.StatusBadRequest, "Account timezone not set")
		return
	}

	// Parse request body
	var updateData struct {
		Message      string          `json:"message"`
		Date         string          `json:"date"`
		Time         string          `json:"time"`
		Recurrence   json.RawMessage `json:"recurrence"`
		Destinations []struct {
			Type     string                 `json:"type"`
			Metadata map[string]interface{} `json:"metadata"`
		} `json:"destinations"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update reminder fields
	if updateData.Message != "" {
		reminder.Message = updateData.Message
	}

	if updateData.Date != "" && updateData.Time != "" {
		// Parse the reminder date and time in user's timezone (same as CreateReminder)
		parsedTime, err := services.ParseReminderDateTimeInTimezone(updateData.Date, updateData.Time, account.Timezone.IANALocation)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid date/time format")
			return
		}
		reminder.RemindAtUTC = parsedTime.UTC()
		reminder.NextFireUTC = &parsedTime
	}

	// Handle recurrence - can be string or int
	if len(updateData.Recurrence) > 0 {
		var recurrenceValue int

		// Try to parse as string first (new format: "DAILY", "WEEKLY", etc.)
		var recurrenceStr string
		if err := json.Unmarshal(updateData.Recurrence, &recurrenceStr); err == nil {
			// It's a string, convert using the map
			if val, exists := services.RecurrenceTypeMap[recurrenceStr]; exists {
				recurrenceValue = val
			} else {
				WriteError(w, http.StatusBadRequest, "Invalid recurrence type")
				return
			}
		} else {
			// Try to parse as int (legacy format)
			if err := json.Unmarshal(updateData.Recurrence, &recurrenceValue); err != nil {
				WriteError(w, http.StatusBadRequest, "Invalid recurrence format")
				return
			}
		}

		if recurrenceValue >= 0 {
			reminder.Recurrence = int16(recurrenceValue)
		}
	}

	// Update destinations if provided
	if len(updateData.Destinations) > 0 {
		// Delete old destinations
		if err := h.destinationRepo.DeleteByReminderID(id); err != nil {
			WriteError(w, http.StatusInternalServerError, "Failed to update destinations")
			return
		}

		effectiveRecurrence := int(reminder.Recurrence)

		// Create new destinations
		newDestinations := make([]models.ReminderDestination, len(updateData.Destinations))
		for i, dest := range updateData.Destinations {
			destType := models.DestinationType(dest.Type)
			if !destType.IsValid() {
				WriteError(w, http.StatusBadRequest, "Invalid destination type")
				return
			}

			if destType == models.DestinationEmail {
				if _, hasEmail := dest.Metadata["email"]; !hasEmail {
					WriteError(w, http.StatusBadRequest, fmt.Sprintf("Email destination requires email in metadata"))
					return
				}
				if services.GetRecurrenceType(effectiveRecurrence) == services.RecurrenceHourly {
					WriteError(w, http.StatusBadRequest, "Email destination cannot be used with hourly recurrence")
					return
				}
			}

			if destType == models.DestinationAndroidPush {
				if dest.Metadata == nil {
					dest.Metadata = map[string]interface{}{}
				}
				dest.Metadata["account_id"] = accountID.String()
			}

			newDestinations[i] = models.ReminderDestination{
				ID:         uuid.New(),
				ReminderID: id,
				Type:       destType,
				Metadata:   models.JSONB(dest.Metadata),
			}
		}

		if err := h.destinationRepo.CreateMultiple(newDestinations); err != nil {
			WriteError(w, http.StatusInternalServerError, "Failed to create destinations")
			return
		}

		reminder.Destinations = newDestinations
	}

	// Save reminder
	if err := h.reminderRepo.Update(reminder, true); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update reminder")
		return
	}

	WriteJSON(w, http.StatusOK, ToReminderResponse(reminder))
}

// DeleteReminder deletes a reminder
func (h *ReminderHandler) DeleteReminder(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)
	reminderID := r.PathValue("id")

	id, err := uuid.Parse(reminderID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid reminder ID")
		return
	}

	// Verify ownership
	reminder, err := h.reminderRepo.GetByID(id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch reminder")
		return
	}

	if reminder == nil || reminder.AccountID != accountID {
		WriteError(w, http.StatusNotFound, "Reminder not found")
		return
	}

	// Delete destinations first
	if err := h.destinationRepo.DeleteByReminderID(id); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to delete reminder")
		return
	}

	// Delete reminder
	if err := h.reminderRepo.Delete(id, true); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to delete reminder")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Reminder deleted successfully"})
}

// PauseReminder pauses a reminder
func (h *ReminderHandler) PauseReminder(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)
	reminderID := r.PathValue("id")

	id, err := uuid.Parse(reminderID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid reminder ID")
		return
	}

	reminder, err := h.reminderRepo.GetByID(id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch reminder")
		return
	}

	if reminder == nil || reminder.AccountID != accountID {
		WriteError(w, http.StatusNotFound, "Reminder not found")
		return
	}

	// Set pause bit
	const pauseBit = 128
	reminder.Recurrence |= pauseBit

	if err := h.reminderRepo.Update(reminder, true); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to pause reminder")
		return
	}

	WriteJSON(w, http.StatusOK, ToReminderResponse(reminder))
}

// ResumeReminder resumes a paused reminder
func (h *ReminderHandler) ResumeReminder(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)
	reminderID := r.PathValue("id")

	id, err := uuid.Parse(reminderID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid reminder ID")
		return
	}

	reminder, err := h.reminderRepo.GetByID(id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch reminder")
		return
	}

	if reminder == nil || reminder.AccountID != accountID {
		WriteError(w, http.StatusNotFound, "Reminder not found")
		return
	}

	// Clear pause bit
	const pauseBit = 128
	reminder.Recurrence &= ^pauseBit

	if err := h.reminderRepo.Update(reminder, true); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to resume reminder")
		return
	}

	WriteJSON(w, http.StatusOK, ToReminderResponse(reminder))
}

// DuplicateReminder duplicates a reminder
func (h *ReminderHandler) DuplicateReminder(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)
	reminderID := r.PathValue("id")

	id, err := uuid.Parse(reminderID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid reminder ID")
		return
	}

	// Fetch original reminder with destinations
	original, err := h.reminderRepo.GetWithAccountAndDestinations(id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch reminder")
		return
	}

	if original == nil || original.AccountID != accountID {
		WriteError(w, http.StatusNotFound, "Reminder not found")
		return
	}

	reminderCount, err := h.reminderRepo.CountByAccountID(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to check reminder count")
		return
	}
	if reminderCount >= services.MaxRemindersPerAccount {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("You have reached the maximum of %d reminders", services.MaxRemindersPerAccount))
		return
	}

	// Create new reminder
	newReminder := &models.Reminder{
		ID:             uuid.New(),
		AccountID:      original.AccountID,
		RemindAtUTC:    original.RemindAtUTC,
		Message:        original.Message,
		Recurrence:     original.Recurrence,
		CreatedAt:      time.Now().UTC(),
		NextFireUTC:    original.NextFireUTC,
		SnoozedAtUTC:   original.SnoozedAtUTC,
	}

	// Create reminder
	if err := h.reminderRepo.Create(newReminder, true); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to duplicate reminder")
		return
	}

	// Duplicate destinations
	if len(original.Destinations) > 0 {
		newDestinations := make([]models.ReminderDestination, len(original.Destinations))
		for i, dest := range original.Destinations {
			newDestinations[i] = models.ReminderDestination{
				ID:        uuid.New(),
				ReminderID: newReminder.ID,
				Type:      dest.Type,
				Metadata:  dest.Metadata,
			}
		}

		if err := h.destinationRepo.CreateMultiple(newDestinations); err != nil {
			WriteError(w, http.StatusInternalServerError, "Failed to duplicate destinations")
			return
		}

		newReminder.Destinations = newDestinations
	}

	WriteJSON(w, http.StatusCreated, ToReminderResponse(newReminder))
}

type snoozeRequest struct {
	Minutes int `json:"minutes"`
}

// SnoozeReminder sets a snooze on a reminder so it re-fires after the given duration.
// @Route: POST /api/reminders/{id}/snooze
func (h *ReminderHandler) SnoozeReminder(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)
	reminderID := r.PathValue("id")

	id, err := uuid.Parse(reminderID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid reminder ID")
		return
	}

	var req snoozeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Minutes <= 0 {
		WriteError(w, http.StatusBadRequest, "minutes must be a positive integer")
		return
	}

	reminder, err := h.reminderRepo.GetByID(id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch reminder")
		return
	}
	if reminder == nil || reminder.AccountID != accountID {
		WriteError(w, http.StatusNotFound, "Reminder not found")
		return
	}

	snoozeUntil := time.Now().UTC().Add(time.Duration(req.Minutes) * time.Minute)
	if err := h.reminderRepo.Snooze(id, snoozeUntil); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to snooze reminder")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("Snoozed for %d minutes", req.Minutes)})
}
