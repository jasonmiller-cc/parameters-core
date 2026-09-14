// Package response provides JSON response helpers for parameters HTTP services.
package response

import (
	"encoding/json"
	"net/http"

	apierrors "github.com/jasonmiller-cc/parameters-core/pkg/errors"
)

// Envelope is the standard JSON wrapper for all API responses.
type Envelope struct {
	Data  any    `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
	Meta  *Meta  `json:"meta,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

type Meta struct {
	Total  int `json:"total,omitempty"`
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Data: data})
}

// List writes a paginated JSON list response.
func List(w http.ResponseWriter, data any, total, limit, offset int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(Envelope{
		Data: data,
		Meta: &Meta{Total: total, Limit: limit, Offset: offset},
	})
}

// OK writes a 200 JSON response.
func OK(w http.ResponseWriter, data any) { JSON(w, http.StatusOK, data) }

// Created writes a 201 JSON response.
func Created(w http.ResponseWriter, data any) { JSON(w, http.StatusCreated, data) }

// NoContent writes a 204 response.
func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

// Err writes an error response from an *apierrors.APIError or falls back to 500.
func Err(w http.ResponseWriter, err error) {
	if apiErr, ok := apierrors.AsAPIError(err); ok {
		writeError(w, apiErr.Status, apiErr.Code, apiErr.Message, apiErr.Detail)
		return
	}
	writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred", "")
}

// ErrStatus writes a plain error with an explicit status code.
func ErrStatus(w http.ResponseWriter, status int, code, message string) {
	writeError(w, status, code, message, "")
}

func writeError(w http.ResponseWriter, status int, code, message, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{
		Error: &Error{Code: code, Message: message, Detail: detail},
	})
}

// DecodeJSON reads and validates a JSON request body into dst.
// Returns an *APIError on parse or size errors.
func DecodeJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1 MiB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return apierrors.BadRequest("invalid request body: " + err.Error())
	}
	return nil
}
