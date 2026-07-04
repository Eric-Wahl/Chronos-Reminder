package logic

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ericp/chronos-bot-reminder/internal/bot/utils"
	"github.com/ericp/chronos-bot-reminder/internal/config"
	"github.com/ericp/chronos-bot-reminder/internal/database"
	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/services"
)

// dfmWebURL returns the web application URL of the DFM page
func dfmWebURL() string {
	cfg := config.Load()
	return strings.TrimSuffix(cfg.WebAppURL, "/") + "/dont-forget-me"
}

// sendDFMEmbed sends an ephemeral embed with a link button to the DFM web page
func sendDFMEmbed(session *discordgo.Session, interaction *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) error {
	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label: "Open in the web app",
							Style: discordgo.LinkButton,
							URL:   dfmWebURL(),
						},
					},
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandleDFMCreate adds a new item to the user's note
func HandleDFMCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	repo := database.GetRepositories()

	var content string
	for _, option := range options {
		if option.Name == "content" {
			content = strings.TrimSpace(option.StringValue())
		}
	}

	if content == "" {
		return utils.SendError(session, interaction, "Invalid Content", "The item content cannot be empty.")
	}

	note, err := repo.DFMNote.GetOrCreateByAccountID(account.ID)
	if err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to access your note.")
	}

	itemCount, err := repo.DFMItem.CountByNoteID(note.ID)
	if err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to check your item count.")
	}
	if itemCount >= services.MaxDFMItemsPerNote {
		return utils.SendError(session, interaction, "Item Limit Reached",
			fmt.Sprintf("You have reached the maximum of %d items. Please remove some before adding new ones.", services.MaxDFMItemsPerNote))
	}

	item := &models.DFMItem{
		NoteID:  note.ID,
		Content: content,
	}
	if err := repo.DFMItem.Create(item); err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to add the item to your note.")
	}

	embed := &discordgo.MessageEmbed{
		Title:       "💭 Don't Forget Me",
		Description: fmt.Sprintf("✅ Added to your note:\n**%s**", content),
		Color:       utils.ColorSuccess,
	}
	return sendDFMEmbed(session, interaction, embed)
}

// HandleDFMList shows the user's note with its items and reminder settings
func HandleDFMList(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account) error {
	repo := database.GetRepositories()

	if _, err := repo.DFMNote.GetOrCreateByAccountID(account.ID); err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to access your note.")
	}

	note, err := repo.DFMNote.GetWithItems(account.ID)
	if err != nil || note == nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to access your note.")
	}

	var description strings.Builder
	if len(note.Items) == 0 {
		description.WriteString("Your note is empty. Use `/dfm create` to add something you don't want to forget.")
	} else {
		for _, item := range note.Items {
			if item.Checked {
				description.WriteString(fmt.Sprintf("✅ ~~%s~~\n", item.Content))
			} else {
				description.WriteString(fmt.Sprintf("⬜ %s\n", item.Content))
			}
		}
	}

	description.WriteString("\n")
	if note.HasReminder() {
		recurrenceLabel := services.GetRecurrenceTypeLabel(services.GetRecurrenceType(int(note.Recurrence)))
		description.WriteString(fmt.Sprintf("🔔 Reminder: **%s** via **%s**", recurrenceLabel, dfmDestinationsLabel(note)))

		if note.NextFireUTC != nil {
			fireTime := *note.NextFireUTC
			fullAccount, err := repo.Account.GetWithTimezone(account.ID)
			if err == nil && fullAccount != nil && fullAccount.Timezone != nil {
				if loc, err := time.LoadLocation(fullAccount.Timezone.IANALocation); err == nil {
					fireTime = fireTime.In(loc)
				}
			}
			description.WriteString(fmt.Sprintf(" - next: %s", fireTime.Format("Jan 02, 15:04 MST")))
		}
	} else {
		description.WriteString("🔕 No reminder set. Use `/dfm set-reminder` to be reminded of your note.")
	}

	embed := &discordgo.MessageEmbed{
		Title:       "💭 Don't Forget Me - Your note",
		Description: description.String(),
		Color:       utils.ColorInfo,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%d item(s)", len(note.Items)),
		},
	}
	return sendDFMEmbed(session, interaction, embed)
}

