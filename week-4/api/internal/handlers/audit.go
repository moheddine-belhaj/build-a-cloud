package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/middleware"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/types"
)

const (
	ActionUserRegistered  = "user.registered"
	ActionUserLoggedIn    = "user.login"
	ActionLoginFailed     = "login.failed"
	ActionInstanceCreated = "instance.created"
	ActionInstanceUpdated = "instance.updated"
	ActionInstanceDeleted = "instance.deleted"
	ActionCredentialsView = "credentials.viewed"
	ActionQueryExecuted   = "query.executed"
)

const (
	defaultAuditLogLimit = 50
	maxAuditLogLimit     = 200
)

// audit is best-effort: a failure to record the trail shouldn't fail the
// action it's recording, so errors are logged, not returned to the caller.
func (s *Server) audit(r *http.Request, userID int64, action string, instanceName *string, metadata map[string]any) {
	if err := s.store.RecordAuditLog(r.Context(), userID, action, instanceName, metadata); err != nil {
		log.Printf("audit log write failed: action=%s user=%d err=%v", action, userID, err)
	}
}

// ListAuditLogs returns the authenticated user's own activity, across every
// instance they own plus account-level events (login/register).
func (s *Server) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	s.writeAuditLogs(w, r, userID, nil)
}

// ListInstanceAuditLogs returns the activity trail for one specific
// instance the caller owns.
func (s *Server) ListInstanceAuditLogs(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorizeOwner(w, r, id) {
		return
	}
	userID, _ := middleware.UserIDFromContext(r.Context())
	s.writeAuditLogs(w, r, userID, &id)
}

func (s *Server) writeAuditLogs(w http.ResponseWriter, r *http.Request, userID int64, instanceName *string) {
	limit := parseLimit(r.URL.Query().Get("limit"))
	offset := parseOffset(r.URL.Query().Get("offset"))

	logs, err := s.store.ListAuditLogs(r.Context(), userID, instanceName, limit, offset)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list audit logs")
		return
	}

	out := make([]types.AuditLogEntry, 0, len(logs))
	for _, l := range logs {
		entry := l
		out = append(out, types.AuditLogEntry{
			Id:           ptr(entry.ID),
			Action:       ptr(entry.Action),
			InstanceName: entry.InstanceName,
			Metadata:     entry.Metadata,
			CreatedAt:    ptr(entry.CreatedAt),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func parseLimit(raw string) int {
	n, err := parsePositiveInt(raw)
	if err != nil || n == 0 {
		return defaultAuditLogLimit
	}
	if n > maxAuditLogLimit {
		return maxAuditLogLimit
	}
	return n
}

func parseOffset(raw string) int {
	n, err := parsePositiveInt(raw)
	if err != nil {
		return 0
	}
	return n
}

func parsePositiveInt(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, strconv.ErrSyntax
	}
	return n, nil
}
