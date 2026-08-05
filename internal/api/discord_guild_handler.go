package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/services"
	"github.com/google/uuid"
)

// DiscordGuildHandler handles Discord guild-related operations
type DiscordGuildHandler struct {
	discordOAuthService *services.DiscordOAuthService
}

// NewDiscordGuildHandler creates a new Discord guild handler
func NewDiscordGuildHandler(
	discordOAuthService *services.DiscordOAuthService,
) *DiscordGuildHandler {
	return &DiscordGuildHandler{
		discordOAuthService: discordOAuthService,
	}
}

// GetUserGuildsRequest represents a request to get user guilds
type GetUserGuildsRequest struct {
	AccountID string `json:"account_id"`
}

// GetUserGuildsResponse represents the response with user guilds
type GetUserGuildsResponse struct {
	Guilds []GuildData `json:"guilds"`
	Error  string      `json:"error,omitempty"`
}

// GuildData represents guild information with metadata
type GuildData struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Icon        string   `json:"icon"`
	Owner       bool     `json:"owner"`
	Permissions int64    `json:"permissions"`
	Features    []string `json:"features"`
}

// GetGuildChannelsResponse represents the response with guild channels
type GetGuildChannelsResponse struct {
	Channels []ChannelData `json:"channels"`
	Error    string        `json:"error,omitempty"`
}

// ChannelData represents channel information
type ChannelData struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     int     `json:"type"`
	Position int     `json:"position"`
	Topic    *string `json:"topic"`
}

// GetGuildRolesResponse represents the response with guild roles
type GetGuildRolesResponse struct {
	Roles []RoleData `json:"roles"`
	Error string     `json:"error,omitempty"`
}

// RoleData represents role information
type RoleData struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Position    int    `json:"position"`
	Permissions int64  `json:"permissions"`
	Managed     bool   `json:"managed"`
	Mentionable bool   `json:"mentionable"`
}

// GetUserGuilds retrieves all guilds for the authenticated user
func (h *DiscordGuildHandler) GetUserGuilds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req GetUserGuildsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if strings.TrimSpace(req.AccountID) == "" {
		WriteError(w, http.StatusBadRequest, "Account ID is required")
		return
	}

	// Parse account ID
	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid account ID format")
		return
	}

	// Get account with identities
	account, err := h.discordOAuthService.GetAccount(r.Context(), accountID)
	if err != nil || account == nil {
		WriteError(w, http.StatusNotFound, "Account not found")
		return
	}

	// Find Discord identity with access token
	var discordIdentity *models.Identity
	for i := range account.Identities {
		if account.Identities[i].Provider == models.ProviderDiscord && account.Identities[i].AccessToken != nil {
			discordIdentity = &account.Identities[i]
			break
		}
	}

	// Note: this is about the Discord OAuth token, not the Chronos session,
	// so it must never be reported as 401 — the frontend's global HTTP
	// client treats any 401 as "your session expired" and force-logs the
	// user out of the whole app.
	if discordIdentity == nil {
		WriteError(w, http.StatusConflict, "No Discord identity with access token found for this account. Please reconnect Discord.")
		return
	}

	accessToken := *discordIdentity.AccessToken

	// Get guilds from Discord
	guilds, err := h.discordOAuthService.GetUserGuilds(r.Context(), accessToken)

	// If we get a 401 from Discord, try to refresh the token
	if err != nil && strings.Contains(err.Error(), "status 401") {
		newAccessToken, refreshErr := h.discordOAuthService.RefreshAndPersistDiscordToken(r.Context(), discordIdentity)
		if refreshErr != nil {
			fmt.Printf("[GUILD_HANDLER] Error refreshing token: %v\n", refreshErr)
			WriteError(w, http.StatusConflict, "Discord token expired. Please reconnect Discord.")
			return
		}

		// Retry fetching guilds with new token
		guilds, err = h.discordOAuthService.GetUserGuilds(r.Context(), newAccessToken)
	}

	if err != nil {
		fmt.Printf("[GUILD_HANDLER] Error fetching guilds: %v\n", err)
		WriteError(w, http.StatusInternalServerError, "Failed to fetch guilds from Discord: "+err.Error())
		return
	}

	// Convert to response format
	guildData := make([]GuildData, len(guilds))
	for i, guild := range guilds {
		guildData[i] = GuildData{
			ID:          guild.ID,
			Name:        guild.Name,
			Icon:        guild.Icon,
			Owner:       guild.Owner,
			Permissions: guild.Permissions,
			Features:    guild.Features,
		}
	}

	resp := GetUserGuildsResponse{
		Guilds: guildData,
	}

	WriteJSON(w, http.StatusOK, resp)
}

