package dispatchers

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/services"
)

// expectedDiscordErrorCodes are Discord API error codes that represent a
// permanent, expected delivery failure (the user closed their DMs, the bot
// lost access, etc.) rather than a bug in our code. Dispatch failures with
// one of these codes don't need a Go stack trace attached — the cause is
// already fully explained by the Discord error message itself.
var expectedDiscordErrorCodes = map[int]bool{
	50001: true, // Missing Access
	50007: true, // Cannot send messages to this user (DMs closed)
	50013: true, // Missing Permissions
	50278: true, // Cannot send messages to this user due to having no mutual guilds
}

// IsExpectedDiscordDeliveryError reports whether err is a Discord REST error
// with a code known to represent a normal, unrecoverable delivery failure
// (as opposed to an unexpected internal error worth a full stack trace).
func IsExpectedDiscordDeliveryError(err error) bool {
	var restErr *discordgo.RESTError
	if !errors.As(err, &restErr) || restErr.Message == nil {
		return false
	}
	return expectedDiscordErrorCodes[restErr.Message.Code]
}

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
		return fmt.Errorf("failed to send reminder embed: %w", err)
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

// NotifyChannelDispatchFailure best-effort DMs the reminder's owner when a
// channel-destination reminder fails to send (e.g. the bot lacks permission
// to post in that channel), so they have a chance to fix it instead of
// silently missing reminders. Failures here (e.g. the user has DMs closed)
// are swallowed — this is a courtesy notification, not the primary error
// reporting path (that's the reminder_errors record created by the caller).
func NotifyChannelDispatchFailure(session *discordgo.Session, account *models.Account, channelID string, dispatchErr error) {
	discordUserID := discordIDFromAccount(account)
	if discordUserID == "" {
		return
	}

	dmChannel, err := session.UserChannelCreate(discordUserID)
	if err != nil {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "⚠️ Reminder delivery failed",
		Description: fmt.Sprintf(
			"I couldn't deliver your reminder to <#%s>.\n\n**Reason:** %s\n\nMake sure I have permission to send messages in that channel, or update the reminder's destination.",
			channelID, dispatchErr.Error(),
		),
		Color: 0xE74C3C,
	}
	_, _ = session.ChannelMessageSendEmbed(dmChannel.ID, embed)
}

// discordIDFromAccount returns the account's linked Discord user ID, or "" if none.
func discordIDFromAccount(account *models.Account) string {
	for _, identity := range account.Identities {
		if identity.Provider == models.ProviderDiscord {
			return identity.ExternalID
		}
	}
	return ""
}
