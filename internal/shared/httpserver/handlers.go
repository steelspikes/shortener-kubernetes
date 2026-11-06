package httpserver

import "net/http"

// VersionHandler devuelve un handler que responde con el nombre y la versión del servicio.
func VersionHandler(name, version, description string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		JSON(w, http.StatusOK, map[string]string{
			"name":        name,
			"service":     "shortener",
			"version":     version,
			"description": description,
		})
	}
}
