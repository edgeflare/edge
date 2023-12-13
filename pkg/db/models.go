package db

import "time"

// K3sCluster represents a K3s cluster
type K3sCluster struct {
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	ID        string    `json:"id"` // Last 12 characters of UUID
	Status    string    `json:"status"`
	Version   string    `json:"version"`
	Apiserver string    `json:"apiserver"`
	IsHA      bool      `json:"is_ha"` // Indicates high availability
}

// K3sNode represents a K3s node
type K3sNode struct {
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	ID        string    `json:"id"`
	ClusterID string    `json:"cluster_id"`
	IP        string    `json:"ip"`
	Role      string    `json:"role"` // Can be "server" or "agent"
	Status    string    `json:"status"`
}
