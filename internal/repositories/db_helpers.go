package repositories

import "strings"

// isDuplicateKeyError returns true if the error indicates a duplicate key or unique constraint violation.
// Used by Postgres/GORM for idempotency checks.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "UNIQUE constraint") ||
		strings.Contains(errStr, "23505")
}
