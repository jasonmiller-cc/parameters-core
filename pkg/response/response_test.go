package response_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	apierrors "github.com/jasonmiller-cc/parameters-core/pkg/errors"
	"github.com/jasonmiller-cc/parameters-core/pkg/response"
)

func TestOK(t *testing.T) {
	w := httptest.NewRecorder()
	response.OK(w, map[string]string{"hello": "world"})

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var env response.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Data == nil {
		t.Error("expected Data to be set")
	}
}

func TestCreated(t *testing.T) {
	w := httptest.NewRecorder()
	response.Created(w, map[string]int{"id": 1})
	if w.Code != 201 {
		t.Errorf("status = %d, want 201", w.Code)
	}
}

func TestNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	response.NoContent(w)
	if w.Code != 204 {
		t.Errorf("status = %d, want 204", w.Code)
	}
	if len(w.Body.Bytes()) != 0 {
		t.Errorf("expected empty body, got %q", w.Body.String())
	}
}

func TestList(t *testing.T) {
	w := httptest.NewRecorder()
	response.List(w, []int{1, 2, 3}, 10, 3, 0)

	var env response.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Meta == nil {
		t.Fatal("expected Meta to be set")
	}
	if env.Meta.Total != 10 || env.Meta.Limit != 3 || env.Meta.Offset != 0 {
		t.Errorf("Meta = %+v, want {Total:10 Limit:3 Offset:0}", env.Meta)
	}
}

func TestErr_APIError(t *testing.T) {
	w := httptest.NewRecorder()
	response.Err(w, apierrors.NotFound("resource missing"))

	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	var env response.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Error == nil {
		t.Fatal("expected Error to be set")
	}
}

func TestErr_PlainError_FallsBackTo500(t *testing.T) {
	w := httptest.NewRecorder()
	response.Err(w, errPlain{})

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

type errPlain struct{}

func (errPlain) Error() string { return "boom" }

func TestErrStatus(t *testing.T) {
	w := httptest.NewRecorder()
	response.ErrStatus(w, 418, "teapot", "I'm a teapot")

	if w.Code != 418 {
		t.Fatalf("status = %d, want 418", w.Code)
	}
	var env response.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if env.Error == nil || env.Error.Code != "teapot" {
		t.Errorf("Error = %+v, want Code=teapot", env.Error)
	}
}

func TestDecodeJSON_Valid(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"svc"}`))
	var dst struct {
		Name string `json:"name"`
	}
	if err := response.DecodeJSON(req, &dst); err != nil {
		t.Fatalf("DecodeJSON() error: %v", err)
	}
	if dst.Name != "svc" {
		t.Errorf("Name = %q, want svc", dst.Name)
	}
}

func TestDecodeJSON_UnknownField(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"unexpected":"field"}`))
	var dst struct {
		Name string `json:"name"`
	}
	err := response.DecodeJSON(req, &dst)
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
	if _, ok := apierrors.AsAPIError(err); !ok {
		t.Errorf("expected *APIError, got %T", err)
	}
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`not json`))
	var dst struct{}
	if err := response.DecodeJSON(req, &dst); err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
