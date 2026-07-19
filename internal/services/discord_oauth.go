package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/database/repositories"
	"github.com/google/uuid"
)

// DiscordOAuthService handles Discord OAuth operations
type DiscordOAuthService struct {
	clientID       string
	clientSecret   string
	redirectURI    string
	botToken       string
	identityRepo   repositories.IdentityRepository
	accountRepo    repositories.AccountRepository
	timezoneRepo   repositories.TimezoneRepository
	sessionService *SessionService
	// verificationService is optional; when set, new Discord accounts whose
	// email is not already verified by Discord receive a verification email.
	verificationService *VerificationService
}

// DiscordUserInfo represents Discord user information from OAuth
type DiscordUserInfo struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	Avatar        string `json:"avatar"`
	Discriminator string `json:"discriminator"`
	// Verified reports whether Discord has verified the user's email address.
	Verified bool `json:"verified"`
}

// DiscordGuild represents a Discord guild (server)
type DiscordGuild struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Owner       bool   `json:"owner"`
	Permissions int64  `json:"permissions"`
	Features    []string `json:"features"`
}

// DiscordGuildChannel represents a Discord guild channel
type DiscordGuildChannel struct {
	ID                   string        `json:"id"`
	Name                 string        `json:"name"`
	Type                 int           `json:"type"` // 0=text, 1=DM, 2=voice, 4=category, etc.
	Position             int           `json:"position"`
	Topic                *string       `json:"topic"`
	PermissionOverwrites []interface{} `json:"permission_overwrites"`
}

// DiscordRole represents a Discord guild role
type DiscordRole struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Hoist       bool   `json:"hoist"`
	Position    int    `json:"position"`
	Permissions int64  `json:"permissions"`
	Managed     bool   `json:"managed"`
	Mentionable bool   `json:"mentionable"`
}

// DiscordTokenResponse represents the response from Discord token exchange
type DiscordTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// NewDiscordOAuthService creates a new Discord OAuth service
func NewDiscordOAuthService(
	clientID string,
	clientSecret string,
	redirectURI string,
	botToken string,
	identityRepo repositories.IdentityRepository,
	accountRepo repositories.AccountRepository,
	timezoneRepo repositories.TimezoneRepository,
	sessionService *SessionService,
	verificationService *VerificationService,
) *DiscordOAuthService {
	return &DiscordOAuthService{
		clientID:            clientID,
		clientSecret:        clientSecret,
		redirectURI:         redirectURI,
		botToken:            botToken,
		identityRepo:        identityRepo,
		accountRepo:         accountRepo,
		timezoneRepo:        timezoneRepo,
		sessionService:      sessionService,
		verificationService: verificationService,
	}
}

// RefreshDiscordSnapshot fetches fresh Discord user info (avatar, username) using
// the stored access/refresh tokens and persists any changes to the identity row.
// It is safe to call in a goroutine — errors are logged but not propagated.
func (s *DiscordOAuthService) RefreshDiscordSnapshot(ctx context.Context, identity *models.Identity) {
	if identity == nil || identity.AccessToken == nil {
		return
	}

	accessToken := *identity.AccessToken

	userInfo, err := s.GetUserInfo(ctx, accessToken)
	if err != nil {
		// Access token may be expired — try to refresh it.
		if identity.RefreshToken == nil {
			return
		}
		newAccess, newRefresh, rerr := s.RefreshDiscordToken(ctx, *identity.RefreshToken)
		if rerr != nil {
			return
		}
		accessToken = newAccess
		identity.AccessToken = &newAccess
		identity.RefreshToken = &newRefresh

		userInfo, err = s.GetUserInfo(ctx, accessToken)
		if err != nil {
			return
		}
	}

	changed := false
	if userInfo.Avatar != "" && (identity.Avatar == nil || *identity.Avatar != userInfo.Avatar) {
		identity.Avatar = &userInfo.Avatar
		changed = true
	}
	if userInfo.Username != "" && (identity.Username == nil || *identity.Username != userInfo.Username) {
		identity.Username = &userInfo.Username
		changed = true
	}

	if changed {
		_ = s.identityRepo.Update(identity)
	}
}

