package services

import (
	"log"

	"github.com/ericp/chronos-bot-reminder/internal/database"
	"github.com/ericp/chronos-bot-reminder/internal/database/models"
)

// LogBotError records an error to the bot_errors table so it's visible
// somewhere other than server stdout logs — used both for errors shown to a
// Discord user (SendError/SendErrorDetailed) and for background job failures
// that currently have nowhere to go but stdout (e.g. the scheduler failing to
// reschedule a recurring reminder, which otherwise silently stops firing
// forever with no trace anywhere).
//
// source identifies where the error came from, e.g. "command:remindus" or
// "engine:scheduler". discordUserID is optional; pass nil for errors with no
// associated Discord user (background jobs). Best-effort: a failure to write
// the log itself only prints to stdout, it never blocks or fails the caller.
func LogBotError(source, title, message string, discordUserID *string) {
	botError := &models.BotError{
		Source:  source,
		Title:   title,
		Message: message,
	}

	repo := database.GetRepositories()

	if discordUserID != nil && *discordUserID != "" {
		botError.DiscordUserID = discordUserID

		// Best-effort account resolution; a lookup failure shouldn't
		// prevent the error itself from being recorded.
		if identity, err := repo.Identity.GetByProviderAndExternalID(models.ProviderDiscord, *discordUserID); err == nil && identity != nil {
			botError.AccountID = &identity.AccountID
		}
	}

	if err := repo.BotError.Create(botError); err != nil {
		log.Printf("[BOT_ERROR_LOG] - ⚠️ Failed to record bot error (source=%s, title=%s): %v", source, title, err)
	}
}
