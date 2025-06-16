package middleware

import (
	"encoding/json"
	"go-server/utils/errors"
	"log"
	"net/http"
)

// ErrorMiddleware handles errors and sends a standardized JSON response
func ErrorMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("Panic recovered: %v", rec)
					WriteError(w, errors.ErrInternal)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// writeError writes an APIError as a JSON response
func WriteError(w http.ResponseWriter, err error) {
	apiErr, ok := err.(*errors.APIError)
	if !ok {
		apiErr = errors.Wrap(err, "UNKNOWN_ERROR", "Unexpected error", errors.ErrInternal.Status)
	}
	// Log server errors
	if apiErr.Status >= 500 {
		log.Printf("Server error %s (Details: %s)", apiErr.Error(), apiErr.Details)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)
	json.NewEncoder(w).Encode(err)
}
