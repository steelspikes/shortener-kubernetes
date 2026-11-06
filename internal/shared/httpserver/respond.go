package httpserver

import (
	"encoding/json"
	"net/http"
)

// JSON escribe v como JSON con el código de estado dado.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error escribe un error con la forma {"error": "mensaje"}.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}
