package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/database/repositories"
	"github.com/ericp/chronos-bot-reminder/internal/services"
	"github.com/google/uuid"
)

// CreateReminderRequest represents the request body for creating a reminder
type CreateReminderRequest struct {
	Date         string          `json:"date"`         // ISO 8601 date format
	Time         string          `json:"time"`         // HH:mm format
	Message      string          `json:"message"`
	Recurrence   json.RawMessage `json:"recurrence"`   // Can be string ("DAILY") or int (4)
	Destinations []CreateDestinationRequest   `json:"destinations"`
}

// CreateDestinationRequest represents a destination to create
type CreateDestinationRequest struct {
	Type     string                 `json:"type"` // "discord_dm", "discord_channel", "webhook"
	Metadata map[string]interface{} `json:"metadata"`
}

// CreateReminderResponse represents the response after creating a reminder
type CreateReminderResponse struct {
	ID              uuid.UUID         `json:"id"`
	Message         string            `json:"message"`
	RemindAtUTC     time.Time         `json:"remind_at_utc"`
	RecurrenceType  string            `json:"recurrence_type"`
	IsPaused        bool              `json:"is_paused"`
	Destinations    []interface{}     `json:"destinations"`
}

// ReminderResponse represents a reminder in API responses with decoded recurrence
type ReminderResponse struct {
	ID              uuid.UUID              `json:"id"`
	AccountID       uuid.UUID              `json:"account_id"`
	RemindAtUTC     time.Time              `json:"remind_at_utc"`
	SnoozedAtUTC    *time.Time             `json:"snoozed_at_utc,omitempty"`
	NextFireUTC     *time.Time             `json:"next_fire_utc,omitempty"`
	Message         string                 `json:"message"`
	CreatedAt       time.Time              `json:"created_at"`
	RecurrenceType  string                 `json:"recurrence_type"`
	IsPaused        bool                   `json:"is_paused"`
	Destinations    []models.ReminderDestination `json:"destinations,omitempty"`
}

// ToReminderResponse converts a Reminder model to ReminderResponse with decoded recurrence
func ToReminderResponse(reminder *models.Reminder) *ReminderResponse {
	recurrenceType := services.GetRecurrenceType(int(reminder.Recurrence))
	return &ReminderResponse{
		ID:             reminder.ID,
		AccountID:      reminder.AccountID,
		RemindAtUTC:    reminder.RemindAtUTC,
		SnoozedAtUTC:   reminder.SnoozedAtUTC,
		NextFireUTC:    reminder.NextFireUTC,
		Message:        reminder.Message,
		CreatedAt:      reminder.CreatedAt,
		RecurrenceType: services.GetRecurrenceTypeName(recurrenceType),
		IsPaused:       services.IsPaused(int(reminder.Recurrence)),
		Destinations:   reminder.Destinations,
	}
}

// UserHandler handles user-related requests
type UserHandler struct {
	reminderRepo            repositories.ReminderRepository
	reminderErrorRepo       repositories.ReminderErrorRepository
	reminderDestinationRepo repositories.ReminderDestinationRepository
	accountRepo             repositories.AccountRepository
	identityRepo            repositories.IdentityRepository
	timezoneRepo            repositories.TimezoneRepository
	sessionService          *services.SessionService
	discordOAuthService     *services.DiscordOAuthService
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	reminderRepo repositories.ReminderRepository,
	reminderErrorRepo repositories.ReminderErrorRepository,
	accountRepo repositories.AccountRepository,
	sessionService *services.SessionService,
) *UserHandler {
	return &UserHandler{
		reminderRepo:      reminderRepo,
		reminderErrorRepo: reminderErrorRepo,
		accountRepo:       accountRepo,
		sessionService:    sessionService,
	}
}

// SetReminderDestinationRepository sets the reminder destination repository
func (h *UserHandler) SetReminderDestinationRepository(repo repositories.ReminderDestinationRepository) {
	h.reminderDestinationRepo = repo
}

