package services

import (
	"naimuBack/internal/models"
	"time"
)

func computeTopFields(raw string, now time.Time) (bool, *time.Time) {
	info, err := models.ParseTopInfo(raw)
	if err != nil || info == nil {
		return false, nil
	}
	expires := info.ExpiresAt
	return info.IsActive(now), &expires
}
