package main

import (
	"net/http"
	"os"
)

const swaggerUIHTML = `<!DOCTYPE html>
<html>
<head>
  <title>PaaS Product API — Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      SwaggerUIBundle({
        url: "/openapi.yaml",
        dom_id: "#swagger-ui",
        presets: [SwaggerUIBundle.presets.apis],
      });
    };
  </script>
</body>
</html>`
// readSpec tries the local dev path first (running via `go run` from
// week-3/api), then the container path (where the Dockerfile copies it to
// the filesystem root) works identically in both environments.
func readSpec() ([]byte, error) {
	for _, path := range []string{"openapi.yaml", "/openapi.yaml"} {
		if b, err := os.ReadFile(path); err == nil {
			return b, nil
		}
	}
	return nil, os.ErrNotExist
}

func registerDocsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		b, err := readSpec()
		if err != nil {
			http.Error(w, "spec not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(b)
	})
	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(swaggerUIHTML))
	})
}