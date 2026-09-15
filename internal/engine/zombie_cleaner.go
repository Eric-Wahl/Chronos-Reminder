package engine

import (
	"context"
	"log"
	"time"

	"github.com/ericp/chronos-bot-reminder/internal/database/repositories"
)

// ZombieThreshold is how long a reminder or DFM delivery channel may sit with
// an unresolved, unfixed error before it's considered abandoned. If the user
// hasn't fixed it (or asked about it) in that time, it's just dead weight.
const ZombieThreshold = 14 * 24 * time.Hour

// zombieCleanerInterval is how often the cleanup pass runs. The threshold is
// measured in days, so hourly is more than precise enough.
const zombieCleanerInterval = time.Hour

// ZombieCleaner periodically removes reminders and disables DFM delivery
// channels that have been failing, unresolved, for longer than
// ZombieThreshold. Reminders are deleted outright (they're just a broken
// schedule); DFM notes are never deleted since they hold the user's actual
// list content — only the failing channel is turned off, and the note's own
// reminder schedule is cleared if that leaves it with nowhere to send to.
type ZombieCleaner struct {
	reminderRepo repositories.ReminderRepository
	noteRepo     repositories.DFMNoteRepository
	stopChan     chan struct{}
	running      bool
}

// NewZombieCleaner creates a new zombie cleaner instance
func NewZombieCleaner(reminderRepo repositories.ReminderRepository, noteRepo repositories.DFMNoteRepository) *ZombieCleaner {
	return &ZombieCleaner{
		reminderRepo: reminderRepo,
		noteRepo:     noteRepo,
		stopChan:     make(chan struct{}),
	}
}

// Start begins the zombie cleaner's periodic loop
func (z *ZombieCleaner) Start(ctx context.Context) {
	if z.running {
		log.Println("[ENGINE] - Zombie cleaner already running")
		return
	}
	z.running = true

	go func() {
		ticker := time.NewTicker(zombieCleanerInterval)
		defer ticker.Stop()

		z.runOnce()

		for {
			select {
			case <-ctx.Done():
				z.running = false
				return
			case <-z.stopChan:
				z.running = false
				return
			case <-ticker.C:
				z.runOnce()
			}
		}
	}()

	log.Println("[ENGINE] - ✅ Zombie cleaner started")
}

// Stop gracefully stops the zombie cleaner
func (z *ZombieCleaner) Stop() {
	if !z.running {
		return
	}
	close(z.stopChan)
	z.running = false
}

// IsRunning returns whether the zombie cleaner is currently running
func (z *ZombieCleaner) IsRunning() bool {
	return z.running
}

// runOnce performs a single cleanup pass over reminders and DFM notes
func (z *ZombieCleaner) runOnce() {
	threshold := time.Now().UTC().Add(-ZombieThreshold)
	z.cleanZombieReminders(threshold)
	z.cleanZombieDFMChannels(threshold)
}

// cleanZombieReminders deletes reminders whose delivery has been broken and
// unresolved for longer than the threshold. Deleting the reminder cascades
// to its reminder_errors and destinations.
func (z *ZombieCleaner) cleanZombieReminders(threshold time.Time) {
	reminders, err := z.reminderRepo.GetZombieReminders(threshold)
	if err != nil {
		log.Printf("[ENGINE] - Error fetching zombie reminders: %v", err)
		return
	}
	if len(reminders) == 0 {
		return
	}

	deleted := 0
	for _, reminder := range reminders {
		// Don't notify the scheduler: these are already excluded from its
		// due-reminders query (they have an unresolved error), so there's
		// nothing to reschedule around.
		if err := z.reminderRepo.Delete(reminder.ID, false); err != nil {
			log.Printf("[ENGINE] - Error deleting zombie reminder %s: %v", reminder.ID, err)
			continue
		}
		deleted++
	}

	if deleted > 0 {
		log.Printf("[ENGINE] - Cleaned up %d zombie reminder(s) with unresolved errors older than %s", deleted, ZombieThreshold)
	}
}

// cleanZombieDFMChannels disables any DFM delivery channel that's been
// failing uninterrupted for longer than the threshold. The note's content is
// never touched. If disabling leaves the note with no enabled channel, its
// reminder schedule is cleared too, since there's nowhere left to send it.
func (z *ZombieCleaner) cleanZombieDFMChannels(threshold time.Time) {
	notes, err := z.noteRepo.GetNotesWithZombieChannels(threshold)
	if err != nil {
		log.Printf("[ENGINE] - Error fetching DFM notes with zombie channels: %v", err)
		return
	}
	if len(notes) == 0 {
		return
	}

	cleaned := 0
	for i := range notes {
		note := &notes[i]
		changed := false

		if note.DiscordFailingSince != nil && note.DiscordFailingSince.Before(threshold) {
			note.SendDiscordDM = false
			note.DiscordFailingSince = nil
			changed = true
		}
		if note.EmailFailingSince != nil && note.EmailFailingSince.Before(threshold) {
			note.SendEmail = false
			note.EmailFailingSince = nil
			changed = true
		}
		if !changed {
			continue
		}

		// Nothing left to send to: stop scheduling this note entirely
		if !note.SendDiscordDM && !note.SendEmail {
			note.RemindAtUTC = nil
			note.NextFireUTC = nil
			note.Recurrence = 0
		}

		if err := z.noteRepo.Update(note); err != nil {
			log.Printf("[ENGINE] - Error updating zombie DFM note %s: %v", note.ID, err)
			continue
		}
		cleaned++
	}

	if cleaned > 0 {
		log.Printf("[ENGINE] - Disabled zombie delivery channel(s) on %d DFM note(s) failing for longer than %s", cleaned, ZombieThreshold)
	}
}
