package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ericp/chronos-bot-reminder/internal/config"
	"github.com/ericp/chronos-bot-reminder/internal/database/repositories"
	"github.com/ericp/chronos-bot-reminder/internal/docs"
	"github.com/ericp/chronos-bot-reminder/internal/services"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Server represents the API server
type Server struct {
	mux           *WrappedMux
	port          string
	server        *http.Server
	cfg           *config.Config
	mailerService *services.MailerService
}

// NewServer creates a new API server instance
func NewServer(cfg *config.Config, repos *repositories.Repositories) *Server {
	// Initialize services
	authService := services.NewAuthService(
		repos.Account,
		repos.Identity,
		repos.Timezone,
	)

	sessionService := services.NewSessionService(
		repos.Identity,
		repos.Account,
	)

	// Initialize mailer service
	mailerService := services.NewMailerService(
		cfg.ResendAPIKey,
		config.EmailNoreply,
	)

	// Initialize verification service (used by auth + Discord OAuth signup)
	verificationService := services.NewVerificationService(
		repos.EmailVerification,
		repos.Account,
		repos.Identity,
		mailerService,
		cfg.WebAppURL,
	)

	discordOAuthService := services.NewDiscordOAuthService(
		cfg.DiscordClientID,
		cfg.DiscordClientSecret,
		cfg.DiscordRedirectURI,
		cfg.DiscordBotToken,
		repos.Identity,
		repos.Account,
		repos.Timezone,
		sessionService,
		verificationService,
	)

	// Initialize rate limiter service
	rateLimiterService := services.NewRateLimiterService(
		cfg.RateLimitRequestsPerWindow,
		cfg.RateLimitWindowSeconds,
	)

	// Initialize password reset service
	passwordResetService := services.NewPasswordResetService(
		repos.PasswordReset,
		repos.Account,
		mailerService,
	)

	// Initialize API key service
	apiKeyService := services.NewAPIKeyService(
		repos.Identity,
		repos.Account,
	)

	// Initialize handlers
	authHandler := NewAuthHandler(authService, sessionService, verificationService, passwordResetService, cfg.WebAppURL)
	discordOAuthHandler := NewDiscordOAuthHandler(discordOAuthService, repos)
	discordGuildHandler := NewDiscordGuildHandler(discordOAuthService)
	userHandler := NewUserHandler(repos.Reminder, repos.ReminderError, repos.Account, sessionService)
	userHandler.SetReminderDestinationRepository(repos.ReminderDestination)
	userHandler.SetIdentityRepository(repos.Identity)
	userHandler.SetTimezoneRepository(repos.Timezone)
	userHandler.SetDiscordOAuthService(discordOAuthService)

	// Initialize reminder handler
	reminderHandler := NewReminderHandler(
		repos.Reminder,
		repos.ReminderDestination,
		repos.ReminderError,
	)
	reminderHandler.SetAccountRepository(repos.Account)
	reminderHandler.SetTimezoneRepository(repos.Timezone)

	// Initialize Don't Forget Me handler
	dfmHandler := NewDFMHandler(
		repos.DFMNote,
		repos.DFMItem,
		repos.Account,
		repos.Identity,
	)

	// Initialize timezone handler
	timezoneHandler := NewTimezoneHandler(repos.Timezone)

	// Initialize API key handler
	apiKeyHandler := NewAPIKeyHandler(apiKeyService)

	// Initialize FCM token handler
	fcmHandler := NewFcmHandler(repos.FcmToken)

	// Initialize health handler
	healthHandler := NewHealthHandler()

	// Initialize contact handler
	contactHandler := NewContactHandler(mailerService)

	// Create wrapped mux with CORS middleware
	wrappedMux := NewWrappedMux()
	wrappedMux.Use(CORSMiddleware(cfg))

	// Apply rate limiter middleware to protected routes (if enabled)
	var rateLimitMiddleware func(http.Handler) http.Handler
	if cfg.RateLimitEnabled {
		rateLimitMiddleware = RateLimitMiddleware(rateLimiterService)
		log.Printf("[API] - ⚡ Rate limiting enabled: %d requests per %d seconds\n",
			cfg.RateLimitRequestsPerWindow, cfg.RateLimitWindowSeconds)
	} else {
		// No-op middleware
		rateLimitMiddleware = func(next http.Handler) http.Handler { return next }
		log.Println("[API] - ⚡ Rate limiting disabled")
	}

	// Register all routes
	registerHealthRoutes(wrappedMux, healthHandler)
	registerSwaggerRoutes(wrappedMux)
	registerAuthRoutes(wrappedMux, authHandler)
	registerDiscordOAuthRoutes(wrappedMux, discordOAuthHandler)
	registerDiscordGuildRoutes(wrappedMux, discordGuildHandler)
	registerUserRoutes(wrappedMux, userHandler, discordOAuthHandler, sessionService, apiKeyService, rateLimitMiddleware)
	registerReminderRoutes(wrappedMux, reminderHandler, sessionService, apiKeyService, rateLimitMiddleware)
	registerDFMRoutes(wrappedMux, dfmHandler, sessionService, apiKeyService, rateLimitMiddleware)
	registerTimezoneRoutes(wrappedMux, timezoneHandler)
	registerAPIKeyRoutes(wrappedMux, apiKeyHandler, sessionService, apiKeyService, rateLimitMiddleware)
	registerFcmRoutes(wrappedMux, fcmHandler, sessionService, apiKeyService, rateLimitMiddleware)
	registerContactRoutes(wrappedMux, contactHandler)

	return &Server{
		mux:           wrappedMux,
		port:          cfg.APIPort,
		cfg:           cfg,
		mailerService: mailerService,
	}
}

