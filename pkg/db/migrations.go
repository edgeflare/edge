package db

import (
	"database/sql"
)

func createMigrations(db *sql.DB) error {
	clusterTableQuery := `
    CREATE TABLE IF NOT EXISTS k3s_clusters (
        id TEXT PRIMARY KEY,
        status TEXT NOT NULL,
        version TEXT NOT NULL,
        is_ha BOOLEAN NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        apiserver TEXT
    );`

	nodeTableQuery := `
    CREATE TABLE IF NOT EXISTS k3s_nodes (
        id TEXT PRIMARY KEY,
        cluster_id TEXT,
        ip TEXT NOT NULL,
        role TEXT CHECK(role IN ('server', 'agent')) NOT NULL,
        status TEXT NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (cluster_id) REFERENCES k3s_clusters(id) ON DELETE CASCADE
    );`

	_, err := db.Exec(clusterTableQuery)
	if err != nil {
		return err
	}

	_, err = db.Exec(nodeTableQuery)
	return err
}