// SetIdentityRepository sets the identity repository
func (h *UserHandler) SetIdentityRepository(repo repositories.IdentityRepository) {
	h.identityRepo = repo
}

// SetTimezoneRepository sets the timezone repository
func (h *UserHandler) SetTimezoneRepository(repo repositories.TimezoneRepository) {
	h.timezoneRepo = repo
}

// SetDiscordOAuthService sets the Discord OAuth service (used for avatar refresh)
func (h *UserHandler) SetDiscordOAuthService(svc *services.DiscordOAuthService) {
	h.discordOAuthService = svc
}

// CreateReminder creates a new reminder for the authenticated user
// @Route: POST /api/reminders
func (h *UserHandler) CreateReminder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract account ID from token
	accountID, err := h.extractAccountIDFromToken(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Get account with timezone to access timezone information
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
	var req CreateReminderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	defer r.Body.Close()

	// Validate required fields
	if req.Message == "" {
		WriteError(w, http.StatusBadRequest, "Message is required")
		return
	}

	if req.Date == "" || req.Time == "" {
		WriteError(w, http.StatusBadRequest, "Date and time are required")
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

	// Parse the reminder date and time in user's timezone
	location, err := time.LoadLocation(account.Timezone.IANALocation)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("Invalid timezone: %s", account.Timezone.IANALocation))
		return
	}

	parsedTime, err := services.ParseReminderDateTimeInTimezone(req.Date, req.Time, account.Timezone.IANALocation)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid date/time format")
		return
	}

	// Check if the reminder time is in the future
	now := time.Now().In(location)
	if parsedTime.Before(now) {
		WriteError(w, http.StatusBadRequest, "Reminder date/time must be in the future")
		return
	}

	// Handle recurrence - can be string or int
	var recurrenceValue int16
	if len(req.Recurrence) > 0 {
		var recurrenceStr string
		if err := json.Unmarshal(req.Recurrence, &recurrenceStr); err == nil {
			// It's a string, convert using the map
			if val, exists := services.RecurrenceTypeMap[recurrenceStr]; exists {
				recurrenceValue = int16(val)
			} else {
				WriteError(w, http.StatusBadRequest, "Invalid recurrence type")
				return
			}
		} else {
			// Try to parse as int (legacy format)
			var recurrenceInt int
			if err := json.Unmarshal(req.Recurrence, &recurrenceInt); err != nil {
				WriteError(w, http.StatusBadRequest, "Invalid recurrence format")
				return
			}
			recurrenceValue = int16(recurrenceInt)
		}
	}

	// Create the reminder with UTC time
	reminder := &models.Reminder{
		AccountID:   accountID,
		RemindAtUTC: parsedTime.UTC(),
		Message:     req.Message,
		Recurrence:  recurrenceValue,
	}

	// Save the reminder to database
	if err := h.reminderRepo.Create(reminder, true); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to create reminder")
		return
	}

	// Process destinations
	var destinations []interface{}
	for _, dest := range req.Destinations {
		destType := models.DestinationType(dest.Type)

		// Validate destination type
		if !destType.IsValid() {
			continue
		}

		// Handle discord_dm destination
		if destType == models.DestinationDiscordDM {
			// If user_id not provided, get it from the Discord identity
			if _, exists := dest.Metadata["user_id"]; !exists {
				account, err := h.accountRepo.GetWithIdentities(accountID)
				if err == nil && account != nil {
					for _, identity := range account.Identities {
						if identity.Provider == "discord" {
							dest.Metadata["user_id"] = identity.ExternalID
							break
						}
					}
				}
				// Skip if still no user_id
				if _, exists := dest.Metadata["user_id"]; !exists {
					continue
				}
			}
		}

		// Handle discord_channel destination
		if destType == models.DestinationDiscordChannel {
			// Validate required fields
			if _, hasGuild := dest.Metadata["guild_id"]; !hasGuild {
				continue
			}
			if _, hasChannel := dest.Metadata["channel_id"]; !hasChannel {
				continue
			}
			// mention_role_id is optional
		}

		// Handle webhook destination
		if destType == models.DestinationWebhook {
			// Validate required fields
			if _, hasURL := dest.Metadata["url"]; !hasURL {
				continue
			}

			// Validate optional platform field
			if platformVal, exists := dest.Metadata["platform"]; exists {
				if platformStr, ok := platformVal.(string); ok {
					platform := models.WebhookPlatform(platformStr)
					if !platform.IsValid() {
						// Invalid platform, skip this destination
						continue
					}
				}
			}
		}

		// Handle email destination
		if destType == models.DestinationEmail {
			if _, hasEmail := dest.Metadata["email"]; !hasEmail {
				// Auto-fill from account-level email
				acct, err := h.accountRepo.GetByID(accountID)
				if err == nil && acct != nil && acct.Email != nil {
					dest.Metadata["email"] = *acct.Email
				}
				if _, still := dest.Metadata["email"]; !still {
					continue
				}
			}

			// Guard: disallow email for hourly (or shorter) recurrence to avoid flooding Resend free tier
			recurrenceType := services.GetRecurrenceType(int(recurrenceValue))
			if recurrenceType == services.RecurrenceHourly {
				WriteError(w, http.StatusBadRequest, "Email destination cannot be used with hourly recurrence")
				return
			}
		}

		// Handle android_push destination - always inject account_id from auth context
		if destType == models.DestinationAndroidPush {
			if dest.Metadata == nil {
				dest.Metadata = models.JSONB{}
			}
			dest.Metadata["account_id"] = accountID.String()
		}

		// Create the destination
		reminderDest := &models.ReminderDestination{
			ReminderID: reminder.ID,
			Type:       destType,
			Metadata:   dest.Metadata,
		}

		if err := h.reminderDestinationRepo.Create(reminderDest); err != nil {
			// Log error but continue - don't fail the entire operation
			fmt.Printf("[CREATE_REMINDER] Failed to create destination: %v\n", err)
			continue
		}

		destinations = append(destinations, reminderDest)
	}

	// If no destinations were provided or valid, create a default discord_dm destination
	// Get the Discord identity for the user
	if len(destinations) == 0 {
		account, err := h.accountRepo.GetWithIdentities(accountID)
		if err == nil && account != nil && len(account.Identities) > 0 {
			// Find Discord identity
			for _, identity := range account.Identities {
				if identity.Provider == "discord" {
					reminderDest := &models.ReminderDestination{
						ReminderID: reminder.ID,
						Type:       models.DestinationDiscordDM,
						Metadata: models.JSONB{
							"user_id": identity.ExternalID,
						},
					}

					if err := h.reminderDestinationRepo.Create(reminderDest); err == nil {
						destinations = append(destinations, reminderDest)
					}
					break
				}
			}
		}
	}

	// Build response with decoded recurrence
	recurrenceType := services.GetRecurrenceType(int(reminder.Recurrence))
	isPaused := services.IsPaused(int(reminder.Recurrence))

	response := CreateReminderResponse{
		ID:             reminder.ID,
		Message:        reminder.Message,
		RemindAtUTC:    reminder.RemindAtUTC,
		RecurrenceType: services.GetRecurrenceTypeName(recurrenceType),
		IsPaused:       isPaused,
		Destinations:   destinations,
	}

	WriteJSON(w, http.StatusCreated, response)
}

