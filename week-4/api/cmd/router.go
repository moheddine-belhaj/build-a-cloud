package main

import (
	"net/http"

	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/handlers"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/middleware"
)

// withID adapts a handler that takes the "{id}" path value as an explicit
// argument into a plain http.HandlerFunc for mux registration.
func withID(h func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h(w, r, r.PathValue("id"))
	}
}

func newRouter(srv *handlers.Server, jwtSecret string) *http.ServeMux {
	mux := http.NewServeMux()
	registerDocsRoutes(mux)

	mux.HandleFunc("POST /v1/auth/register", srv.Register)
	mux.HandleFunc("POST /v1/auth/login", srv.Login)

	requireAuth := middleware.RequireAuth(jwtSecret)
	mux.HandleFunc("GET /v1/instances", requireAuth(srv.ListInstances))
	mux.HandleFunc("POST /v1/instances", requireAuth(srv.CreateInstance))
	mux.HandleFunc("GET /v1/instances/{id}", requireAuth(withID(srv.GetInstance)))
	mux.HandleFunc("PATCH /v1/instances/{id}", requireAuth(withID(srv.UpdateInstance)))
	mux.HandleFunc("DELETE /v1/instances/{id}", requireAuth(withID(srv.DeleteInstance)))
	mux.HandleFunc("GET /v1/instances/{id}/connection", requireAuth(withID(srv.GetInstanceConnection)))
	mux.HandleFunc("GET /v1/instances/{id}/services", requireAuth(withID(srv.ListInstanceServices)))
	mux.HandleFunc("POST /v1/instances/{id}/query", requireAuth(withID(srv.RunInstanceQuery)))
	mux.HandleFunc("GET /v1/audit-logs", requireAuth(srv.ListAuditLogs))
	mux.HandleFunc("GET /v1/instances/{id}/audit-logs", requireAuth(withID(srv.ListInstanceAuditLogs)))
	mux.HandleFunc("GET /v1/request-logs", requireAuth(srv.ListRequestLogs))
	mux.HandleFunc("GET /v1/instances/{id}/request-logs", requireAuth(withID(srv.ListInstanceRequestLogs)))

	return mux
}
