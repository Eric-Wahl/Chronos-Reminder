package utils

import "github.com/bwmarrin/discordgo"

func BuildEmbed(session *discordgo.Session, title, description string, footerText *string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       ColorInfo,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: ClockLogo,
		},
	}

	if footerText != nil {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: *footerText,
		}
	}

	return embed
}

func SendEmbed(session *discordgo.Session, interaction *discordgo.InteractionCreate, title, description string, footerText *string) error {
	return SendEmbedDeferred(session, interaction, title, description, footerText, false)
}

func SendEmbedDeferred(session *discordgo.Session, interaction *discordgo.InteractionCreate, title, description string, footerText *string, deferred bool) error {
	embed := BuildEmbed(session, title, description, footerText)

	if deferred {
		// Use edit for deferred interactions
		_, err := session.InteractionResponseEdit(interaction.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})
		return err
	}

	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func BuildInfoEmbed(session *discordgo.Session, title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: "ℹ️ - " + description,
		Color:       ColorInfo,
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: session.State.User.AvatarURL(""),
			Text:    "Chronos Bot Reminder",
		},
	}
}

// SendInfo sends an info message as an interaction response
func SendInfo(session *discordgo.Session, interaction *discordgo.InteractionCreate, title, description string) error {
	embed := BuildInfoEmbed(session, title, description)

	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// BuildSuccessEmbed builds a success embed message
func BuildSuccessEmbed(session *discordgo.Session, title, description string, footerText *string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: "✅ - " + description,
		Color:       ColorSuccess,
		Author: &discordgo.MessageEmbedAuthor{
			IconURL: session.State.User.AvatarURL(""),
			Name:    "Chronos Bot Reminder",
		},
	}

	if footerText != nil {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: *footerText,
		}
	}

	return embed
}

// SendSuccess sends a success message as an interaction response
func SendSuccess(session *discordgo.Session, interaction *discordgo.InteractionCreate, title, description string, footerText *string) error {
	embed := BuildSuccessEmbed(session, title, description, footerText)

	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// BuildErrorEmbed builds an error embed message
func BuildErrorEmbed(session *discordgo.Session, title, description string, footerText *string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: "❌ - " + description,
		Color:       ColorError,
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: session.State.User.AvatarURL(""),
			Text:    "Chronos Bot Reminder",
		},
	}

	if footerText != nil {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: *footerText,
		}
	}

	return embed
}

// SendError without the footer
func SendError(session *discordgo.Session, interaction *discordgo.InteractionCreate, title, description string) error {
	return SendErrorDeferred(session, interaction, title, description, nil, false)
}

// SendErrorDeferred sends an error message as an interaction response with deferred support
func SendErrorDeferred(session *discordgo.Session, interaction *discordgo.InteractionCreate, title, description string, footerText *string, deferred bool) error {
	LogBotError(interaction, sourceFromInteraction(interaction), title, description)

	embed := BuildErrorEmbed(session, title, description, footerText)

	if deferred {
		// Use edit for deferred interactions
		_, err := session.InteractionResponseEdit(interaction.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})
		return err
	}

	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

// SendError sends an error message as an interaction response
func SendErrorDetailed(session *discordgo.Session, interaction *discordgo.InteractionCreate, title, description string, footerText *string) error {
	return SendErrorDetailedDeferred(session, interaction, title, description, footerText, false)
}

// SendErrorDetailedDeferred sends detailed error with deferred support
func SendErrorDetailedDeferred(session *discordgo.Session, interaction *discordgo.InteractionCreate, title, description string, footerText *string, deferred bool) error {
	LogBotError(interaction, sourceFromInteraction(interaction), title, description)

	embed := BuildErrorEmbed(session, title, description, footerText)

	if deferred {
		_, err := session.InteractionResponseEdit(interaction.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})
		return err
	}

	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

// BuildWarningEmbed builds a warning embed message
func BuildWarningEmbed(session *discordgo.Session, title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: "⚠️ - " + description,
		Color:       ColorWarning,
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: session.State.User.AvatarURL(""),
			Text:    "Chronos Bot Reminder",
		},
	}
}

// SendWarning sends a warning message as an interaction response
func SendWarning(session *discordgo.Session, interaction *discordgo.InteractionCreate, title, description string) error {
	embed := BuildWarningEmbed(session, title, description)

	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}
