package dispatchers

import (
	"bytes"
	"fmt"
	"image/png"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/services"
)

// =====================================================================
// Contains everything that may be used in multiple dispatchers
// =====================================================================

// DiscordSend handles sending reminders via Discord
func DiscordSend(session *discordgo.Session, reminder *models.Reminder, channelID string, account *models.Account, roleMentionID ...string) error {
	// Convert the due date to the user's local timezone if available
	loc, err := time.LoadLocation(account.Timezone.IANALocation)
	if err == nil {
		reminder.RemindAtUTC = reminder.RemindAtUTC.In(loc)
	}

	// Add a Snooze button to the message, unless the account has disabled it
	var components []discordgo.MessageComponent
	if account.DiscordSnoozeEnabled() {
		button := discordgo.Button{
			Label:    "Snooze",
			Style:    discordgo.SecondaryButton,
			CustomID: "reminder_request_snooze_" + fmt.Sprint(reminder.ID),
		}
		components = []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{button},
			},
		}
	}

	// Build message content with role mention if provided
	var messageContent string
	if len(roleMentionID) > 0 && roleMentionID[0] != "" {
		messageContent = fmt.Sprintf("<@&%s>", roleMentionID[0])
	}

	// Some users prefer a single plain text reminder over the generated image
	if !account.DiscordSendImage() {
		textEmbed := &discordgo.MessageEmbed{
			Title:       "⌛ Reminder",
			Description: fmt.Sprintf("**%s**\n\n🕒 %s", reminder.Message, reminder.RemindAtUTC.Format("Monday, January 2, 2006 at 15:04")),
			Color:       0xCEA04D,
		}
		msg := &discordgo.MessageSend{
			Content:    messageContent,
			Embeds:     []*discordgo.MessageEmbed{textEmbed},
			Components: components,
		}
		if _, err := session.ChannelMessageSendComplex(channelID, msg); err != nil {
			return fmt.Errorf("failed to send reminder: %w", err)
		}
		return nil
	}

	// Create the reminder message
	embed := &discordgo.MessageEmbed{
		Title: "⌛ | You have a new reminder ! ⌛",
		Color: 0xCEA04D,
	}

	// Send the message
	if _, err := session.ChannelMessageSendEmbed(channelID, embed); err != nil {
		return fmt.Errorf("failed to send DM  %w", err)
	}

	img, err := services.NewDrawService("./assets").GenerateReminderImage(services.TextOverlay{
		Label: reminder.Message,
		Date:  reminder.RemindAtUTC,
	})

	// Check for errors
	if err != nil {
		return fmt.Errorf("failed to generate reminder image: %w", err)
	}

	// Encode img (image.Image) to PNG and wrap in io.Reader
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("failed to encode reminder image: %w", err)
	}

	msg := &discordgo.MessageSend{
		Content: messageContent,
		File: &discordgo.File{
			Name:        "reminder.png",
			ContentType: "image/png",
			Reader:      &buf,
		},
		Components: components,
	}
	_, err = session.ChannelMessageSendComplex(channelID, msg)
	if err != nil {
		return fmt.Errorf("failed to send reminder: %w", err)
	}

	return nil
}
