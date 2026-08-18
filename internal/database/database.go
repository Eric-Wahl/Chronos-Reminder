package database

import (
	"fmt"
	"log"
	"os"

	appConfig "github.com/ericp/chronos-bot-reminder/internal/config"
	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/database/repositories"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
var repos *repositories.Repositories

// Initialize sets up the database connection and runs migrations
func Initialize() error {
	var err error

	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_NAME"), os.Getenv("DB_PASSWORD"))

	// Configure GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	// Connect to database
	DB, err = gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return fmt.Errorf("[DATABASE] - Failed to connect to database: %w", err)
	}

	log.Println("[DATABASE] - ✅ Connection established")

	// Initialize Redis connection
	if err := InitializeRedis(); err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}

	// Flush cache in development mode to avoid stale data after restarts
	if appConfig.IsDebugMode() {
		if err := FlushCache(); err != nil {
			log.Printf("[REDIS] - ⚠️ Failed to flush cache: %v", err)
		} else {
			log.Println("[REDIS] - ✅ Cache flushed in development mode")
		}
	}

	// Run auto migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Seed initial data
	if err := seedData(); err != nil {
		return fmt.Errorf("[DATABASE] - Failed to seed data: %w", err)
	}

	// Initialize repositories after database connection is established
	repos = repositories.NewRepositories(DB)

	return nil
}

