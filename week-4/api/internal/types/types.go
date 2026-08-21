// Package types holds the API's request/response models, matching openapi.yaml.
package types

import "time"

type InstancePhase string

const (
	Provisioning InstancePhase = "Provisioning"
	Healthy      InstancePhase = "Healthy"
	Degraded     InstancePhase = "Degraded"
	Deleting     InstancePhase = "Deleting"
)

type CreateInstanceRequest struct {
	// Name must be <=11 chars (SKE-style naming constraint you already hit once)
	Name         string   `json:"name"`
	Instances    *int     `json:"instances,omitempty"`
	StorageSize  *string  `json:"storageSize,omitempty"`
	StorageClass *string  `json:"storageClass,omitempty"`
	Database     *string  `json:"database,omitempty"`
	Username     *string  `json:"username,omitempty"`
	AllowedIPs   []string `json:"allowedIPs,omitempty"`
}

// UpdateInstanceRequest carries a partial update only name, storageClass,
// database, and username are truly immutable on a live CNPG Cluster (fixed
// at initdb bootstrap or by Kubernetes object-name semantics), so those
// aren't represented here. A nil field means "leave unchanged"; AllowedIPs
// is a plain slice (not a pointer) so an explicit `"allowedIPs": []` can be
// told apart from an omitted key the former clears external access, the
// latter leaves it as-is.
type UpdateInstanceRequest struct {
	Instances   *int     `json:"instances,omitempty"`
	StorageSize *string  `json:"storageSize,omitempty"`
	AllowedIPs  []string `json:"allowedIPs,omitempty"`
}

type Instance struct {
	Id             *string        `json:"id,omitempty"`
	Uid            *string        `json:"uid,omitempty"`
	Name           *string        `json:"name,omitempty"`
	Phase          *InstancePhase `json:"phase,omitempty"`
	Instances      *int           `json:"instances,omitempty"`
	ReadyInstances *int           `json:"readyInstances,omitempty"`
	External       *bool          `json:"external,omitempty"`
	AllowedIPs     []string       `json:"allowedIPs,omitempty"`
	Version        *string        `json:"version,omitempty"`
	StorageSize    *string        `json:"storageSize,omitempty"`
	CreatedAt      *time.Time     `json:"createdAt,omitempty"`
}

// ServicePort mirrors a single entry of a Kubernetes Service's spec.ports.
type ServicePort struct {
	Name     *string `json:"name,omitempty"`
	Port     *int    `json:"port,omitempty"`
	Protocol *string `json:"protocol,omitempty"`
}

// ServiceInfo describes one of the Postgres endpoints CNPG exposes for an
// instance e.g. "<name>-rw" (primary), "<name>-ro" (read-only replicas),
// "<name>-r" (any replica).
type ServiceInfo struct {
	Name       *string       `json:"name,omitempty"`
	Type       *string       `json:"type,omitempty"`
	ClusterIP  *string       `json:"clusterIP,omitempty"`
	ExternalIP *string       `json:"externalIP,omitempty"`
	Ports      []ServicePort `json:"ports,omitempty"`
}

type ConnectionInfo struct {
	Host     *string `json:"host,omitempty"`
	Port     *int    `json:"port,omitempty"`
	Database *string `json:"database,omitempty"`
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
}

type Error struct {
	Message *string `json:"message,omitempty"`
}

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// QueryRequest carries a single arbitrary SQL statement to run against one
// instance's own database not the control-plane database.
type QueryRequest struct {
	Query string `json:"query"`
	// ReadOnly, if true, runs the statement inside a BEGIN READ ONLY
	// transaction, so Postgres itself rejects any data/schema-modifying
	// statement rather than relying on client-side inspection of the SQL text.
	ReadOnly *bool `json:"readOnly,omitempty"`
}

// QueryResult reports either a result set (Columns/Rows, for SELECT-like
// statements) or an affected-row count (RowsAffected, for INSERT/UPDATE/
// DELETE/DDL)  never both. Which shape applies is determined purely from
// Postgres's own response, not by inspecting the query text.
type QueryResult struct {
	Columns      []string `json:"columns,omitempty"`
	Rows         [][]any  `json:"rows,omitempty"`
	RowsAffected *int64   `json:"rowsAffected,omitempty"`
	// Command is Postgres's own command tag (e.g. "SELECT 3", "INSERT 0 1",
	// "CREATE TABLE")  a human-readable confirmation of what ran.
	Command *string `json:"command,omitempty"`
}

// AuditLogEntry is one recorded user action, returned newest-first from
// GET /v1/audit-logs and GET /v1/instances/{id}/audit-logs.
type AuditLogEntry struct {
	Id           *int64         `json:"id,omitempty"`
	Action       *string        `json:"action,omitempty"`
	InstanceName *string        `json:"instanceName,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    *time.Time     `json:"createdAt,omitempty"`
}

// RequestLogEntry is one recorded HTTP request/response cycle, returned
// newest-first from GET /v1/request-logs and
// GET /v1/instances/{id}/request-logs.
type RequestLogEntry struct {
	Id           *int64     `json:"id,omitempty"`
	Method       *string    `json:"method,omitempty"`
	Path         *string    `json:"path,omitempty"`
	InstanceName *string    `json:"instanceName,omitempty"`
	Status       *int       `json:"status,omitempty"`
	DurationMs   *float64   `json:"durationMs,omitempty"`
	CreatedAt    *time.Time `json:"createdAt,omitempty"`
}

// RequestLogPage is one page of RequestLogEntry results plus the total
// number of matching rows, so the UI can render "Page X of Y" pagination
// without guessing from page size alone.
type RequestLogPage struct {
	Items []RequestLogEntry `json:"items"`
	Total int               `json:"total"`
}
