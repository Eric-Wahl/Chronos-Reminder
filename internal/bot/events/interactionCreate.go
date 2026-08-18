package events

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/ericp/chronos-bot-reminder/internal/bot/commands"
	"github.com/ericp/chronos-bot-reminder/internal/bot/handlers"
	"github.com/ericp/chronos-bot-reminder/internal/bot/utils"
)

func InteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		err := commands.HandleCommand(s, i)
		if err != nil {
			log.Printf("[DISCORD_BOT] - ❌ Error handling command: %v", err)
			// Reaching here means the command returned an error the user was
			// never shown (SendError succeeds with a nil return, so this is
			// either a failed Discord response or a bug that skipped
			// SendError entirely) — record it so it's not only in stdout.
			utils.LogBotError(i, "command:"+i.ApplicationCommandData().Name, "Unhandled command error", err.Error())
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		err := commands.HandleAutocomplete(s, i)
		if err != nil {
			log.Printf("[DISCORD_BOT] - ❌ Error handling autocomplete: %v", err)
			utils.LogBotError(i, "autocomplete:"+i.ApplicationCommandData().Name, "Unhandled autocomplete error", err.Error())
		}
	case discordgo.InteractionMessageComponent:
		err := handlers.HandleMessageComponent(s, i)
		if err != nil {
			log.Printf("[DISCORD_BOT] - ❌ Error handling message component: %v", err)
			utils.LogBotError(i, "component:"+i.MessageComponentData().CustomID, "Unhandled component error", err.Error())
		}
	}
}
