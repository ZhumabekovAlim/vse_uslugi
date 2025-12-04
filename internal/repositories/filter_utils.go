package repositories

import (
	"fmt"
	"strings"
	"time"
)

func matchesLanguageFilter(required []string, available []string) bool {
	if len(required) == 0 {
		return true
	}
	availableSet := make(map[string]struct{}, len(available))
	for _, lang := range available {
		availableSet[strings.ToLower(strings.TrimSpace(lang))] = struct{}{}
	}
	for _, lang := range required {
		if _, ok := availableSet[strings.ToLower(strings.TrimSpace(lang))]; ok {
			return true
		}
	}
	return false
}

func parseDailyTime(value string) (time.Time, error) {
	layouts := []string{"15:04:05", "15:04"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time format: %s", value)
}

func isRoundTheClock(workTimeFrom, workTimeTo string) bool {
	from, errFrom := parseDailyTime(workTimeFrom)
	to, errTo := parseDailyTime(workTimeTo)
	if errFrom != nil || errTo != nil {
		return false
	}
	return from.Hour() == 0 && from.Minute() == 0 && to.Hour() == 23 && to.Minute() >= 59
}

func isCurrentlyOpen(workTimeFrom, workTimeTo string, now time.Time) bool {
	from, errFrom := parseDailyTime(workTimeFrom)
	to, errTo := parseDailyTime(workTimeTo)
	if errFrom != nil || errTo != nil {
		return false
	}

	current := time.Date(0, 1, 1, now.Hour(), now.Minute(), now.Second(), 0, time.UTC)
	start := time.Date(0, 1, 1, from.Hour(), from.Minute(), from.Second(), 0, time.UTC)
	end := time.Date(0, 1, 1, to.Hour(), to.Minute(), to.Second(), 0, time.UTC)

	if end.Before(start) {
		end = end.Add(24 * time.Hour)
		if current.Before(start) {
			current = current.Add(24 * time.Hour)
		}
	}

	return (current.Equal(start) || current.After(start)) && current.Before(end)
}
