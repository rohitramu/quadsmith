package blob

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"quadsmith/api/internal/authcontext"
)

func TestNormalizeMIMEType(t *testing.T) {
	tests := []struct {
		filename    string
		headerBytes []byte
		wantMIME    string
		wantMax     int64
		wantErr     bool
	}{
		{
			filename:    "frame.stl",
			headerBytes: []byte("solid OpenSCAD_Model"),
			wantMIME:    "model/stl",
			wantMax:     MaxModelSize,
			wantErr:     false,
		},
		{
			filename:    "mount.3mf",
			headerBytes: []byte("PK\x03\x04"),
			wantMIME:    "model/3mf",
			wantMax:     MaxModelSize,
			wantErr:     false,
		},
		{
			filename:    "antenna.step",
			headerBytes: []byte("ISO-10303-21;"),
			wantMIME:    "model/step",
			wantMax:     MaxModelSize,
			wantErr:     false,
		},
		{
			filename:    "photo.png",
			headerBytes: []byte("\x89PNG\r\n\x1a\n"),
			wantMIME:    "image/png",
			wantMax:     MaxImageSize,
			wantErr:     false,
		},
		{
			filename:    "photo.jpg",
			headerBytes: []byte("\xff\xd8\xff"),
			wantMIME:    "image/jpeg",
			wantMax:     MaxImageSize,
			wantErr:     false,
		},
		{
			filename:    "flight.mp4",
			headerBytes: []byte("\x00\x00\x00\x18ftypmp42"),
			wantMIME:    "video/mp4",
			wantMax:     MaxVideoSize,
			wantErr:     false,
		},
		{
			filename:    "malware.exe",
			headerBytes: []byte("MZ\x90\x00"),
			wantErr:     true,
		},
		{
			filename:    "script.sh",
			headerBytes: []byte("#!/bin/bash"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			mime, max, err := NormalizeMIMEType(tt.filename, tt.headerBytes)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %s, got nil", tt.filename)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.filename, err)
			}
			if mime != tt.wantMIME {
				t.Errorf("expected MIME %q, got %q", tt.wantMIME, mime)
			}
			if max != tt.wantMax {
				t.Errorf("expected max %d, got %d", tt.wantMax, max)
			}
		})
	}
}

func TestExtractBlobKeys(t *testing.T) {
	urls := []string{
		"/blobs/e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"https://quadsmith.dev/blobs/e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855/gopro.stl",
		"/blobs/0192e4a8-f7b2-7000-8000-000000000001",
		"https://youtube.com/watch?v=123",
		"https://example.com/some/other/image.png",
	}

	keys := ExtractBlobKeys(urls)
	if len(keys) != 2 {
		t.Fatalf("expected 2 unique blob keys, got %d: %v", len(keys), keys)
	}

	expectedKey1 := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	expectedKey2 := "0192e4a8-f7b2-7000-8000-000000000001"

	if keys[0] != expectedKey1 || keys[1] != expectedKey2 {
		t.Errorf("unexpected extracted keys: %v", keys)
	}
}

func TestHTTPHandlerHeadersAndRange(t *testing.T) {
	// Create mock blob data
	testData := []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
	hasher := sha256.New()
	hasher.Write(testData)
	shaHex := hex.EncodeToString(hasher.Sum(nil))

	blob := &Blob{
		UUID:      authcontext.NewUUIDv7(),
		ID:        shaHex,
		Filename:  "test_data.stl",
		MIMEType:  "model/stl",
		SizeBytes: int64(len(testData)),
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		http.ServeContent(w, r, blob.Filename, blob.CreatedAt, bytes.NewReader(testData))
	})

	// 1. Test full GET
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/blobs/"+shaHex, nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Errorf("unexpected Cache-Control: %s", rec.Header().Get("Cache-Control"))
	}
	if rec.Header().Get("Content-Type") != "model/stl" {
		t.Errorf("unexpected Content-Type: %s", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Accept-Ranges") != "bytes" {
		t.Errorf("expected Accept-Ranges: bytes")
	}
	if rec.Body.String() != string(testData) {
		t.Errorf("body mismatch: got %q, want %q", rec.Body.String(), string(testData))
	}

	// 2. Test ETag 304 Not Modified
	rec304 := httptest.NewRecorder()
	req304 := httptest.NewRequest(http.MethodGet, "/blobs/"+shaHex, nil)
	req304.Header.Set("If-None-Match", fmt.Sprintf("%q", shaHex))
	handler.ServeHTTP(rec304, req304)

	if rec304.Code != http.StatusNotModified {
		t.Fatalf("expected status 304, got %d", rec304.Code)
	}

	// 3. Test HTTP Range request (Partial Content 206)
	recRange := httptest.NewRecorder()
	reqRange := httptest.NewRequest(http.MethodGet, "/blobs/"+shaHex, nil)
	reqRange.Header.Set("Range", "bytes=0-9")
	handler.ServeHTTP(recRange, reqRange)

	if recRange.Code != http.StatusPartialContent {
		t.Fatalf("expected status 206, got %d", recRange.Code)
	}
	if recRange.Body.String() != "0123456789" {
		t.Errorf("expected partial slice '0123456789', got %q", recRange.Body.String())
	}
	if recRange.Header().Get("Content-Range") != fmt.Sprintf("bytes 0-9/%d", len(testData)) {
		t.Errorf("unexpected Content-Range header: %s", recRange.Header().Get("Content-Range"))
	}
}

func TestUnauthenticatedUploadRejection(t *testing.T) {
	h := NewHandler(nil, func(ctx context.Context, r *http.Request) *authcontext.UserContext {
		return nil // Not authenticated
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/blobs", bytes.NewReader([]byte("test")))
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized, got %d", rec.Code)
	}
}
