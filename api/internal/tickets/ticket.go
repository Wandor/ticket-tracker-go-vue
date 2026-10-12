// Package tickets implements the ticket resource: validation, persistence and HTTP handlers.
package tickets

import (
	"strings"
	"time"
	"unicode/utf8"
)

type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusClosed     Status = "closed"
)

// Statuses lists every status in workflow order; stats responses include each one.
var Statuses = []Status{StatusOpen, StatusInProgress, StatusClosed}

func (s Status) Valid() bool {
	return s == StatusOpen || s == StatusInProgress || s == StatusClosed
}

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

func (p Priority) Valid() bool {
	return p == PriorityLow || p == PriorityMedium || p == PriorityHigh
}

const (
	TitleMaxLength = 200
	DefaultLimit   = 20
	MaxLimit       = 100

	statusMessage   = "must be one of open, in_progress, closed"
	priorityMessage = "must be one of low, medium, high"
)

type Ticket struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Status      Status    `json:"status"`
	Priority    Priority  `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateRequest is the POST body. Pointers distinguish "omitted" from "empty".
type CreateRequest struct {
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	Status      *Status   `json:"status"`
	Priority    *Priority `json:"priority"`
}

// NewTicket is a validated CreateRequest with defaults applied.
type NewTicket struct {
	Title       string
	Description *string
	Status      Status
	Priority    Priority
}

// Validate trims input, applies defaults and returns field errors, if any.
func (r CreateRequest) Validate() (NewTicket, map[string]string) {
	errs := map[string]string{}
	t := NewTicket{Status: StatusOpen, Priority: PriorityMedium}

	if r.Title != nil {
		t.Title = strings.TrimSpace(*r.Title)
	}
	switch {
	case t.Title == "":
		errs["title"] = "is required"
	case utf8.RuneCountInString(t.Title) > TitleMaxLength:
		errs["title"] = "must be at most 200 characters"
	}

	// A blank description is stored as NULL so the UI can always send the field.
	if r.Description != nil {
		if d := strings.TrimSpace(*r.Description); d != "" {
			t.Description = &d
		}
	}
	if r.Status != nil {
		if !r.Status.Valid() {
			errs["status"] = statusMessage
		}
		t.Status = *r.Status
	}
	if r.Priority != nil {
		if !r.Priority.Valid() {
			errs["priority"] = priorityMessage
		}
		t.Priority = *r.Priority
	}
	return t, errs
}

// UpdateRequest is the PATCH body; only status and priority are editable.
type UpdateRequest struct {
	Status   *Status   `json:"status"`
	Priority *Priority `json:"priority"`
}

func (r UpdateRequest) Validate() map[string]string {
	errs := map[string]string{}
	if r.Status == nil && r.Priority == nil {
		errs["body"] = "provide status and/or priority"
	}
	if r.Status != nil && !r.Status.Valid() {
		errs["status"] = statusMessage
	}
	if r.Priority != nil && !r.Priority.Valid() {
		errs["priority"] = priorityMessage
	}
	return errs
}

// ListParams are the parsed query parameters for GET /tickets.
type ListParams struct {
	Status *Status
	Page   int
	Limit  int
}

func (p ListParams) Offset() int { return (p.Page - 1) * p.Limit }

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ListResponse struct {
	Data       []Ticket   `json:"data"`
	Pagination Pagination `json:"pagination"`
}
