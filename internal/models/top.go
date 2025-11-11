package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidTopType     = errors.New("invalid listing type for top promotion")
	ErrInvalidTopDuration = errors.New("invalid top duration")
	ErrInvalidTopID       = errors.New("invalid listing id")
)

var allowedTopTypes = map[string]string{
	"service": "service",
	"ad":      "ad",
	"work":    "work",
	"work_ad": "work_ad",
	"rent":    "rent",
	"rent_ad": "rent_ad",
}

type TopInfo struct {
	ActivatedAt  time.Time `json:"activated_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	DurationDays int       `json:"duration_days"`
}

type TopActivationRequest struct {
	Type         string `json:"type"`
	ID           int    `json:"id"`
	DurationDays int    `json:"duration_days"`
}

func (r TopActivationRequest) Validate() error {
	listingType := strings.TrimSpace(r.Type)
	if listingType == "" {
		return ErrInvalidTopType
	}
	if _, ok := allowedTopTypes[listingType]; !ok {
		return fmt.Errorf("%w: %s", ErrInvalidTopType, listingType)
	}
	if r.ID <= 0 {
		return fmt.Errorf("%w: %d", ErrInvalidTopID, r.ID)
	}
	if r.DurationDays <= 0 {
		return fmt.Errorf("%w: %d", ErrInvalidTopDuration, r.DurationDays)
	}
	return nil
}

func (t TopInfo) Marshal() (string, error) {
	payload, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func NewTopInfo(now time.Time, durationDays int) (TopInfo, error) {
	if durationDays <= 0 {
		return TopInfo{}, fmt.Errorf("%w: %d", ErrInvalidTopDuration, durationDays)
	}
	now = now.UTC()
	return TopInfo{
		ActivatedAt:  now,
		ExpiresAt:    now.AddDate(0, 0, durationDays).UTC(),
		DurationDays: durationDays,
	}, nil
}

func ParseTopInfo(raw string) (*TopInfo, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var info TopInfo
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		if ts, parseErr := time.Parse(time.RFC3339, raw); parseErr == nil {
			inferred := TopInfo{
				ActivatedAt: ts.UTC(),
				ExpiresAt:   ts.UTC(),
			}
			return &inferred, nil
		}
		return nil, nil
	}
	if !info.ActivatedAt.IsZero() {
		info.ActivatedAt = info.ActivatedAt.UTC()
	}
	if !info.ExpiresAt.IsZero() {
		info.ExpiresAt = info.ExpiresAt.UTC()
	}
	return &info, nil
}

func (t TopInfo) IsActive(now time.Time) bool {
	if t.ActivatedAt.IsZero() || t.ExpiresAt.IsZero() {
		return false
	}
	now = now.UTC()
	return now.Before(t.ExpiresAt)
}

func AllowedTopTypes() map[string]string {
	result := make(map[string]string, len(allowedTopTypes))
	for k, v := range allowedTopTypes {
		result[k] = v
	}
	return result
}

func ResolveTopTable(listingType string) (string, bool) {
	listingType = strings.TrimSpace(listingType)
	table, ok := allowedTopTypes[listingType]
	return table, ok
}