// GetReminders retrieves all reminders for the authenticated user with their destinations
// @Route: GET /api/reminders
func (h *UserHandler) GetReminders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accountID, err := h.extractAccountIDFromToken(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	reminders, err := h.reminderRepo.GetByAccountIDWithDestinations(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve reminders")
		return
	}

	// Convert reminders to response format with decoded recurrence
	reminderResponses := make([]*ReminderResponse, len(reminders))
	for i, reminder := range reminders {
		reminderResponses[i] = ToReminderResponse(&reminder)
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"reminders": reminderResponses,
		"count":     len(reminderResponses),
	})
}

// GetReminder retrieves a single reminder by ID for the authenticated user
// @Route: GET /api/reminders/{id}
func (h *UserHandler) GetReminder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accountID, err := h.extractAccountIDFromToken(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	reminderIDStr := strings.TrimPrefix(r.URL.Path, "/api/reminders/")
	reminderID, err := uuid.Parse(reminderIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid reminder ID")
		return
	}

	reminder, err := h.reminderRepo.GetWithAccountAndDestinations(reminderID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve reminder")
		return
	}

	if reminder == nil {
		WriteError(w, http.StatusNotFound, "Reminder not found")
		return
	}

	// Verify ownership
	if reminder.AccountID != accountID {
		WriteError(w, http.StatusForbidden, "You do not have permission to access this reminder")
		return
	}

	WriteJSON(w, http.StatusOK, ToReminderResponse(reminder))
}

