package blob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/api/pkg/authcontext"
)

// Authenticator validates an incoming HTTP request and returns UserContext if valid.
type Authenticator func(ctx context.Context, r *http.Request) *authcontext.UserContext

// Handler serves HTTP endpoints for blob uploads, downloads, and metadata retrieval.
type Handler struct {
	pool   *pgxpool.Pool
	authFn Authenticator
}

func NewHandler(pool *pgxpool.Pool, authFn Authenticator) *Handler {
	return &Handler{
		pool:   pool,
		authFn: authFn,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case r.Method == http.MethodPost && (path == "/api/blobs" || path == "/api/blobs/"):
		h.handleUpload(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/api/blobs/") && strings.HasSuffix(path, "/metadata"):
		h.handleMetadata(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/blobs/"):
		h.handleDownload(w, r)
	default:
		http.NotFound(w, r)
	}
}

type uploadResponse struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	MIMEType    string `json:"mime_type"`
	SizeBytes   int64  `json:"size_bytes"`
	IsDuplicate bool   `json:"is_duplicate,omitempty"`
}

func (h *Handler) handleUpload(w http.ResponseWriter, r *http.Request) {
	if h.authFn == nil {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	userCtx := h.authFn(r.Context(), r)
	if userCtx == nil {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var filename string
	var bodyReader = r.Body

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Parse multipart form up to 100MB
		if err := r.ParseMultipartForm(MaxVideoSize); err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid multipart form: %v", err))
			return
		}
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "file field is required in multipart form")
			return
		}
		defer file.Close()
		filename = fileHeader.Filename
		bodyReader = file
	} else {
		// Raw binary stream
		filename = r.URL.Query().Get("filename")
		if filename == "" {
			filename = r.Header.Get("X-Filename")
		}
		if filename == "" {
			filename = "upload.bin"
		}
	}

	blob, isDuplicate, err := PutBlob(r.Context(), h.pool, userCtx.UserUUID, filename, bodyReader)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnsupportedMediaType):
			writeJSONError(w, http.StatusUnsupportedMediaType, err.Error())
		case errors.Is(err, ErrPayloadTooLarge):
			writeJSONError(w, http.StatusRequestEntityTooLarge, err.Error())
		default:
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("upload failed: %v", err))
		}
		return
	}

	statusCode := http.StatusCreated
	if isDuplicate {
		statusCode = http.StatusOK
	}

	resp := uploadResponse{
		ID:          blob.ID,
		URL:         "/blobs/" + blob.ID,
		Filename:    blob.Filename,
		MIMEType:    blob.MIMEType,
		SizeBytes:   blob.SizeBytes,
		IsDuplicate: isDuplicate,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleDownload(w http.ResponseWriter, r *http.Request) {
	// Path pattern: /blobs/<id> or /blobs/<id>/<filename>
	subPath := strings.TrimPrefix(r.URL.Path, "/blobs/")
	parts := strings.SplitN(subPath, "/", 2)
	idOrUUID := strings.TrimSpace(parts[0])

	if idOrUUID == "" {
		http.NotFound(w, r)
		return
	}

	blob, reader, err := GetBlob(r.Context(), h.pool, idOrUUID)
	if err != nil {
		if errors.Is(err, ErrBlobNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Content-addressed blobs are immutable
	eTag := fmt.Sprintf("%q", blob.ID)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("ETag", eTag)
	w.Header().Set("Content-Type", blob.MIMEType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", blob.Filename))
	w.Header().Set("Accept-Ranges", "bytes")

	if r.Header.Get("If-None-Match") == eTag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	modTime := blob.CreatedAt
	http.ServeContent(w, r, blob.Filename, modTime, reader)
}

func (h *Handler) handleMetadata(w http.ResponseWriter, r *http.Request) {
	// Path pattern: /api/blobs/<id>/metadata
	subPath := strings.TrimPrefix(r.URL.Path, "/api/blobs/")
	idOrUUID := strings.TrimSuffix(subPath, "/metadata")
	idOrUUID = strings.TrimSpace(strings.Trim(idOrUUID, "/"))

	if idOrUUID == "" {
		http.NotFound(w, r)
		return
	}

	blob, err := GetBlobMetadata(r.Context(), h.pool, idOrUUID)
	if err != nil {
		if errors.Is(err, ErrBlobNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := uploadResponse{
		ID:        blob.ID,
		URL:       "/blobs/" + blob.ID,
		Filename:  blob.Filename,
		MIMEType:  blob.MIMEType,
		SizeBytes: blob.SizeBytes,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func writeJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
