package emqx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/edgeflare/pgo/pkg/util"
	"github.com/edgeflare/pgo/pkg/util/rand"
)

// EMQXClient struct for holding base URL, HTTP client, and admin token
type EMQXClient struct {
	BaseURL    string
	Client     *http.Client
	AdminToken string
}

var (
	url               = util.GetEnvOrDefault("EDGE_EMQX_HTTP_API", "http://emqx:18083/api/v5")
	pgPassword        = util.GetEnvOrDefault("EDGE_EMQX_PGPASSWORD", "")
	pgUser            = util.GetEnvOrDefault("EDGE_EMQX_PGUSER", "emqx")
	dashboardPassword = util.GetEnvOrDefault("EMQX_DASHBOARD__DEFAULT_PASSWORD", "public")
	dashboardUsername = util.GetEnvOrDefault("EMQX_DASHBOARD__DEFAULT_USERNAME", "admin")
	edgeMqttPassword  = util.GetEnvOrDefault("EDGE_MQTT_PASSWORD", "")
)

// NewEMQXClient initializes a new EMQX client
// Requires the EMQX_HTTP_API, EMQX_DASHBOARD__DEFAULT_PASSWORD, and EMQX_DASHBOARD__DEFAULT_USERNAME, EMQX_MQTT_CLIENT_SUPERUSER_PASSWORD env vars
// If not set, defaults are http://emqx:18083/api/v5, admin, public and public, respectively
func NewEMQXClient() *EMQXClient {
	client := &EMQXClient{
		BaseURL: url,
		Client:  &http.Client{},
	}

	if err := client.obtainAdminToken(dashboardUsername, dashboardPassword); err != nil {
		log.Fatalf("Error obtaining admin token: %v", err)
	}

	return client
}

// PostRequest sends a POST request to the EMQX API with authorization header
func (c *EMQXClient) PostRequest(endpoint string, payload interface{}) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s", c.BaseURL, endpoint)
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AdminToken))

	return c.Client.Do(req)
}

// Modify AddBuiltInDBAuth to be idempotent
func (c *EMQXClient) AddBuiltInDBAuth() error {
	exists, err := c.authMethodExists("password_based:built_in_database")
	if err != nil {
		return err
	}
	if exists {
		fmt.Println("Built-in database auth already exists, skipping...")
		return nil
	}

	payload := map[string]interface{}{
		"user_id_type": "username",
		"password_hash_algorithm": map[string]string{
			"name":          "sha512",
			"salt_position": "suffix",
		},
		"backend":   "built_in_database",
		"mechanism": "password_based",
	}

	res, err := c.PostRequest("authentication", payload)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add built-in database auth: %s", res.Status)
	}

	return nil
}

// Modify AddJWTAuth to be idempotent
func (c *EMQXClient) AddJWTAuth() error {
	exists, err := c.authMethodExists("jwt")
	if err != nil {
		return err
	}
	if exists {
		fmt.Println("JWT auth already exists, skipping...")
		return nil
	}

	payload := map[string]interface{}{
		"from":     "password",
		"use_jwks": true,
		"verify_claims": map[string]string{
			"sub": "${username}",
		},
		"disconnect_after_expire": true,
		"endpoint":                fmt.Sprintf("https://iam-%s.%s.edgeflare.dev/oauth/v2/keys", os.Getenv("EF_PROJECT_NS"), os.Getenv("EF_REGION")),
		"refresh_interval":        5,
		"ssl": map[string]interface{}{
			"enable": true,
			"verify": "verify_none",
		},
		"mechanism": "jwt",
	}

	res, err := c.PostRequest("authentication", payload)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add JWT auth: %s", res.Status)
	}

	return nil
}

// Modify AddPostgresAuth to be idempotent
func (c *EMQXClient) AddPostgresAuth() error {
	exists, err := c.authMethodExists("password_based:postgresql")
	if err != nil {
		return err
	}
	if exists {
		fmt.Println("PostgreSQL auth already exists, skipping...")
		return nil
	}

	payload := map[string]interface{}{
		"backend":                     "postgresql",
		"database":                    "main",
		"disable_prepared_statements": false,
		"enable":                      true,
		"mechanism":                   "password_based",
		"password":                    pgPassword,
		"password_hash_algorithm": map[string]string{
			"name":          "sha512",
			"salt_position": "suffix",
		},
		"pool_size": 8,
		"query":     "SELECT password_hash, salt, is_superuser FROM edge.mqtt_users WHERE username = ${username} LIMIT 1",
		"server":    "postgres-replica:5432",
		"ssl": map[string]interface{}{
			"ciphers":                []string{},
			"depth":                  10,
			"enable":                 true,
			"hibernate_after":        "5s",
			"log_level":              "notice",
			"reuse_sessions":         true,
			"secure_renegotiate":     true,
			"server_name_indication": "postgres-replica",
			"verify":                 "verify_none",
			"versions":               []string{"tlsv1.3", "tlsv1.2"},
		},
		"username": pgUser,
	}

	res, err := c.PostRequest("authentication", payload)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add PostgreSQL auth client: %s", res.Status)
	}

	return nil
}