// GetReminderErrors retrieves all reminders with errors for the authenticated user
// @Route: GET /api/reminders/errors
func (h *UserHandler) GetReminderErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accountID, err := h.extractAccountIDFromToken(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Get all reminders for the user
	reminders, err := h.reminderRepo.GetByAccountIDWithDestinations(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve reminders")
		return
	}

	// Collect all reminder IDs
	reminderIDs := make([]uuid.UUID, 0, len(reminders))
	for _, reminder := range reminders {
		reminderIDs = append(reminderIDs, reminder.ID)
	}

	// Get all errors for these reminders
	reminderErrors := make([]models.ReminderError, 0)
	for _, reminderID := range reminderIDs {
		errs, err := h.reminderErrorRepo.GetByReminderID(reminderID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "Failed to retrieve reminder errors")
			return
		}
		// Append each error individually and handle both value and pointer element types
		for _, e := range errs {
			switch v := interface{}(e).(type) {
			case models.ReminderError:
				reminderErrors = append(reminderErrors, v)
			case *models.ReminderError:
				if v != nil {
					reminderErrors = append(reminderErrors, *v)
				}
			default:
				// ignore unexpected types
			}
		}
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"errors": reminderErrors,
		"count":  len(reminderErrors),
	})
}

// GetAccount retrieves the authenticated user's account information with identities
// @Route: GET /api/account
func (h *UserHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accountID, err := h.extractAccountIDFromToken(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	account, err := h.accountRepo.GetWithIdentities(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve account")
		return
	}

	if account == nil {
		WriteError(w, http.StatusNotFound, "Account not found")
		return
	}

	// Background-refresh the Discord avatar/username so the snapshot stays current
	// even when the user changes their Discord profile picture between logins.
	if h.discordOAuthService != nil {
		for i := range account.Identities {
			if account.Identities[i].Provider == models.ProviderDiscord {
				identity := &account.Identities[i]
				go h.discordOAuthService.RefreshDiscordSnapshot(context.Background(), identity)
				break
			}
		}
	}

	WriteJSON(w, http.StatusOK, account)
}

// AddAppIdentity creates an email/password (app) identity for the currently
// authenticated account. This is the inverse of linking Discord: it lets a
// Discord-first (or mobile-first) account add email/password login so the same
// account can be reached from every surface.
// @Route: POST /api/account/identity/app
func (h *UserHandler) AddAppIdentity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accountID, err := h.extractAccountIDFromToken(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req struct {
		Email    string `json:"email"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	defer r.Body.Close()

	req.Email = strings.TrimSpace(req.Email)
	req.Username = strings.TrimSpace(req.Username)
	if req.Email == "" {
		WriteError(w, http.StatusBadRequest, "Email is required")
		return
	}
	if req.Username == "" {
		WriteError(w, http.StatusBadRequest, "Username is required")
		return
	}
	if len(req.Password) < 8 {
		WriteError(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	account, err := h.accountRepo.GetWithIdentities(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve account")
		return
	}
	if account == nil {
		WriteError(w, http.StatusNotFound, "Account not found")
		return
	}

	// Reject if this account already has email/password credentials.
	if account.Email != nil {
		WriteError(w, http.StatusConflict, "This account already has email/password login")
		return
	}

	// Reject if the email is already used by another account.
	existing, err := h.accountRepo.GetByEmail(req.Email)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to check email availability")
		return
	}
	if existing != nil {
		WriteError(w, http.StatusConflict, "This email is already in use")
		return
	}

	hashedPassword, err := services.HashPassword(req.Password)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to secure password")
		return
	}

	account.Email = &req.Email
	account.Username = &req.Username
	account.PasswordHash = &hashedPassword
	account.EmailVerified = true
	if err := h.accountRepo.Update(account); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to add login credentials")
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]string{"message": "Login credentials added successfully"})
}

// ChangeAppIdentityPassword changes the password for the app identity
// @Route: POST /api/account/identity/app/change-password
func (h *UserHandler) ChangeAppIdentityPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accountID, err := h.extractAccountIDFromToken(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Parse request body
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	defer r.Body.Close()

	// Validate inputs
	if req.CurrentPassword == "" || req.NewPassword == "" {
		WriteError(w, http.StatusBadRequest, "Current and new passwords are required")
		return
	}

	if len(req.NewPassword) < 8 {
		WriteError(w, http.StatusBadRequest, "New password must be at least 8 characters long")
		return
	}

	account, err := h.accountRepo.GetByID(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve account")
		return
	}

	if account == nil {
		WriteError(w, http.StatusNotFound, "Account not found")
		return
	}

	if account.PasswordHash == nil {
		WriteError(w, http.StatusBadRequest, "This account does not have a password set")
		return
	}

	// Verify current password
	if err := services.VerifyPassword(*account.PasswordHash, req.CurrentPassword); err != nil {
		WriteError(w, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	// Hash the new password
	hashedPassword, err := services.HashPassword(req.NewPassword)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	account.PasswordHash = &hashedPassword
	if err := h.accountRepo.Update(account); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Password updated successfully",
	})
}

// UpdateAccountTimezone updates the timezone for the authenticated user's account
// @Route: PUT /api/account/timezone
func (h *UserHandler) UpdateAccountTimezone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accountID, err := h.extractAccountIDFromToken(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Parse request body
	var req struct {
		Timezone string `json:"timezone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	defer r.Body.Close()

	// Validate timezone is not empty
	if req.Timezone == "" {
		WriteError(w, http.StatusBadRequest, "Timezone is required")
		return
	}

	// Validate timezone is a valid IANA location
	if _, err := time.LoadLocation(req.Timezone); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid timezone")
		return
	}

	// Get the timezone from database
	timezone, err := h.timezoneRepo.GetByIANALocation(req.Timezone)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve timezone")
		return
	}

	if timezone == nil {
		WriteError(w, http.StatusBadRequest, "Timezone not found in database")
		return
	}

	// Get the account
	account, err := h.accountRepo.GetByID(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve account")
		return
	}

	if account == nil {
		WriteError(w, http.StatusNotFound, "Account not found")
		return
	}

	// Update the timezone ID
	account.TimezoneID = &timezone.ID

	if err := h.accountRepo.Update(account); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update timezone")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Timezone updated successfully",
	})
}

