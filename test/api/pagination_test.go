package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/encoding/prototext"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
)

func resolveSeedPath(t *testing.T) string {
	t.Helper()
	// Relative from test/api/
	p := filepath.Join("..", "..", "src", "backend", "db", "seeds", "motors.textproto")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	// Relative from test/
	p = filepath.Join("..", "src", "backend", "db", "seeds", "motors.textproto")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	// Relative from root
	p = filepath.Join("src", "backend", "db", "seeds", "motors.textproto")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	t.Fatalf("Failed to resolve motors.textproto path")
	return ""
}

func setupTestServer(t *testing.T) (*httptest.Server, *pgxpool.Pool) {
	t.Helper()
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "postgres://postgres:postgres@localhost:5432/quadsmith"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("Database ping failed: %v", err)
	}

	mux := http.NewServeMux()
	path, handler := quadsmithconnect.NewMotorServiceHandler(pb.NewMotorServiceHandler(pool))
	mux.Handle(path, handler)

	pathFrame, handlerFrame := quadsmithconnect.NewFrameServiceHandler(pb.NewFrameServiceHandler(pool))
	mux.Handle(pathFrame, handlerFrame)

	srv := httptest.NewServer(h2c.NewHandler(mux, &http2.Server{}))
	return srv, pool
}

func TestPagination_BasicFlow(t *testing.T) {
	srv, pool := setupTestServer(t)
	defer srv.Close()
	defer pool.Close()

	client := quadsmithconnect.NewMotorServiceClient(http.DefaultClient, srv.URL)
	ctx := context.Background()

	// 1. Fetch first page of 5 motors
	pageSize := int32(5)
	res1, err := client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		PageSize: pageSize,
	}))
	if err != nil {
		t.Fatalf("failed to list motors page 1: %v", err)
	}

	if len(res1.Msg.Motors) != 5 {
		t.Fatalf("expected 5 motors on page 1, got %d", len(res1.Msg.Motors))
	}
	if res1.Msg.NextPageToken == "" {
		t.Fatalf("expected non-empty next_page_token on page 1")
	}

	// 2. Fetch second page using next_page_token
	res2, err := client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		PageSize:  pageSize,
		PageToken: res1.Msg.NextPageToken,
	}))
	if err != nil {
		t.Fatalf("failed to list motors page 2: %v", err)
	}

	if len(res2.Msg.Motors) != 5 {
		t.Fatalf("expected 5 motors on page 2, got %d", len(res2.Msg.Motors))
	}
	if res2.Msg.NextPageToken == "" {
		t.Fatalf("expected non-empty next_page_token on page 2")
	}

	// 3. Verify no overlapping items between page 1 and page 2
	seen := make(map[string]bool)
	for _, m := range res1.Msg.Motors {
		seen[m.Id] = true
	}
	for _, m := range res2.Msg.Motors {
		if seen[m.Id] {
			t.Errorf("duplicate motor %s found across page 1 and page 2", m.Id)
		}
	}
}

func TestPagination_FullIteration(t *testing.T) {
	srv, pool := setupTestServer(t)
	defer srv.Close()
	defer pool.Close()

	client := quadsmithconnect.NewMotorServiceClient(http.DefaultClient, srv.URL)
	ctx := context.Background()

	// Iterate through all pages with page_size=25
	var allIds []string
	seen := make(map[string]bool)
	pageToken := ""
	pageCount := 0

	for {
		pageCount++
		res, err := client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
			PageSize:  25,
			PageToken: pageToken,
		}))
		if err != nil {
			t.Fatalf("failed on page %d: %v", pageCount, err)
		}

		for _, m := range res.Msg.Motors {
			if seen[m.Id] {
				t.Fatalf("duplicate motor %s on page %d", m.Id, pageCount)
			}
			seen[m.Id] = true
			allIds = append(allIds, m.Id)
		}

		if res.Msg.NextPageToken == "" {
			// Last page reached
			break
		}
		pageToken = res.Msg.NextPageToken
	}

	// Verify against total seeded motors
	expectedMotors := 143
	seedPath := resolveSeedPath(t)
	if b, err := os.ReadFile(seedPath); err == nil {
		var seedResp pb.ListMotorsResponse
		if err := prototext.Unmarshal(b, &seedResp); err == nil && len(seedResp.Motors) > 0 {
			expectedMotors = len(seedResp.Motors)
		}
	}
	if len(allIds) != expectedMotors {
		t.Fatalf("expected %d total motors across all pages, got %d", expectedMotors, len(allIds))
	}
}