// Configure sets up the EMQX
func Configure() {
	client := NewEMQXClient()

	err := client.AddBuiltInDBAuth()
	if err != nil {
		log.Fatalf("Error adding built-in database auth: %v", err)
	}

	err = client.AddJWTAuth()
	if err != nil {
		log.Fatalf("Error adding JWT auth: %v", err)
	}

	err = client.AddPostgresAuth()
	if err != nil {
		log.Fatalf("Error adding PostgreSQL auth client: %v", err)
	}

	err = client.CreatePublicUser()
	if err != nil {
		log.Fatalf("Error creating public user: %v", err)
	}

	err = client.CreateSuperuser()
	if err != nil {
		log.Fatalf("Error creating superuser: %v", err)
	}

	fmt.Println("Successfully configured EMQX authentication clients.")
}

// obtainAdminToken gets the admin token and stores it in the client
func (c *EMQXClient) obtainAdminToken(username, password string) error {
	values := map[string]string{
		"password": password,
		"username": username,
	}
	jsonData, err := json.Marshal(values)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/login", c.BaseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to obtain admin token: %s", res.Status)
	}

	var body struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(res.Body).Decode(&body)
	if err != nil {
		return err
	}

	c.AdminToken = body.Token
	return nil
}

// createUser creates a user with specific details in EMQX.
// It first checks if the password-based authentication method exists.
// If not, creating a user will likely fail.
// Then, it checks if the specified user ID already exists.
func (c *EMQXClient) createUser(user map[string]interface{}) error {
	// Check if the password-based authentication method exists
	exists, err := c.authMethodExists("password_based:built_in_database")
	if err != nil {
		return fmt.Errorf("failed to check authentication method: %w", err)
	}

	// If the method doesn't exist, we can't create users
	if !exists {
		return fmt.Errorf("password-based authentication method not found. User creation requires this method")
	}

	// Check if the user ID already exists
	userID, ok := user["user_id"].(string)
	if !ok {
		return fmt.Errorf("invalid user ID: %v", user["user_id"])
	}
	exists, err = c.userExists(userID)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	// If the user already exists, return an error
	if exists {
		fmt.Printf("user %s already exists\n", userID)
		return nil
	}

	// Proceed with user creation if the user doesn't exist
	res, err := c.PostRequest("authentication/password_based%3Abuilt_in_database/users", user)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create user %s: %s", userID, res.Status)
	}
	return nil
}

func (c *EMQXClient) userExists(userID string) (bool, error) {
	// Construct the URL for the GET request
	url := c.BaseURL + "/authentication/password_based%3Abuilt_in_database/users"

	// Create the HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	// Add authorization header
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AdminToken))

	// Send the request
	res, err := c.Client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = res.Body.Close() }()

	// Check the response status
	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to check user existence: %s", res.Status)
	}

	// Decode the response body
	type UserResponse struct {
		Data []struct {
			UserID string `json:"user_id"`
		}
	}
	var response UserResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return false, err
	}

	// Search for the user ID in the response data
	for _, user := range response.Data {
		if user.UserID == userID {
			return true, nil
		}
	}

	return false, nil
}

// Check if an authentication method exists
func (c *EMQXClient) authMethodExists(methodId string) (bool, error) {
	url := fmt.Sprintf("%s/authentication", c.BaseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AdminToken))
	res, err := c.Client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to check authentication: %s", res.Status)
	}

	// Inline struct definition
	type authMethod struct {
		ID string `json:"id"`
	}

	// Decode response directly into a slice of the inline struct
	var authMethods []authMethod
	err = json.NewDecoder(res.Body).Decode(&authMethods)
	if err != nil {
		return false, err
	}

	// Search for method ID within the slice
	for _, method := range authMethods {
		if method.ID == methodId {
			return true, nil
		}
	}
	return false, nil
}

func (c *EMQXClient) CreateSuperuser() error {
	if edgeMqttPassword == "" {
		edgeMqttPassword = rand.NewPassword()
	}

	superuser := map[string]interface{}{
		"user_id":      "edge",
		"password":     edgeMqttPassword,
		"is_superuser": true,
	}

	// Check if the "edge" user already exists
	exists, err := c.userExists("edge")
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if exists {
		fmt.Println("superuser 'edge' already exists, skipping creation.")
		return nil // User already exists, skip creation
	}

	// Proceed with user creation if the user doesn't exist
	err = c.createUser(superuser)
	if err != nil {
		return err
	}

	// Create kubernetes secret using kubectl
	cmd := exec.Command("kubectl", "-n", os.Getenv("EF_PROJECT_NS"), "create", "secret", "generic",
		"emqx-mqttuser-edge",
		fmt.Sprintf("--from-literal=password=%s", edgeMqttPassword))

	// Capture both stdout and stderr
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create kubernetes secret: %v\nstdout: %s\nstderr: %s", err, out.String(), stderr.String())
	}

	return nil
}

func (c *EMQXClient) CreatePublicUser() error {
	user := map[string]interface{}{
		"user_id":      "public",
		"password":     "public",
		"is_superuser": false,
	}

	err := c.createUser(user)
	if err != nil {
		return err
	}

	return nil
}