// getDFMItemFromOptions resolves the selected item and verifies ownership
func getDFMItemFromOptions(account *models.Account, options []*discordgo.ApplicationCommandInteractionDataOption) (*models.DFMItem, error) {
	repo := database.GetRepositories()

	var itemIDStr string
	for _, option := range options {
		if option.Name == "item" {
			itemIDStr = option.StringValue()
		}
	}

	note, err := repo.DFMNote.GetByAccountID(account.ID)
	if err != nil || note == nil {
		return nil, fmt.Errorf("note not found")
	}

	items, err := repo.DFMItem.GetByNoteID(note.ID)
	if err != nil {
		return nil, err
	}

	for i := range items {
		if items[i].ID.String() == itemIDStr {
			return &items[i], nil
		}
	}

	return nil, fmt.Errorf("item not found")
}

// HandleDFMSetChecked checks or unchecks an item of the note
func HandleDFMSetChecked(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account, options []*discordgo.ApplicationCommandInteractionDataOption, checked bool) error {
	repo := database.GetRepositories()

	item, err := getDFMItemFromOptions(account, options)
	if err != nil {
		return utils.SendError(session, interaction, "Item Not Found", "The specified item could not be found in your note.")
	}

	item.Checked = checked
	if err := repo.DFMItem.Update(item); err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to update the item.")
	}

	action := "⬜ Item unchecked"
	if checked {
		action = "✅ Item checked"
	}
	embed := &discordgo.MessageEmbed{
		Title:       "💭 Don't Forget Me",
		Description: fmt.Sprintf("%s:\n**%s**", action, item.Content),
		Color:       utils.ColorSuccess,
	}
	return sendDFMEmbed(session, interaction, embed)
}

// HandleDFMDelete removes an item from the note
func HandleDFMDelete(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	repo := database.GetRepositories()

	item, err := getDFMItemFromOptions(account, options)
	if err != nil {
		return utils.SendError(session, interaction, "Item Not Found", "The specified item could not be found in your note.")
	}

	if err := repo.DFMItem.Delete(item.ID); err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to delete the item.")
	}

	embed := &discordgo.MessageEmbed{
		Title:       "💭 Don't Forget Me",
		Description: fmt.Sprintf("🗑️ Removed from your note:\n**%s**", item.Content),
		Color:       utils.ColorSuccess,
	}
	return sendDFMEmbed(session, interaction, embed)
}

// dfmDestinationsLabel returns a human readable list of the note's delivery channels
func dfmDestinationsLabel(note *models.DFMNote) string {
	var labels []string
	if note.SendDiscordDM {
		labels = append(labels, "Discord DM")
	}
	if note.SendEmail {
		labels = append(labels, "Email")
	}
	if len(labels) == 0 {
		return "Discord DM"
	}
	return strings.Join(labels, " and ")
}

