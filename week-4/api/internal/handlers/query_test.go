package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/handlers"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/types"
)

// fakeRows is a minimal pgx.Rows  the pgx docs note Rows is deliberately an
// interface "to allow tests to mock Query", which is exactly what this does.
type fakeRows struct {
	columns    []string
	rows       [][]any
	commandTag string
	idx        int
}

func (r *fakeRows) Close()     {}
func (r *fakeRows) Err() error { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag(r.commandTag)
}
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	fds := make([]pgconn.FieldDescription, len(r.columns))
	for i, c := range r.columns {
		fds[i] = pgconn.FieldDescription{Name: c}
	}
	return fds
}
func (r *fakeRows) Next() bool {
	if r.idx >= len(r.rows) {
		return false
	}
	r.idx++
	return true
}
func (r *fakeRows) Scan(dest ...any) error { return nil }
func (r *fakeRows) Values() ([]any, error) { return r.rows[r.idx-1], nil }
func (r *fakeRows) RawValues() [][]byte    { return nil }
func (r *fakeRows) Conn() *pgx.Conn        { return nil }

// fakeQueryRunner stands in for a real *pgx.Conn. The housekeeping
// statements query.go always issues (SET statement_timeout, and BEGIN
// READ ONLY / COMMIT when readOnly is requested) succeed automatically with
// an empty result; only the caller's actual query reaches respond.
type fakeQueryRunner struct {
	queries []string
	respond func(sql string) (*fakeRows, error)
}

func (f *fakeQueryRunner) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	f.queries = append(f.queries, sql)
	if strings.HasPrefix(sql, "SET statement_timeout") || sql == "BEGIN READ ONLY" || sql == "COMMIT" {
		return &fakeRows{}, nil
	}
	rows, err := f.respond(sql)
	if err != nil {
		return nil, err
	}
	return rows, nil
}
func (f *fakeQueryRunner) Close(ctx context.Context) error { return nil }

// withFakeQueryRunner substitutes handlers.ConnectInstanceDB for the
// duration of one test and restores the original afterward  the seam
// query.go exposes specifically so tests never open a real network
// connection to a Postgres instance.
func withFakeQueryRunner(t *testing.T, runner *fakeQueryRunner, connectErr error) {
	t.Helper()
	original := handlers.ConnectInstanceDB
	handlers.ConnectInstanceDB = func(ctx context.Context, conn types.ConnectionInfo) (handlers.QueryRunner, error) {
		if connectErr != nil {
			return nil, connectErr
		}
		return runner, nil
	}
	t.Cleanup(func() { handlers.ConnectInstanceDB = original })
}

// refuseToConnect guards the two authorization-failure cases: it fails the
// test outright if RunInstanceQuery ever attempts to open a connection,
// proving those requests are rejected before touching any database.
func refuseToConnect(t *testing.T) {
	t.Helper()
	original := handlers.ConnectInstanceDB
	handlers.ConnectInstanceDB = func(ctx context.Context, conn types.ConnectionInfo) (handlers.QueryRunner, error) {
		t.Fatal("ConnectInstanceDB should not be called")
		return nil, nil
	}
	t.Cleanup(func() { handlers.ConnectInstanceDB = original })
}