// registerSwaggerRoutes registers Swagger documentation routes
func registerSwaggerRoutes(mux *WrappedMux) {
	// Swagger JSON endpoint
	mux.HandleFunc("GET /swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(docs.ReadDoc()))
	})

	// Swagger UI handler - handles all swagger UI requests including assets
	swaggerHandler := httpSwagger.Handler(
		httpSwagger.URL("https://api.chronosrmd.com/swagger/doc.json"),
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	)

	// Register swagger handler for the entire /swagger/ path including nested paths
	mux.Handle("GET /swagger/", swaggerHandler)
}

// registerHealthRoutes registers health check routes
func registerHealthRoutes(mux *WrappedMux, healthHandler *HealthHandler) {
	mux.HandleFunc("GET /", healthHandler.Root)
	mux.HandleFunc("GET /api/health", healthHandler.Health)
}

// registerAuthRoutes registers authentication routes
func registerAuthRoutes(mux *WrappedMux, authHandler *AuthHandler) {
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/verify", authHandler.VerifyEmail)
	mux.HandleFunc("POST /api/auth/verify/resend", authHandler.ResendVerification)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	mux.HandleFunc("POST /api/auth/password-reset/request", authHandler.RequestPasswordReset)
	mux.HandleFunc("POST /api/auth/password-reset/verify-token", authHandler.VerifyResetToken)
	mux.HandleFunc("POST /api/auth/password-reset/reset", authHandler.ResetPassword)
}

// registerDiscordOAuthRoutes registers Discord OAuth routes
func registerDiscordOAuthRoutes(mux *WrappedMux, discordOAuthHandler *DiscordOAuthHandler) {
	mux.HandleFunc("POST /api/auth/discord/callback", discordOAuthHandler.DiscordCallback)
	mux.HandleFunc("POST /api/auth/discord/setup", discordOAuthHandler.CompleteDiscordSetup)
}

// registerDiscordGuildRoutes registers Discord guild-related routes
func registerDiscordGuildRoutes(mux *WrappedMux, discordGuildHandler *DiscordGuildHandler) {
	mux.HandleFunc("POST /api/discord/guilds", discordGuildHandler.GetUserGuilds)
	mux.HandleFunc("POST /api/discord/guilds/channels", discordGuildHandler.GetGuildChannels)
	mux.HandleFunc("POST /api/discord/guilds/roles", discordGuildHandler.GetGuildRoles)
}

