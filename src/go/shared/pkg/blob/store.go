package blob

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/shared/pkg/authcontext"
)

var (
	ErrUnsupportedMediaType = errors.New("unsupported media type")
	ErrPayloadTooLarge      = errors.New("payload too large")
	ErrBlobNotFound         = errors.New("blob not found")
)

const (
	MaxImageSize = 15 * 1024 * 1024  // 15 MB
	MaxModelSize = 35 * 1024 * 1024  // 35 MB
	MaxVideoSize = 100 * 1024 * 1024 // 100 MB
)

// Blob represents binary metadata and domain model.
type Blob struct {
	UUID      string    `json:"uuid"`
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	MIMEType  string    `json:"mime_type"`
	SizeBytes int64     `json:"size_bytes"`
	UserUUID  string    `json:"user_uuid"`
	CreatedAt time.Time `json:"created_at"`
}

// Allowed MIME types mapped to their category and maximum allowed size
var allowedMIMETypes = map[string]int64{
	// Images (15MB)
	"image/jpeg":    MaxImageSize,
	"image/png":     MaxImageSize,
	"image/webp":    MaxImageSize,
	"image/gif":     MaxImageSize,
	"image/svg+xml": MaxImageSize,
	"image/avif":    MaxImageSize,

	// 3D Models (35MB)
	"model/stl":                  MaxModelSize,
	"application/sla":            MaxModelSize,
	"application/vnd.ms-pki.stl": MaxModelSize,
	"model/3mf":                  MaxModelSize,
	"application/vnd.ms-package.3dmanufacturing-3dmodel+xml": MaxModelSize,
	"model/step":         MaxModelSize,
	"application/step":   MaxModelSize,
	"application/x-step": MaxModelSize,
	"model/obj":          MaxModelSize,

	// Videos (100MB)
	"video/mp4":       MaxVideoSize,
	"video/webm":      MaxVideoSize,
	"video/quicktime": MaxVideoSize,
}

// NormalizeMIMEType sniffs the content type from the initial bytes and checks filename extension for known types.
func NormalizeMIMEType(filename string, headerBytes []byte) (string, int64, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	sniffed := http.DetectContentType(headerBytes)
	mimeType := sniffed

	// Extension overrides for formats where sniffed content type defaults to text/plain or application/octet-stream
	switch ext {
	case ".stl":
		mimeType = "model/stl"
	case ".3mf":
		mimeType = "model/3mf"
	case ".step", ".stp":
		mimeType = "model/step"
	case ".obj":
		mimeType = "model/obj"
	case ".mp4":
		mimeType = "video/mp4"
	case ".webm":
		mimeType = "video/webm"
	case ".mov":
		mimeType = "video/quicktime"
	case ".png":
		mimeType = "image/png"
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".webp":
		mimeType = "image/webp"
	case ".gif":
		mimeType = "image/gif"
	case ".svg":
		mimeType = "image/svg+xml"
	case ".avif":
		mimeType = "image/avif"
	}

	maxSize, allowed := allowedMIMETypes[mimeType]
	if !allowed {
		// Fallback check: if sniffed is allowed
		if sMax, sAllowed := allowedMIMETypes[sniffed]; sAllowed {
			return sniffed, sMax, nil
		}
		return "", 0, fmt.Errorf("%w: detected %q for file %q", ErrUnsupportedMediaType, mimeType, filename)
	}

	return mimeType, maxSize, nil
}