// HandleDFMSetReminder configures the recurring reminder of the note
func HandleDFMSetReminder(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	repo := database.GetRepositories()

	recurrenceStr := ""
	timeStr := "09:00"
	dateStr := ""
	destinationChoice := "discord_dm"
	for _, option := range options {
		switch option.Name {
		case "recurrence":
			recurrenceStr = strings.ToUpper(option.StringValue())
		case "time":
			timeStr = strings.TrimSpace(option.StringValue())
		case "date":
			dateStr = strings.TrimSpace(option.StringValue())
		case "destination":
			destinationChoice = option.StringValue()
		}
	}

	recurrenceValue, exists := services.RecurrenceTypeMap[recurrenceStr]
	if !exists || recurrenceValue == services.RecurrenceOnce {
		return utils.SendError(session, interaction, "Invalid Recurrence", "Please choose a valid recurrence.")
	}

	sendDiscordDM := destinationChoice == "discord_dm" || destinationChoice == "both"
	sendEmail := destinationChoice == "email" || destinationChoice == "both"
	if !sendDiscordDM && !sendEmail {
		return utils.SendError(session, interaction, "Invalid Destination", "The destination must be Discord DM, Email or Both.")
	}

	// Email delivery requires an account-level email address
	if sendEmail && account.Email == nil {
		return utils.SendError(session, interaction, "No Email Linked", "You need a Chronos web account with an email address to receive your note by email. You can link one from the web application.")
	}

	fullAccount, err := repo.Account.GetWithTimezone(account.ID)
	if err != nil || fullAccount == nil || fullAccount.Timezone == nil {
		return utils.SendError(session, interaction, "Timezone Missing", "Please set your timezone first with `/timezones`.")
	}

	firstFire, err := services.ComputeDFMReminderSchedule(dateStr, timeStr, recurrenceValue, fullAccount.Timezone.IANALocation)
	if err != nil {
		return utils.SendError(session, interaction, "Invalid Date or Time", "The provided date or time could not be parsed. Use YYYY-MM-DD for the date and HH:MM for the time, for example 2026-06-19 09:00.")
	}

	note, err := repo.DFMNote.GetOrCreateByAccountID(account.ID)
	if err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to access your note.")
	}

	note.RemindAtUTC = &firstFire
	note.NextFireUTC = &firstFire
	note.Recurrence = int16(recurrenceValue)
	note.SendDiscordDM = sendDiscordDM
	note.SendEmail = sendEmail
	if err := repo.DFMNote.Update(note); err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to set the reminder.")
	}

	localFire := firstFire
	if loc, err := time.LoadLocation(fullAccount.Timezone.IANALocation); err == nil {
		localFire = firstFire.In(loc)
	}

	embed := &discordgo.MessageEmbed{
		Title: "💭 Don't Forget Me",
		Description: fmt.Sprintf(
			"🔔 Your note will now be sent to you **%s** via **%s**.\nNext delivery: **%s**",
			strings.ToLower(services.GetRecurrenceTypeLabel(recurrenceValue)),
			dfmDestinationsLabel(note),
			localFire.Format("Jan 02, 15:04 MST"),
		),
		Color: utils.ColorSuccess,
	}
	return sendDFMEmbed(session, interaction, embed)
}

// HandleDFMSend dispatches the note to the user immediately
func HandleDFMSend(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account) error {
	repo := database.GetRepositories()
	const cooldownDuration = 5 * time.Minute

	note, err := repo.DFMNote.GetOrCreateByAccountID(account.ID)
	if err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to access your note.")
	}

	// Check cooldown
	if note.LastSentAt != nil {
		timeSinceLastSend := time.Since(*note.LastSentAt)
		if timeSinceLastSend < cooldownDuration {
			remainingTime := cooldownDuration - timeSinceLastSend
			minutes := int(remainingTime.Minutes())
			seconds := int(remainingTime.Seconds()) % 60
			return utils.SendError(session, interaction, "Cooldown Active", 
				fmt.Sprintf("Please wait %d:%02d before sending your note again.", minutes, seconds))
		}
	}

	if services.DFMSendNow == nil {
		return utils.SendError(session, interaction, "Send Failed", "The reminder engine is not available. Please try again later.")
	}
	if err := services.DFMSendNow(account.ID); err != nil {
		return utils.SendError(session, interaction, "Send Failed", "Your note could not be sent. Please try again later.")
	}

	// Update last sent timestamp
	now := time.Now()
	note.LastSentAt = &now
	if err := repo.DFMNote.Update(note); err != nil {
		// Log the error but don't fail the send - the note was already sent
		fmt.Printf("Warning: Failed to update LastSentAt for note %s: %v\n", note.ID, err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "💭 Don't Forget Me",
		Description: "📨 Your note has been sent!",
		Color:       utils.ColorSuccess,
	}
	return sendDFMEmbed(session, interaction, embed)
}

// HandleDFMRemoveReminder clears the reminder of the note
func HandleDFMRemoveReminder(session *discordgo.Session, interaction *discordgo.InteractionCreate, account *models.Account) error {
	repo := database.GetRepositories()

	note, err := repo.DFMNote.GetOrCreateByAccountID(account.ID)
	if err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to access your note.")
	}

	if !note.HasReminder() {
		return utils.SendError(session, interaction, "No Reminder", "Your note has no reminder to remove.")
	}

	note.RemindAtUTC = nil
	note.NextFireUTC = nil
	note.Recurrence = 0
	if err := repo.DFMNote.Update(note); err != nil {
		return utils.SendError(session, interaction, "Database Error", "Failed to remove the reminder.")
	}

	embed := &discordgo.MessageEmbed{
		Title:       "💭 Don't Forget Me",
		Description: "🔕 The reminder of your note has been removed.",
		Color:       utils.ColorSuccess,
	}
	return sendDFMEmbed(session, interaction, embed)
}