func TestRunInstanceQuery(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		runner     *fakeQueryRunner
		wantStatus int
		check      func(t *testing.T, got types.QueryResult)
	}{
		{
			name: "valid SELECT",
			body: `{"query":"SELECT id, name FROM widgets"}`,
			runner: &fakeQueryRunner{respond: func(sql string) (*fakeRows, error) {
				return &fakeRows{
					columns:    []string{"id", "name"},
					rows:       [][]any{{int64(1), "a"}, {int64(2), "b"}},
					commandTag: "SELECT 2",
				}, nil
			}},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, got types.QueryResult) {
				if len(got.Columns) != 2 || got.Columns[0] != "id" || got.Columns[1] != "name" {
					t.Errorf("columns = %v, want [id name]", got.Columns)
				}
				if len(got.Rows) != 2 {
					t.Errorf("len(rows) = %d, want 2", len(got.Rows))
				}
				if got.RowsAffected != nil {
					t.Errorf("rowsAffected = %v, want nil for a SELECT", got.RowsAffected)
				}
			},
		},
		{
			name: "valid write query",
			body: `{"query":"INSERT INTO widgets (name) VALUES ('a')"}`,
			runner: &fakeQueryRunner{respond: func(sql string) (*fakeRows, error) {
				return &fakeRows{commandTag: "INSERT 0 1"}, nil
			}},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, got types.QueryResult) {
				if got.Columns != nil {
					t.Errorf("columns = %v, want nil for an INSERT", got.Columns)
				}
				if got.RowsAffected == nil || *got.RowsAffected != 1 {
					t.Errorf("rowsAffected = %v, want 1", got.RowsAffected)
				}
			},
		},
		{
			name: "invalid SQL",
			body: `{"query":"SELCT nonsense"}`,
			runner: &fakeQueryRunner{respond: func(sql string) (*fakeRows, error) {
				return nil, errors.New(`syntax error at or near "SELCT"`)
			}},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _, st := newFakeServer(
				newCluster("api-test-1", "Cluster in healthy state"),
				newAppSecret("api-test-1", "appuser", "s3cret", "appdb"),
			)
			st.instances["api-test-1"] = testUserID
			withFakeQueryRunner(t, tt.runner, nil)

			req := withUser(httptest.NewRequest(http.MethodPost, "/v1/instances/api-test-1/query", strings.NewReader(tt.body)))
			w := httptest.NewRecorder()

			srv.RunInstanceQuery(w, req, "api-test-1")

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.check != nil {
				got := decode[types.QueryResult](t, w.Body)
				tt.check(t, got)
			}
		})
	}
}

func TestRunInstanceQuery_ReadOnlyUsesTransaction(t *testing.T) {
	srv, _, st := newFakeServer(
		newCluster("api-test-1", "Cluster in healthy state"),
		newAppSecret("api-test-1", "appuser", "s3cret", "appdb"),
	)
	st.instances["api-test-1"] = testUserID
	runner := &fakeQueryRunner{respond: func(sql string) (*fakeRows, error) {
		return &fakeRows{columns: []string{"id"}, rows: [][]any{{int64(1)}}, commandTag: "SELECT 1"}, nil
	}}
	withFakeQueryRunner(t, runner, nil)

	body := `{"query":"SELECT id FROM widgets","readOnly":true}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/v1/instances/api-test-1/query", strings.NewReader(body)))
	w := httptest.NewRecorder()

	srv.RunInstanceQuery(w, req, "api-test-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if len(runner.queries) < 3 || runner.queries[1] != "BEGIN READ ONLY" {
		t.Errorf("queries = %v, want [SET..., BEGIN READ ONLY, <query>, COMMIT]", runner.queries)
	}
	if runner.queries[len(runner.queries)-1] != "COMMIT" {
		t.Errorf("last query = %q, want COMMIT", runner.queries[len(runner.queries)-1])
	}
}

func TestRunInstanceQuery_UnauthorizedInstanceAccess(t *testing.T) {
	srv, _, st := newFakeServer(
		newCluster("api-test-1", "Cluster in healthy state"),
		newAppSecret("api-test-1", "appuser", "s3cret", "appdb"),
	)
	st.instances["api-test-1"] = testUserID + 1 // owned by someone else
	refuseToConnect(t)

	req := withUser(httptest.NewRequest(http.MethodPost, "/v1/instances/api-test-1/query", strings.NewReader(`{"query":"SELECT 1"}`)))
	w := httptest.NewRecorder()

	srv.RunInstanceQuery(w, req, "api-test-1")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRunInstanceQuery_MissingInstance(t *testing.T) {
	srv, _, _ := newFakeServer()
	refuseToConnect(t)

	req := withUser(httptest.NewRequest(http.MethodPost, "/v1/instances/does-not-exist/query", strings.NewReader(`{"query":"SELECT 1"}`)))
	w := httptest.NewRecorder()

	srv.RunInstanceQuery(w, req, "does-not-exist")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRunInstanceQuery_MissingQuery(t *testing.T) {
	srv, _, st := newFakeServer(newCluster("api-test-1", "Cluster in healthy state"))
	st.instances["api-test-1"] = testUserID
	refuseToConnect(t)

	req := withUser(httptest.NewRequest(http.MethodPost, "/v1/instances/api-test-1/query", strings.NewReader(`{"query":""}`)))
	w := httptest.NewRecorder()

	srv.RunInstanceQuery(w, req, "api-test-1")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
