package tickets

import (
	"errors"
	"net/http"
	"strconv"

	"tickettracker/internal/httpx"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// Register mounts the ticket routes. ServeMux prefers the literal /tickets/stats over
// /tickets/{id}, so the two never collide.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /tickets", httpx.Handle(h.create))
	mux.HandleFunc("GET /tickets", httpx.Handle(h.list))
	mux.HandleFunc("GET /tickets/stats", httpx.Handle(h.stats))
	mux.HandleFunc("PATCH /tickets/{id}", httpx.Handle(h.update))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	var req CreateRequest
	if err := httpx.DecodeJSON(r, &req, false); err != nil {
		return err
	}
	ticket, errs := req.Validate()
	if len(errs) > 0 {
		return httpx.ValidationError(errs)
	}
	created, err := h.store.Create(r.Context(), ticket)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, created)
	return nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	params, errs := parseListParams(r)
	if len(errs) > 0 {
		return httpx.ValidationError(errs)
	}
	tickets, total, err := h.store.List(r.Context(), params)
	if err != nil {
		return err
	}
	// Encode an empty page as [] rather than null.
	if tickets == nil {
		tickets = []Ticket{}
	}
	httpx.WriteJSON(w, http.StatusOK, ListResponse{
		Data: tickets,
		Pagination: Pagination{
			Page:       params.Page,
			Limit:      params.Limit,
			Total:      total,
			TotalPages: (total + params.Limit - 1) / params.Limit,
		},
	})
	return nil
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return httpx.NotFound("Ticket not found")
	}
	var req UpdateRequest
	if err := httpx.DecodeJSON(r, &req, true); err != nil {
		return err
	}
	if errs := req.Validate(); len(errs) > 0 {
		return httpx.ValidationError(errs)
	}
	ticket, err := h.store.Update(r.Context(), id, req)
	if errors.Is(err, ErrNotFound) {
		return httpx.NotFound("Ticket not found")
	}
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, ticket)
	return nil
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) error {
	counts, err := h.store.CountByStatus(r.Context())
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, counts)
	return nil
}

func parseListParams(r *http.Request) (ListParams, map[string]string) {
	q := r.URL.Query()
	errs := map[string]string{}
	p := ListParams{Page: 1, Limit: DefaultLimit}

	if raw := q.Get("status"); raw != "" {
		st := Status(raw)
		if st.Valid() {
			p.Status = &st
		} else {
			errs["status"] = statusMessage
		}
	}
	if raw := q.Get("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			errs["page"] = "must be a positive integer"
		}
		p.Page = n
	}
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > MaxLimit {
			errs["limit"] = "must be an integer between 1 and 100"
		}
		p.Limit = n
	}
	return p, errs
}
