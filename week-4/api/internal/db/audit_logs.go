package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/store"
)

func (p *Postgres) RecordAuditLog(ctx context.Context, userID int64, action string, instanceName *string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}

	_, err = p.conn.ExecContext(ctx,
		`INSERT INTO audit_logs (user_id, action, instance_name, metadata) VALUES ($1, $2, $3, $4)`,
		userID, action, instanceName, raw,
	)
	return err
}

func (p *Postgres) ListAuditLogs(ctx context.Context, userID int64, instanceName *string, limit, offset int) ([]store.AuditLog, error) {
	query := `SELECT id, user_id, action, instance_name, metadata, created_at
	          FROM audit_logs WHERE user_id = $1`
	args := []any{userID}

	if instanceName != nil {
		query += fmt.Sprintf(" AND instance_name = $%d", len(args)+1)
		args = append(args, *instanceName)
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := p.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]store.AuditLog, 0, limit)
	for rows.Next() {
		var l store.AuditLog
		var raw []byte
		if err := rows.Scan(&l.ID, &l.UserID, &l.Action, &l.InstanceName, &raw, &l.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &l.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal audit metadata: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}
