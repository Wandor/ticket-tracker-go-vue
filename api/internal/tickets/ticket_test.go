package tickets

import (
	"strings"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestCreateRequestValidate(t *testing.T) {
	tests := []struct {
		name      string
		req       CreateRequest
		wantError map[string]string
	}{
		{"valid with defaults", CreateRequest{Title: ptr("Printer jam")}, nil},
		{"missing title", CreateRequest{}, map[string]string{"title": "is required"}},
		{"blank title", CreateRequest{Title: ptr("   ")}, map[string]string{"title": "is required"}},
		{"200 multibyte runes fit", CreateRequest{Title: ptr(strings.Repeat("é", 200))}, nil},
		{"title too long", CreateRequest{Title: ptr(strings.Repeat("a", 201))}, map[string]string{"title": "must be at most 200 characters"}},
		{"bad enums", CreateRequest{Title: ptr("x"), Status: ptr(Status("done")), Priority: ptr(Priority("p1"))},
			map[string]string{"status": statusMessage, "priority": priorityMessage}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, errs := tc.req.Validate()
			if len(errs) != len(tc.wantError) {
				t.Fatalf("errors = %v, want %v", errs, tc.wantError)
			}
			for k, v := range tc.wantError {
				if errs[k] != v {
					t.Errorf("errors[%s] = %q, want %q", k, errs[k], v)
				}
			}
			if len(errs) == 0 && (got.Status != StatusOpen || got.Priority != PriorityMedium) {
				t.Errorf("defaults not applied: %+v", got)
			}
		})
	}
}

func TestBlankDescriptionBecomesNil(t *testing.T) {
	got, _ := CreateRequest{Title: ptr("x"), Description: ptr("  ")}.Validate()
	if got.Description != nil {
		t.Errorf("description = %q, want nil", *got.Description)
	}
}
