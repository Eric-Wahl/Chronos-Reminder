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


const RemindersPerPage = 10

// HandleListReminders handles the list subcommand
func HandleListReminders(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	repo := database.GetRepositories()

	// Get all reminders for the account
	reminders, err := repo.Reminder.GetByAccountIDWithDestinations(account.ID)
	if err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to retrieve reminders.")
	}

	if len(reminders) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "📝 Your Reminders",
			Description: "You don't have any reminders yet. Use `/remindme` to create your first reminder!",
			Color:       0x3498db,
		}

		return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
	}

	// Convert to pointer slice for buildRemindersListEmbed. GetByAccountIDWithDestinations
	// doesn't preload Account, so attach the already-loaded account (these are
	// all its own reminders) — otherwise the embed can't convert times to the
	// user's timezone and silently omits them.
	remindersPointers := make([]*models.Reminder, len(reminders))
	for i := range reminders {
		reminders[i].Account = account
		remindersPointers[i] = &reminders[i]
	}

	// Build the reminders list embed
	embed := BuildRemindersListEmbed(remindersPointers, 1, len(reminders))

	// Create components with "Show First Reminder" button
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: fmt.Sprintf("show_reminder_%s", reminders[0].ID.String()),
					Label:    "Show First Reminder",
					Style:    discordgo.PrimaryButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "👁️"},
				},
			},
		},
	}

	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

// buildRemindersListEmbed creates an embed with a list of reminders
func BuildRemindersListEmbed(reminders []*models.Reminder, page, total int) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: "📝 Your Reminders",
		Color: 0x3498db,
	}

	if total == 0 {
		embed.Description = "You don't have any reminders yet."
		return embed
	}

	var description strings.Builder
	description.WriteString(fmt.Sprintf("You have **%d** reminder", total))
	if total != 1 {
		description.WriteString("s")
	}
	description.WriteString(":\n\n")

	for i, reminder := range reminders {
		// Status emoji
		statusEmoji := "✅"
		if services.IsPaused(int(reminder.Recurrence)) {
			statusEmoji = "⏸️"
		}

		// Truncate message if too long
		message := reminder.Message
		if len(message) > 50 {
			message = message[:47] + "..."
		}

		description.WriteString(fmt.Sprintf("%s **%d.** %s\n", statusEmoji, i+1, message))
		
		// Add schedule info with correct recurrence type
		recurrenceType := services.GetRecurrenceType(int(reminder.Recurrence))
		recurrenceLabel := services.GetRecurrenceTypeLabel(recurrenceType)
		description.WriteString(fmt.Sprintf("    🕐 %s\n", recurrenceLabel))
		
		// Add time in user's timezone
		if reminder.Account != nil && reminder.Account.Timezone != nil {
			loc, err := time.LoadLocation(reminder.Account.Timezone.IANALocation)
			if err == nil {
				userTime := reminder.RemindAtUTC.In(loc)
				description.WriteString(fmt.Sprintf("    � %s\n", userTime.Format("Jan 02, 15:04 MST")))
			}
		}
		
		description.WriteString("\n")
	}

	embed.Description = description.String()
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("Page %d - Use the buttons to navigate", page),
	}

	return embed
}

// HandleShowReminderFromList handles showing a specific reminder from the list with navigation
func HandleShowReminderFromList(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account) error {
	customID := interaction.MessageComponentData().CustomID
	reminderIDStr := strings.TrimPrefix(customID, "show_reminder_")

	repo := database.GetRepositories()

	// Get all reminders for the account to determine position
	allReminders, err := repo.Reminder.GetByAccountIDWithDestinations(account.ID)
	if err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to retrieve reminders.")
	}

	// Find the current reminder index
	var currentIndex int = -1
	var currentReminder *models.Reminder
	
	for i, reminder := range allReminders {
		if reminder.ID.String() == reminderIDStr {
			currentIndex = i
			currentReminder = &reminder
			break
		}
	}

	if currentReminder == nil {
		return utils.SendError(session, interaction, "Reminder Not Found", "The specified reminder could not be found.")
	}

	// Check permissions
	if !CanAccessReminder(interaction, account, currentReminder) {
		return utils.SendError(session, interaction, "Permission Denied", "You don't have permission to access this reminder.")
	}

	// GetByAccountIDWithDestinations doesn't preload Account, so attach the
	// already-loaded account so the embed can convert to the user's timezone.
	currentReminder.Account = account

	// Build the reminder embed
	embed := BuildReminderEmbed(session, currentReminder)

	// Create navigation components
	var components []discordgo.MessageComponent
	var buttons []discordgo.MessageComponent

	// Previous button (if not first)
	if currentIndex > 0 {
		buttons = append(buttons, discordgo.Button{
			CustomID: fmt.Sprintf("show_reminder_%s", allReminders[currentIndex-1].ID.String()),
			Label:    "Previous",
			Style:    discordgo.SecondaryButton,
			Emoji:    &discordgo.ComponentEmoji{Name: "⬅️"},
		})
	}

	// Back to list button
	buttons = append(buttons, discordgo.Button{
		CustomID: "back_to_list",
		Label:    "Back to List",
		Style:    discordgo.SecondaryButton,
		Emoji:    &discordgo.ComponentEmoji{Name: "📝"},
	})

	// Next button (if not last)
	if currentIndex < len(allReminders)-1 {
		buttons = append(buttons, discordgo.Button{
			CustomID: fmt.Sprintf("show_reminder_%s", allReminders[currentIndex+1].ID.String()),
			Label:    "Next",
			Style:    discordgo.SecondaryButton,
			Emoji:    &discordgo.ComponentEmoji{Name: "➡️"},
		})
	}

	if len(buttons) > 0 {
		components = append(components, discordgo.ActionsRow{
			Components: buttons,
		})
	}

	// Add reminder position info to embed
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("Reminder %d of %d", currentIndex+1, len(allReminders)),
	}

	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

// HandleBackToList handles going back to the reminders list
func HandleBackToList(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account) error {
	repo := database.GetRepositories()

	// Get all reminders for the account
	reminders, err := repo.Reminder.GetByAccountIDWithDestinations(account.ID)
	if err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to retrieve reminders.")
	}

	if len(reminders) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "📝 Your Reminders",
			Description: "You don't have any reminders yet. Use `/remindme` to create your first reminder!",
			Color:       0x3498db,
		}

		return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: []discordgo.MessageComponent{},
			},
		})
	}

	// Convert to pointer slice for buildRemindersListEmbed. GetByAccountIDWithDestinations
	// doesn't preload Account, so attach the already-loaded account.
	remindersPointers := make([]*models.Reminder, len(reminders))
	for i := range reminders {
		reminders[i].Account = account
		remindersPointers[i] = &reminders[i]
	}

	// Build the reminders list embed
	embed := BuildRemindersListEmbed(remindersPointers, 1, len(reminders))

	// Create components with "Show First Reminder" button
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: fmt.Sprintf("show_reminder_%s", reminders[0].ID.String()),
					Label:    "Show First Reminder",
					Style:    discordgo.PrimaryButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "👁️"},
				},
			},
		},
	}

	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}