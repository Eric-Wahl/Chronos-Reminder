package services

import (
	"fmt"

	"github.com/ericp/chronos-bot-reminder/internal/database"
	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/google/uuid"
)

// UpdateAccountPreferences merges the given preference values into the
// account's Preferences and persists them. It always re-reads the account
// fresh from the database rather than trusting a possibly-stale snapshot
// (e.g. one resolved through the Discord bot's account cache, which can be
// up to 12h old). Without this, setting one preference from a stale
// snapshot would silently revert any other preference changed since that
// snapshot was fetched, because Account.Update saves the whole Preferences
// column. The account's KV cache entries are invalidated afterwards so the
// next bot command sees the fresh preferences instead of the old snapshot.
func UpdateAccountPreferences(accountID uuid.UUID, updates map[string]bool) (*models.Account, error) {
	repo := database.GetRepositories()

	account, err := repo.Account.GetWithIdentities(accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("account not found")
	}

	if account.Preferences == nil {
		account.Preferences = models.JSONB{}
	}
	for key, value := range updates {
		account.Preferences[key] = value
	}

	if err := repo.Account.Update(account); err != nil {
		return nil, err
	}

	if err := InvalidateAccountCache(account); err != nil {
		fmt.Printf("[CACHE] Warning: failed to invalidate account cache for %s: %v\n", account.ID, err)
	}

	return account, nil
}
