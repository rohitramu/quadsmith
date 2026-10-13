package web

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func getWebURL() string {
	if u := os.Getenv("QS_WEB_URL"); u != "" {
		return u
	}
	return "http://127.0.0.1:3001"
}

func TestWeb_Healthz(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(getWebURL() + "/healthz")
	if err != nil {
		t.Fatalf("failed to query web healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from web healthz, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", string(body))
	}
}

func TestWeb_ConfigEndpoint(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(getWebURL() + "/api/v1/config")
	if err != nil {
		t.Fatalf("failed to query web config endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from web config, got %d", resp.StatusCode)
	}

	var data map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode web config json: %v", err)
	}

	if data["environment"] != "sandbox" {
		t.Errorf("expected environment sandbox, got %q", data["environment"])
	}
	if data["apiUrl"] == "" {
		t.Error("expected non-empty apiUrl")
	}
}

func TestWeb_SPAInjection(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(getWebURL() + "/login")
	if err != nil {
		t.Fatalf("failed to get /login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from /login, got %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	body := string(bodyBytes)

	if !strings.Contains(body, "window.__QUADSMITH_CONFIG__") {
		t.Errorf("expected window.__QUADSMITH_CONFIG__ in index.html, got: %s", body)
	}
}
