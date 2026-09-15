package dispatchers

import (
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/ericp/chronos-bot-reminder/internal/bot"
	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/services"
)

// DFMDispatcher sends "Don't Forget Me" notes to their owner. The note is
// always private: it goes to the user's Discord DM when a Discord identity
// exists, otherwise it falls back to the app identity email.
type DFMDispatcher struct {
	session   *discordgo.Session
	mailer    *services.MailerService
	webAppURL string
}

// NewDFMDispatcher creates a new DFM dispatcher
func NewDFMDispatcher(mailer *services.MailerService, webAppURL string) *DFMDispatcher {
	return &DFMDispatcher{
		session:   bot.GetDiscordSession(),
		mailer:    mailer,
		webAppURL: webAppURL,
	}
}

// NoteWebURL returns the web application URL of the DFM page
func (d *DFMDispatcher) NoteWebURL() string {
	return strings.TrimSuffix(d.webAppURL, "/") + "/dont-forget-me"
}

// DFMDeliveryResult reports the outcome of each channel attempted by a
// Dispatch call, so the caller can track per-channel failure streaks without
// re-parsing a joined error string.
type DFMDeliveryResult struct {
	DiscordAttempted bool
	DiscordErr       error
	EmailAttempted   bool
	EmailErr         error
}

// Err joins whichever channel errors occurred, or nil if all attempted
// channels succeeded.
func (r DFMDeliveryResult) Err() error {
	return errors.Join(r.DiscordErr, r.EmailErr)
}

// Dispatch sends the note to its owner on every enabled private channel
// (Discord DM and/or email). discordUserID and email may be empty string if
// the respective channel is not available. A failure on one channel does not
// prevent the other from being attempted.
func (d *DFMDispatcher) Dispatch(note *models.DFMNote, discordUserID string, email string) error {
	if !note.SendDiscordDM && !note.SendEmail {
		return fmt.Errorf("no destination enabled for DFM note %s", note.ID)
	}
	return d.DispatchDetailed(note, discordUserID, email).Err()
}

// DispatchDetailed behaves like Dispatch but reports the per-channel outcome
// instead of a single joined error.
func (d *DFMDispatcher) DispatchDetailed(note *models.DFMNote, discordUserID string, email string) DFMDeliveryResult {
	var result DFMDeliveryResult

	if note.SendDiscordDM {
		result.DiscordAttempted = true
		if discordUserID == "" || d.session == nil {
			result.DiscordErr = fmt.Errorf("no Discord identity linked for DFM note %s", note.ID)
		} else {
			result.DiscordErr = d.dispatchDiscordDM(note, discordUserID)
		}
	}

	if note.SendEmail {
		result.EmailAttempted = true
		if email == "" {
			result.EmailErr = fmt.Errorf("no email linked for DFM note %s", note.ID)
		} else {
			result.EmailErr = d.dispatchEmail(note, email)
		}
	}

	return result
}

// RenderDFMNoteText renders the note items as a plain text checklist
func RenderDFMNoteText(note *models.DFMNote) string {
	if len(note.Items) == 0 {
		return "Your note is empty."
	}

	var builder strings.Builder
	for _, item := range note.Items {
		if item.Checked {
			builder.WriteString(fmt.Sprintf("[x] %s\n", item.Content))
		} else {
			builder.WriteString(fmt.Sprintf("[ ] %s\n", item.Content))
		}
	}
	return strings.TrimRight(builder.String(), "\n")
}

// dispatchDiscordDM sends the note content as a private Discord message
func (d *DFMDispatcher) dispatchDiscordDM(note *models.DFMNote, discordUserID string) error {
	dmChannel, err := d.session.UserChannelCreate(discordUserID)
	if err != nil {
		return fmt.Errorf("failed to create DM channel with user %s: %w", discordUserID, err)
	}

	var description strings.Builder
	if len(note.Items) == 0 {
		description.WriteString("Your note is empty.")
	} else {
		for _, item := range note.Items {
			if item.Checked {
				description.WriteString(fmt.Sprintf("✅ ~~%s~~\n", item.Content))
			} else {
				description.WriteString(fmt.Sprintf("⬜ %s\n", item.Content))
			}
		}
	}
	description.WriteString("\nYou can edit your note, check items and manage the reminder from the web application.")

	embed := &discordgo.MessageEmbed{
		Title:       "💭 Don't Forget Me - Your note",
		Description: description.String(),
		Color:       0xCEA04D,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Chronos Bot Reminder",
		},
	}

	msg := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label: "Open in the web app",
						Style: discordgo.LinkButton,
						URL:   d.NoteWebURL(),
					},
				},
			},
		},
	}

	if _, err := d.session.ChannelMessageSendComplex(dmChannel.ID, msg); err != nil {
		return fmt.Errorf("failed to send DFM note DM: %w", err)
	}
	return nil
}

// dispatchEmail sends the note content to the user's email address
func (d *DFMDispatcher) dispatchEmail(note *models.DFMNote, email string) error {
	var itemsHTML strings.Builder
	if len(note.Items) == 0 {
		itemsHTML.WriteString("<p>Your note is empty.</p>")
	} else {
		itemsHTML.WriteString("<ul style=\"list-style: none; padding-left: 0;\">")
		for _, item := range note.Items {
			content := html.EscapeString(item.Content)
			if item.Checked {
				itemsHTML.WriteString(fmt.Sprintf("<li style=\"margin: 6px 0;\">[x] <span style=\"text-decoration: line-through; color: #999;\">%s</span></li>", content))
			} else {
				itemsHTML.WriteString(fmt.Sprintf("<li style=\"margin: 6px 0;\">[ ] %s</li>", content))
			}
		}
		itemsHTML.WriteString("</ul>")
	}

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Don't Forget Me</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
	<div style="max-width: 600px; margin: 0 auto; padding: 20px;">
		<h2 style="color: #CEA04D;">Don't Forget Me - Your note</h2>
		%s
		<p style="margin: 30px 0;">
			<a href="%s" style="background-color: #CEA04D; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; display: inline-block;">
				Open in the web app
			</a>
		</p>
		<p style="margin-top: 30px; color: #999; font-size: 12px;">This is an automated reminder from Chronos Reminder</p>
	</div>
</body>
</html>
	`, itemsHTML.String(), d.NoteWebURL())

	textBody := fmt.Sprintf("Don't Forget Me - Your note\n\n%s\n\nEdit your note: %s", RenderDFMNoteText(note), d.NoteWebURL())

	_, err := d.mailer.SendEmail(&services.EmailRequest{
		To:       email,
		Subject:  "Don't Forget Me: your note reminder",
		HtmlBody: htmlBody,
		TextBody: textBody,
	})
	return err
}
