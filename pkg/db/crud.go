package db

import (
	"database/sql"

	"github.com/google/uuid"
)

// GenerateID generates a new ID for a cluster or node
func GenerateID() string {
	newUUID := uuid.New()
	return newUUID.String()[len(newUUID.String())-12:]
}

// InsertCluster inserts a new cluster into the database
func InsertCluster(db *sql.DB, cluster K3sCluster) error {
	query := `INSERT INTO k3s_clusters (id, status, version, is_ha, created_at, updated_at, apiserver) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(query, cluster.ID, cluster.Status, cluster.Version, cluster.IsHA, cluster.CreatedAt, cluster.UpdatedAt, cluster.Apiserver)
	return err
}

// SelectClusters selects all clusters from the database
func SelectClusters(db *sql.DB) ([]K3sCluster, error) {
	var clusters []K3sCluster

	query := `SELECT id, status, version, is_ha, created_at, updated_at, apiserver FROM k3s_clusters`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cluster K3sCluster
		err = rows.Scan(&cluster.ID, &cluster.Status, &cluster.Version, &cluster.IsHA, &cluster.CreatedAt, &cluster.UpdatedAt, &cluster.Apiserver)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, cluster)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return clusters, nil
}

// SelectCluster selects a cluster from the database by ID
func SelectCluster(db *sql.DB, id string) (*K3sCluster, error) {
	var cluster K3sCluster
	query := `SELECT id, status, version, is_ha, created_at, updated_at, apiserver FROM k3s_clusters WHERE id = ?`
	row := db.QueryRow(query, id)
	err := row.Scan(&cluster.ID, &cluster.Status, &cluster.Version, &cluster.IsHA, &cluster.CreatedAt, &cluster.UpdatedAt, &cluster.Apiserver)
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

// UpdateCluster updates a cluster in the database
func UpdateCluster(db *sql.DB, cluster K3sCluster) error {
	query := `UPDATE k3s_clusters SET status = ?, version = ?, is_ha = ?, updated_at = CURRENT_TIMESTAMP, apiserver = ? WHERE id = ?`
	_, err := db.Exec(query, cluster.Status, cluster.Version, cluster.IsHA, cluster.Apiserver, cluster.ID)
	return err
}

// DeleteCluster deletes a cluster from the database
func DeleteCluster(db *sql.DB, id string) error {
	query := `DELETE FROM k3s_clusters WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}

// InsertNode inserts a new node into the database
func InsertNode(db *sql.DB, node K3sNode) error {
	query := `INSERT INTO k3s_nodes (id, cluster_id, ip, role, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(query, node.ID, node.ClusterID, node.IP, node.Role, node.Status, node.CreatedAt, node.UpdatedAt)
	return err
}

// SelectNodes selects all nodes from the database
func SelectNodes(db *sql.DB) ([]K3sNode, error) {
	var nodes []K3sNode

	query := `SELECT id, cluster_id, ip, role, status, created_at, updated_at FROM k3s_nodes`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var node K3sNode
		err = rows.Scan(&node.ID, &node.ClusterID, &node.IP, &node.Role, &node.Status, &node.CreatedAt, &node.UpdatedAt)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return nodes, nil
}

// SelectNodesByCluster selects all nodes from the database by cluster ID
func SelectNodesByCluster(db *sql.DB, clusterID string) ([]K3sNode, error) {
	var nodes []K3sNode

	query := `SELECT id, cluster_id, ip, role, status, created_at, updated_at FROM k3s_nodes WHERE cluster_id = ?`
	rows, err := db.Query(query, clusterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var node K3sNode
		err = rows.Scan(&node.ID, &node.ClusterID, &node.IP, &node.Role, &node.Status, &node.CreatedAt, &node.UpdatedAt)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return nodes, nil
}

// SelectNode selects a node from the database by ID
func SelectNode(db *sql.DB, id string) (*K3sNode, error) {
	var node K3sNode
	query := `SELECT id, cluster_id, ip, role, status, created_at, updated_at FROM k3s_nodes WHERE id = ? LIMIT 1`
	row := db.QueryRow(query, id)
	err := row.Scan(&node.ID, &node.ClusterID, &node.IP, &node.Role, &node.Status, &node.CreatedAt, &node.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// SelectNodeByIP selects a node from the database by IP
func SelectNodeByIP(db *sql.DB, ip string) (K3sNode, error) {
	var node K3sNode

	query := `SELECT id, cluster_id, ip, role, status, created_at, updated_at FROM k3s_nodes WHERE ip = ? LIMIT 1`
	row := db.QueryRow(query, ip)

	err := row.Scan(&node.ID, &node.ClusterID, &node.IP, &node.Role, &node.Status, &node.CreatedAt, &node.UpdatedAt)
	if err != nil {
		return K3sNode{}, err // Return an empty K3sNode and the error
	}

	return node, nil
}

// UpdateNode updates a node in the database
func UpdateNode(db *sql.DB, node K3sNode) error {
	query := `UPDATE k3s_nodes SET cluster_id = ?, ip = ?, role = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Exec(query, node.ClusterID, node.IP, node.Role, node.Status, node.ID)
	return err
}

// DeleteNode deletes a node from the database
func DeleteNode(db *sql.DB, id string) error {
	query := `DELETE FROM k3s_nodes WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}