// ExchangeCodeForToken exchanges Discord authorization code for access token and refresh token
func (s *DiscordOAuthService) ExchangeCodeForToken(ctx context.Context, code string) (string, string, error) {
	if s.clientID == "" || s.clientSecret == "" {
		return "", "", fmt.Errorf("discord credentials not configured: client_id or client_secret is empty")
	}

	data := url.Values{}
	data.Set("client_id", s.clientID)
	data.Set("client_secret", s.clientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", s.redirectURI)

	resp, err := http.PostForm("https://discord.com/api/oauth2/token", data)
	if err != nil {
		return "", "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("discord token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp DiscordTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return tokenResp.AccessToken, tokenResp.RefreshToken, nil
}

// GetUserInfo retrieves Discord user information using access token
func (s *DiscordOAuthService) GetUserInfo(ctx context.Context, accessToken string) (*DiscordUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://discord.com/api/users/@me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord user info request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var userInfo DiscordUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// ProcessDiscordAuth handles the complete Discord authentication flow with four cases:
//   - Case 1: Existing Discord identity -> Check if app identity exists, prompt setup if needed
//   - Case 2: Email exists as app provider -> Link Discord identity in background, login
//   - Case 3: Email exists as Discord provider -> Login with existing account
//   - Case 4: New user (no Discord ID or email) -> Create new account with Discord identity, prompt setup
func (s *DiscordOAuthService) ProcessDiscordAuth(ctx context.Context, userInfo *DiscordUserInfo, accessToken, refreshToken string) (*models.Account, string, error) {
	if userInfo == nil {
		return nil, "", errors.New("user info is nil")
	}

	// Step 1: Check if Discord identity already exists
	discordIdentity, err := s.identityRepo.GetByProviderAndExternalID(models.ProviderDiscord, userInfo.ID)
	if err != nil {
		return nil, "", fmt.Errorf("error checking discord identity: %w", err)
	}

	if discordIdentity != nil {
		// Case 1: Existing Discord identity - check if account needs setup
		account, err := s.accountRepo.GetWithIdentities(discordIdentity.AccountID)
		if err != nil {
			return nil, "", fmt.Errorf("error loading account: %w", err)
		}
		if account == nil {
			return nil, "", errors.New("account not found for existing discord identity")
		}

		// Update access token for the Discord identity
		discordIdentity.AccessToken = &accessToken
		if refreshToken != "" {
			discordIdentity.RefreshToken = &refreshToken
		}
		if err := s.identityRepo.Update(discordIdentity); err != nil {
			fmt.Printf("[DISCORD_AUTH] Warning: Failed to update access token: %v\n", err)
		}

		// If account has no password set, prompt for setup. Checking Email
		// alone isn't enough: an account can have Email populated (e.g.
		// copied over from a legacy migration) without ever having a usable
		// password, which would otherwise never be re-prompted to fix.
		if !account.HasPassword() && len(account.Identities) == 1 {
			return account, "SETUP_REQUIRED", nil
		}

		// Create session token for existing account
		token, err := s.sessionService.generateTokenForAccount(account, discordIdentity, 30*24*time.Hour)
		if err != nil {
			return nil, "", fmt.Errorf("error creating session: %w", err)
		}

		return account, token, nil
	}

	// Step 2: Check if an account already exists with this email (account-level
	// credential). If so, this is the same person — link Discord and log in.
	if userInfo.Email != "" {
		existingAccount, err := s.accountRepo.GetByEmail(userInfo.Email)
		if err != nil {
			return nil, "", fmt.Errorf("error checking account email: %w", err)
		}

		if existingAccount != nil {
			// Case 2: account with this email exists - link Discord, login
			account, err := s.accountRepo.GetWithIdentities(existingAccount.ID)
			if err != nil {
				return nil, "", fmt.Errorf("error loading account: %w", err)
			}
			if account == nil {
				return nil, "", errors.New("account not found for existing email")
			}

			// Create session token for this account
			token, err := s.sessionService.generateTokenForAccount(account, nil, 30*24*time.Hour)
			if err != nil {
				return nil, "", fmt.Errorf("error creating session: %w", err)
			}

			// Link Discord identity in the background (non-blocking)
			go func() {
				newDiscordIdentity := &models.Identity{
					ID:          uuid.New(),
					AccountID:   account.ID,
					Provider:    models.ProviderDiscord,
					ExternalID:  userInfo.ID,
					Username:    &userInfo.Username,
					Avatar:      &userInfo.Avatar,
					AccessToken: &accessToken,
				}
				if refreshToken != "" {
					newDiscordIdentity.RefreshToken = &refreshToken
				}
				_ = s.identityRepo.Create(newDiscordIdentity)
			}()

			return account, token, nil
		}
	}

	// Case 4: New user - create new account with Discord identity only
	timezone, err := s.timezoneRepo.GetByIANALocation("UTC")
	if err != nil {
		return nil, "", fmt.Errorf("error fetching timezone: %w", err)
	}
	if timezone == nil {
		return nil, "", errors.New("UTC timezone not found")
	}

	// Create new account. Discord already verifies user emails, so when Discord
	// reports the email as verified we trust it and mark the account verified
	// immediately — this avoids leaving the user stuck behind an "email not
	// verified" wall they have no way to clear (they signed up via Discord and
	// never received a Chronos verification email).
	emailVerified := userInfo.Email != "" && userInfo.Verified
	account := &models.Account{
		ID:            uuid.New(),
		TimezoneID:    &timezone.ID,
		EmailVerified: emailVerified,
	}

	if err := s.accountRepo.Create(account); err != nil {
		return nil, "", fmt.Errorf("error creating account: %w", err)
	}

	// Create Discord identity
	newDiscordIdentity := &models.Identity{
		ID:           uuid.New(),
		AccountID:    account.ID,
		Provider:     models.ProviderDiscord,
		ExternalID:   userInfo.ID,
		Username:     &userInfo.Username,
		Avatar:       &userInfo.Avatar,
		AccessToken:  &accessToken,
	}
	if refreshToken != "" {
		newDiscordIdentity.RefreshToken = &refreshToken
	}

	if err := s.identityRepo.Create(newDiscordIdentity); err != nil {
		// Clean up the created account on identity creation failure
		s.accountRepo.Delete(account.ID)
		return nil, "", fmt.Errorf("error creating discord identity: %w", err)
	}

	// Load full account
	account.Identities = []models.Identity{*newDiscordIdentity}
	account.Timezone = timezone

	// If we could not auto-verify the email (Discord reported it unverified, or
	// no email scope was granted), send a Chronos verification email so the user
	// has a way to verify and unblock email/password login. Best-effort only.
	if !emailVerified && userInfo.Email != "" && s.verificationService != nil {
		if err := s.verificationService.SendAccountVerification(userInfo.Email, account.ID.String()); err != nil {
			fmt.Printf("[DISCORD_OAUTH] - ⚠️ Failed to send verification email for new Discord account: %v\n", err)
		}
	}

	// New account created with Discord identity only - prompt for setup
	return account, "SETUP_REQUIRED", nil
}

// LinkDiscordToAccount links a Discord identity to an existing account.
// ErrDiscordLinkedToOtherAccount is returned by LinkDiscordToAccount when the
// Discord ID belongs to a different account. The handler can use this sentinel
// to distinguish a "merge required" situation from hard errors.
var ErrDiscordLinkedToOtherAccount = errors.New("discord account already linked to another account")

// LinkDiscordResult carries the outcome of a link attempt.
type LinkDiscordResult struct {
	// OtherAccountID is set when ErrDiscordLinkedToOtherAccount is returned.
	OtherAccountID uuid.UUID
	// OtherDiscordUsername is the Discord username of the conflicting identity.
	OtherDiscordUsername string
}

// LinkDiscordToAccount links a Discord identity to an existing account.
// accessToken/refreshToken are stored so Discord guild features work for the
// linked account; pass empty strings if unavailable.
// Returns (LinkDiscordResult{}, nil) on success.
// Returns (result, ErrDiscordLinkedToOtherAccount) when the Discord ID already
// belongs to a different account — the caller should offer a merge.
func (s *DiscordOAuthService) LinkDiscordToAccount(ctx context.Context, accountID uuid.UUID, userInfo *DiscordUserInfo, accessToken, refreshToken string) (LinkDiscordResult, error) {
	// Check if this Discord ID is already linked to an account
	existingIdentity, err := s.identityRepo.GetByProviderAndExternalID(models.ProviderDiscord, userInfo.ID)
	if err != nil {
		return LinkDiscordResult{}, fmt.Errorf("error checking discord identity: %w", err)
	}
	if existingIdentity != nil {
		// Already linked to THIS account: refresh the snapshot/tokens instead of erroring.
		if existingIdentity.AccountID == accountID {
			existingIdentity.Username = &userInfo.Username
			existingIdentity.Avatar = &userInfo.Avatar
			if accessToken != "" {
				existingIdentity.AccessToken = &accessToken
			}
			if refreshToken != "" {
				existingIdentity.RefreshToken = &refreshToken
			}
			if err := s.identityRepo.Update(existingIdentity); err != nil {
				return LinkDiscordResult{}, fmt.Errorf("error updating discord identity: %w", err)
			}
			return LinkDiscordResult{}, nil
		}
		username := userInfo.Username
		if existingIdentity.Username != nil {
			username = *existingIdentity.Username
		}
		return LinkDiscordResult{
			OtherAccountID:       existingIdentity.AccountID,
			OtherDiscordUsername: username,
		}, ErrDiscordLinkedToOtherAccount
	}

	// Create Discord identity for this account
	discordIdentity := &models.Identity{
		ID:         uuid.New(),
		AccountID:  accountID,
		Provider:   models.ProviderDiscord,
		ExternalID: userInfo.ID,
		Username:   &userInfo.Username,
		Avatar:     &userInfo.Avatar,
	}
	if accessToken != "" {
		discordIdentity.AccessToken = &accessToken
	}
	if refreshToken != "" {
		discordIdentity.RefreshToken = &refreshToken
	}

	if err := s.identityRepo.Create(discordIdentity); err != nil {
		return LinkDiscordResult{}, fmt.Errorf("error creating discord identity: %w", err)
	}

	return LinkDiscordResult{}, nil
}

// CreateAppIdentityForDiscordAccount sets the account-level email/password for a
// Discord-only account. This is called during the setup flow after Discord OAuth
// to enable email/password login on web and mobile.
func (s *DiscordOAuthService) CreateAppIdentityForDiscordAccount(
	ctx context.Context,
	accountIDStr string,
	email string,
	username string,
	password string,
	timezone string,
) (string, error) {
	// Parse account ID
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return "", fmt.Errorf("invalid account ID: %w", err)
	}

	// Verify account exists
	account, err := s.accountRepo.GetWithIdentities(accountID)
	if err != nil {
		return "", fmt.Errorf("error loading account: %w", err)
	}
	if account == nil {
		return "", errors.New("account not found")
	}

	// Reject if this account already has credentials.
	if account.PasswordHash != nil {
		return "", errors.New("account already has email/password login")
	}

	// Reject if the email is already used by another account.
	existingAccount, err := s.accountRepo.GetByEmail(email)
	if err != nil {
		return "", fmt.Errorf("error checking email: %w", err)
	}
	if existingAccount != nil && existingAccount.ID != accountID {
		return "", errors.New("email already in use")
	}

	// Hash password
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("error hashing password: %w", err)
	}

	// Set account-level credentials. The user has verified via Discord OAuth, so
	// mark the email verified immediately.
	account.Email = &email
	account.Username = &username
	account.PasswordHash = &hashedPassword
	account.EmailVerified = true

	// Update timezone if provided and valid
	if timezone != "" {
		tz, err := s.timezoneRepo.GetByIANALocation(timezone)
		if err == nil && tz != nil {
			account.TimezoneID = &tz.ID
		}
	}

	if err := s.accountRepo.Update(account); err != nil {
		return "", fmt.Errorf("error updating account: %w", err)
	}

	// Create session token for the account
	token, err := s.sessionService.generateTokenForAccount(account, nil, 30*24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("error creating session: %w", err)
	}

	return token, nil
}

// GetAccount retrieves an account with all its identities
func (s *DiscordOAuthService) GetAccount(ctx context.Context, accountID uuid.UUID) (*models.Account, error) {
	return s.accountRepo.GetWithIdentities(accountID)
}

// RefreshDiscordToken refreshes a Discord access token using the refresh token
func (s *DiscordOAuthService) RefreshDiscordToken(ctx context.Context, refreshToken string) (string, string, error) {
	if refreshToken == "" {
		return "", "", errors.New("refresh token is empty")
	}

	data := url.Values{}
	data.Set("client_id", s.clientID)
	data.Set("client_secret", s.clientSecret)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	resp, err := http.PostForm("https://discord.com/api/oauth2/token", data)
	if err != nil {
		return "", "", fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("discord token refresh failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp DiscordTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return tokenResp.AccessToken, tokenResp.RefreshToken, nil
}

// GetUserGuilds retrieves all guilds the user has access to
func (s *DiscordOAuthService) GetUserGuilds(ctx context.Context, accessToken string) ([]DiscordGuild, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://discord.com/api/users/@me/guilds", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get guilds: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord guilds request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var guilds []DiscordGuild
	if err := json.NewDecoder(resp.Body).Decode(&guilds); err != nil {
		return nil, fmt.Errorf("failed to decode guilds: %w", err)
	}

	return guilds, nil
}

// GetGuildChannels retrieves all channels for a specific guild
// Uses bot token instead of user OAuth token as Discord API requires bot authentication for this endpoint
func (s *DiscordOAuthService) GetGuildChannels(ctx context.Context, accessToken string, guildID string) ([]DiscordGuildChannel, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://discord.com/api/guilds/%s/channels", guildID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Use bot token for this endpoint (user OAuth tokens don't work for guild endpoints)
	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", s.botToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get channels: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord channels request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var channels []DiscordGuildChannel
	if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
		return nil, fmt.Errorf("failed to decode channels: %w", err)
	}

	return channels, nil
}

// GetGuildRoles retrieves all roles for a specific guild
// Uses bot token instead of user OAuth token as Discord API requires bot authentication for this endpoint
func (s *DiscordOAuthService) GetGuildRoles(ctx context.Context, accessToken string, guildID string) ([]DiscordRole, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://discord.com/api/guilds/%s/roles", guildID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Use bot token for this endpoint (user OAuth tokens don't work for guild endpoints)
	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", s.botToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord roles request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var roles []DiscordRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode roles: %w", err)
	}

	return roles, nil
}

// IsBotInGuild checks if the bot is a member of the specified guild
func (s *DiscordOAuthService) IsBotInGuild(ctx context.Context, guildID string) (bool, error) {
	if s.botToken == "" {
		return false, errors.New("bot token not configured")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://discord.com/api/guilds/%s", guildID), nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", s.botToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check guild: %w", err)
	}
	defer resp.Body.Close()

	// 200 = bot is in the guild, 404 = bot is not in the guild, other = error
	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("failed to check guild membership (status %d): %s", resp.StatusCode, string(body))
}
