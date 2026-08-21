package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/store"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/types"
)

func TestListRequestLogs(t *testing.T) {
	srv, _, st := newFakeServer()
	callerID := int64(testUserID)
	otherUser := callerID + 1
	st.requestLogs = []store.RequestLog{
		{ID: 1, UserID: &callerID, Method: "GET", Path: "/v1/instances", Status: 200, DurationMs: 26.724082},
		{ID: 2, UserID: &otherUser, Method: "GET", Path: "/v1/instances", Status: 200, DurationMs: 5},
		{ID: 3, UserID: nil, Method: "POST", Path: "/v1/auth/login", Status: 401, DurationMs: 1.2},
	}

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/request-logs", nil))
	w := httptest.NewRecorder()

	srv.ListRequestLogs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	got := decode[types.RequestLogPage](t, w.Body)
	if got.Total != 1 || len(got.Items) != 1 {
		t.Fatalf("got %+v, want exactly the caller's own 1 entry", got)
	}
	if got.Items[0].Method == nil || *got.Items[0].Method != "GET" {
		t.Errorf("method = %v, want GET", got.Items[0].Method)
	}
	if got.Items[0].Status == nil || *got.Items[0].Status != 200 {
		t.Errorf("status = %v, want 200", got.Items[0].Status)
	}
}

func TestListRequestLogs_Unauthorized(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/request-logs", nil)
	w := httptest.NewRecorder()

	srv.ListRequestLogs(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestListRequestLogs_Pagination(t *testing.T) {
	srv, _, st := newFakeServer()
	callerID := int64(testUserID)
	for i := 0; i < 25; i++ {
		st.requestLogs = append(st.requestLogs, store.RequestLog{
			ID: int64(i), UserID: &callerID, Method: "GET", Path: "/v1/instances", Status: 200, DurationMs: 1,
		})
	}

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/request-logs?limit=20&offset=20", nil))
	w := httptest.NewRecorder()

	srv.ListRequestLogs(w, req)

	got := decode[types.RequestLogPage](t, w.Body)
	if got.Total != 25 {
		t.Fatalf("total = %d, want 25", got.Total)
	}
	if len(got.Items) != 5 {
		t.Fatalf("len(items) = %d, want 5 (second page remainder)", len(got.Items))
	}
}

func TestListInstanceRequestLogs(t *testing.T) {
	srv, _, st := newFakeServer(newCluster("api-test-1", "Healthy"))
	st.instances["api-test-1"] = testUserID
	callerID := int64(testUserID)
	other := "other-instance"
	st.requestLogs = []store.RequestLog{
		{ID: 1, UserID: &callerID, Method: "GET", Path: "/v1/instances/api-test-1", Status: 200, DurationMs: 3, InstanceName: strPtr("api-test-1")},
		{ID: 2, UserID: &callerID, Method: "GET", Path: "/v1/instances/other-instance", Status: 200, DurationMs: 3, InstanceName: &other},
	}

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/instances/api-test-1/request-logs", nil))
	w := httptest.NewRecorder()

	srv.ListInstanceRequestLogs(w, req, "api-test-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	got := decode[types.RequestLogPage](t, w.Body)
	if got.Total != 1 || len(got.Items) != 1 {
		t.Fatalf("got %+v, want exactly the 1 entry scoped to api-test-1", got)
	}
	if got.Items[0].InstanceName == nil || *got.Items[0].InstanceName != "api-test-1" {
		t.Errorf("instanceName = %v, want api-test-1", got.Items[0].InstanceName)
	}
}

func TestListInstanceRequestLogs_NotOwner(t *testing.T) {
	srv, _, st := newFakeServer(newCluster("api-test-1", "Healthy"))
	st.instances["api-test-1"] = testUserID + 1

	req := withUser(httptest.NewRequest(http.MethodGet, "/v1/instances/api-test-1/request-logs", nil))
	w := httptest.NewRecorder()

	srv.ListInstanceRequestLogs(w, req, "api-test-1")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func strPtr(s string) *string { return &s }
