package commands

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

// reminderHandler handles the reminder creation command
func reminderHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account) error {
	options := interaction.ApplicationCommandData().Options

	var message string
	var dateStr string
	var timeStr string
	var recurrenceType string = "ONCE" // Default to ONCE

	// Parse command options
	for _, option := range options {
		switch option.Name {
		case "message":
			message = option.StringValue()
		case "date":
			dateStr = option.StringValue()
		case "time":
			timeStr = option.StringValue()
		case "recurrence":
			if option.StringValue() != "" {
				recurrenceType = option.StringValue()
			}
		}
	}

	// Load account timezone for parsing
	repo := database.GetRepositories()

	// Parse the reminder date and time in user's timezone
	parsedTime, err := services.ParseReminderDateTimeInTimezone(dateStr, timeStr, account.Timezone.IANALocation)
	if err != nil {
		return utils.SendError(session, interaction, "Invalid Date/Time Format", 
			fmt.Sprintf("Could not parse the date '%s' and time '%s'. Please check your date and time formats.", dateStr, timeStr))
	}

	location, err := time.LoadLocation(account.Timezone.IANALocation)
	if err != nil {
		return utils.SendError(session, interaction, "Invalid Timezone", 
			fmt.Sprintf("Could not load timezone '%s'. Please check your timezone settings.", account.Timezone.IANALocation))
	}
	now := time.Now().In(location)
	// If the parsed reminder time is before the current time, return an error
	if parsedTime.Before(now) {
		return utils.SendError(session, interaction, "Invalid Date/Time", 
			"The specified date and time is in the past. Please provide a future date and time for the reminder. You entered: " + parsedTime.Format(time.RFC3339) +
			". Current time is: " + now.Format(time.RFC3339))
	}

	// Get recurrence type value
	recurrenceTypeValue, exists := services.RecurrenceTypeMap[strings.ToUpper(recurrenceType)]
	if !exists {
		return utils.SendError(session, interaction, "Invalid Recurrence Type",
			fmt.Sprintf("Invalid recurrence type '%s'. Valid options are: ONCE, YEARLY, MONTHLY, WEEKLY, DAILY, HOURLY, WORKDAYS, WEEKEND.", recurrenceType))
	}

	reminderCount, err := repo.Reminder.CountByAccountID(account.ID)
	if err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to check your reminder count.")
	}
	if reminderCount >= services.MaxRemindersPerAccount {
		return utils.SendError(session, interaction, "Reminder Limit Reached",
			fmt.Sprintf("You have reached the maximum of %d reminders. Please delete some before creating new ones.", services.MaxRemindersPerAccount))
	}

	// Create the reminder with UTC time
	reminder := &models.Reminder{
		AccountID:   account.ID,
		RemindAtUTC: parsedTime.UTC(),
		Message:     message,
		Recurrence:  int16(services.BuildRecurrenceState(recurrenceTypeValue, false)),
	}

	// Save the reminder to database
	if err := repo.Reminder.Create(reminder, true); err != nil {
		return utils.SendError(session, interaction, "Database Error", 
			"Failed to save the reminder. Please try again later.")
	}

	// userId is either the interaction user ID or the member user ID
	var userID string
	if interaction.Member != nil && interaction.Member.User != nil {
		userID = interaction.Member.User.ID
	} else if interaction.User != nil {
		userID = interaction.User.ID
	}

	// Create the discord_dm destination
	destination := &models.ReminderDestination{
		ReminderID: reminder.ID,
		Type:       models.DestinationDiscordDM,
		Metadata: models.JSONB{
			"user_id": userID,
		},
	}

	if err := repo.ReminderDestination.Create(destination); err != nil {
		// If destination creation fails, we should clean up the reminder
		repo.Reminder.Delete(reminder.ID, true)
		return utils.SendError(session, interaction, "Database Error", 
			"Failed to set up reminder destination. Please try again later.")
	}

	// Format response message
	var recurrenceText string
	if recurrenceType == "ONCE" {
		recurrenceText = "This is a one-time reminder."
	} else {
		recurrenceText = fmt.Sprintf("This reminder will repeat: %s", strings.ToLower(recurrenceType))
	}

	// Load account timezone for display
	var displayTime string
	if account != nil && account.Timezone != nil {
		// Display the local time as entered by the user
		displayTime = parsedTime.Format("Monday, January 2, 2006 at 15:04")
	} else {
		// Display in the same timezone as the parsed time was created
		displayTime = parsedTime.Format("Monday, January 2, 2006 at 15:04")
	}

	description := fmt.Sprintf("**Content:** %s\n**Remind Time:** %s", 
		message, displayTime)

	return utils.SendEmbed(session, interaction, "Reminder Created! ⏰", description, &recurrenceText)
}

// Register the reminder command
func init() {
	autocompleteFunc := AutocompleteFunc(DateAutocompleteHandler)
	
	RegisterCommand(&Command{
		Description: Description{
			Name:             "remindme",
			Emoji:            "⏰",
			CategoryName:     "Reminders",
			ShortDescription: "Create a new reminder",
			FullDescription:  "Create a new reminder that will be sent to you via direct message at the specified date and time",
			Usage:            "/remindme message:<text> date:<date> time:<time> [recurrence:<type>]",
			Example:          "/remindme message:\"Take medicine\" date:\"25/12/2024\" time:\"15:30\" recurrence:daily",
		},
		Data: &discordgo.ApplicationCommand{
			Name:        "remindme",
			Description: "Create a new reminder",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "message",
					Description: "The reminder message",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "date",
					Description: "The date for the reminder (e.g., 'today', 'tomorrow', '25/12/2024', '2024-12-25')",
					Required:    true,
					Autocomplete: true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "time",
					Description: "The time for the reminder (e.g., '15:30', '3pm', '9:30am')",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "recurrence",
					Description: "How often to repeat (default: once)",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "Once",
							Value: "ONCE",
						},
						{
							Name:  "Hourly",
							Value: "HOURLY",
						},
						{
							Name:  "Daily",
							Value: "DAILY",
						},
						{
							Name:  "Weekly",
							Value: "WEEKLY",
						},
						{
							Name:  "Monthly",
							Value: "MONTHLY",
						},
						{
							Name:  "Yearly",
							Value: "YEARLY",
						},
						{
							Name:  "Workdays (Mon-Fri)",
							Value: "WORKDAYS",
						},
						{
							Name:  "Weekends (Sat-Sun)",
							Value: "WEEKEND",
						},
					},
				},
			},
		},
		NeedsAccount: true,
		Run:          reminderHandler,
		Autocomplete: &autocompleteFunc,
	})
}
