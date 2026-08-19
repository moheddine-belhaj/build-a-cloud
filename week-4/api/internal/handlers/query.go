package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/middleware"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/types"
)

const (
	instanceQueryConnectTimeout = 5 * time.Second
	instanceQueryExecTimeout    = 30 * time.Second
	instanceQueryStatementTimeout = "25s"
)

// QueryRunner is the subset of *pgx.Conn this file actually uses, narrowed
// to an interface so tests can substitute a fake instead of a real network
// connection to a customer's Postgres instance.
type QueryRunner interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Close(ctx context.Context) error
}

// ConnectInstanceDB opens a connection to one instance's own database from
// already-resolved connection info.
var ConnectInstanceDB = func(ctx context.Context, conn types.ConnectionInfo) (QueryRunner, error) {
	cfg, err := pgx.ParseConfig("")
	if err != nil {
		return nil, err
	}
	cfg.Host = *conn.Host
	cfg.Port = uint16(*conn.Port)
	cfg.Database = *conn.Database
	cfg.User = *conn.Username
	cfg.Password = *conn.Password
	cfg.ConnectTimeout = instanceQueryConnectTimeout

	return pgx.ConnectConfig(ctx, cfg)
}

// RunInstanceQuery executes one caller-supplied SQL statement against the
// instance's own database and returns either its result
// set or its affected-row count.

func (s *Server) RunInstanceQuery(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorizeOwner(w, r, id) {
		return
	}

	var req types.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		writeJSONError(w, http.StatusBadRequest, "query is required")
		return
	}

	connInfo, err := s.resolveInstanceConnection(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}

	connectCtx, cancelConnect := context.WithTimeout(r.Context(), instanceQueryConnectTimeout)
	defer cancelConnect()
	conn, err := ConnectInstanceDB(connectCtx, connInfo)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to connect to instance database")
		return
	}
	defer conn.Close(context.Background())

	execCtx, cancelExec := context.WithTimeout(r.Context(), instanceQueryExecTimeout)
	defer cancelExec()

	readOnly := req.ReadOnly != nil && *req.ReadOnly
	result, queryType, execErr := runQuery(execCtx, conn, req.Query, readOnly)

	userID, _ := middleware.UserIDFromContext(r.Context())
	s.audit(r, userID, ActionQueryExecuted, &id, auditMetadataForQuery(queryType, readOnly, result, execErr))

	if execErr != nil {
		writeJSONError(w, http.StatusBadRequest, "query failed: "+execErr.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// runQuery executes one statement and returns its result plus a coarse
// classification of what kind of statement it was 
func runQuery(ctx context.Context, conn QueryRunner, query string, readOnly bool) (types.QueryResult, string, error) {
	if err := execAndDrain(ctx, conn, "SET statement_timeout = '"+instanceQueryStatementTimeout+"'"); err != nil {
		return types.QueryResult{}, "", err
	}

	if readOnly {
		if err := execAndDrain(ctx, conn, "BEGIN READ ONLY"); err != nil {
			return types.QueryResult{}, "", err
		}
	}

	result, queryType, err := execAndShape(ctx, conn, query)

	if readOnly {
		// COMMIT unconditionally: on success it closes the read-only
		// transaction normally; on a failed statement, Postgres treats a
		// COMMIT on an already-aborted transaction as an implicit ROLLBACK
		// either way, nothing here needs to change based on err.
		_ = execAndDrain(ctx, conn, "COMMIT")
	}

	return result, queryType, err
}

// execAndDrain runs a statement whose result is never used (SET, BEGIN,
// COMMIT) but must still be fully consumed for the command to complete.
func execAndDrain(ctx context.Context, conn QueryRunner, sql string) error {
	rows, err := conn.Query(ctx, sql)
	if err != nil {
		return err
	}
	for rows.Next() {
	}
	err = rows.Err()
	rows.Close()
	return err
}

// execAndShape runs the caller's actual statement and shapes the response
// purely from what Postgres reports back never by inspecting the query
// text so this is correct for CTEs, EXPLAIN, RETURNING clauses, and
// anything else that might produce a result set or an affected-row count in a way that doesn't match the first word of the query.
func execAndShape(ctx context.Context, conn QueryRunner, query string) (types.QueryResult, string, error) {
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return types.QueryResult{}, "", err
	}

	fields := rows.FieldDescriptions()
	columns := make([]string, len(fields))
	for i, f := range fields {
		columns[i] = f.Name
	}

	var resultRows [][]any
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			rows.Close()
			return types.QueryResult{}, "", err
		}
		resultRows = append(resultRows, vals)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return types.QueryResult{}, "", err
	}
	tag := rows.CommandTag()
	rows.Close()

	queryType := "OTHER"
	if parts := strings.Fields(tag.String()); len(parts) > 0 {
		queryType = strings.ToUpper(parts[0])
	}

	result := types.QueryResult{Command: ptr(tag.String())}
	if len(columns) > 0 {
		result.Columns = columns
		result.Rows = resultRows
	} else {
		affected := tag.RowsAffected()
		result.RowsAffected = &affected
	}
	return result, queryType, nil
}

// auditMetadataForQuery deliberately never includes the raw query text or
// any of its results. Arbitrary SQL can carry sensitive literal values
// (real customer data in an INSERT, for instance) writing that verbatim
// into audit_logs would be a serious security/privacy problem. Instead, this returns
// a small set of coarse-grained metadata about the query that is useful
// for auditing without exposing sensitive data.
func auditMetadataForQuery(queryType string, readOnly bool, result types.QueryResult, execErr error) map[string]any {
	meta := map[string]any{
		"readOnly": readOnly,
		"success":  execErr == nil,
	}
	// queryType is only known once Postgres has returned a command tag
	// for a query that failed before that point (a syntax error, a
	// read-only-transaction rejection)
	if queryType != "" {
		meta["queryType"] = queryType
	}
	if execErr == nil {
		if result.RowsAffected != nil {
			meta["rowsAffected"] = *result.RowsAffected
		}
		if result.Rows != nil {
			meta["rowCount"] = len(result.Rows)
		}
	}
	return meta
}
