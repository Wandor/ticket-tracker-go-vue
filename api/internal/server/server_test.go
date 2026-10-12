package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tickettracker/internal/database"
	"tickettracker/internal/server"
)

const defaultTestURL = "postgres://tickets:tickets@localhost:55432/tickets_test?sslmode=disable"

var (
	pool    *pgxpool.Pool
	handler http.Handler
)

// TestMain creates the test database if needed, migrates it and shares one pool across tests.
func TestMain(m *testing.M) {
	ctx := context.Background()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultTestURL
	}
	if err := ensureDatabase(ctx, dbURL); err != nil {
		fmt.Fprintf(os.Stderr, "test database unavailable (%v); start it with `docker compose up -d db`\n", err)
		os.Exit(1)
	}

	var err error
	pool, err = database.Connect(ctx, dbURL)
	if err == nil {
		err = database.Migrate(ctx, pool)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	handler = server.New(pool)

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func ensureDatabase(ctx context.Context, dbURL string) error {
	u, err := url.Parse(dbURL)
	if err != nil {
		return err
	}
	name := u.Path[1:]
	u.Path = "/postgres"
	conn, err := pgx.Connect(ctx, u.String())
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	var exists bool
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		_, err = conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	}
	return err
}

func resetTickets(t *testing.T) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), "TRUNCATE tickets RESTART IDENTITY"); err != nil {
		t.Fatal(err)
	}
}

// do sends a request through the full handler stack and decodes the JSON response.
func do(t *testing.T, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(method, path, &buf))

	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s %s: invalid JSON %q", method, path, rec.Body.String())
	}
	return rec.Code, out
}

func create(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	code, ticket := do(t, http.MethodPost, "/tickets", body)
	if code != http.StatusCreated {
		t.Fatalf("create: status %d, body %v", code, ticket)
	}
	return ticket
}

func fieldErrors(resp map[string]any) map[string]any {
	errObj, _ := resp["error"].(map[string]any)
	fields, _ := errObj["fields"].(map[string]any)
	return fields
}

func TestCreateTicket(t *testing.T) {
	resetTickets(t)

	t.Run("applies defaults and trims input", func(t *testing.T) {
		ticket := create(t, map[string]any{"title": "  VPN drops every hour  ", "description": ""})

		want := map[string]any{"title": "VPN drops every hour", "description": nil, "status": "open", "priority": "medium"}
		for k, v := range want {
			if ticket[k] != v {
				t.Errorf("%s = %v, want %v", k, ticket[k], v)
			}
		}
		if ticket["id"] == nil || ticket["created_at"] == nil {
			t.Errorf("missing id or created_at: %v", ticket)
		}
	})

	t.Run("rejects invalid input without writing", func(t *testing.T) {
		code, resp := do(t, http.MethodPost, "/tickets", map[string]any{"title": "   ", "priority": "urgent"})
		if code != http.StatusBadRequest {
			t.Fatalf("status %d, want 400", code)
		}
		fields := fieldErrors(resp)
		if fields["title"] != "is required" || fields["priority"] == nil {
			t.Errorf("unexpected field errors: %v", fields)
		}

		var n int
		_ = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM tickets").Scan(&n)
		if n != 1 {
			t.Errorf("tickets in db = %d, want 1", n)
		}
	})
}

func TestListTicketsFiltersAndPaginates(t *testing.T) {
	resetTickets(t)
	for _, tc := range []struct{ title, status string }{{"a", "open"}, {"b", "closed"}, {"c", "open"}, {"d", "open"}} {
		create(t, map[string]any{"title": tc.title, "status": tc.status})
	}

	titles := func(resp map[string]any) []any {
		var out []any
		for _, item := range resp["data"].([]any) {
			out = append(out, item.(map[string]any)["title"])
		}
		return out
	}

	code, page1 := do(t, http.MethodGet, "/tickets?status=open&limit=2&page=1", nil)
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if got := fmt.Sprint(titles(page1)); got != "[d c]" {
		t.Errorf("page 1 titles = %s, want [d c]", got)
	}
	wantPagination := map[string]any{"page": 1.0, "limit": 2.0, "total": 3.0, "total_pages": 2.0}
	if fmt.Sprint(page1["pagination"]) != fmt.Sprint(wantPagination) {
		t.Errorf("pagination = %v, want %v", page1["pagination"], wantPagination)
	}

	_, page2 := do(t, http.MethodGet, "/tickets?status=open&limit=2&page=2", nil)
	if got := fmt.Sprint(titles(page2)); got != "[a]" {
		t.Errorf("page 2 titles = %s, want [a]", got)
	}

	code, resp := do(t, http.MethodGet, "/tickets?status=pending", nil)
	if code != http.StatusBadRequest || fieldErrors(resp)["status"] == nil {
		t.Errorf("unknown status: got %d %v, want 400 with status error", code, resp)
	}
}

func TestUpdateTicket(t *testing.T) {
	resetTickets(t)
	ticket := create(t, map[string]any{"title": "Reset password", "priority": "high"})
	path := fmt.Sprintf("/tickets/%v", ticket["id"])

	code, updated := do(t, http.MethodPatch, path, map[string]any{"status": "in_progress"})
	if code != http.StatusOK || updated["status"] != "in_progress" || updated["priority"] != "high" {
		t.Errorf("partial update: got %d %v", code, updated)
	}

	if code, _ := do(t, http.MethodPatch, path, map[string]any{"title": "Renamed"}); code != http.StatusBadRequest {
		t.Errorf("updating title: status %d, want 400", code)
	}
	if code, _ := do(t, http.MethodPatch, "/tickets/9999", map[string]any{"status": "closed"}); code != http.StatusNotFound {
		t.Errorf("unknown id: status %d, want 404", code)
	}
}

func TestStatsGroupsByStatus(t *testing.T) {
	resetTickets(t)

	_, empty := do(t, http.MethodGet, "/tickets/stats", nil)
	if fmt.Sprint(empty) != "map[closed:0 in_progress:0 open:0]" {
		t.Errorf("empty stats = %v", empty)
	}

	create(t, map[string]any{"title": "one"})
	create(t, map[string]any{"title": "two"})
	third := create(t, map[string]any{"title": "three"})
	do(t, http.MethodPatch, fmt.Sprintf("/tickets/%v", third["id"]), map[string]any{"status": "closed"})

	code, stats := do(t, http.MethodGet, "/tickets/stats", nil)
	if code != http.StatusOK || fmt.Sprint(stats) != "map[closed:1 in_progress:0 open:2]" {
		t.Errorf("stats = %d %v", code, stats)
	}
}
