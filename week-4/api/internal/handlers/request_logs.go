package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/middleware"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/types"
)

// ListRequestLogs returns the authenticated user's own HTTP request log,
// across every instance they own plus account-level requests.
func (s *Server) ListRequestLogs(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	s.writeRequestLogs(w, r, userID, nil)
}

// ListInstanceRequestLogs returns the HTTP request log for one specific
// instance the caller owns.
func (s *Server) ListInstanceRequestLogs(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorizeOwner(w, r, id) {
		return
	}
	userID, _ := middleware.UserIDFromContext(r.Context())
	s.writeRequestLogs(w, r, userID, &id)
}

func (s *Server) writeRequestLogs(w http.ResponseWriter, r *http.Request, userID int64, instanceName *string) {
	limit := parseLimit(r.URL.Query().Get("limit"))
	offset := parseOffset(r.URL.Query().Get("offset"))

	logs, total, err := s.store.ListRequestLogs(r.Context(), userID, instanceName, limit, offset)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list request logs")
		return
	}

	out := types.RequestLogPage{Items: make([]types.RequestLogEntry, 0, len(logs)), Total: total}
	for _, l := range logs {
		entry := l
		out.Items = append(out.Items, types.RequestLogEntry{
			Id:           ptr(entry.ID),
			Method:       ptr(entry.Method),
			Path:         ptr(entry.Path),
			InstanceName: entry.InstanceName,
			Status:       ptr(entry.Status),
			DurationMs:   ptr(entry.DurationMs),
			CreatedAt:    ptr(entry.CreatedAt),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
