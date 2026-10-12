// Package server wires handlers into one http.Handler, shared by main and the tests.
package server

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"tickettracker/internal/httpx"
	"tickettracker/internal/tickets"
)

func New(pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	tickets.NewHandler(tickets.NewStore(pool)).Register(mux)

	// Unmatched paths get the same JSON error shape as everything else.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteError(w, httpx.NotFound("No route for "+r.Method+" "+r.URL.Path))
	})
	return mux
}