// runMigrations performs automatic database migrations
func runMigrations() error {
	// Create custom enum types first
	if err := createEnumTypes(); err != nil {
		return err
	}

	// AutoMigrate will create tables, missing columns and missing indexes
	err := DB.AutoMigrate(
		&models.Timezone{},
		&models.Account{},
		&models.Identity{},
		&models.Reminder{},
		&models.ReminderDestination{},
		&models.ReminderError{},
		&models.BotError{},
		&models.EmailVerification{},
		&models.PasswordReset{},
		&models.DFMNote{},
		&models.DFMItem{},
		&models.FcmToken{},
	)

	if err != nil {
		return err
	}

	// Create unique constraint for provider + external_id combination
	if err := DB.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_identities_provider_external_id
		ON identities(provider, external_id)
	`).Error; err != nil {
		return err
	}

	return nil
}

// createEnumTypes creates custom PostgreSQL enum types if they don't already exist.
func createEnumTypes() error {
	if err := DB.Exec(`
		DO $$ BEGIN
			CREATE TYPE provider_type AS ENUM ('discord', 'app', 'api_key', 'mobile');
		EXCEPTION WHEN duplicate_object THEN NULL;
		END $$;
	`).Error; err != nil {
		return fmt.Errorf("failed to create provider_type enum: %w", err)
	}

	if err := DB.Exec(`
		DO $$ BEGIN
			CREATE TYPE destination_type AS ENUM ('discord_dm', 'discord_channel', 'webhook', 'email', 'android_push');
		EXCEPTION WHEN duplicate_object THEN NULL;
		END $$;
	`).Error; err != nil {
		return fmt.Errorf("failed to create destination_type enum: %w", err)
	}

	return nil
}

// seedData inserts initial timezone data if it doesn't exist
func seedData() error {
	// Check if UTC timezone already exists
	var utcTimezone models.Timezone
	result := DB.Where("iana_location = ?", "UTC").First(&utcTimezone)

	// If UTC doesn't exist, we need to seed all timezones
	if result.Error == gorm.ErrRecordNotFound {
		// Check if we have any timezones at all
		var count int64
		DB.Model(&models.Timezone{}).Count(&count)

		// If we have timezones but no UTC, something is wrong - clear and reseed
		if count > 0 {
			log.Println("[DATABASE] - ⚠️ Timezones exist but UTC not found, reseeding...")
			DB.Exec("DELETE FROM timezones")
		}

		timezones := []models.Timezone{
			{ID: 1, Name: "UTC (Coordinated Universal Time)", GMTOffset: 0.0, IANALocation: "UTC"},
			{ID: 2, Name: "International Date Line West", GMTOffset: -12.0, IANALocation: "Pacific/Kwajalein"},
			{ID: 3, Name: "Midway Island, Samoa", GMTOffset: -11.0, IANALocation: "Pacific/Midway"},
			{ID: 4, Name: "Hawaii", GMTOffset: -10.0, IANALocation: "Pacific/Honolulu"},
			{ID: 5, Name: "Alaska", GMTOffset: -9.0, IANALocation: "America/Anchorage"},
			{ID: 6, Name: "Pacific Time (US & Canada)", GMTOffset: -8.0, IANALocation: "America/Los_Angeles"},
			{ID: 7, Name: "Mountain Time (US & Canada)", GMTOffset: -7.0, IANALocation: "America/Denver"},
			{ID: 8, Name: "Central Time (US & Canada), Mexico City", GMTOffset: -6.0, IANALocation: "America/Chicago"},
			{ID: 9, Name: "Eastern Time (US & Canada), Bogota, Lima", GMTOffset: -5.0, IANALocation: "America/New_York"},
			{ID: 10, Name: "Atlantic Time (Canada), Caracas, La Paz", GMTOffset: -4.0, IANALocation: "America/Halifax"},
			{ID: 11, Name: "Newfoundland", GMTOffset: -3.5, IANALocation: "America/St_Johns"},
			{ID: 12, Name: "Brazil, Buenos Aires, Georgetown", GMTOffset: -3.0, IANALocation: "America/Sao_Paulo"},
			{ID: 13, Name: "Mid-Atlantic", GMTOffset: -2.0, IANALocation: "Atlantic/South_Georgia"},
			{ID: 14, Name: "Azores, Cape Verde Islands", GMTOffset: -1.0, IANALocation: "Atlantic/Azores"},
			{ID: 15, Name: "Western Europe Time, London, Lisbon, Casablanca", GMTOffset: 0.0, IANALocation: "Europe/London"},
			{ID: 16, Name: "Brussels, Copenhagen, Madrid, Paris", GMTOffset: 1.0, IANALocation: "Europe/Paris"},
			{ID: 17, Name: "Kaliningrad, South Africa", GMTOffset: 2.0, IANALocation: "Europe/Kaliningrad"},
			{ID: 18, Name: "Baghdad, Riyadh, Moscow, St. Petersburg", GMTOffset: 3.0, IANALocation: "Europe/Moscow"},
			{ID: 19, Name: "Tehran", GMTOffset: 3.5, IANALocation: "Asia/Tehran"},
			{ID: 20, Name: "Abu Dhabi, Muscat, Baku, Tbilisi", GMTOffset: 4.0, IANALocation: "Asia/Dubai"},
			{ID: 21, Name: "Kabul", GMTOffset: 4.5, IANALocation: "Asia/Kabul"},
			{ID: 22, Name: "Ekaterinburg, Islamabad, Karachi, Tashkent", GMTOffset: 5.0, IANALocation: "Asia/Karachi"},
			{ID: 23, Name: "Bombay, Calcutta, Madras, New Delhi", GMTOffset: 5.5, IANALocation: "Asia/Kolkata"},
			{ID: 24, Name: "Kathmandu", GMTOffset: 5.75, IANALocation: "Asia/Kathmandu"},
			{ID: 25, Name: "Almaty, Dhaka, Colombo", GMTOffset: 6.0, IANALocation: "Asia/Almaty"},
			{ID: 26, Name: "Yangon, Bangkok, Hanoi, Jakarta", GMTOffset: 6.5, IANALocation: "Asia/Yangon"},
			{ID: 27, Name: "Bangkok, Hanoi, Jakarta", GMTOffset: 7.0, IANALocation: "Asia/Bangkok"},
			{ID: 28, Name: "Beijing, Perth, Singapore, Hong Kong", GMTOffset: 8.0, IANALocation: "Asia/Shanghai"},
			{ID: 29, Name: "Tokyo, Seoul, Osaka, Sapporo, Yakutsk", GMTOffset: 9.0, IANALocation: "Asia/Tokyo"},
			{ID: 30, Name: "Darwin", GMTOffset: 9.5, IANALocation: "Australia/Darwin"},
			{ID: 31, Name: "Eastern Australia, Guam, Vladivostok", GMTOffset: 10.0, IANALocation: "Australia/Sydney"},
			{ID: 32, Name: "Magadan, Solomon Islands, New Caledonia", GMTOffset: 11.0, IANALocation: "Pacific/Guadalcanal"},
			{ID: 33, Name: "Auckland, Wellington, Fiji, Kamchatka", GMTOffset: 12.0, IANALocation: "Pacific/Auckland"},
		}

		if err := DB.Create(&timezones).Error; err != nil {
			return err
		}

		log.Printf("[DATABASE] - ✅ Seeded %d timezones", len(timezones))
	}

	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// GetRepositories returns the repository instances
func GetRepositories() *repositories.Repositories {
	return repos
}

// Close closes the database connection
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
