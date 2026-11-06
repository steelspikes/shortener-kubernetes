package httpserver

import (
	"encoding/json"
	"net/http"
)

// maxBodyBytes es el tamaño máximo permitido para el body (1 MiB).
const maxBodyBytes = 1 << 20

// DecodeBody decodifica el body JSON de la petición en el tipo T, con un límite de tamaño.
func DecodeBody[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var v T
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	err := json.NewDecoder(r.Body).Decode(&v)
	return v, err
}
