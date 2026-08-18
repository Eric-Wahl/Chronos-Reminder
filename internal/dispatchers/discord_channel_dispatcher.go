package dispatchers

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/ericp/chronos-bot-reminder/internal/bot"
	"github.com/ericp/chronos-bot-reminder/internal/database/models"
)

type DiscordChannelDispatcher struct {
	session *discordgo.Session
}

func NewDiscordChannelDispatcher() *DiscordChannelDispatcher {
	return &DiscordChannelDispatcher{
		session: bot.GetDiscordSession(),
	}
}

// GetSupportedType returns the destination type this dispatcher supports
func (d *DiscordChannelDispatcher) GetSupportedType() models.DestinationType {
	return models.DestinationDiscordChannel
}

// Dispatch sends a reminder message to a Discord channel
func (d *DiscordChannelDispatcher) Dispatch(reminder *models.Reminder, destination *models.ReminderDestination, account *models.Account) error {
	// Validate destination type
	if destination.Type != models.DestinationDiscordChannel {
		return fmt.Errorf("invalid destination type for Discord channel dispatcher: %s", destination.Type)
	}

	// Build the data from the metadata
	// {"guild_id": "912661874871533588", "channel_id": "913222458251837441", "mention_role_id": "role_id"}
	channelID, exists := destination.Metadata["channel_id"]
	if !exists {
		return fmt.Errorf("channel_id not found in destination metadata")
	}
	channelIDStr, ok := channelID.(string)
	if !ok {
		return fmt.Errorf("channel_id is not a string: %v", channelID)
	}

	// Extract optional role mention ID
	var roleMentionID string
	if mentionRole, exists := destination.Metadata["mention_role_id"]; exists {
		if mentionRoleStr, ok := mentionRole.(string); ok {
			roleMentionID = mentionRoleStr
		}
	}

	if err := DiscordSend(d.session, reminder, channelIDStr, account, roleMentionID); err != nil {
		NotifyChannelDispatchFailure(d.session, account, channelIDStr, err)
		return fmt.Errorf("failed to send reminder to channel %s: %w", channelIDStr, err)
	}

	return nil
}