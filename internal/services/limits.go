package services

// Per-account caps that prevent a single account from flooding the system
// with an unbounded number of reminders or "Don't Forget Me" items.
const (
	MaxRemindersPerAccount = 100
	MaxDFMItemsPerNote     = 100
)
