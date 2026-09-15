package engine

import (
	"context"
	"log"
	"sync"

	"github.com/ericp/chronos-bot-reminder/internal/config"
	"github.com/ericp/chronos-bot-reminder/internal/database"
	"github.com/ericp/chronos-bot-reminder/internal/database/repositories"
	"github.com/ericp/chronos-bot-reminder/internal/dispatchers"
	"github.com/ericp/chronos-bot-reminder/internal/services"
)

// SchedulerService manages the complete scheduling system
type SchedulerService struct {
	Scheduler          *Scheduler
	GarbageCollector   *GarbageCollector
	DispatcherRegistry *DispatcherRegistry
	ReminderRepo       repositories.ReminderRepository
	DFMScheduler       *DFMScheduler
	ZombieCleaner      *ZombieCleaner
}

var (
	schedulerService *SchedulerService
	schedulerCtx     context.Context
	schedulerCancel  context.CancelFunc
	schedulerMutex   sync.Mutex
)

// GetSchedulerService returns a singleton scheduler service
func GetSchedulerService() *SchedulerService {
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()

	if schedulerService == nil {
		repos := database.GetRepositories()
		if repos == nil {
			log.Fatalf("[ENGINE] - ❌ Cannot get repositories")
		}

		schedulerService = NewSchedulerService(repos.Reminder, repos.ReminderError)
	}

	return schedulerService
}

// StartSchedulerService initializes and starts the scheduler service
func StartSchedulerService() {
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()

	if schedulerService == nil {
		repos := database.GetRepositories()
		if repos == nil {
			log.Fatalf("[ENGINE] - ❌ Cannot get repositories")
		}

		schedulerService = NewSchedulerService(repos.Reminder, repos.ReminderError)
	}

	if schedulerService.Scheduler.IsRunning() {
		log.Println("[ENGINE] - Already running")
		return
	}

	// Create context for the scheduler
	schedulerCtx, schedulerCancel = context.WithCancel(context.Background())

	// Initialize the repository notifier
	initializeRepositoryNotifier(schedulerService)

	// Start the scheduler
	schedulerService.Scheduler.Start(schedulerCtx)
	log.Println("[ENGINE] - ✅ Scheduler started")

	// Start the garbage collector
	schedulerService.GarbageCollector.Start(schedulerCtx)

	// Start the Don't Forget Me scheduler
	if schedulerService.DFMScheduler != nil {
		schedulerService.DFMScheduler.Start(schedulerCtx)
	}

	// Start the zombie cleaner
	if schedulerService.ZombieCleaner != nil {
		schedulerService.ZombieCleaner.Start(schedulerCtx)
	}
}

// StopSchedulerService gracefully stops the scheduler service
func StopSchedulerService() {
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()

	if schedulerService != nil {
		if schedulerService.Scheduler.IsRunning() {
			schedulerService.Scheduler.Stop()
		}
		if schedulerService.GarbageCollector.IsRunning() {
			schedulerService.GarbageCollector.Stop()
		}
		if schedulerService.DFMScheduler != nil && schedulerService.DFMScheduler.IsRunning() {
			schedulerService.DFMScheduler.Stop()
		}
		if schedulerService.ZombieCleaner != nil && schedulerService.ZombieCleaner.IsRunning() {
			schedulerService.ZombieCleaner.Stop()
		}
	}

	if schedulerCancel != nil {
		schedulerCancel()
		schedulerCancel = nil
	}

	// Reset the service
	schedulerService = nil
}

// GetReminderRepository returns the reminder repository for use by other parts of the application
func GetReminderRepository() repositories.ReminderRepository {
	service := GetSchedulerService()
	if service == nil {
		return nil
	}
	return service.ReminderRepo
}

// InitializeRepositoryNotifier sets up the scheduler notifier in the base repository
// This should be called after the scheduler service is created
func InitializeRepositoryNotifier() {
	service := GetSchedulerService()
	if service == nil {
		log.Printf("[ENGINE] - ⚠️ Cannot initialize repository notifier: scheduler service not available")
		return
	}

	initializeRepositoryNotifier(service)
}

