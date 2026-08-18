package utils

import (
	"github.com/bwmarrin/discordgo"
	"github.com/ericp/chronos-bot-reminder/internal/services"
)

// sourceFromInteraction derives a short, human-readable identifier of where
// an interaction originated, e.g. "command:remindus" or
// "component:show_reminder_...". Falls back to the raw interaction type name
// when it can't be more specific.
func sourceFromInteraction(interaction *discordgo.InteractionCreate) string {
	if interaction == nil {
		return "unknown"
	}
	switch interaction.Type {
	case discordgo.InteractionApplicationCommand, discordgo.InteractionApplicationCommandAutocomplete:
		return "command:" + interaction.ApplicationCommandData().Name
	case discordgo.InteractionMessageComponent:
		return "component:" + interaction.MessageComponentData().CustomID
	case discordgo.InteractionModalSubmit:
		return "modal:" + interaction.ModalSubmitData().CustomID
	default:
		return "interaction"
	}
}

// LogBotError records an error to the bot_errors table, deriving the source
// and Discord user from the interaction when available. See
// services.LogBotError for the underlying storage/best-effort behavior.
// interaction may be nil when none is available (e.g. a background job).
func LogBotError(interaction *discordgo.InteractionCreate, source, title, message string) {
	var discordUserID *string
	if interaction != nil {
		if interaction.Member != nil && interaction.Member.User != nil {
			discordUserID = &interaction.Member.User.ID
		} else if interaction.User != nil {
			discordUserID = &interaction.User.ID
		}
	}
	services.LogBotError(source, title, message, discordUserID)
}
