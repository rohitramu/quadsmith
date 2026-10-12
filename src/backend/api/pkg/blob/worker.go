package blob

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/api/pkg/worker"
)

// RegisterWorkers registers all blob-related background tasks into the worker registry.
func RegisterWorkers(r *worker.Registry, pool *pgxpool.Pool) {
	// Periodic 5m task: clean up unreferenced/orphaned blobs older than 5m grace period
	worker.RegisterPeriodic(r, "blob.cleanup_orphaned_blobs", 5*time.Minute, worker.EmptyPayload{}, func(ctx context.Context, job worker.Job[worker.EmptyPayload]) error {
		return HandleCleanupOrphanedBlobs(ctx, pool)
	})
}

// HandleCleanupOrphanedBlobs deletes blobs that are not referenced in blob_references and were created > 5 minutes ago.
func HandleCleanupOrphanedBlobs(ctx context.Context, pool *pgxpool.Pool) error {
	pruned, err := PruneOrphanBlobs(ctx, pool, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to prune orphaned blobs: %w", err)
	}
	if pruned > 0 {
		log.Printf("[BlobCleanupWorker] Pruned %d orphaned blobs", pruned)
	}
	return nil
}
