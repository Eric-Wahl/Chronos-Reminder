package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/ericp/chronos-bot-reminder/internal/bot/logic"
)

// Register the preferences command with subcommands
func init() {
	RegisterCommand(&Command{
		Description: Description{
			Name:             "preferences",
			Emoji:            "⚙️",
			CategoryName:     "User",
			ShortDescription: "Manage your preferences",
			FullDescription:  "View your preferences, toggle whether Discord reminders (remindme/remindus) include a generated image in addition to the text, or toggle the Snooze button.",
			Usage:            "/preferences <show|discord-image|discord-snooze>",
			Example:          "/preferences show, /preferences discord-image enabled:false, /preferences discord-snooze enabled:false",
		},
		Data: &discordgo.ApplicationCommand{
			Name:        "preferences",
			Description: "Manage your preferences",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "show",
					Description: "Show your current preferences",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "discord-image",
					Description: "Enable or disable sending a reminder image on Discord",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "enabled",
							Description: "Whether to include a generated image with Discord reminders",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "discord-snooze",
					Description: "Enable or disable the Snooze button on Discord reminders",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "enabled",
							Description: "Whether to show a Snooze button on Discord reminders",
							Required:    true,
						},
					},
				},
			},
		},
		NeedsAccount: true,
		Run:          logic.PreferencesHandler,
	})
}
