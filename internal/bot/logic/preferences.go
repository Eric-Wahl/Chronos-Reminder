package logic

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/ericp/chronos-bot-reminder/internal/bot/utils"
	"github.com/ericp/chronos-bot-reminder/internal/database"
	"github.com/ericp/chronos-bot-reminder/internal/database/models"
)

// PreferencesHandler routes the /preferences command to its subcommands
func PreferencesHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account) error {
	options := interaction.ApplicationCommandData().Options
	if len(options) == 0 {
		return utils.SendError(session, interaction, "Invalid Command", "Please specify a subcommand.")
	}

	subcommand := options[0]
	switch subcommand.Name {
	case "show":
		return preferencesShowHandler(session, interaction, account)
	case "discord-image":
		return preferencesDiscordImageHandler(session, interaction, account, subcommand.Options)
	default:
		return utils.SendError(session, interaction, "Unknown Subcommand", "The specified subcommand is not recognized.")
	}
}

// preferencesShowHandler displays the account's current preferences
func preferencesShowHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account) error {
	status := "✅ Enabled"
	if !account.DiscordSendImage() {
		status = "❌ Disabled"
	}

	return utils.SendInfo(session, interaction, "⚙️ Your Preferences",
		fmt.Sprintf("**Send reminder image:** %s", status))
}

// preferencesDiscordImageHandler toggles whether reminders sent to Discord
// (via /remindme and /remindus) include a generated image in addition to the
// text embed.
func preferencesDiscordImageHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	var enabled bool
	var provided bool
	for _, option := range options {
		if option.Name == "enabled" {
			enabled = option.BoolValue()
			provided = true
		}
	}

	if !provided {
		return utils.SendError(session, interaction, "Missing Option", "Please specify whether to enable or disable the reminder image.")
	}

	if account.Preferences == nil {
		account.Preferences = models.JSONB{}
	}
	account.Preferences[models.PrefDiscordSendImage] = enabled

	repo := database.GetRepositories()
	if err := repo.Account.Update(account); err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to update your preference. Please try again later.")
	}

	status := "enabled"
	if !enabled {
		status = "disabled"
	}
	return utils.SendSuccess(session, interaction, "⚙️ Preference Updated",
		fmt.Sprintf("Sending a reminder image on Discord is now **%s**.", status), nil)
}
