package com.chronos.reminder.core

/**
 * Per-account caps mirrored from the backend (see internal/services/limits.go)
 * so the UI can proactively block creation instead of relying solely on the
 * API's rejection.
 */
const val MAX_REMINDERS_PER_ACCOUNT = 100
const val MAX_DFM_ITEMS_PER_NOTE = 100
