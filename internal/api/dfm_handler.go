package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/database/repositories"
	"github.com/ericp/chronos-bot-reminder/internal/engine"
	"github.com/ericp/chronos-bot-reminder/internal/services"
	"github.com/google/uuid"
)

// DFMHandler handles "Don't Forget Me" note HTTP requests
type DFMHandler struct {
	noteRepo     repositories.DFMNoteRepository
	itemRepo     repositories.DFMItemRepository
	accountRepo  repositories.AccountRepository
	identityRepo repositories.IdentityRepository
}

// NewDFMHandler creates a new DFM handler
func NewDFMHandler(
	noteRepo repositories.DFMNoteRepository,
	itemRepo repositories.DFMItemRepository,
	accountRepo repositories.AccountRepository,
	identityRepo repositories.IdentityRepository,
) *DFMHandler {
	return &DFMHandler{
		noteRepo:     noteRepo,
		itemRepo:     itemRepo,
		accountRepo:  accountRepo,
		identityRepo: identityRepo,
	}
}

// DFMNoteResponse represents a DFM note in API responses with decoded recurrence
type DFMNoteResponse struct {
	ID             uuid.UUID        `json:"id"`
	RemindAtUTC    *time.Time       `json:"remind_at_utc,omitempty"`
	NextFireUTC    *time.Time       `json:"next_fire_utc,omitempty"`
	RecurrenceType string           `json:"recurrence_type"`
	HasReminder    bool             `json:"has_reminder"`
	Destinations   []string         `json:"destinations"`
	Items          []models.DFMItem `json:"items"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// ToDFMNoteResponse converts a DFMNote model to its API representation
func ToDFMNoteResponse(note *models.DFMNote) *DFMNoteResponse {
	items := note.Items
	if items == nil {
		items = []models.DFMItem{}
	}
	recurrenceType := services.GetRecurrenceType(int(note.Recurrence))
	destinations := []string{}
	for _, destination := range note.Destinations() {
		destinations = append(destinations, destination.String())
	}
	return &DFMNoteResponse{
		ID:             note.ID,
		RemindAtUTC:    note.RemindAtUTC,
		NextFireUTC:    note.NextFireUTC,
		RecurrenceType: services.GetRecurrenceTypeName(recurrenceType),
		HasReminder:    note.HasReminder(),
		Destinations:   destinations,
		Items:          items,
		CreatedAt:      note.CreatedAt,
		UpdatedAt:      note.UpdatedAt,
	}
}

// getOrCreateNote loads the account's note with items, creating it on first access
func (h *DFMHandler) getOrCreateNote(accountID uuid.UUID) (*models.DFMNote, error) {
	if _, err := h.noteRepo.GetOrCreateByAccountID(accountID); err != nil {
		return nil, err
	}
	return h.noteRepo.GetWithItems(accountID)
}

// GetNote returns the account's note with its items
func (h *DFMHandler) GetNote(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)

	note, err := h.getOrCreateNote(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch note")
		return
	}

	WriteJSON(w, http.StatusOK, ToDFMNoteResponse(note))
}

// AddItem appends a new item to the account's note
func (h *DFMHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		WriteError(w, http.StatusBadRequest, "Content is required")
		return
	}

	note, err := h.noteRepo.GetOrCreateByAccountID(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch note")
		return
	}

	itemCount, err := h.itemRepo.CountByNoteID(note.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to check item count")
		return
	}
	if itemCount >= services.MaxDFMItemsPerNote {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("You have reached the maximum of %d items", services.MaxDFMItemsPerNote))
		return
	}

	item := &models.DFMItem{
		NoteID:  note.ID,
		Content: req.Content,
	}
	if err := h.itemRepo.Create(item); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to create item")
		return
	}

	WriteJSON(w, http.StatusCreated, item)
}

// getOwnedItem fetches an item and verifies it belongs to the account's note
func (h *DFMHandler) getOwnedItem(accountID uuid.UUID, itemIDStr string) (*models.DFMItem, int, string) {
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		return nil, http.StatusBadRequest, "Invalid item ID"
	}

	item, err := h.itemRepo.GetByID(itemID)
	if err != nil {
		return nil, http.StatusInternalServerError, "Failed to fetch item"
	}
	if item == nil {
		return nil, http.StatusNotFound, "Item not found"
	}

	note, err := h.noteRepo.GetByID(item.NoteID)
	if err != nil {
		return nil, http.StatusInternalServerError, "Failed to fetch note"
	}
	if note == nil || note.AccountID != accountID {
		return nil, http.StatusNotFound, "Item not found"
	}

	return item, 0, ""
}

// UpdateItem updates an item's content and/or checked state
func (h *DFMHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)

	item, status, msg := h.getOwnedItem(accountID, r.PathValue("id"))
	if item == nil {
		WriteError(w, status, msg)
		return
	}

	var req struct {
		Content *string `json:"content"`
		Checked *bool   `json:"checked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Content != nil {
		content := strings.TrimSpace(*req.Content)
		if content == "" {
			WriteError(w, http.StatusBadRequest, "Content cannot be empty")
			return
		}
		item.Content = content
	}
	if req.Checked != nil {
		item.Checked = *req.Checked
	}

	if err := h.itemRepo.Update(item); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update item")
		return
	}

	WriteJSON(w, http.StatusOK, item)
}