// GetGuildChannels retrieves all channels for a specific guild
func (h *DiscordGuildHandler) GetGuildChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		AccountID string `json:"account_id"`
		GuildID   string `json:"guild_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if strings.TrimSpace(req.AccountID) == "" {
		WriteError(w, http.StatusBadRequest, "Account ID is required")
		return
	}

	if strings.TrimSpace(req.GuildID) == "" {
		WriteError(w, http.StatusBadRequest, "Guild ID is required")
		return
	}

	// Parse account ID
	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid account ID format")
		return
	}

	// Get account with identities
	account, err := h.discordOAuthService.GetAccount(r.Context(), accountID)
	if err != nil || account == nil {
		WriteError(w, http.StatusNotFound, "Account not found")
		return
	}

	// Find Discord identity with access token
	var discordIdentity *models.Identity
	for i := range account.Identities {
		if account.Identities[i].Provider == models.ProviderDiscord && account.Identities[i].AccessToken != nil {
			discordIdentity = &account.Identities[i]
			break
		}
	}

	// Note: this is about the Discord OAuth token, not the Chronos session,
	// so it must never be reported as 401 — the frontend's global HTTP
	// client treats any 401 as "your session expired" and force-logs the
	// user out of the whole app.
	if discordIdentity == nil {
		WriteError(w, http.StatusConflict, "No Discord identity with access token found for this account. Please reconnect Discord.")
		return
	}

	// Check if bot is in the guild
	botInGuild, err := h.discordOAuthService.IsBotInGuild(r.Context(), req.GuildID)
	if err != nil {
		fmt.Printf("[GUILD_HANDLER] Error checking bot guild membership: %v\n", err)
		WriteError(w, http.StatusInternalServerError, "Failed to verify bot guild membership")
		return
	}

	if !botInGuild {
		WriteError(w, http.StatusForbidden, "Bot is not a member of this guild. Please invite the bot to the guild first.")
		return
	}

	accessToken := *discordIdentity.AccessToken

	// Get channels from Discord
	channels, err := h.discordOAuthService.GetGuildChannels(r.Context(), accessToken, req.GuildID)

	// If we get a 401 from Discord, try to refresh the token
	if err != nil && strings.Contains(err.Error(), "status 401") {
		newAccessToken, refreshErr := h.discordOAuthService.RefreshAndPersistDiscordToken(r.Context(), discordIdentity)
		if refreshErr != nil {
			fmt.Printf("[GUILD_HANDLER] Error refreshing token: %v\n", refreshErr)
			WriteError(w, http.StatusConflict, "Discord token expired. Please reconnect Discord.")
			return
		}

		// Retry fetching channels with new token
		channels, err = h.discordOAuthService.GetGuildChannels(r.Context(), newAccessToken, req.GuildID)
	}

	if err != nil {
		fmt.Printf("[GUILD_HANDLER] Error fetching channels: %v\n", err)
		WriteError(w, http.StatusInternalServerError, "Failed to fetch channels from Discord: "+err.Error())
		return
	}

	// Convert to response format
	channelData := make([]ChannelData, len(channels))
	for i, channel := range channels {
		channelData[i] = ChannelData{
			ID:       channel.ID,
			Name:     channel.Name,
			Type:     channel.Type,
			Position: channel.Position,
			Topic:    channel.Topic,
		}
	}

	resp := GetGuildChannelsResponse{
		Channels: channelData,
	}

	WriteJSON(w, http.StatusOK, resp)
}

// GetGuildRoles retrieves all roles for a specific guild
func (h *DiscordGuildHandler) GetGuildRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		AccountID string `json:"account_id"`
		GuildID   string `json:"guild_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if strings.TrimSpace(req.AccountID) == "" {
		WriteError(w, http.StatusBadRequest, "Account ID is required")
		return
	}

	if strings.TrimSpace(req.GuildID) == "" {
		WriteError(w, http.StatusBadRequest, "Guild ID is required")
		return
	}

	// Parse account ID
	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid account ID format")
		return
	}

	// Get account with identities
	account, err := h.discordOAuthService.GetAccount(r.Context(), accountID)
	if err != nil || account == nil {
		WriteError(w, http.StatusNotFound, "Account not found")
		return
	}

	// Find Discord identity with access token
	var discordIdentity *models.Identity
	for i := range account.Identities {
		if account.Identities[i].Provider == models.ProviderDiscord && account.Identities[i].AccessToken != nil {
			discordIdentity = &account.Identities[i]
			break
		}
	}

	// Note: this is about the Discord OAuth token, not the Chronos session,
	// so it must never be reported as 401 — the frontend's global HTTP
	// client treats any 401 as "your session expired" and force-logs the
	// user out of the whole app.
	if discordIdentity == nil {
		WriteError(w, http.StatusConflict, "No Discord identity with access token found for this account. Please reconnect Discord.")
		return
	}

	// Check if bot is in the guild
	botInGuild, err := h.discordOAuthService.IsBotInGuild(r.Context(), req.GuildID)
	if err != nil {
		fmt.Printf("[GUILD_HANDLER] Error checking bot guild membership: %v\n", err)
		WriteError(w, http.StatusInternalServerError, "Failed to verify bot guild membership")
		return
	}

	if !botInGuild {
		WriteError(w, http.StatusForbidden, "Bot is not a member of this guild. Please invite the bot to the guild first.")
		return
	}

	accessToken := *discordIdentity.AccessToken

	// Get roles from Discord
	roles, err := h.discordOAuthService.GetGuildRoles(r.Context(), accessToken, req.GuildID)

	// If we get a 401 from Discord, try to refresh the token
	if err != nil && strings.Contains(err.Error(), "status 401") {
		newAccessToken, refreshErr := h.discordOAuthService.RefreshAndPersistDiscordToken(r.Context(), discordIdentity)
		if refreshErr != nil {
			fmt.Printf("[GUILD_HANDLER] Error refreshing token: %v\n", refreshErr)
			WriteError(w, http.StatusConflict, "Discord token expired. Please reconnect Discord.")
			return
		}

		// Retry fetching roles with new token
		roles, err = h.discordOAuthService.GetGuildRoles(r.Context(), newAccessToken, req.GuildID)
	}

	if err != nil {
		fmt.Printf("[GUILD_HANDLER] Error fetching roles: %v\n", err)
		WriteError(w, http.StatusInternalServerError, "Failed to fetch roles from Discord: "+err.Error())
		return
	}

	// Convert to response format
	roleData := make([]RoleData, len(roles))
	for i, role := range roles {
		roleData[i] = RoleData{
			ID:          role.ID,
			Name:        role.Name,
			Color:       role.Color,
			Position:    role.Position,
			Permissions: role.Permissions,
			Managed:     role.Managed,
			Mentionable: role.Mentionable,
		}
	}

	resp := GetGuildRolesResponse{
		Roles: roleData,
	}

	WriteJSON(w, http.StatusOK, resp)
}
