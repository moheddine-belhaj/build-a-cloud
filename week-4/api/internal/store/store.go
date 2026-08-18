// Package store defines the persistence interface handlers.Server depends
// on, so tests can substitute an in-memory fake instead of a real Postgres.
package store

import (
	"context"
	"errors"
	"time"
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

// AuditLog records a single user-initiated action for later retrieval by
// that same user — a permanent, per-account activity record, distinct from
// operational logging (which goes to Loki, not Postgres, and isn't scoped to
// an individual user or retrievable through the API).
type AuditLog struct {
	ID           int64
	UserID       int64
	Action       string
	InstanceName *string
	Metadata     map[string]any
	CreatedAt    time.Time
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

	// RecordAuditLog is best-effort from the caller's perspective: a failure
	// here shouldn't fail the user-facing action it's recording.
	RecordAuditLog(ctx context.Context, userID int64, action string, instanceName *string, metadata map[string]any) error
	// ListAuditLogs returns userID's own audit trail, newest first. If
	// instanceName is non-nil, results are scoped to just that instance.
	ListAuditLogs(ctx context.Context, userID int64, instanceName *string, limit, offset int) ([]AuditLog, error)
}
