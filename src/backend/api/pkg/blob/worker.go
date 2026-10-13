package blob

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HandleCleanupOrphanedBlobs deletes blobs that are not referenced in blob_references and were created > 5 minutes ago.
func HandleCleanupOrphanedBlobs(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	pruned, err := PruneOrphanBlobs(ctx, pool, 5*time.Minute)
	if err != nil {
		return 0, fmt.Errorf("failed to prune orphaned blobs: %w", err)
	}
	if pruned > 0 {
		log.Printf("[BlobCleanupWorker] Pruned %d orphaned blobs", pruned)
	}
	return pruned, nil
}