// DeleteItem removes an item from the note
func (h *DFMHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)

	item, status, msg := h.getOwnedItem(accountID, r.PathValue("id"))
	if item == nil {
		WriteError(w, status, msg)
		return
	}

	if err := h.itemRepo.Delete(item.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to delete item")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Item deleted successfully"})
}

// SetReminder configures the recurring reminder of the note
func (h *DFMHandler) SetReminder(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)

	var req struct {
		Date         string   `json:"date"`         // optional, ISO 8601, defaults to today
		Time         string   `json:"time"`         // HH:mm format
		Recurrence   string   `json:"recurrence"`   // "DAILY", "WEEKLY", "MONTHLY", "YEARLY", ...
		Destinations []string `json:"destinations"` // any of "discord_dm", "email"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Time == "" {
		WriteError(w, http.StatusBadRequest, "Time is required")
		return
	}

	recurrenceValue, exists := services.RecurrenceTypeMap[strings.ToUpper(req.Recurrence)]
	if !exists {
		WriteError(w, http.StatusBadRequest, "Invalid recurrence type")
		return
	}

	if len(req.Destinations) == 0 {
		WriteError(w, http.StatusBadRequest, "At least one destination is required")
		return
	}
	sendDiscordDM, sendEmail := false, false
	for _, destinationStr := range req.Destinations {
		destination := models.DestinationType(destinationStr)
		if !models.IsValidDFMDestination(destination) {
			WriteError(w, http.StatusBadRequest, "Invalid destination: must be discord_dm or email")
			return
		}
		if destination == models.DestinationDiscordDM {
			sendDiscordDM = true
		} else {
			sendEmail = true
		}
	}

	account, err := h.accountRepo.GetWithTimezone(accountID)
	if err != nil || account == nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve account")
		return
	}
	if account.Timezone == nil {
		WriteError(w, http.StatusBadRequest, "Account timezone not set")
		return
	}

	// Each chosen destination must be available on the account
	identities, err := h.identityRepo.GetByAccountID(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve identities")
		return
	}
	hasDiscord := false
	for _, identity := range identities {
		if identity.Provider == models.ProviderDiscord {
			hasDiscord = true
		}
	}
	if sendDiscordDM && !hasDiscord {
		WriteError(w, http.StatusBadRequest, "No Discord account linked")
		return
	}
	if sendEmail && account.Email == nil {
		WriteError(w, http.StatusBadRequest, "No email linked to this account")
		return
	}

	firstFire, err := services.ComputeDFMReminderSchedule(req.Date, req.Time, recurrenceValue, account.Timezone.IANALocation)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid date/time: "+err.Error())
		return
	}

	note, err := h.noteRepo.GetOrCreateByAccountID(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch note")
		return
	}

	note.RemindAtUTC = &firstFire
	note.NextFireUTC = &firstFire
	note.Recurrence = int16(recurrenceValue)
	note.SendDiscordDM = sendDiscordDM
	note.SendEmail = sendEmail

	if err := h.noteRepo.Update(note); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to set reminder")
		return
	}

	noteWithItems, err := h.noteRepo.GetWithItems(accountID)
	if err != nil || noteWithItems == nil {
		WriteJSON(w, http.StatusOK, ToDFMNoteResponse(note))
		return
	}
	WriteJSON(w, http.StatusOK, ToDFMNoteResponse(noteWithItems))
}

// RemoveReminder clears the reminder of the note
func (h *DFMHandler) RemoveReminder(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)

	note, err := h.noteRepo.GetOrCreateByAccountID(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch note")
		return
	}

	note.RemindAtUTC = nil
	note.NextFireUTC = nil
	note.Recurrence = 0

	if err := h.noteRepo.Update(note); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to remove reminder")
		return
	}

	noteWithItems, err := h.noteRepo.GetWithItems(accountID)
	if err != nil || noteWithItems == nil {
		WriteJSON(w, http.StatusOK, ToDFMNoteResponse(note))
		return
	}
	WriteJSON(w, http.StatusOK, ToDFMNoteResponse(noteWithItems))
}

// SendNow dispatches the note immediately without touching the schedule
func (h *DFMHandler) SendNow(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)

	// Make sure the note exists before trying to send it
	if _, err := h.noteRepo.GetOrCreateByAccountID(accountID); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch note")
		return
	}

	if err := engine.SendDFMNoteNow(accountID); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to send the note: "+err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Note sent successfully"})
}

