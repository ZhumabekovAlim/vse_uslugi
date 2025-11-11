package main

import (
	"context"
	"log"
	"time"

	"naimuBack/internal/services"
)

const (
	topCleanerInterval = 5 * time.Minute
	topCleanerTimeout  = 30 * time.Second
)

func startTopCleaner(ctx context.Context, svc *services.TopService, infoLog, errorLog *log.Logger) {
	if svc == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(topCleanerInterval)
		defer ticker.Stop()

		run := func() {
			runCtx, cancel := context.WithTimeout(ctx, topCleanerTimeout)
			defer cancel()

			cleared, err := svc.ClearExpiredTop(runCtx, time.Now())
			if err != nil {
				if errorLog != nil {
					errorLog.Printf("top cleaner: failed to clear expired promotions: %v", err)
				}
				return
			}
			if cleared > 0 && infoLog != nil {
				infoLog.Printf("top cleaner: cleared %d expired promotions", cleared)
			}
		}

		run()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}