// registerUserRoutes registers authenticated user routes with auth and rate limit middleware
func registerUserRoutes(mux *WrappedMux, userHandler *UserHandler, discordOAuthHandler *DiscordOAuthHandler, sessionService *services.SessionService, apiKeyService *services.APIKeyService, rateLimitMiddleware func(http.Handler) http.Handler) {
	// Apply auth middleware to user routes
	authMiddleware := AuthMiddleware(sessionService, apiKeyService)

	// Chain middlewares: rate limit -> auth
	chainMiddleware := func(handler http.Handler) http.Handler {
		return rateLimitMiddleware(authMiddleware(handler))
	}

	// Wrap each user route handler with both middlewares
	mux.Handle("GET /api/reminders", chainMiddleware(http.HandlerFunc(userHandler.GetReminders)))
	mux.Handle("POST /api/reminders", chainMiddleware(http.HandlerFunc(userHandler.CreateReminder)))
	mux.Handle("GET /api/reminders/errors", chainMiddleware(http.HandlerFunc(userHandler.GetReminderErrors)))
	mux.Handle("GET /api/account", chainMiddleware(http.HandlerFunc(userHandler.GetAccount)))
	mux.Handle("POST /api/account/identity/app/change-password", chainMiddleware(http.HandlerFunc(userHandler.ChangeAppIdentityPassword)))
	mux.Handle("PUT /api/account/timezone", chainMiddleware(http.HandlerFunc(userHandler.UpdateAccountTimezone)))
	mux.Handle("PUT /api/account/preferences", chainMiddleware(http.HandlerFunc(userHandler.UpdatePreferences)))
	mux.Handle("PUT /api/account/identity/app/username", chainMiddleware(http.HandlerFunc(userHandler.UpdateAppIdentityUsername)))
	mux.Handle("PUT /api/account/identity/app/email", chainMiddleware(http.HandlerFunc(userHandler.UpdateAppIdentityEmail)))
	mux.Handle("DELETE /api/account", chainMiddleware(http.HandlerFunc(userHandler.DeleteAccount)))
	mux.Handle("POST /api/account/identity/mobile", chainMiddleware(http.HandlerFunc(userHandler.EnsureMobileIdentity)))
	mux.Handle("POST /api/account/identity/app", chainMiddleware(http.HandlerFunc(userHandler.AddAppIdentity)))
	mux.Handle("POST /api/account/identity/discord/link", chainMiddleware(http.HandlerFunc(discordOAuthHandler.LinkDiscordIdentity)))
	mux.Handle("POST /api/account/merge", chainMiddleware(http.HandlerFunc(discordOAuthHandler.MergeDiscordAccounts)))
}

// registerReminderRoutes registers reminder-specific routes with auth and rate limit middleware
func registerReminderRoutes(mux *WrappedMux, reminderHandler *ReminderHandler, sessionService *services.SessionService, apiKeyService *services.APIKeyService, rateLimitMiddleware func(http.Handler) http.Handler) {
	authMiddleware := AuthMiddleware(sessionService, apiKeyService)

	// Chain middlewares: rate limit -> auth
	chainMiddleware := func(handler http.Handler) http.Handler {
		return rateLimitMiddleware(authMiddleware(handler))
	}

	// Reminder CRUD operations
	mux.Handle("GET /api/reminders/{id}", chainMiddleware(http.HandlerFunc(reminderHandler.GetReminder)))
	mux.Handle("PUT /api/reminders/{id}", chainMiddleware(http.HandlerFunc(reminderHandler.UpdateReminder)))
	mux.Handle("DELETE /api/reminders/{id}", chainMiddleware(http.HandlerFunc(reminderHandler.DeleteReminder)))

	// Reminder state operations
	mux.Handle("POST /api/reminders/{id}/pause", chainMiddleware(http.HandlerFunc(reminderHandler.PauseReminder)))
	mux.Handle("POST /api/reminders/{id}/resume", chainMiddleware(http.HandlerFunc(reminderHandler.ResumeReminder)))
	mux.Handle("POST /api/reminders/{id}/duplicate", chainMiddleware(http.HandlerFunc(reminderHandler.DuplicateReminder)))
	mux.Handle("POST /api/reminders/{id}/snooze", chainMiddleware(http.HandlerFunc(reminderHandler.SnoozeReminder)))
}

