//go:build integration

package http_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	appdb "challenge/internal/db"
	apphttp "challenge/internal/http"
	"challenge/internal/repository"
)

func setupServer(t *testing.T) *httptest.Server {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	goose.SetBaseFS(appdb.MigrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		t.Fatalf("goose up: %v", err)
	}

	webFS := os.DirFS("../../web")

	repo := repository.New(db)
	handlers := apphttp.NewHandlers(repo, webFS)

	return httptest.NewServer(apphttp.NewRouter(handlers, webFS))
}

// Calls POST /api/packs with a JSON body and returns the response
func postPacks(t *testing.T, srv *httptest.Server, sizes string) *http.Response {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"sizes": sizes})
	resp, err := http.Post(srv.URL+"/api/packs", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("POST /api/packs: %v", err)
	}
	return resp
}

// Sets known pack sizes via the API
func seedPacks(t *testing.T, srv *httptest.Server, sizes string) {
	t.Helper()
	resp := postPacks(t, srv, sizes)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("seed /api/packs status %d: %s", resp.StatusCode, body)
	}
}

// Tests GET / returns the main page
func TestGetIndex(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Order Packs Calculator") {
		t.Error("response body missing expected page title")
	}
}

// Tests GET /api/health returns the health check
func TestGetHealth(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"status":"ok"`) {
		t.Errorf("unexpected health response: %s", body)
	}
}

// Tests GET /api/packs returns the current pack sizes
func TestAPIListPacks(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	seedPacks(t, srv, "100, 200, 500")

	resp, err := http.Get(srv.URL + "/api/packs")
	if err != nil {
		t.Fatalf("GET /api/packs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body: %s", resp.StatusCode, body)
	}

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body struct {
		Sizes []int `json:"sizes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}

	want := map[int]bool{100: true, 200: true, 500: true}
	if len(body.Sizes) != len(want) {
		t.Fatalf("got %d sizes, want %d: %v", len(body.Sizes), len(want), body.Sizes)
	}
	for _, s := range body.Sizes {
		if !want[s] {
			t.Errorf("unexpected size %d in response", s)
		}
	}

	// Verify descending order.
	for i := 1; i < len(body.Sizes); i++ {
		if body.Sizes[i] > body.Sizes[i-1] {
			t.Errorf("sizes not in descending order: %v", body.Sizes)
			break
		}
	}
}

// Tests GET /api/calculate with known pack sizes
func TestAPICalculate(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	seedPacks(t, srv, "250, 500, 1000, 2000, 5000")

	tests := []struct {
		name         string
		amount       string
		wantMinTotal int
	}{
		{"exact fit", "250", 250},
		{"one over boundary", "251", 500},
		{"large order", "12001", 12250},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(srv.URL + "/api/calculate?amount=" + tt.amount)
			if err != nil {
				t.Fatalf("GET /api/calculate: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("status = %d, body: %s", resp.StatusCode, body)
			}

			if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			var result map[string]int
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("decode JSON: %v", err)
			}

			total := 0
			for sizeStr, qty := range result {
				var size int
				if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
					t.Errorf("non-integer key %q in result: %v", sizeStr, err)
					continue
				}
				total += size * qty
			}

			if total != tt.wantMinTotal {
				t.Errorf("total shipped = %d, want %d; result: %v", total, tt.wantMinTotal, result)
			}
		})
	}
}

// Tests GET /api/calculate with bad input
func TestAPICalculateInvalidAmount(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	tests := []struct{ name, amount string }{
		{"non-numeric", "abc"},
		{"zero", "0"},
		{"negative", "-5"},
		{"missing", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(srv.URL + "/api/calculate?amount=" + tt.amount)
			if err != nil {
				t.Fatalf("GET /api/calculate: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
			}

			var body map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode JSON: %v", err)
			}
			if body["error"] == "" {
				t.Errorf("expected non-empty error field in response")
			}
		})
	}
}

// Tests POST /api/packs with valid sizes
func TestAPIPacksSave(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	resp := postPacks(t, srv, "100, 200, 500")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body: %s", resp.StatusCode, body)
	}

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if body["ok"] != true {
		t.Errorf("expected {\"ok\":true}, got %v", body)
	}
}

// Tests POST /api/packs with invalid input
func TestAPIPacksInvalidSizes(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	tests := []struct {
		name       string
		sizes      string
		wantErrMsg string
	}{
		{"non-numeric token", "250, abc, 1000", "invalid size"},
		{"negative size", "250, -5, 1000", "must be positive"},
		{"empty input", "", "at least one"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := postPacks(t, srv, tt.sizes)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("status = %d, want 400; body: %s", resp.StatusCode, body)
			}

			var body map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode JSON: %v", err)
			}
			if !strings.Contains(body["error"], tt.wantErrMsg) {
				t.Errorf("error = %q, want it to contain %q", body["error"], tt.wantErrMsg)
			}
		})
	}
}
