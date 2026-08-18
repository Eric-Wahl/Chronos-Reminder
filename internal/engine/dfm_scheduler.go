package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ericp/chronos-bot-reminder/internal/config"
	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/database/repositories"
	"github.com/ericp/chronos-bot-reminder/internal/dispatchers"
	"github.com/ericp/chronos-bot-reminder/internal/services"
	"github.com/google/uuid"
)

// dfmPollInterval is how often the DFM scheduler checks for due notes.
// DFM reminders are coarse-grained (daily and above), so minute precision is enough.
const dfmPollInterval = time.Minute

// DFMScheduler periodically dispatches due "Don't Forget Me" notes
type DFMScheduler struct {
	noteRepo     repositories.DFMNoteRepository
	identityRepo repositories.IdentityRepository
	accountRepo  repositories.AccountRepository
	dispatcher   *dispatchers.DFMDispatcher
	stopChan     chan struct{}
	running      bool
}

// NewDFMScheduler creates a new DFM scheduler instance
func NewDFMScheduler(
	noteRepo repositories.DFMNoteRepository,
	identityRepo repositories.IdentityRepository,
	accountRepo repositories.AccountRepository,
	dispatcher *dispatchers.DFMDispatcher,
) *DFMScheduler {
	return &DFMScheduler{
		noteRepo:     noteRepo,
		identityRepo: identityRepo,
		accountRepo:  accountRepo,
		dispatcher:   dispatcher,
		stopChan:     make(chan struct{}),
	}
}

// Start begins the DFM scheduling loop
func (s *DFMScheduler) Start(ctx context.Context) {
	if s.running {
		log.Println("[ENGINE] - DFM scheduler already running")
		return
	}
	s.running = true

	go func() {
		ticker := time.NewTicker(dfmPollInterval)
		defer ticker.Stop()

		s.processDueNotes()

		for {
			select {
			case <-ctx.Done():
				s.running = false
				return
			case <-s.stopChan:
				s.running = false
				return
			case <-ticker.C:
				s.processDueNotes()
			}
		}
	}()

	log.Println("[ENGINE] - ✅ DFM scheduler started")
}

// Stop gracefully stops the DFM scheduler
func (s *DFMScheduler) Stop() {
	if !s.running {
		return
	}
	close(s.stopChan)
	s.running = false
}

// IsRunning returns whether the DFM scheduler is currently running
func (s *DFMScheduler) IsRunning() bool {
	return s.running
}

// SendNoteNow dispatches the account's note immediately, without touching the
// reminder schedule. Used to test the DFM delivery.
func (s *DFMScheduler) SendNoteNow(accountID uuid.UUID) error {
	note, err := s.noteRepo.GetWithItems(accountID)
	if err != nil {
		return err
	}
	if note == nil {
		return fmt.Errorf("no DFM note found for account %s", accountID)
	}

	discordID, email, err := s.resolveDeliveryAddresses(accountID)
	if err != nil {
		return err
	}

	return s.dispatcher.Dispatch(note, discordID, email)
}

// SendDFMNoteNow dispatches the account's note immediately through the running scheduler service
func SendDFMNoteNow(accountID uuid.UUID) error {
	service := GetSchedulerService()
	if service == nil || service.DFMScheduler == nil {
		return fmt.Errorf("DFM scheduler not available")
	}
	return service.DFMScheduler.SendNoteNow(accountID)
}

// processDueNotes dispatches every note whose reminder is due and reschedules it
func (s *DFMScheduler) processDueNotes() {
	now := time.Now().UTC()
	notes, err := s.noteRepo.GetDueNotes(now)
	if err != nil {
		log.Printf("[ENGINE] - Error fetching due DFM notes: %v", err)
		services.LogBotError("engine:dfm_scheduler", "Failed to fetch due DFM notes", err.Error(), nil)
		return
	}

	for i := range notes {
		s.processNote(&notes[i])
	}
}

// resolveDeliveryAddresses returns the Discord user ID and email for an account.
// Either may be empty string if not linked.
func (s *DFMScheduler) resolveDeliveryAddresses(accountID uuid.UUID) (discordID string, email string, err error) {
	identities, err := s.identityRepo.GetByAccountID(accountID)
	if err != nil {
		return "", "", err
	}
	for _, id := range identities {
		if id.Provider == models.ProviderDiscord {
			discordID = id.ExternalID
		}
	}
	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		return "", "", err
	}
	if account != nil && account.Email != nil {
		email = *account.Email
	}
	return discordID, email, nil
}

// processNote dispatches a single note, records the delivery and computes the next fire time
func (s *DFMScheduler) processNote(note *models.DFMNote) {
	discordID, email, err := s.resolveDeliveryAddresses(note.AccountID)
	if err != nil {
		log.Printf("[ENGINE] - Error fetching delivery addresses for DFM note %s: %v", note.ID, err)
		services.LogBotError("engine:dfm_scheduler", "Failed to fetch delivery addresses for DFM note", fmt.Sprintf("note %s: %v", note.ID, err), nil)
		return
	}

	if err := s.dispatcher.Dispatch(note, discordID, email); err != nil {
		log.Printf("[ENGINE] - Error dispatching DFM note %s: %v", note.ID, err)
		services.LogBotError("engine:dfm_scheduler", "Failed to dispatch DFM note", fmt.Sprintf("note %s: %v", note.ID, err), nil)
	} else if config.IsDebugMode() {
		log.Printf("[ENGINE] - DFM note %s dispatched", note.ID)
	}

	// Always reschedule, even after a dispatch failure, so a broken
	// destination cannot make the scheduler retry every poll
	s.rescheduleNote(note)
}

// rescheduleNote computes the next occurrence of the note reminder, or clears
// it for one-time reminders
func (s *DFMScheduler) rescheduleNote(note *models.DFMNote) {
	if note.RemindAtUTC == nil {
		return
	}

	recurrenceType := services.GetRecurrenceType(int(note.Recurrence))
	if recurrenceType == services.RecurrenceOnce {
		note.RemindAtUTC = nil
		note.NextFireUTC = nil
	} else {
		ianaLocation := "UTC"
		if note.Account != nil && note.Account.Timezone != nil {
			ianaLocation = note.Account.Timezone.IANALocation
		}

		nextTime, err := services.GetNextOccurrence(*note.RemindAtUTC, int(note.Recurrence), ianaLocation)
		if err != nil {
			log.Printf("[ENGINE] - Error computing next occurrence for DFM note %s: %v", note.ID, err)
			// Critical: if this fails, the recurring DFM note never gets
			// rescheduled and silently stops firing forever with no other trace.
			services.LogBotError("engine:dfm_scheduler", "Failed to compute next occurrence — recurring DFM note will stop firing", fmt.Sprintf("note %s: %v", note.ID, err), nil)
			return
		}

		next := nextTime.UTC()
		note.RemindAtUTC = &next
		note.NextFireUTC = &next
	}

	if err := s.noteRepo.Update(note); err != nil {
		log.Printf("[ENGINE] - Error rescheduling DFM note %s: %v", note.ID, err)
		// Critical: same as above — the note will not fire again.
		services.LogBotError("engine:dfm_scheduler", "Failed to reschedule DFM note — it will stop firing", fmt.Sprintf("note %s: %v", note.ID, err), nil)
	}
}
