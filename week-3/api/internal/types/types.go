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
	Name        string  `json:"name"`
	Instances   *int    `json:"instances,omitempty"`
	StorageSize *string `json:"storageSize,omitempty"`
}

type Instance struct {
	Id             *string        `json:"id,omitempty"`
	Name           *string        `json:"name,omitempty"`
	Phase          *InstancePhase `json:"phase,omitempty"`
	Instances      *int           `json:"instances,omitempty"`
	ReadyInstances *int           `json:"readyInstances,omitempty"`
	CreatedAt      *time.Time     `json:"createdAt,omitempty"`
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
