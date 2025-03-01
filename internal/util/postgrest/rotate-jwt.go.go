package postgrest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"net/http"
	"os"
	"time"

	"log"

	pg "github.com/edgeflare/pgo/pkg/pgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultJWKURL = "http://iam.example.local/oauth/v2/keys"
	cacheFilePath = "/tmp/postgrest-jwks-kids.txt"
)

// RotateJwtKey periodically fetches JWKS data, updates the PostgREST configuration,
// and triggers a configuration reload.
func RotateJwtKey(ctx context.Context, conn pg.Conn, jwkURL string) error {
	if jwkURL == "" {
		jwkURL = defaultJWKURL
	}

	waitForJWKSURL(jwkURL)

	for {
		cachedKids, err := readCacheFile(cacheFilePath)
		if err != nil {
			log.Printf("Error reading cache file: %v", err)
		}

		jwksResponse, fetchedKids, err := fetchJWKS(jwkURL)
		if err != nil {
			log.Printf("Error fetching JWKS: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if cachedKids != fetchedKids {
			date := time.Now().Format(time.RFC3339)
			log.Println("Date:", date)

			if err := updateDatabase(conn, jwksResponse); err != nil {
				log.Printf("Error updating database: %v", err)
			}

			if err := writeCacheFile(cacheFilePath, fetchedKids); err != nil {
				log.Printf("Error writing cache file: %v", err)
			}

			if err := reloadPostgrest(conn); err != nil {
				log.Printf("Error reloading PostgREST: %v", err)
			}
		}

		time.Sleep(5 * time.Second)
	}
}

// waitForJWKSURL waits for the given JWKS URL to become available,
// retrying every 5 seconds until a successful response is received.
func waitForJWKSURL(url string) {
	for {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return
		}
		log.Printf("Waiting for %s to be available...", url)
		time.Sleep(5 * time.Second)
	}
}

// fetchJWKS retrieves JWKS data from the specified URL and extracts
// the 'kid' values from keys used for signature verification ('use' = 'sig').
func fetchJWKS(url string) (string, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var jwks map[string]interface{}
	if err := json.Unmarshal(body, &jwks); err != nil {
		return "", "", err
	}

	keys, ok := jwks["keys"].([]interface{})
	if !ok {
		return "", "", fmt.Errorf("unexpected format for keys")
	}

	var fetchedKids []string
	for _, key := range keys {
		keyMap, ok := key.(map[string]interface{})
		if !ok {
			continue
		}
		use, ok := keyMap["use"].(string)
		if ok && use == "sig" {
			kid, ok := keyMap["kid"].(string)
			if ok {
				fetchedKids = append(fetchedKids, kid)
			}
		}
	}

	jwksResponse := string(body)
	fetchedKidsStr := fmt.Sprintf("%v", fetchedKids)

	return jwksResponse, fetchedKidsStr, nil
}

// readCacheFile reads the contents of the cache file at the given path.
// If the file doesn't exist, it returns an empty string and no error.
func readCacheFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// writeCacheFile writes the specified content to the cache file at the given path.
func writeCacheFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// updateDatabase updates the 'pgrst.jwt_secret' configuration for the 'postgrest'
// role in the specified database with the provided JWKS response.
func updateDatabase(conn pg.Conn, jwksResponse string) error {
	ctx := context.Background()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var dbName string

	if pgxConn, ok := conn.(*pgx.Conn); ok {
		dbName = pgxConn.Config().Database
	} else if pgxpoolConn, ok := conn.(*pgxpool.Pool); ok {
		dbName = pgxpoolConn.Config().ConnConfig.Database
	}

	if dbName == "" {
		return fmt.Errorf("dbname not set")
	}

	query := fmt.Sprintf(
		`ALTER ROLE postgrest IN DATABASE %[1]s SET pgrst.jwt_secret = '%[2]s';
		ALTER ROLE postgrest IN DATABASE %[1]s SET pgrst.jwt_secret_is_base64 = false;`,
		dbName, jwksResponse,
	)
	_, err = tx.Exec(ctx, query)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// reloadPostgrest sends a notification to PostgREST to reload its configuration.
func reloadPostgrest(conn pg.Conn) error {
	_, err := conn.Exec(context.Background(), "NOTIFY pgrst, 'reload config'")
	return err
}
