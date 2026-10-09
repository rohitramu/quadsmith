package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
)

func TestSearch_GlobalSearch(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	searchClient := quadsmithconnect.NewSearchServiceClient(http.DefaultClient, apiUrl)
	ctx := context.Background()

	// 1. Search without selectors (global search across all collections)
	res, err := searchClient.Search(ctx, connect.NewRequest(&pb.SearchRequest{
		Query: "tattu",
		Limit: 10,
	}))
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(res.Msg.Results) == 0 {
		t.Fatal("expected results for query 'tattu', got 0")
	}

	t.Logf("Got %d results for query 'tattu'", len(res.Msg.Results))
	for _, item := range res.Msg.Results {
		if !strings.Contains(strings.ToLower(item.Name), "tattu") &&
			!strings.Contains(strings.ToLower(item.Id), "tattu") &&
			!strings.Contains(strings.ToLower(item.Metadata["manufacturer"]), "tattu") {
			t.Errorf("result %s (%s) does not seem to match 'tattu'", item.Name, item.Id)
		}
		if item.MatchScore <= 0 {
			t.Errorf("expected positive match score, got %f", item.MatchScore)
		}
	}
}

func TestSearch_SingleSelectorWithCELFilter(t *testing.T) {
	apiUrl := getAPIURL()
	searchClient := quadsmithconnect.NewSearchServiceClient(http.DefaultClient, apiUrl)
	ctx := context.Background()

	filter := "cell_count_s == 6"
	res, err := searchClient.Search(ctx, connect.NewRequest(&pb.SearchRequest{
		Query: "tattu",
		Selectors: []*pb.SearchSelector{
			{
				Path:   "components/hardware/batteries",
				Filter: &filter,
			},
		},
		Limit: 10,
	}))
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(res.Msg.Results) == 0 {
		t.Fatal("expected results for 6S tattu batteries, got 0")
	}

	for _, item := range res.Msg.Results {
		if item.Path != "components/hardware/batteries" {
			t.Errorf("expected path components/hardware/batteries, got %s", item.Path)
		}
		if cellCount, ok := item.Metadata["cell_count_s"]; ok {
			if cellCount != "6" {
				t.Errorf("expected cell_count_s == 6, got %s for item %s", cellCount, item.Name)
			}
		}
	}
}

func TestSearch_DuplicatePath_FilterMerging(t *testing.T) {
	apiUrl := getAPIURL()
	searchClient := quadsmithconnect.NewSearchServiceClient(http.DefaultClient, apiUrl)
	ctx := context.Background()

	filter1 := "cell_count_s == 6 || cell_count_s == 4"
	filter2 := "capacity_mah >= 1000"

	// Two selectors targeting the same collection (canonical path and alias)
	res, err := searchClient.Search(ctx, connect.NewRequest(&pb.SearchRequest{
		Query: "tattu",
		Selectors: []*pb.SearchSelector{
			{
				Path:   "components/hardware/batteries",
				Filter: &filter1,
			},
			{
				Path:   "batteries", // Alias
				Filter: &filter2,
			},
		},
		Limit: 10,
	}))
	if err != nil {
		t.Fatalf("Search with duplicate paths failed: %v", err)
	}

	if len(res.Msg.Results) == 0 {
		t.Fatal("expected results for merged filter, got 0")
	}

	seenUUIDs := make(map[string]bool)
	for _, item := range res.Msg.Results {
		if seenUUIDs[item.Uuid] {
			t.Errorf("duplicate item returned in search: %s", item.Id)
		}
		seenUUIDs[item.Uuid] = true
	}
}

func TestSearch_FuzzyTypoTolerance(t *testing.T) {
	apiUrl := getAPIURL()
	searchClient := quadsmithconnect.NewSearchServiceClient(http.DefaultClient, apiUrl)
	ctx := context.Background()

	// "tatu" is a typo for "tattu"
	res, err := searchClient.Search(ctx, connect.NewRequest(&pb.SearchRequest{
		Query: "tatu",
		Limit: 5,
	}))
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	foundTattu := false
	for _, item := range res.Msg.Results {
		if strings.Contains(strings.ToLower(item.Name), "tattu") ||
			strings.Contains(strings.ToLower(item.Id), "tattu") {
			foundTattu = true
			break
		}
	}

	if !foundTattu {
		t.Errorf("expected fuzzy search for 'tatu' to match 'Tattu' items, got results: %+v", res.Msg.Results)
	}
}

func TestSearch_AllFieldSearch(t *testing.T) {
	apiUrl := getAPIURL()
	searchClient := quadsmithconnect.NewSearchServiceClient(http.DefaultClient, apiUrl)
	ctx := context.Background()

	// Search for connector specification "XT60"
	res, err := searchClient.Search(ctx, connect.NewRequest(&pb.SearchRequest{
		Query: "XT60",
		Limit: 5,
	}))
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(res.Msg.Results) == 0 {
		t.Fatal("expected results for spec query 'XT60', got 0")
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	apiUrl := getAPIURL()
	searchClient := quadsmithconnect.NewSearchServiceClient(http.DefaultClient, apiUrl)
	ctx := context.Background()

	// Empty query targeting batteries returns initial items for combobox
	res, err := searchClient.Search(ctx, connect.NewRequest(&pb.SearchRequest{
		Query: "",
		Selectors: []*pb.SearchSelector{
			{Path: "components/hardware/batteries"},
		},
		Limit: 5,
	}))
	if err != nil {
		t.Fatalf("Empty query search failed: %v", err)
	}

	if len(res.Msg.Results) == 0 {
		t.Fatal("expected items returned on empty query, got 0")
	}

	for _, item := range res.Msg.Results {
		if item.MatchScore != 1.0 {
			t.Errorf("expected match score 1.0 for empty query, got %f", item.MatchScore)
		}
	}
}

func TestSearch_UnknownPath_ReturnsInvalidArgument(t *testing.T) {
	apiUrl := getAPIURL()
	searchClient := quadsmithconnect.NewSearchServiceClient(http.DefaultClient, apiUrl)
	ctx := context.Background()

	_, err := searchClient.Search(ctx, connect.NewRequest(&pb.SearchRequest{
		Query: "test",
		Selectors: []*pb.SearchSelector{
			{Path: "nonexistent/collection"},
		},
	}))
	if err == nil {
		t.Fatal("expected error for nonexistent collection path, got nil")
	}

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", connect.CodeOf(err))
	}
}
