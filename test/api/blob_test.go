package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	connect "connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	pb "quadsmith/api/gen/quadsmith/api"
	"quadsmith/api/gen/quadsmith/api/apiconnect"
)

func TestCLIBlobUploadAndGet(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	if _, err := client.Get(apiUrl); err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	tempDir := t.TempDir()
	sampleSTL := filepath.Join(tempDir, "mount.stl")
	stlContent := []byte("solid OpenSCAD_Model\nendsolid OpenSCAD_Model")
	if err := os.WriteFile(sampleSTL, stlContent, 0644); err != nil {
		t.Fatalf("failed to write test STL: %v", err)
	}

	// 1. Upload using CLI
	cmd := newQSCommand(t, "blob", "upload", sampleSTL, "--json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qs blob upload failed: %v, output: %s", err, string(out))
	}

	var uploadResp map[string]interface{}
	if err := json.Unmarshal(out, &uploadResp); err != nil {
		t.Fatalf("failed to parse upload json: %v, raw output: %s", err, string(out))
	}

	blobID, ok := uploadResp["id"].(string)
	if !ok || blobID == "" {
		t.Fatalf("expected valid blob id in upload output: %v", uploadResp)
	}

	mimeType, _ := uploadResp["mime_type"].(string)
	if mimeType != "model/stl" {
		t.Errorf("expected mime_type 'model/stl', got %q", mimeType)
	}

	// 2. Get blob content using CLI
	downloadedPath := filepath.Join(tempDir, "downloaded.stl")
	getCmd := newQSCommand(t, "blob", "get", blobID, "--output", downloadedPath)
	getOut, err := getCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qs blob get failed: %v, output: %s", err, string(getOut))
	}

	getContent, err := os.ReadFile(downloadedPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}

	if !bytes.Equal(getContent, stlContent) {
		t.Errorf("retrieved blob content mismatch. expected %q, got %q", string(stlContent), string(getContent))
	}

	// 3. Get metadata using CLI
	metaCmd := newQSCommand(t, "blob", "get", blobID, "--metadata", "--json")
	metaOut, err := metaCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qs blob get --metadata failed: %v, output: %s", err, string(metaOut))
	}

	var metaResp map[string]interface{}
	if err := json.Unmarshal(metaOut, &metaResp); err != nil {
		t.Fatalf("failed to parse metadata json: %v, raw output: %s", err, string(metaOut))
	}
	if metaResp["id"] != blobID {
		t.Errorf("expected metadata id %q, got %q", blobID, metaResp["id"])
	}
}

func TestBlobHTTPEndpointsAndRange(t *testing.T) {
	apiUrl := getAPIURL()
	payload := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR0123456789abcdefghijklmnopqrstuvwxyz")

	// 1. Upload via multipart HTTP POST
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "test.png")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatalf("failed to write payload: %v", err)
	}
	writer.Close()

	req, err := http.NewRequest("POST", apiUrl+"/api/blobs", &body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if testAccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+testAccessToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/blobs failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected upload status %d: %s", resp.StatusCode, string(respBody))
	}

	var uploadRes struct {
		ID       string `json:"id"`
		MimeType string `json:"mime_type"`
		ByteSize int64  `json:"byte_size"`
		URL      string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&uploadRes); err != nil {
		t.Fatalf("failed to decode upload response: %v", err)
	}
	if uploadRes.ID == "" {
		t.Fatalf("expected non-empty blob ID")
	}

	// 2. Unauthenticated GET /blobs/{id}
	getResp, err := http.Get(apiUrl + "/blobs/" + uploadRes.ID)
	if err != nil {
		t.Fatalf("GET /blobs/%s failed: %v", uploadRes.ID, err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", getResp.StatusCode)
	}
	if cc := getResp.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("expected immutable cache-control, got %q", cc)
	}
	gotBytes, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if !bytes.Equal(gotBytes, payload) {
		t.Errorf("payload mismatch: expected %d bytes, got %d bytes", len(payload), len(gotBytes))
	}

	// 3. HTTP Range Request (206 Partial Content)
	rangeReq, err := http.NewRequest("GET", apiUrl+"/blobs/"+uploadRes.ID, nil)
	if err != nil {
		t.Fatalf("failed to create range request: %v", err)
	}
	rangeReq.Header.Set("Range", "bytes=0-3")
	rangeResp, err := http.DefaultClient.Do(rangeReq)
	if err != nil {
		t.Fatalf("Range request failed: %v", err)
	}
	defer rangeResp.Body.Close()

	if rangeResp.StatusCode != http.StatusPartialContent {
		t.Fatalf("expected 206 Partial Content, got %d", rangeResp.StatusCode)
	}
	rangeBytes, err := io.ReadAll(rangeResp.Body)
	if err != nil {
		t.Fatalf("failed to read range response: %v", err)
	}
	if !bytes.Equal(rangeBytes, payload[:4]) {
		t.Errorf("expected first 4 bytes %q, got %q", payload[:4], rangeBytes)
	}
}

