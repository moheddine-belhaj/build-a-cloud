package db

import (
	"context"
	"fmt"

	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/store"
)

func (p *Postgres) RecordRequestLog(ctx context.Context, userID *int64, method, path string, instanceName *string, status int, durationMs float64) error {
	_, err := p.conn.ExecContext(ctx,
		`INSERT INTO request_logs (user_id, method, path, instance_name, status, duration_ms) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, method, path, instanceName, status, durationMs,
	)
	return err
}

func (p *Postgres) ListRequestLogs(ctx context.Context, userID int64, instanceName *string, limit, offset int) ([]store.RequestLog, int, error) {
	countQuery := `SELECT count(*) FROM request_logs WHERE user_id = $1`
	listQuery := `SELECT id, user_id, method, path, instance_name, status, duration_ms, created_at
	          FROM request_logs WHERE user_id = $1`
	args := []any{userID}

	if instanceName != nil {
		countQuery += " AND instance_name = $2"
		listQuery += fmt.Sprintf(" AND instance_name = $%d", len(args)+1)
		args = append(args, *instanceName)
	}

	var total int
	countArgs := []any{userID}
	if instanceName != nil {
		countArgs = append(countArgs, *instanceName)
	}
	if err := p.conn.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := p.conn.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs := make([]store.RequestLog, 0, limit)
	for rows.Next() {
		var l store.RequestLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.Method, &l.Path, &l.InstanceName, &l.Status, &l.DurationMs, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}
