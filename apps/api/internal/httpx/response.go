// Package httpx contains shared HTTP transport helpers.
package httpx

import (
	"encoding/json"
	"net/http"
)

// WriteJSON encodes data as JSON and writes it with status.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteError writes the API-standard JSON error response.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}