// registerDFMRoutes registers "Don't Forget Me" routes with auth and rate limit middleware
func registerDFMRoutes(mux *WrappedMux, dfmHandler *DFMHandler, sessionService *services.SessionService, apiKeyService *services.APIKeyService, rateLimitMiddleware func(http.Handler) http.Handler) {
	authMiddleware := AuthMiddleware(sessionService, apiKeyService)

	// Chain middlewares: rate limit -> auth
	chainMiddleware := func(handler http.Handler) http.Handler {
		return rateLimitMiddleware(authMiddleware(handler))
	}

	// Note and items
	mux.Handle("GET /api/dfm", chainMiddleware(http.HandlerFunc(dfmHandler.GetNote)))
	mux.Handle("POST /api/dfm/items", chainMiddleware(http.HandlerFunc(dfmHandler.AddItem)))
	mux.Handle("PUT /api/dfm/items/{id}", chainMiddleware(http.HandlerFunc(dfmHandler.UpdateItem)))
	mux.Handle("DELETE /api/dfm/items/{id}", chainMiddleware(http.HandlerFunc(dfmHandler.DeleteItem)))

	// Reminder configuration
	mux.Handle("PUT /api/dfm/reminder", chainMiddleware(http.HandlerFunc(dfmHandler.SetReminder)))
	mux.Handle("DELETE /api/dfm/reminder", chainMiddleware(http.HandlerFunc(dfmHandler.RemoveReminder)))
	mux.Handle("POST /api/dfm/send", chainMiddleware(http.HandlerFunc(dfmHandler.SendNow)))
}

// registerTimezoneRoutes registers timezone routes (public, no auth required)
func registerTimezoneRoutes(mux *WrappedMux, timezoneHandler *TimezoneHandler) {
	mux.HandleFunc("GET /api/timezones", timezoneHandler.GetAvailableTimezones)
}

// registerAPIKeyRoutes registers API key management routes with auth and rate limit middleware
func registerAPIKeyRoutes(mux *WrappedMux, apiKeyHandler *APIKeyHandler, sessionService *services.SessionService, apiKeyService *services.APIKeyService, rateLimitMiddleware func(http.Handler) http.Handler) {
	authMiddleware := AuthMiddleware(sessionService, apiKeyService)

	// Chain middlewares: rate limit -> auth
	chainMiddleware := func(handler http.Handler) http.Handler {
		return rateLimitMiddleware(authMiddleware(handler))
	}

	// API key management routes
	mux.Handle("POST /api/api-keys", chainMiddleware(http.HandlerFunc(apiKeyHandler.CreateAPIKey)))
	mux.Handle("GET /api/api-keys", chainMiddleware(http.HandlerFunc(apiKeyHandler.GetAPIKeys)))
	mux.Handle("DELETE /api/api-keys/{id}", chainMiddleware(http.HandlerFunc(apiKeyHandler.RevokeAPIKey)))
}

// registerFcmRoutes registers FCM token routes with auth and rate limit middleware
func registerFcmRoutes(mux *WrappedMux, fcmHandler *FcmHandler, sessionService *services.SessionService, apiKeyService *services.APIKeyService, rateLimitMiddleware func(http.Handler) http.Handler) {
	authMiddleware := AuthMiddleware(sessionService, apiKeyService)

	// Chain middlewares: rate limit -> auth
	chainMiddleware := func(handler http.Handler) http.Handler {
		return rateLimitMiddleware(authMiddleware(handler))
	}

	mux.Handle("GET /api/fcm/status", chainMiddleware(http.HandlerFunc(fcmHandler.HasTokens)))
	mux.Handle("POST /api/fcm/token", chainMiddleware(http.HandlerFunc(fcmHandler.RegisterToken)))
	mux.Handle("DELETE /api/fcm/token", chainMiddleware(http.HandlerFunc(fcmHandler.UnregisterToken)))
}

// registerContactRoutes registers contact form routes (public, no auth required)
func registerContactRoutes(mux *WrappedMux, contactHandler *ContactHandler) {
	mux.HandleFunc("POST /api/contact", contactHandler.SubmitContact)
}

// Start starts the API server and listens for incoming requests
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:    ":" + s.port,
		Handler: s.mux,
	}

	log.Printf("[API] - 🚀 Starting API server on port %s\n", s.port)
	log.Printf("[API] - 📡 Server running at http://localhost:%s\n", s.port)
	log.Printf("[API] - 📚 Swagger documentation available at http://localhost:%s/swagger/\n", s.port)
	log.Printf("[API] - 🌐 WebClient available at %s", s.cfg.WebAppURL)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("[API] - ❌ Failed to start server: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the API server
func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}

	log.Println("[API] - 🛑 Shutting down API server...")
	return s.server.Close()
}

// GetPort returns the port the server is listening on
func (s *Server) GetPort() string {
	return s.port
}

// GetMailerService returns the mailer service instance
func (s *Server) GetMailerService() *services.MailerService {
	return s.mailerService
}
