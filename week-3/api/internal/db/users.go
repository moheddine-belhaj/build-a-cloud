package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/moheddine-belhaj/build-a-cloud/week-3/api/internal/store"
)

func (p *Postgres) CreateUser(ctx context.Context, email, passwordHash, firstName, lastName string) (store.User, error) {
	var u store.User
	err := p.conn.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash, first_name, last_name)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, email, password_hash, first_name, last_name`,
		email, passwordHash, firstName, lastName,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName)
	if isUniqueViolation(err) {
		return store.User{}, store.ErrAlreadyExists
	}
	if err != nil {
		return store.User{}, err
	}
	return u, nil
}

func (p *Postgres) GetUserByEmail(ctx context.Context, email string) (store.User, error) {
	var u store.User
	err := p.conn.QueryRowContext(ctx,
		`SELECT id, email, password_hash, first_name, last_name FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName)
	if errors.Is(err, sql.ErrNoRows) {
		return store.User{}, store.ErrNotFound
	}
	if err != nil {
		return store.User{}, err
	}
	return u, nil
}
