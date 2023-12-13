package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	// pure go SQLite driver
	_ "github.com/glebarez/go-sqlite"
	"go.uber.org/zap"
)

var db *sql.DB

// InitializeDB initializes the database
func InitializeDB() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to get user home directory: %w", err)
	}

	edgeDir := filepath.Join(home, ".edge")

	// Check if the .edge directory exists, if not, create it
	if _, err = os.Stat(edgeDir); os.IsNotExist(err) {
		if err = os.MkdirAll(edgeDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", edgeDir, err)
		}
	}

	dbPath := filepath.Join(edgeDir, "edge.db")

	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		zap.L().Error("Error opening database", zap.Error(err))
		return err
	}
	zap.L().Info("Database opened")

	err = createMigrations(db)
	if err != nil {
		zap.L().Error("Error creating migrations", zap.Error(err))
		return err
	}
	zap.L().Info("Database initialized")

	return nil
}

// GetDB returns the database connection
func GetDB() *sql.DB {
	return db
}
