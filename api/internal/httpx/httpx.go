// Package httpx holds the JSON response and error conventions shared by every handler.
package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

// Error is an API error rendered as {"error": {...}}.
type Error struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (e *Error) Error() string { return e.Message }

// ValidationError reports field-level problems with a request.
func ValidationError(fields map[string]string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: "validation_error", Message: "Request validation failed", Fields: fields}
}

// NotFound reports a missing resource.
func NotFound(message string) *Error {
	return &Error{Status: http.StatusNotFound, Code: "not_found", Message: message}
}

// WriteJSON writes v with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}

// WriteError renders err; anything that is not an *Error becomes an opaque 500.
func WriteError(w http.ResponseWriter, err error) {
	apiErr, ok := err.(*Error)
	if !ok {
		slog.Error("unhandled error", "err", err)
		apiErr = &Error{Status: http.StatusInternalServerError, Code: "internal_error", Message: "Something went wrong"}
	}
	WriteJSON(w, apiErr.Status, map[string]*Error{"error": apiErr})
}

// HandlerFunc is an http.HandlerFunc that returns an error instead of writing it.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Handle adapts h so returned errors are rendered consistently.
func Handle(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			WriteError(w, err)
		}
	}
}

// DecodeJSON reads a JSON object body into v. With strict set, unknown fields are rejected so
// a client cannot believe it changed something the endpoint ignores.
func DecodeJSON(r *http.Request, v any, strict bool) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 100<<10))
	if strict {
		dec.DisallowUnknownFields()
	}
	err := dec.Decode(v)
	if err == nil {
		return nil
	}

	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &typeErr) && typeErr.Field != "":
		return ValidationError(map[string]string{typeErr.Field: "must be a " + typeErr.Type.String()})
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.Trim(strings.TrimPrefix(err.Error(), "json: unknown field "), `"`)
		return ValidationError(map[string]string{field: "cannot be updated"})
	default:
		return &Error{Status: http.StatusBadRequest, Code: "invalid_json", Message: "Request body is not valid JSON"}
	}
}
