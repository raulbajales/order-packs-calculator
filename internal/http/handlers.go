package http

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	nethttp "net/http"
	"strconv"
	"strings"

	"challenge/internal/repository"
	"challenge/internal/service"
)

// Holds dependencies shared across all HTTP handlers
type Handlers struct {
	repo  *repository.PackRepository
	webFS fs.FS
}

func NewHandlers(repo *repository.PackRepository, webFS fs.FS) *Handlers {
	return &Handlers{repo: repo, webFS: webFS}
}

// Serves the main application page as a static HTML file.
func (h *Handlers) Index(w nethttp.ResponseWriter, r *nethttp.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	nethttp.FileServer(nethttp.FS(h.webFS)).ServeHTTP(w, r)
}

// ListPacksAPI handles GET /api/packs.
// It returns the current pack sizes stored in the database.
//
// Response (200):
//
//	{"sizes":[5000,2000,1000,500,250]}
//
// Error responses:
//   - 500 {"error":"..."}  — database error
func (h *Handlers) ListPacksAPI(w nethttp.ResponseWriter, r *nethttp.Request) {
	w.Header().Set("Content-Type", "application/json")

	sizes, err := h.repo.ListSizes(r.Context())
	if err != nil {
		slog.Error("list pack sizes", "error", err)
		w.WriteHeader(nethttp.StatusInternalServerError)
		w.Write([]byte(`{"error":"failed to fetch pack sizes"}`))
		return
	}

	// Build the response JSON
	var buf strings.Builder
	buf.WriteString(`{"sizes":[`)
	for i, s := range sizes {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(strconv.Itoa(s))
	}
	buf.WriteString(`]}`)

	w.WriteHeader(nethttp.StatusOK)
	w.Write([]byte(buf.String()))
}

// UpdatePacksAPI handles POST /api/packs.
// Replaces all stored pack sizes with those provided in the JSON request body.
//
// Request body (application/json):
//
//	{"sizes":"250,500,1000"}
//
// Responses:
//   - 200 {"ok":true}       — sizes saved successfully
//   - 400 {"error":"..."}   — invalid JSON, or validation failure
//   - 500 {"error":"..."}   — database error
func (h *Handlers) UpdatePacksAPI(w nethttp.ResponseWriter, r *nethttp.Request) {
	w.Header().Set("Content-Type", "application/json")

	var body struct {
		Sizes string `json:"sizes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(nethttp.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid JSON body"}`))
		return
	}

	sizes, err := parseSizes(body.Sizes)
	if err != nil {
		w.WriteHeader(nethttp.StatusBadRequest)
		fmt.Fprintf(w, `{"error":%q}`, err.Error())
		return
	}

	// Replace the pack sizes in the database
	if err := h.repo.Replace(r.Context(), sizes); err != nil {
		slog.Error("replace packs via API", "error", err)
		w.WriteHeader(nethttp.StatusInternalServerError)
		w.Write([]byte(`{"error":"failed to save pack sizes"}`))
		return
	}

	w.WriteHeader(nethttp.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}

// CalculateAPI handles GET /api/calculate?amount=N.
// It reads pack sizes from the database and returns the optimal distribution
// for the requested order quantity.
//
// Query parameters:
//   - amount — positive integer order quantity (required)
//
// Response (200):
//
//	{"53":9429,"31":7,"23":2}
//
// Keys are pack sizes (as strings); values are the quantity of each pack.
// Error responses:
//   - 400 {"error":"..."}   — invalid or missing amount
//   - 422 {"error":"..."}   — no valid pack combination found
//   - 500 {"error":"..."}   — database error
func (h *Handlers) CalculateAPI(w nethttp.ResponseWriter, r *nethttp.Request) {
	w.Header().Set("Content-Type", "application/json")

	amountStr := r.URL.Query().Get("amount")
	amount, err := strconv.Atoi(amountStr)
	if err != nil || amount <= 0 {
		w.WriteHeader(nethttp.StatusBadRequest)
		w.Write([]byte(`{"error":"amount must be a positive integer"}`))
		return
	}

	sizes, err := h.repo.ListSizes(r.Context())
	if err != nil {
		slog.Error("list pack sizes", "error", err)
		w.WriteHeader(nethttp.StatusInternalServerError)
		w.Write([]byte(`{"error":"failed to fetch pack sizes"}`))
		return
	}

	results, err := service.Calculate(amount, sizes)
	if err != nil {
		w.WriteHeader(nethttp.StatusUnprocessableEntity)
		fmt.Fprintf(w, `{"error":%q}`, err.Error())
		return
	}

	var buf strings.Builder
	buf.WriteByte('{')
	for i, pr := range results {
		if i > 0 {
			buf.WriteByte(',')
		}
		fmt.Fprintf(&buf, "%q:%d", strconv.Itoa(pr.Size), pr.Quantity)
	}
	buf.WriteByte('}')

	w.WriteHeader(nethttp.StatusOK)
	w.Write([]byte(buf.String()))
}

// Health handles GET /api/health.
// It returns a static JSON payload confirming the server is reachable.
func (h *Handlers) Health(w nethttp.ResponseWriter, r *nethttp.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(nethttp.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// parseSizes parses a comma-separated string of positive integers.
// It returns an error if any token is non-numeric, non-positive, or if the
// input is empty or contains only whitespace.
func parseSizes(raw string) ([]int, error) {
	var sizes []int
	for _, part := range strings.Split(raw, ",") {
		s := strings.TrimSpace(part)
		if s == "" {
			continue
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("invalid size %q: must be a whole number", s)
		}
		if n <= 0 {
			return nil, fmt.Errorf("pack size must be positive, got %d", n)
		}
		sizes = append(sizes, n)
	}
	if len(sizes) == 0 {
		return nil, fmt.Errorf("at least one pack size is required")
	}
	return sizes, nil
}