// PutBlob streams the incoming reader, validates content type and size limits, hashes via SHA-256,
// and idempotently inserts or retrieves the blob record.
func PutBlob(ctx context.Context, pool *pgxpool.Pool, userUUID, filename string, r io.Reader) (*Blob, bool, error) {
	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "" || filename == "." || filename == "/" {
		filename = "blob.bin"
	}

	// Sniff header bytes (up to 512 bytes)
	var headerBuf [512]byte
	n, err := io.ReadFull(r, headerBuf[:])
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, false, fmt.Errorf("failed reading upload stream: %w", err)
	}

	mimeType, maxSize, err := NormalizeMIMEType(filename, headerBuf[:n])
	if err != nil {
		return nil, false, err
	}

	// Read entire stream up to maxSize + 1 to detect overflows
	hasher := sha256.New()
	hasher.Write(headerBuf[:n])

	buf := bytes.NewBuffer(make([]byte, 0, n+1024))
	buf.Write(headerBuf[:n])

	limitReader := io.LimitReader(r, maxSize-int64(n)+1)
	written, err := io.Copy(io.MultiWriter(buf, hasher), limitReader)
	if err != nil {
		return nil, false, fmt.Errorf("failed reading blob stream: %w", err)
	}

	totalSize := int64(n) + written
	if totalSize > maxSize {
		return nil, false, fmt.Errorf("%w: size %d exceeds %d bytes limit for %s", ErrPayloadTooLarge, totalSize, maxSize, mimeType)
	}

	shaHex := hex.EncodeToString(hasher.Sum(nil))
	blobUUID := authcontext.NewUUIDv7()

	// Idempotent upsert into PostgreSQL
	query := `
		INSERT INTO blobs (
			uuid, id, filename, mime_type, size_bytes, bytes, user_uuid, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (id) DO UPDATE
		SET filename = EXCLUDED.filename
		RETURNING uuid, id, filename, mime_type, size_bytes, user_uuid, created_at, (xmax != 0) AS is_duplicate;`

	var (
		resUUID, resID, resFilename, resMimeType, resUserUUID string
		resSizeBytes                                          int64
		resCreatedAt                                          time.Time
		isDuplicate                                           bool
	)

	err = pool.QueryRow(ctx, query,
		blobUUID,
		shaHex,
		filename,
		mimeType,
		totalSize,
		buf.Bytes(),
		userUUID,
	).Scan(
		&resUUID,
		&resID,
		&resFilename,
		&resMimeType,
		&resSizeBytes,
		&resUserUUID,
		&resCreatedAt,
		&isDuplicate,
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to save blob: %w", err)
	}

	blob := &Blob{
		UUID:      resUUID,
		ID:        resID,
		Filename:  resFilename,
		MIMEType:  resMimeType,
		SizeBytes: resSizeBytes,
		UserUUID:  resUserUUID,
		CreatedAt: resCreatedAt,
	}

	return blob, isDuplicate, nil
}

// GetBlob retrieves the blob metadata and raw bytes reader.
func GetBlob(ctx context.Context, pool *pgxpool.Pool, idOrUUID string) (*Blob, io.ReadSeeker, error) {
	query := `
		SELECT uuid, id, filename, mime_type, size_bytes, bytes, user_uuid, created_at
		FROM blobs
		WHERE id = $1 OR uuid::text = $1
		LIMIT 1;`

	var (
		uuid, id, filename, mimeType, userUUIDStr string
		sizeBytes                                 int64
		rawBytes                                  []byte
		createdAt                                 time.Time
	)

	err := pool.QueryRow(ctx, query, idOrUUID).Scan(
		&uuid,
		&id,
		&filename,
		&mimeType,
		&sizeBytes,
		&rawBytes,
		&userUUIDStr,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrBlobNotFound
		}
		return nil, nil, fmt.Errorf("failed querying blob: %w", err)
	}

	b := &Blob{
		UUID:      uuid,
		ID:        id,
		Filename:  filename,
		MIMEType:  mimeType,
		SizeBytes: sizeBytes,
		UserUUID:  userUUIDStr,
		CreatedAt: createdAt,
	}

	return b, bytes.NewReader(rawBytes), nil
}

// GetBlobMetadata retrieves the blob metadata without loading the BYTEA payload.
func GetBlobMetadata(ctx context.Context, pool *pgxpool.Pool, idOrUUID string) (*Blob, error) {
	query := `
		SELECT uuid, id, filename, mime_type, size_bytes, user_uuid, created_at
		FROM blobs
		WHERE id = $1 OR uuid::text = $1
		LIMIT 1;`

	var (
		uuid, id, filename, mimeType, userUUIDStr string
		sizeBytes                                 int64
		createdAt                                 time.Time
	)

	err := pool.QueryRow(ctx, query, idOrUUID).Scan(
		&uuid,
		&id,
		&filename,
		&mimeType,
		&sizeBytes,
		&userUUIDStr,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBlobNotFound
		}
		return nil, fmt.Errorf("failed querying blob metadata: %w", err)
	}

	b := &Blob{
		UUID:      uuid,
		ID:        id,
		Filename:  filename,
		MIMEType:  mimeType,
		SizeBytes: sizeBytes,
		UserUUID:  userUUIDStr,
		CreatedAt: createdAt,
	}

	return b, nil
}

