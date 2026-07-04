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

func hasPerm(perms int64, perm int64) bool {
    return perms&perm == perm
}

// getGuildRolesCache fetches guild roles once and caches them
func getGuildRolesCache(session *discordgo.Session, guildID string) (map[string]*discordgo.Role, error) {
	roles, err := session.GuildRoles(guildID)
	if err != nil {
		return nil, err
	}
	roleMap := make(map[string]*discordgo.Role)
	for _, role := range roles {
		roleMap[role.ID] = role
	}
	return roleMap, nil
}

// getRoleFromCache retrieves a role from cache or state
func getRoleFromCache(session *discordgo.Session, guildID, roleID string, roleCache map[string]*discordgo.Role) *discordgo.Role {
	// Check cache first
	if role, exists := roleCache[roleID]; exists {
		return role
	}
	// Try state
	if role, err := session.State.Role(guildID, roleID); err == nil {
		return role
	}
	return nil
}

// remindUsHandler handles the remind us command
func remindUsHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account) error {
	// Defer the interaction response immediately to avoid timeout
	session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Check if the command is being used in a server (not DM)
	if interaction.GuildID == "" {
		return utils.SendErrorDeferred(session, interaction, "Server Required", 
			"The `/remindus` command can only be used in a server, not in direct messages. Use `/remindme` for personal reminders.", nil, true)
	}

	options := interaction.ApplicationCommandData().Options

	var message string
	var dateStr string
	var timeStr string
	var channelID string
	var roleID string
	var recurrenceType string = "ONCE"

	// Parse command options
	for _, option := range options {
		switch option.Name {
		case "message":
			message = option.StringValue()
		case "date":
			dateStr = option.StringValue()
		case "time":
			timeStr = option.StringValue()
		case "channel":
			if channel := option.ChannelValue(session); channel != nil {
				channelID = channel.ID
			} else if option.Value != nil {
				if channelIDStr, ok := option.Value.(string); ok {
					channelID = channelIDStr
				}
			}
		case "role":
			if role := option.RoleValue(session, interaction.GuildID); role != nil {
				roleID = role.ID
			} else if option.Value != nil {
				if roleIDStr, ok := option.Value.(string); ok {
					roleID = roleIDStr
				}
			}
		case "recurrence":
			if option.StringValue() != "" {
				recurrenceType = option.StringValue()
			}
		}
	}

	// Validate that a channel was selected
	if channelID == "" {
		return utils.SendErrorDeferred(session, interaction, "Channel Required", 
			"Please select a channel where the reminder should be sent.", nil, true)
	}

	// Validate required fields
	if message == "" {
		return utils.SendErrorDeferred(session, interaction, "Message Required", 
			"Please provide a message for the reminder.", nil, true)
	}
	
	if dateStr == "" {
		return utils.SendErrorDeferred(session, interaction, "Date Required", 
			"Please provide a date for the reminder.", nil, true)
	}
	
	if timeStr == "" {
		return utils.SendErrorDeferred(session, interaction, "Time Required", 
			"Please provide a time for the reminder.", nil, true)
	}

	// Get the user ID for permission checking
	var userID string
	if interaction.Member != nil && interaction.Member.User != nil {
		userID = interaction.Member.User.ID
	} else if interaction.User != nil {
		userID = interaction.User.ID
	} else {
		return utils.SendErrorDeferred(session, interaction, "User Information Missing", 
			"Could not determine user information for permission check.", nil, true)
	}

	// Cache guild roles to avoid multiple API calls
	roleCache, err := getGuildRolesCache(session, interaction.GuildID)
	if err != nil {
		return utils.SendErrorDeferred(session, interaction, "Permission Check Failed", 
			"Could not verify your permissions for the selected channel.", nil, true)
	}

	// Verify the user has manage channel permissions, administrator permissions, or is the server owner
	channelPerms, err := session.UserChannelPermissions(userID, channelID)
	if err != nil {
		return utils.SendErrorDeferred(session, interaction, "Permission Check Failed", 
			"Could not verify your permissions for the selected channel.", nil, true)
	}

	userPerms := interaction.Member.Permissions

	// Check if user is server owner
	guild, err := session.Guild(interaction.GuildID)
	isAllowed := err == nil && (guild.OwnerID == userID || hasPerm(userPerms, discordgo.PermissionAdministrator) || hasPerm(userPerms, discordgo.PermissionManageChannels) || hasPerm(channelPerms, discordgo.PermissionManageChannels))

	if !isAllowed {
		return utils.SendErrorDeferred(session, interaction, "Insufficient Permissions", 
			"You need 'Manage Channel', 'Administrator' permission, or be the server owner to create reminders in the selected channel.", nil, true)
	}

	// If a role is specified, validate role mention permissions
	if roleID != "" {
		botMember, err := session.GuildMember(interaction.GuildID, session.State.User.ID)
		if err != nil {
			return utils.SendErrorDeferred(session, interaction, "Bot Permission Check Failed", 
				"Could not verify bot's permissions to mention roles.", nil, true)
		}

		// Check guild-wide permissions
		var botHasPermission bool
		for _, botRoleID := range botMember.Roles {
			role := getRoleFromCache(session, interaction.GuildID, botRoleID, roleCache)
			if role == nil {
				continue
			}
			
			if hasPerm(role.Permissions, discordgo.PermissionAdministrator) || 
				hasPerm(role.Permissions, discordgo.PermissionMentionEveryone) || 
				hasPerm(role.Permissions, discordgo.PermissionManageRoles) {
				botHasPermission = true
				break
			}
		}

		// If no guild-wide permission found, check channel-specific permissions
		if !botHasPermission {
			botPerms, err := session.UserChannelPermissions(session.State.User.ID, channelID)
			if err == nil && (hasPerm(botPerms, discordgo.PermissionMentionEveryone) || 
				hasPerm(botPerms, discordgo.PermissionManageRoles) || 
				hasPerm(botPerms, discordgo.PermissionAdministrator)) {
				botHasPermission = true
			}
		}

		if !botHasPermission {
			return utils.SendErrorDeferred(session, interaction, "Bot Insufficient Permissions", 
				"The bot needs 'Mention Everyone', 'Manage Roles', or 'Administrator' permission to mention roles in reminders.", nil, true)
		}

		// Get the role to validate
		role := getRoleFromCache(session, interaction.GuildID, roleID, roleCache)
		if role == nil {
			return utils.SendErrorDeferred(session, interaction, "Invalid Role", 
				"The specified role could not be found.", nil, true)
		}

		// Check if user has manage roles permission
		if !hasPerm(userPerms, discordgo.PermissionManageRoles) {
			return utils.SendErrorDeferred(session, interaction, "Role Permission Required", 
				"You need 'Manage Roles' permission to mention roles in reminders.", nil, true)
		}

		// Bot must be able to mention the role (role must be lower than bot's highest role)
		botHighestRolePos := -1
		for _, botRoleID := range botMember.Roles {
			botRole := getRoleFromCache(session, interaction.GuildID, botRoleID, roleCache)
			if botRole != nil && botRole.Position > botHighestRolePos {
				botHighestRolePos = botRole.Position
			}
		}

		if role.Position >= botHighestRolePos {
			return utils.SendErrorDeferred(session, interaction, "Bot Role Hierarchy Insufficient", 
				"The bot's highest role must be higher than the specified role to mention it in reminders.", nil, true)
		}
	}

	// Parse the reminder date and time in user's timezone
	parsedTime, err := services.ParseReminderDateTimeInTimezone(dateStr, timeStr, account.Timezone.IANALocation)
	if err != nil {
		return utils.SendErrorDeferred(session, interaction, "Invalid Date/Time Format", 
			fmt.Sprintf("Could not parse the date '%s' and time '%s'. Please check your date and time formats.", dateStr, timeStr), nil, true)
	}

	location, err := time.LoadLocation(account.Timezone.IANALocation)
	if err != nil {
		return utils.SendErrorDeferred(session, interaction, "Invalid Timezone", 
			fmt.Sprintf("Could not load timezone '%s'. Please check your timezone settings.", account.Timezone.IANALocation), nil, true)
	}
	now := time.Now().In(location)
	// If the parsed reminder time is before the current time, return an error
	if parsedTime.Before(now) {
		return utils.SendErrorDeferred(session, interaction, "Invalid Date/Time", 
			"The specified date and time is in the past. Please provide a future date and time for the reminder. You entered: "+parsedTime.Format(time.RFC3339)+
			". Current time is: "+now.Format(time.RFC3339), nil, true)
	}

	// Get recurrence type value
	recurrenceTypeValue, exists := services.RecurrenceTypeMap[strings.ToUpper(recurrenceType)]
	if !exists {
		return utils.SendErrorDeferred(session, interaction, "Invalid Recurrence Type",
			fmt.Sprintf("Invalid recurrence type '%s'. Valid options are: ONCE, YEARLY, MONTHLY, WEEKLY, DAILY, HOURLY, WORKDAYS, WEEKEND.", recurrenceType), nil, true)
	}

	repo := database.GetRepositories()

	reminderCount, err := repo.Reminder.CountByAccountID(account.ID)
	if err != nil {
		return utils.SendErrorDeferred(session, interaction, "Database Error", "Failed to check your reminder count.", nil, true)
	}
	if reminderCount >= services.MaxRemindersPerAccount {
		return utils.SendErrorDeferred(session, interaction, "Reminder Limit Reached",
			fmt.Sprintf("You have reached the maximum of %d reminders. Please delete some before creating new ones.", services.MaxRemindersPerAccount), nil, true)
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
		return utils.SendErrorDeferred(session, interaction, "Database Error",
			"Failed to save the reminder. Please try again later.", nil, true)
	}

	// Create the discord_channel destination
	destinationMetadata := models.JSONB{
		"guild_id":   interaction.GuildID,
		"channel_id": channelID,
	}
	
	// Add role mention if specified
	if roleID != "" {
		destinationMetadata["mention_role_id"] = roleID
	}
	
	destination := &models.ReminderDestination{
		ReminderID: reminder.ID,
		Type:       models.DestinationDiscordChannel,
		Metadata:   destinationMetadata,
	}

	if err := repo.ReminderDestination.Create(destination); err != nil {
		// If destination creation fails, we should clean up the reminder
		repo.Reminder.Delete(reminder.ID, true)
		return utils.SendErrorDeferred(session, interaction, "Database Error", 
			"Failed to set up reminder destination. Please try again later.", nil, true)
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

	description := fmt.Sprintf("**Content:** %s\n**Remind Time:** %s\n**Channel:** <#%s>", 
		message, displayTime, channelID)
	
	// Add role mention info if specified
	if roleID != "" {
		description += fmt.Sprintf("\n**Role Mention:** <@&%s>", roleID)
	}

	return utils.SendEmbedDeferred(session, interaction, "Channel Reminder Created! 📢", description, &recurrenceText, true)
}

func init() {
	autocompleteFunc := AutocompleteFunc(DateAutocompleteHandler)

	RegisterCommand(&Command{
		Description: Description{
			Name:             "remindus",
			Emoji:            "📢",
			CategoryName:     "Reminders",
			ShortDescription: "Create a new reminder in a channel",
			FullDescription:  "Create a new reminder that will be sent in a specified channel at the specified date and time. Requires 'Manage Channel', 'Administrator' permission, or server ownership.",
			Usage:            "/remindus message:<text> date:<date> time:<time> channel:<channel> [role:<role>] [recurrence:<type>]",
			Example:          "/remindus message:\"Team meeting\" date:\"25/12/2024\" time:\"10:00\" channel:#general role:@developers recurrence:weekly",
		},
		Data: &discordgo.ApplicationCommand{
			Name:        "remindus",
			Description: "Create a new reminder in a channel",
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
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "The channel to send the reminder in",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "role",
					Description: "Role to mention in the reminder (optional)",
					Required:    false,
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
		Run:          remindUsHandler,
		Autocomplete: &autocompleteFunc,
	})
}