func TestPagination_WithSorting(t *testing.T) {
	srv, pool := setupTestServer(t)
	defer srv.Close()
	defer pool.Close()

	client := quadsmithconnect.NewMotorServiceClient(http.DefaultClient, srv.URL)
	ctx := context.Background()

	// Sort descending by KV (^kv)
	res1, err := client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		PageSize: 10,
		Sort:     []string{"^kv"},
	}))
	if err != nil {
		t.Fatalf("page 1 failed: %v", err)
	}

	res2, err := client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		PageSize:  10,
		PageToken: res1.Msg.NextPageToken,
		Sort:      []string{"^kv"},
	}))
	if err != nil {
		t.Fatalf("page 2 failed: %v", err)
	}

	all := append(res1.Msg.Motors, res2.Msg.Motors...)
	for i := 0; i < len(all)-1; i++ {
		if all[i].Kv < all[i+1].Kv {
			t.Fatalf("sorting violated at index %d: kv %d < kv %d (expected descending)", i, all[i].Kv, all[i+1].Kv)
		}
	}
}

func TestPagination_WithFiltering(t *testing.T) {
	srv, pool := setupTestServer(t)
	defer srv.Close()
	defer pool.Close()

	client := quadsmithconnect.NewMotorServiceClient(http.DefaultClient, srv.URL)
	ctx := context.Background()

	filter := `manufacturer == "T-Motor"`
	pageSize := int32(5)

	var totalFiltered []string
	pageToken := ""

	for {
		res, err := client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
			Filter:    filter,
			PageSize:  pageSize,
			PageToken: pageToken,
		}))
		if err != nil {
			t.Fatalf("list motors failed: %v", err)
		}

		for _, m := range res.Msg.Motors {
			if m.Manufacturer != "T-Motor" {
				t.Fatalf("filter violated: got manufacturer %q, expected 'T-Motor'", m.Manufacturer)
			}
			totalFiltered = append(totalFiltered, m.Id)
		}

		if res.Msg.NextPageToken == "" {
			break
		}
		pageToken = res.Msg.NextPageToken
	}

	if len(totalFiltered) == 0 {
		t.Fatalf("expected to find T-Motor motors, found 0")
	}
}

func TestPagination_WithFilteringAndSorting(t *testing.T) {
	srv, pool := setupTestServer(t)
	defer srv.Close()
	defer pool.Close()

	client := quadsmithconnect.NewMotorServiceClient(http.DefaultClient, srv.URL)
	ctx := context.Background()

	filter := `kv >= 2000`
	sort := []string{"^kv"}

	res1, err := client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		Filter:   filter,
		Sort:     sort,
		PageSize: 5,
	}))
	if err != nil {
		t.Fatalf("failed page 1: %v", err)
	}

	res2, err := client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		Filter:    filter,
		Sort:      sort,
		PageSize:  5,
		PageToken: res1.Msg.NextPageToken,
	}))
	if err != nil {
		t.Fatalf("failed page 2: %v", err)
	}

	combined := append(res1.Msg.Motors, res2.Msg.Motors...)
	for i, m := range combined {
		if m.Kv < 2000 {
			t.Fatalf("motor %s has kv %d, expected >= 2000", m.Id, m.Kv)
		}
		if i > 0 && combined[i-1].Kv < m.Kv {
			t.Fatalf("ordering violated at %d: %d < %d", i, combined[i-1].Kv, m.Kv)
		}
	}
}

