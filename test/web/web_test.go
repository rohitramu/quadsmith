package web

import (
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
	return "http://127.0.0.1:8080"
}

func getOpsWebURL() string {
	if u := os.Getenv("QS_OPS_WEB_URL"); u != "" {
		return u
	}
	return "http://127.0.0.1:8083"
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
	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	if !strings.Contains(body, `"status":"HEALTHY"`) || !strings.Contains(body, `"service":"server-web"`) {
		t.Errorf("expected structured HEALTHY JSON for server-web, got %q", body)
	}
}

func TestOpsWeb_HealthzAndSPA(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(getOpsWebURL() + "/healthz")
	if err != nil {
		t.Fatalf("failed to query ops-web healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from ops-web healthz, got %d", resp.StatusCode)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	if !strings.Contains(body, `"status":"HEALTHY"`) || !strings.Contains(body, `"service":"server-ops-web"`) {
		t.Errorf("expected structured HEALTHY JSON for server-ops-web, got %q", body)
	}

	spaResp, err := client.Get(getOpsWebURL() + "/")
	if err != nil {
		t.Fatalf("failed to get ops-web /: %v", err)
	}
	defer spaResp.Body.Close()
	if spaResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from ops-web /, got %d", spaResp.StatusCode)
	}
	spaBody, _ := io.ReadAll(spaResp.Body)
	if !strings.Contains(string(spaBody), "<!doctype html>") && !strings.Contains(string(spaBody), "Quadsmith") {
		t.Errorf("expected Ops Console SPA index.html in response, got: %s", string(spaBody))
	}
}

func TestWeb_SPAFallback(t *testing.T) {
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

	if !strings.Contains(body, "Quadsmith") && !strings.Contains(body, "<!doctype html>") {
		t.Errorf("expected SPA index.html in response, got: %s", body)
	}
}
