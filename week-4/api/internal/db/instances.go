package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/store"
)

func (p *Postgres) RecordInstance(ctx context.Context, name string, ownerID int64) error {
	_, err := p.conn.ExecContext(ctx,
		`INSERT INTO instances (name, owner_id) VALUES ($1, $2)`,
		name, ownerID,
	)
	if isUniqueViolation(err) {
		return store.ErrAlreadyExists
	}
	return err
}

func (p *Postgres) DeleteInstanceRecord(ctx context.Context, name string) error {
	_, err := p.conn.ExecContext(ctx, `DELETE FROM instances WHERE name = $1`, name)
	return err
}

func (p *Postgres) OwnerOf(ctx context.Context, name string) (int64, error) {
	var ownerID int64
	err := p.conn.QueryRowContext(ctx,
		`SELECT owner_id FROM instances WHERE name = $1`, name,
	).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, store.ErrNotFound
	}
	return ownerID, err
}

func (p *Postgres) ListOwned(ctx context.Context, ownerID int64) (map[string]bool, error) {
	rows, err := p.conn.QueryContext(ctx, `SELECT name FROM instances WHERE owner_id = $1`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	owned := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		owned[name] = true
	}
	return owned, rows.Err()
}
