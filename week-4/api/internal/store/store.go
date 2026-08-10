// Package store defines the persistence interface handlers.Server depends
// on, so tests can substitute an in-memory fake instead of a real Postgres.
package store

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("not found")
var ErrAlreadyExists = errors.New("already exists")

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
}

type Store interface {
	CreateUser(ctx context.Context, email, passwordHash, firstName, lastName string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)

	// RecordInstance claims an instance name for an owner, failing with
	// ErrAlreadyExists if the name is already taken by anyone.
	RecordInstance(ctx context.Context, name string, ownerID int64) error
	DeleteInstanceRecord(ctx context.Context, name string) error
	// OwnerOf returns ErrNotFound if no instance with this name is recorded.
	OwnerOf(ctx context.Context, name string) (int64, error)
	// ListOwned returns the names of every instance owned by ownerID.
	ListOwned(ctx context.Context, ownerID int64) (map[string]bool, error)
}
