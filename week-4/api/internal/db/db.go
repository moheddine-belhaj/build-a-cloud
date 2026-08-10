// Package db is the Postgres-backed implementation of store.Store, owning
// the connection and the users/instances tables behind user accounts and
// per-user instance ownership.
package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed schema.sql
var schema string

// Postgres implements store.Store on top of a *sql.DB.
type Postgres struct {
	conn *sql.DB
}

func Connect(ctx context.Context, dsn string) (*Postgres, error) {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if _, err := conn.ExecContext(ctx, schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("run schema: %w", err)
	}

	return &Postgres{conn: conn}, nil
}

func (p *Postgres) Close() error {
	return p.conn.Close()
}