var blobIDRegex = regexp.MustCompile(`(?i)(?:^|/blobs/)([a-f0-9]{64}|[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})`)

// ExtractBlobKeys finds all blob SHA-256 hashes or UUIDs from arbitrary URLs.
func ExtractBlobKeys(urls []string) []string {
	seen := make(map[string]bool)
	var keys []string

	for _, u := range urls {
		matches := blobIDRegex.FindStringSubmatch(strings.TrimSpace(u))
		if len(matches) > 1 {
			k := strings.ToLower(matches[1])
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	return keys
}

// LinkEntityBlobs creates relational references between an entity (or user) and any referenced blobs.
func LinkEntityBlobs(ctx context.Context, tx pgx.Tx, entityType, targetUUID string, rawURLs []string) error {
	keys := ExtractBlobKeys(rawURLs)
	if len(keys) == 0 {
		return nil
	}

	rows, err := tx.Query(ctx, `SELECT uuid FROM blobs WHERE id = ANY($1) OR uuid::text = ANY($1)`, keys)
	if err != nil {
		return fmt.Errorf("failed resolving blob uuids: %w", err)
	}
	defer rows.Close()

	var blobUUIDs []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err == nil {
			blobUUIDs = append(blobUUIDs, u)
		}
	}

	for _, bu := range blobUUIDs {
		refUUID := authcontext.NewUUIDv7()
		refID := "ref-" + refUUID
		var execErr error
		if entityType == "user_avatar" {
			_, execErr = tx.Exec(ctx, `
				INSERT INTO blob_references (uuid, id, blob_uuid, entity_type, user_uuid, created_at)
				VALUES ($1, $2, $3, $4, $5, NOW())
				ON CONFLICT (blob_uuid, entity_type, user_uuid) DO NOTHING;`,
				refUUID, refID, bu, entityType, targetUUID)
		} else {
			_, execErr = tx.Exec(ctx, `
				INSERT INTO blob_references (uuid, id, blob_uuid, entity_type, entity_uuid, created_at)
				VALUES ($1, $2, $3, $4, $5, NOW())
				ON CONFLICT (blob_uuid, entity_type, entity_uuid) DO NOTHING;`,
				refUUID, refID, bu, entityType, targetUUID)
		}
		if execErr != nil {
			return fmt.Errorf("failed creating blob reference: %w", execErr)
		}
	}

	return nil
}

// UnlinkEntityBlobs removes all references from blob_references for the given entity or user.
func UnlinkEntityBlobs(ctx context.Context, tx pgx.Tx, entityType, targetUUID string) error {
	var err error
	if entityType == "user_avatar" {
		_, err = tx.Exec(ctx, `DELETE FROM blob_references WHERE entity_type = $1 AND user_uuid = $2;`, entityType, targetUUID)
	} else {
		_, err = tx.Exec(ctx, `DELETE FROM blob_references WHERE entity_type = $1 AND entity_uuid = $2;`, entityType, targetUUID)
	}
	if err != nil {
		return fmt.Errorf("failed deleting blob references: %w", err)
	}
	return nil
}

// PruneOrphanBlobs deletes blobs that are not referenced in blob_references and are older than gracePeriod.
func PruneOrphanBlobs(ctx context.Context, pool *pgxpool.Pool, gracePeriod time.Duration) (int64, error) {
	intervalSeconds := int64(gracePeriod.Seconds())
	if intervalSeconds <= 0 {
		intervalSeconds = 300 // default 5 minutes
	}

	query := fmt.Sprintf(`
		DELETE FROM blobs
		WHERE uuid NOT IN (SELECT blob_uuid FROM blob_references)
		  AND created_at < NOW() - INTERVAL '%d seconds';`, intervalSeconds)

	cmd, err := pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed pruning orphaned blobs: %w", err)
	}

	return cmd.RowsAffected(), nil
}
