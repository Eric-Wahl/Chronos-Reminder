package logic

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ericp/chronos-bot-reminder/internal/bot/utils"
	"github.com/ericp/chronos-bot-reminder/internal/database"
	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/services"
)

// BuildReminderEmbed creates a detailed embed for a reminder with its destinations
func BuildReminderEmbed(session *discordgo.Session, reminder *models.Reminder) *discordgo.MessageEmbed {
	// Format the reminder time in the account's timezone. RemindAtUTC is
	// stored in UTC, so formatting it directly (as this used to do) shows
	// the wrong wall-clock time to anyone not in UTC.
	remindTime := reminder.RemindAtUTC
	tzLabel := "UTC"
	if reminder.Account != nil && reminder.Account.Timezone != nil {
		if loc, err := time.LoadLocation(reminder.Account.Timezone.IANALocation); err == nil {
			remindTime = remindTime.In(loc)
			tzLabel = reminder.Account.Timezone.IANALocation
		}
	}
	remindTimeStr := fmt.Sprintf("%s (%s)", remindTime.Format("Monday, January 2, 2006 at 15:04"), tzLabel)

	// Determine status
	status := "✅ Active"
	if services.IsPaused(int(reminder.Recurrence)) {
		status = "⏸️ Paused"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📝 Reminder Details",
		Description: fmt.Sprintf("**Message:** %s", reminder.Message),
		Color:       utils.ColorInfo,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: utils.ClockLogo,
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "⏰ Remind Time",
				Value:  remindTimeStr,
				Inline: true,
			},
			{
				Name:   "📊 Status",
				Value:  status,
				Inline: true,
			},
			{
				Name:   "🆔 Reminder ID",
				Value:  reminder.ID.String(),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: session.State.User.AvatarURL(""),
			Text:    "Chronos Bot Reminder",
		},
	}

	// Add owner information if different from viewer
	if reminder.Account != nil {
		ownerInfo := fmt.Sprintf("Account ID: %s", reminder.Account.ID.String())
		// Try to get Discord identity for better display
		repo := database.GetRepositories()
		identities, err := repo.Identity.GetByAccountID(reminder.Account.ID)
		if err == nil && len(identities) > 0 {
			// Look for Discord identity
			for _, identity := range identities {
				if identity.Provider == models.ProviderDiscord {
					ownerInfo = fmt.Sprintf("<@%s>", identity.ExternalID)
					break
				}
			}
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "👤 Created By",
			Value:  ownerInfo,
			Inline: true,
		})
	}

	// If the reminder has a recurrence different from one-time, show it
	recurrenceType := services.GetRecurrenceType(int(reminder.Recurrence))
	if recurrenceType != services.RecurrenceOnce {
		recurrenceStr := services.GetRecurrenceTypeLabel(recurrenceType)
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🔁 Recurrence",
			Value:  recurrenceStr,
			Inline: true,
		})
	}

	// Add destinations
	if len(reminder.Destinations) > 0 {
		for i, dest := range reminder.Destinations {
			destField := buildDestinationField(dest, i+1)
			embed.Fields = append(embed.Fields, destField)
		}
	}

	return embed
}

// buildDestinationField creates an embed field for a destination
func buildDestinationField(dest models.ReminderDestination, index int) *discordgo.MessageEmbedField {
	fieldName := fmt.Sprintf("📍 Destination %d", index)
	var fieldValue string

	switch dest.Type {
	case models.DestinationDiscordDM:
		if userID, exists := dest.Metadata["user_id"]; exists {
			if userIDStr, ok := userID.(string); ok {
				fieldValue = fmt.Sprintf("**Type:** Discord DM\n**User:** <@%s>", userIDStr)
			} else {
				fieldValue = "**Type:** Discord DM\n**User:** Unknown"
			}
		} else {
			fieldValue = "**Type:** Discord DM\n**User:** Invalid configuration"
		}

	case models.DestinationDiscordChannel:
		channelInfo := "**Type:** Discord Channel\n"
		
		if channelID, exists := dest.Metadata["channel_id"]; exists {
			if channelIDStr, ok := channelID.(string); ok {
				channelInfo += fmt.Sprintf("**Channel:** <#%s>\n", channelIDStr)
			} else {
				channelInfo += "**Channel:** Unknown\n"
			}
		} else {
			channelInfo += "**Channel:** Invalid configuration\n"
		}

		if guildID, exists := dest.Metadata["guild_id"]; exists {
			if guildIDStr, ok := guildID.(string); ok {
				channelInfo += fmt.Sprintf("**Server ID:** %s", guildIDStr)
			}
		}

		fieldValue = channelInfo

	case models.DestinationWebhook:
		if url, exists := dest.Metadata["url"]; exists {
			if urlStr, ok := url.(string); ok {
				// Mask the webhook URL for security
				maskedURL := maskWebhookURL(urlStr)
				fieldValue = fmt.Sprintf("**Type:** Webhook\n**URL:** %s", maskedURL)
			} else {
				fieldValue = "**Type:** Webhook\n**URL:** Invalid configuration"
			}
		} else {
			fieldValue = "**Type:** Webhook\n**URL:** Invalid configuration"
		}

		// Add platform information if specified
		if platform, exists := dest.Metadata["platform"]; exists {
			if platformStr, ok := platform.(string); ok && platformStr != "" && platformStr != "generic" {
				fieldValue += fmt.Sprintf("\n**Platform:** %s", platformStr)
			}
		}

		// Add any additional webhook metadata
		if name, exists := dest.Metadata["name"]; exists {
			if nameStr, ok := name.(string); ok {
				fieldValue += fmt.Sprintf("\n**Name:** %s", nameStr)
			}
		}

	default:
		fieldValue = fmt.Sprintf("**Type:** Unknown (%s)\n**Configuration:** %v", dest.Type, dest.Metadata)
	}

	return &discordgo.MessageEmbedField{
		Name:   fieldName,
		Value:  fieldValue,
		Inline: false,
	}
}

// maskWebhookURL masks sensitive parts of a webhook URL
func maskWebhookURL(url string) string {
	// Split by '/' and mask the token part
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		// Find the token part (usually the last part and quite long)
		for i := len(parts) - 1; i >= 0; i-- {
			if len(parts[i]) > 20 { // Webhook tokens are typically long
				parts[i] = parts[i][:8] + "..." + parts[i][len(parts[i])-4:]
				break
			}
		}
	}
	return strings.Join(parts, "/")
}

// CanAccessReminder checks if the user has permission to access the reminder
func CanAccessReminder(interaction *discordgo.InteractionCreate, account *models.Account, reminder *models.Reminder) bool {
	// User can always access their own reminders
	if reminder.AccountID == account.ID {
		return true
	}

	// If in a server and user is admin, check if the reminder has a server destination
	if interaction.GuildID != "" && interaction.Member != nil {
		// Check if user has administrator permissions
		permissions := interaction.Member.Permissions
		isAdmin := (permissions & discordgo.PermissionAdministrator) == discordgo.PermissionAdministrator

		if isAdmin {
			// Check if reminder has a destination for this server
			for _, dest := range reminder.Destinations {
				if dest.Type == models.DestinationDiscordChannel {
					if guildID, exists := dest.Metadata["guild_id"]; exists {
						if guildIDStr, ok := guildID.(string); ok && guildIDStr == interaction.GuildID {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

// CanSnoozeReminder checks if a reminder can be snoozed
func CanSnoozeReminder(reminder *models.Reminder) (bool, string) {
	// Check if reminder is already snoozed
	if reminder.SnoozedAtUTC != nil {
		return false, "This reminder is already snoozed."
	}

	return true, ""
}