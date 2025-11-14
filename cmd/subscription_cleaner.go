package main

import (
	"context"
	"log"
	"time"

	"naimuBack/internal/repositories"
)

const (
	subscriptionCleanerTimeout = 1 * time.Minute
)

func startSubscriptionCleaner(ctx context.Context, repo *repositories.SubscriptionRepository, infoLog, errorLog *log.Logger) {
	if repo == nil {
		return
	}

	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		if errorLog != nil {
			errorLog.Printf("subscription cleaner: failed to load location Asia/Almaty: %v", err)
		}
		loc = time.FixedZone("Asia/Almaty", 6*60*60)
	}

	go func() {
		timer := time.NewTimer(durationUntilNextMidnight(time.Now().In(loc)))
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				runCtx, cancel := context.WithTimeout(ctx, subscriptionCleanerTimeout)
				processed, err := repo.ArchiveExpiredExecutorListings(runCtx, time.Now().In(loc).UTC())
				cancel()
				if err != nil {
					if errorLog != nil {
						errorLog.Printf("subscription cleaner: failed to archive expired subscriptions: %v", err)
					}
				} else if processed > 0 && infoLog != nil {
					infoLog.Printf("subscription cleaner: archived listings for %d expired subscriptions", processed)
				}
				timer.Reset(durationUntilNextMidnight(time.Now().In(loc)))
			}
		}
	}()
}

func durationUntilNextMidnight(now time.Time) time.Duration {
	next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	return next.Sub(now)
}
