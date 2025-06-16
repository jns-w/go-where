package errors

import (
	"fmt"
	"net/http"
)

// APIError represents a custom error type for API responses
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
	Details string `json:"details,omitempty"`
}

// Error returns the error message
func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewAPIError(code, message string, status int, details ...string) *APIError {
	err := &APIError{
		Code:    code,
		Message: message,
		Status:  status,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

var (
	ErrInvalidInput = NewAPIError("INVALID_INPUT", "Invalid request data", http.StatusBadRequest)
	ErrUnauthorized = NewAPIError("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized)
	ErrNotFound     = NewAPIError("NOT_FOUND", "Resource not found", http.StatusNotFound)
	ErrInternal     = NewAPIError("INTERNAL_SERVER_ERROR", "Internal server error", http.StatusInternalServerError)
	ErrConflict     = NewAPIError("CONFLICT", "Resource conflict", http.StatusConflict)
)

func Wrap(err error, code, message string, status int) *APIError {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr
	}
	return NewAPIError(code, message, status, err.Error())
}