// UpdatePreferences updates a subset of the account's free-form preferences.
// Currently supports "discord_send_image", which requires a linked Discord
// identity since it only affects Discord reminder delivery.
// @Route: PUT /api/account/preferences
func (h *UserHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accountID, err := h.extractAccountIDFromToken(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req struct {
		DiscordSendImage *bool `json:"discord_send_image"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	defer r.Body.Close()

	if req.DiscordSendImage == nil {
		WriteError(w, http.StatusBadRequest, "No preference provided")
		return
	}

	account, err := h.accountRepo.GetWithIdentities(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve account")
		return
	}
	if account == nil {
		WriteError(w, http.StatusNotFound, "Account not found")
		return
	}

	hasDiscord := false
	for _, identity := range account.Identities {
		if identity.Provider == models.ProviderDiscord {
			hasDiscord = true
			break
		}
	}
	if !hasDiscord {
		WriteError(w, http.StatusBadRequest, "A linked Discord identity is required to change this preference")
		return
	}

	if account.Preferences == nil {
		account.Preferences = models.JSONB{}
	}
	account.Preferences[models.PrefDiscordSendImage] = *req.DiscordSendImage

	if err := h.accountRepo.Update(account); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update preferences")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message":     "Preferences updated successfully",
		"preferences": account.Preferences,
	})
}

// UpdateAppIdentityUsername updates the username for the app identity
// @Route: PUT /api/account/identity/app/username
func (h *UserHandler) UpdateAppIdentityUsername(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accountID, err := h.extractAccountIDFromToken(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Parse request body
	var req struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	defer r.Body.Close()

	// Validate username is not empty
	if req.Username == "" {
		WriteError(w, http.StatusBadRequest, "Username is required")
		return
	}

	account, err := h.accountRepo.GetByID(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve account")
		return
	}

	if account == nil {
		WriteError(w, http.StatusNotFound, "Account not found")
		return
	}

	account.Username = &req.Username
	if err := h.accountRepo.Update(account); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update username")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Username updated successfully",
	})
}

// UpdateAppIdentityEmail updates the email for the app identity
// @Route: PUT /api/account/identity/app/email
func (h *UserHandler) UpdateAppIdentityEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accountID, err := h.extractAccountIDFromToken(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Parse request body
	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	defer r.Body.Close()

	// Validate email is not empty
	if req.Email == "" {
		WriteError(w, http.StatusBadRequest, "Email is required")
		return
	}

	// Get the account with identities
	account, err := h.accountRepo.GetWithIdentities(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve account")
		return
	}

	if account == nil {
		WriteError(w, http.StatusNotFound, "Account not found")
		return
	}

	if account.Email == nil {
		WriteError(w, http.StatusNotFound, "This account does not have an email login")
		return
	}

	account.Email = &req.Email
	if err := h.accountRepo.Update(account); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update email")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Email updated successfully",
	})
}

// DeleteAccount deletes the authenticated user's account and all associated data
// @Route: DELETE /api/account
func (h *UserHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accountID, err := h.extractAccountIDFromToken(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Get the account to verify it exists
	account, err := h.accountRepo.GetByID(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve account")
		return
	}

	if account == nil {
		WriteError(w, http.StatusNotFound, "Account not found")
		return
	}

	// Delete the account (cascade deletes should handle related data)
	if err := h.accountRepo.Delete(accountID); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to delete account")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Account deleted successfully",
	})
}

// DeleteReminder deletes a reminder for the authenticated user
// @Route: DELETE /api/reminders/{id}
func (h *UserHandler) DeleteReminder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accountID, err := h.extractAccountIDFromToken(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	reminderIDStr := strings.TrimPrefix(r.URL.Path, "/api/reminders/")
	reminderID, err := uuid.Parse(reminderIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid reminder ID")
		return
	}

	// Get the reminder to verify ownership
	reminder, err := h.reminderRepo.GetByID(reminderID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to retrieve reminder")
		return
	}

	if reminder == nil {
		WriteError(w, http.StatusNotFound, "Reminder not found")
		return
	}

	// Verify ownership
	if reminder.AccountID != accountID {
		WriteError(w, http.StatusForbidden, "You do not have permission to delete this reminder")
		return
	}

	// Delete the reminder with notification enabled
	if err := h.reminderRepo.Delete(reminderID, true); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to delete reminder")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Reminder deleted successfully",
	})
}

// EnsureMobileIdentity creates a "mobile" identity for the calling account the
// first time it connects from the mobile app. Idempotent: a single mobile
// identity per account (keyed by account id), so repeated logins are no-ops.
// @Route: POST /api/account/identity/mobile
func (h *UserHandler) EnsureMobileIdentity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	accountID, err := h.extractAccountIDFromToken(r)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Optional friendly device label from the client.
	var req struct {
		DeviceName string `json:"device_name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	defer r.Body.Close()

	// One mobile identity per account; external_id is the account id so the
	// (provider, external_id) unique index makes this idempotent.
	externalID := accountID.String()
	existing, err := h.identityRepo.GetByProviderAndExternalID(models.ProviderMobile, externalID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to check mobile identity")
		return
	}

	if existing != nil {
		// Refresh the device label if a new one was provided.
		if req.DeviceName != "" && (existing.Username == nil || *existing.Username != req.DeviceName) {
			existing.Username = &req.DeviceName
			_ = h.identityRepo.Update(existing)
		}
		WriteJSON(w, http.StatusOK, existing)
		return
	}

	username := req.DeviceName
	if username == "" {
		username = "Mobile App"
	}

	identity := &models.Identity{
		AccountID:  accountID,
		Provider:   models.ProviderMobile,
		ExternalID: externalID,
		Username:   &username,
	}
	if err := h.identityRepo.Create(identity); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to create mobile identity")
		return
	}

	WriteJSON(w, http.StatusCreated, identity)
}

// extractAccountIDFromToken extracts the account ID from the JWT token in the request or context
func (h *UserHandler) extractAccountIDFromToken(r *http.Request) (uuid.UUID, error) {
	// First, check if account ID is already in context (set by API key middleware)
	if accountID, ok := r.Context().Value(AccountIDKey).(uuid.UUID); ok {
		return accountID, nil
	}

	// Try to get token from Authorization header first
	authHeader := r.Header.Get("Authorization")
	var token string

	if authHeader != "" {
		// Extract Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			token = parts[1]
		}
	}

	// If no token in header, try cookie
	if token == "" {
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			return uuid.Nil, ErrUnauthorized
		}
		token = cookie.Value
	}

	if token == "" {
		return uuid.Nil, ErrUnauthorized
	}

	// Validate token and extract claims
	claims, err := h.sessionService.ValidateToken(token)
	if err != nil {
		return uuid.Nil, ErrUnauthorized
	}

	// Parse account ID from claims
	accountID, err := uuid.Parse(claims.AccountID)
	if err != nil {
		return uuid.Nil, ErrUnauthorized
	}

	return accountID, nil
}