func TestPagination_TokenPreservesFilterAndSort(t *testing.T) {
	srv, pool := setupTestServer(t)
	defer srv.Close()
	defer pool.Close()

	client := quadsmithconnect.NewMotorServiceClient(http.DefaultClient, srv.URL)
	ctx := context.Background()

	filter := `manufacturer == "BETAFPV"`
	sort := []string{"^kv"}

	res1, err := client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		Filter:   filter,
		Sort:     sort,
		PageSize: 5,
	}))
	if err != nil {
		t.Fatalf("failed page 1: %v", err)
	}
	if res1.Msg.NextPageToken == "" {
		t.Fatalf("expected non-empty next_page_token on page 1")
	}

	// Request page 2 WITHOUT passing filter or sort explicitly — token should preserve them
	res2, err := client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		PageSize:  5,
		PageToken: res1.Msg.NextPageToken,
	}))
	if err != nil {
		t.Fatalf("failed page 2 with preserved token params: %v", err)
	}

	if len(res2.Msg.Motors) != 5 {
		t.Fatalf("expected 5 motors on page 2, got %d", len(res2.Msg.Motors))
	}
	for _, m := range res2.Msg.Motors {
		if m.Manufacturer != "BETAFPV" {
			t.Fatalf("expected manufacturer 'BETAFPV', got %q", m.Manufacturer)
		}
	}
}

func TestPagination_ValidationErrors(t *testing.T) {
	srv, pool := setupTestServer(t)
	defer srv.Close()
	defer pool.Close()

	client := quadsmithconnect.NewMotorServiceClient(http.DefaultClient, srv.URL)
	ctx := context.Background()

	// 1. Negative page_size
	_, err := client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		PageSize: -1,
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument for negative page_size, got %v", err)
	}

	// 2. Malformed page_token
	_, err = client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		PageToken: "not-a-valid-base64-token",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument for malformed page_token, got %v", err)
	}

	// 3. Invalid sort column
	_, err = client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		Sort: []string{"non_existent_column"},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument for invalid sort column, got %v", err)
	}

	// 4. Mismatched filter with page_token
	res1, err := client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		Filter:   `manufacturer == "BETAFPV"`,
		PageSize: 5,
	}))
	if err != nil {
		t.Fatalf("failed page 1: %v", err)
	}
	if res1.Msg.NextPageToken == "" {
		t.Fatalf("expected non-empty next_page_token")
	}

	_, err = client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		Filter:    `manufacturer == "HGLRC"`,
		PageToken: res1.Msg.NextPageToken,
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument for mismatched filter, got %v", err)
	}

	// 5. Mismatched sort with page_token
	resSort, err := client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		Sort:     []string{"^kv"},
		PageSize: 5,
	}))
	if err != nil {
		t.Fatalf("failed page 1 with sort: %v", err)
	}

	_, err = client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{
		Sort:      []string{"kv"},
		PageToken: resSort.Msg.NextPageToken,
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument for mismatched sort, got %v", err)
	}
}

func TestPagination_DefaultPageSize(t *testing.T) {
	srv, pool := setupTestServer(t)
	defer srv.Close()
	defer pool.Close()

	client := quadsmithconnect.NewMotorServiceClient(http.DefaultClient, srv.URL)
	ctx := context.Background()

	// Default page size (unspecified = 0) should return 20 items
	res, err := client.ListMotors(ctx, connect.NewRequest(&pb.ListMotorsRequest{}))
	if err != nil {
		t.Fatalf("failed list motors: %v", err)
	}

	if len(res.Msg.Motors) != 20 {
		t.Fatalf("expected default 20 motors, got %d", len(res.Msg.Motors))
	}
	if res.Msg.NextPageToken == "" {
		t.Fatalf("expected next_page_token for default pagination, got empty")
	}
}

func TestCLI_Pagination(t *testing.T) {
	srv, pool := setupTestServer(t)
	defer srv.Close()
	defer pool.Close()

	qsPath := resolveQSPath(t)

	// 1. List with --limit, --filter, --sort
	cmd := exec.Command(qsPath, "components", "hardware", "motors", "list", "--json", "--limit", "5", "--filter", `manufacturer == "BETAFPV"`, "--sort", "^kv")
	cmd.Env = append(cmd.Env, "QS_API_URL="+srv.URL)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("CLI command failed: %v\nStderr: %s", err, stderr.String())
	}

	var motors []map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &motors); err != nil {
		t.Fatalf("Failed to parse CLI JSON output: %v\nOutput: %s", err, stdout.String())
	}

	if len(motors) != 5 {
		t.Fatalf("expected 5 motors from CLI, got %d", len(motors))
	}

	for _, m := range motors {
		if m["manufacturer"] != "BETAFPV" {
			t.Fatalf("expected BETAFPV manufacturer, got %v", m["manufacturer"])
		}
	}
}
