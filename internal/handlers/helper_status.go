package handlers

import "strings"

// normalizeListingStatus ensures that a listing has a persisted status value.
// Some clients omit the status field when creating a listing which resulted in
// empty strings being stored in the database. Empty statuses were ignored by the
// subscription counters, so slots were never consumed. To avoid this, default
// to "pending" which still represents a non-archived listing but can be counted
// as an active paid slot.
func normalizeListingStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "pending"
	}
	return status
}