// AuthMiddleware creates a middleware that validates JWT tokens or API keys
func AuthMiddleware(sessionService *services.SessionService, apiKeyService *services.APIKeyService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get token from Authorization header first
			authHeader := r.Header.Get("Authorization")
			var token string

			if authHeader != "" {
				// Extract Bearer token
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					token = parts[1]
				}
			}

			// If no token in header, try cookie
			if token == "" {
				cookie, err := r.Cookie("auth_token")
				if err != nil {
					WriteError(w, http.StatusUnauthorized, "No authentication token found")
					return
				}
				token = cookie.Value
			}

			if token == "" {
				WriteError(w, http.StatusUnauthorized, "No authentication token found")
				return
			}

			// Check if this is an API key (starts with "ck_")
			if strings.HasPrefix(token, "ck_") {
				// Validate API key
				accountID, err := apiKeyService.ValidateAPIKey(token)
				if err != nil {
					WriteError(w, http.StatusUnauthorized, "Invalid API key")
					return
				}
				if accountID == uuid.Nil {
					WriteError(w, http.StatusUnauthorized, "Invalid API key")
					return
				}
				// Add account ID to request context
				ctx := context.WithValue(r.Context(), AccountIDKey, accountID)
				*r = *r.WithContext(ctx)
				next.ServeHTTP(w, r)
				return
			}

			// Validate token and extract claims
			claims, err := sessionService.ValidateToken(token)
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			// Extract account ID from claims
			accountID, err := uuid.Parse(claims.AccountID)
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "Invalid token claims")
				return
			}

			// Add account ID to request context
			ctx := context.WithValue(r.Context(), AccountIDKey, accountID)
			*r = *r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
