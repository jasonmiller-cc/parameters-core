package errors_test

import (
	"net/http"
	"testing"

	"github.com/jasonmiller-cc/parameters-core/pkg/errors"
)

func TestAPIError_Error(t *testing.T) {
	err := errors.NotFound("user")
	if err.Status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", err.Status)
	}
	if err.Code != "not_found" {
		t.Fatalf("expected not_found, got %s", err.Code)
	}
}

func TestAsAPIError(t *testing.T) {
	err := errors.BadRequest("bad")
	apiErr, ok := errors.AsAPIError(err)
	if !ok {
		t.Fatal("expected APIError")
	}
	if apiErr.Status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", apiErr.Status)
	}
}

func TestInternal_Wraps(t *testing.T) {
	inner := errors.BadRequest("inner")
	outer := errors.Internal(inner)
	if outer.Status != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", outer.Status)
	}
	if !errors.Is(outer, inner) {
		t.Fatal("expected Unwrap chain to include inner")
	}
}
