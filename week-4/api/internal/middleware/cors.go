package middleware

import "net/http"

// CORS allows the web UI to call this API from a browser. The UI is served
// from a different origin than the API (a different port in local dev, a
// different subdomain once deployed), so without these headers the browser
// blocks every request at the preflight stage before it reaches our routes.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
