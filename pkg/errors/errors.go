package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// APIError is a structured HTTP error with a status code, machine-readable code, and human message.
type APIError struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
	Err     error  `json:"-"`
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *APIError) Unwrap() error { return e.Err }

func New(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

func Wrap(status int, code, message string, err error) *APIError {
	return &APIError{Status: status, Code: code, Message: message, Err: err}
}

// Common constructors.
func BadRequest(message string) *APIError {
	return New(http.StatusBadRequest, "bad_request", message)
}

func Unauthorized(message string) *APIError {
	return New(http.StatusUnauthorized, "unauthorized", message)
}

func Forbidden(message string) *APIError {
	return New(http.StatusForbidden, "forbidden", message)
}

func NotFound(resource string) *APIError {
	return New(http.StatusNotFound, "not_found", fmt.Sprintf("%s not found", resource))
}

func Conflict(message string) *APIError {
	return New(http.StatusConflict, "conflict", message)
}

func UnprocessableEntity(message string) *APIError {
	return New(http.StatusUnprocessableEntity, "unprocessable_entity", message)
}

func Internal(err error) *APIError {
	return Wrap(http.StatusInternalServerError, "internal_error", "an internal error occurred", err)
}

func ServiceUnavailable(message string) *APIError {
	return New(http.StatusServiceUnavailable, "service_unavailable", message)
}

// As unwraps to *APIError from arbitrary errors.
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	ok := errors.As(err, &apiErr)
	return apiErr, ok
}

// Is forwards to stdlib.
var Is = errors.Is
var As = errors.As
var Unwrap = errors.Unwrap