// initializeRepositoryNotifier is the internal implementation that doesn't call GetSchedulerService
// to avoid deadlocks when called from within StartSchedulerService
func initializeRepositoryNotifier(service *SchedulerService) {
	// Get the base repositories
	repos := database.GetRepositories()
	if repos == nil {
		log.Printf("[ENGINE] - ⚠️ Cannot initialize repository notifier: repositories not available")
		return
	}

	// Set the scheduler notifier in the base reminder repository
	if reminderRepo, ok := repos.Reminder.(interface{ SetScheduler(repositories.SchedulerNotifier) }); ok {
		reminderRepo.SetScheduler(service.Scheduler)
		log.Printf("[ENGINE] - ✅ Repository notifier initialized")
	} else {
		log.Printf("[ENGINE] - ⚠️ Reminder repository does not support scheduler notification")
	}
}

// GetSchedulerAwareReminderRepository returns the reminder repository with scheduler notification support
func GetSchedulerAwareReminderRepository() repositories.ReminderRepository {
	// Try to get the repository from scheduler service
	reminderRepo := GetReminderRepository()
	if reminderRepo != nil {
		return reminderRepo
	}

	// Fallback to base repository
	repos := database.GetRepositories()
	if repos != nil {
		return repos.Reminder
	}

	return nil
}

// IsSchedulerNotificationEnabled returns true if the scheduler is running and can receive notifications
func IsSchedulerNotificationEnabled() bool {
	service := GetSchedulerService()
	return service != nil && service.Scheduler.IsRunning()
}

// NewSchedulerService creates a new complete scheduler service with all dispatchers registered
func NewSchedulerService(reminderRepo repositories.ReminderRepository, reminderErrorRepo repositories.ReminderErrorRepository) *SchedulerService {
	// Create dispatcher registry
	dispatcherRegistry := NewDispatcherRegistry(reminderErrorRepo)

	// Register all dispatchers
	dispatcherRegistry.RegisterDispatcher(dispatchers.NewDiscordDMDispatcher())
	dispatcherRegistry.RegisterDispatcher(dispatchers.NewWebhookDispatcher())
	dispatcherRegistry.RegisterDispatcher(dispatchers.NewDiscordChannelDispatcher())

	cfg := config.Load()
	mailer := services.NewMailerService(cfg.ResendAPIKey, config.EmailNoreply)
	dispatcherRegistry.RegisterDispatcher(dispatchers.NewEmailDispatcher(mailer))

	// Android push delivery via Firebase Cloud Messaging
	if repos := database.GetRepositories(); repos != nil {
		fcmService := services.NewFcmService(cfg.GoogleAppCredentials)
		dispatcherRegistry.RegisterDispatcher(dispatchers.NewAndroidPushDispatcher(fcmService, repos.FcmToken))
	}

	// Create garbage collector
	garbageCollector := NewGarbageCollector(reminderRepo)

	// Create scheduler
	scheduler := NewScheduler(reminderRepo, reminderErrorRepo, dispatcherRegistry, garbageCollector)

	// Set the scheduler in the repository if it supports it
	if schedulerAwareRepo, ok := reminderRepo.(interface{ SetScheduler(repositories.SchedulerNotifier) }); ok {
		schedulerAwareRepo.SetScheduler(scheduler)
	}

	// Create the Don't Forget Me scheduler
	var dfmScheduler *DFMScheduler
	var zombieCleaner *ZombieCleaner
	if repos := database.GetRepositories(); repos != nil {
		dfmDispatcher := dispatchers.NewDFMDispatcher(mailer, cfg.WebAppURL)
		dfmScheduler = NewDFMScheduler(repos.DFMNote, repos.Identity, repos.Account, dfmDispatcher)
		// Expose the immediate send for callers that cannot import the engine (bot commands)
		services.DFMSendNow = dfmScheduler.SendNoteNow

		zombieCleaner = NewZombieCleaner(reminderRepo, repos.DFMNote)
	}

	return &SchedulerService{
		Scheduler:          scheduler,
		GarbageCollector:   garbageCollector,
		DispatcherRegistry: dispatcherRegistry,
		ReminderRepo:       reminderRepo,
		DFMScheduler:       dfmScheduler,
		ZombieCleaner:      zombieCleaner,
	}
}
