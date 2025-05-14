package utils

import (
	"context"
	"database/sql"
	"dk/db"
	"log"
	"time"
)

// StartPolicyWorker begins a background worker that periodically checks for and applies
// scheduled policy changes that have reached their effective date.
func StartPolicyWorker(ctx context.Context, database *sql.DB, checkInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("Policy worker shutting down")
				return
			case <-ticker.C:
				applyPendingPolicyChanges(ctx, database)
			}
		}
	}()

	log.Printf("Policy worker started with check interval of %v", checkInterval)
}

// applyPendingPolicyChanges checks for and applies any pending policy changes
func applyPendingPolicyChanges(ctx context.Context, database *sql.DB) {
	pendingChanges, err := db.GetPendingPolicyChanges(database)
	if err != nil {
		log.Printf("Error getting pending policy changes: %v", err)
		return
	}

	if len(pendingChanges) == 0 {
		// No pending changes, nothing to do
		return
	}

	log.Printf("Found %d pending policy changes to apply", len(pendingChanges))

	for _, change := range pendingChanges {
		if err := db.ApplyPendingPolicyChange(database, change); err != nil {
			log.Printf("Error applying policy change %s: %v", change.ID, err)
			continue
		}

		log.Printf("Applied policy change %s (API: %s, New Policy: %s)",
			change.ID, change.APIID, *change.NewPolicyID)
	}
}