func TestBlobBuildLifecycleLinking(t *testing.T) {
	apiUrl := getAPIURL()
	dbUrl := getDBURL()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		t.Fatalf("failed to connect to DB: %v", err)
	}
	defer pool.Close()

	// 1. Upload a blob
	stlPayload := []byte("solid QuadPart\nendsolid QuadPart")
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("file", "part.stl")
	part.Write(stlPayload)
	writer.Close()

	req, _ := http.NewRequest("POST", apiUrl+"/api/blobs", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if testAccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+testAccessToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/blobs failed: %v", err)
	}
	defer resp.Body.Close()

	var uploadRes struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	json.NewDecoder(resp.Body).Decode(&uploadRes)

	blobURL := fmt.Sprintf("%s/blobs/%s/part.stl", apiUrl, uploadRes.ID)

	// 2. Create a Build linking this blob in its media
	desc := "3D Printable Mount"
	buildClient := apiconnect.NewBuildServiceClient(http.DefaultClient, apiUrl)
	createReq := connect.NewRequest(&pb.CreateBuildRequest{
		Build: &pb.Build{
			Name:                 "Blob Lifecycle Drone",
			Description:          "Drone with attached 3D model blob",
			FrameUuid:            "01923019-3008-7001-8001-000000000001",
			MotorUuid:            "01923019-3001-7001-8001-000000000001",
			PropellerUuid:        "bac19aec-99fc-43d4-b96d-d1feb30e6b6e",
			FlightControllerUuid: "01923019-3006-7001-8001-000000000001",
			VideoTransmitterUuid: "01923019-3004-7001-8001-000000000008",
			Media: []*pb.Media{
				{
					Url:         blobURL,
					Type:        pb.MediaType_MEDIA_TYPE_MODEL_3D,
					Description: &desc,
				},
			},
		},
	})

	createRes, err := buildClient.CreateBuild(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateBuild failed: %v", err)
	}
	createdBuild := createRes.Msg

	// 3. Verify in DB that blob_references has an entry linking the blob to this build
	var count int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM blob_references br
		JOIN blobs b ON br.blob_uuid = b.uuid
		WHERE b.id = $1 AND br.entity_type = 'build'
	`, uploadRes.ID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query blob_references: %v", err)
	}
	if count == 0 {
		t.Errorf("expected at least 1 blob_reference for blob %s on build, got 0", uploadRes.ID)
	}

	// 4. Delete Build and verify blob_references is cleaned up
	_, err = buildClient.DeleteBuild(ctx, connect.NewRequest(&pb.DeleteBuildRequest{
		Id: createdBuild.Id,
	}))
	if err != nil {
		t.Fatalf("DeleteBuild failed: %v", err)
	}

	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM blob_references br
		JOIN blobs b ON br.blob_uuid = b.uuid
		WHERE b.id = $1 AND br.entity_type = 'build'
	`, uploadRes.ID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query blob_references after build delete: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 blob_references after build delete, got %d", count)
	}
}
