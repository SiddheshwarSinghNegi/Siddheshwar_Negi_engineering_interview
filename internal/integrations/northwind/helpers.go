package northwind

import (
	"time"

	"github.com/array/banking-api/internal/models"
)

// MapStatus maps NorthWind API status strings to our local status constants
func MapStatus(apiStatus string) string {
	switch apiStatus {
	case "COMPLETED", "completed":
		return models.NWTransferStatusCompleted
	case "FAILED", "failed":
		return models.NWTransferStatusFailed
	case "CANCELLED", "cancelled":
		return models.NWTransferStatusCancelled
	case "REVERSED", "reversed":
		return models.NWTransferStatusReversed
	case "PROCESSING", "processing":
		return models.NWTransferStatusProcessing
	default:
		return models.NWTransferStatusPending
	}
}

// ParseRFC3339Optional parses an RFC3339 date string; returns nil if empty or invalid
func ParseRFC3339Optional(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
