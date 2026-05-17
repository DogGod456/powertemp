package api

import (
	"encoding/json"
	"net/http"
)

// writeJSON выставляет единый JSON Content-Type и сериализует ответ API.
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// writeError приводит ошибки backend к единому JSON-формату { "error": "..." }.
func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}
