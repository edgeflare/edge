package emqx

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/edgeflare/pgo/pkg/util/rand"
)

// EMQXClient struct for holding base URL, HTTP client, and admin token
type EMQXClient struct {
	BaseURL    string
	Client     *http.Client
	AdminToken string
}

var (
	url               = cmp.Or(os.Getenv("EDGE_EMQX_HTTP_API"), fmt.Sprintf("http://emqx-%s:18083/api/v5", os.Getenv("EDGE_DOMAIN_ROOT")), "http://emqx:18083/api/v5")
	pgPassword        = cmp.Or(os.Getenv("EDGE_EMQX_PGPASSWORD"), "")
	pgUser            = cmp.Or(os.Getenv("EDGE_EMQX_PGUSER"), "emqx")
	dashboardPassword = cmp.Or(os.Getenv("EMQX_DASHBOARD__DEFAULT_PASSWORD"), "public")
	dashboardUsername = cmp.Or(os.Getenv("EMQX_DASHBOARD__DEFAULT_USERNAME"), "admin")
	edgeMqttPassword  = cmp.Or(os.Getenv("EDGE_MQTT_PASSWORD"), "")
	idPJwksEndpoint   = cmp.Or(os.Getenv("EDGE_IAM_ISSUER_JWKS_ENDPOINT"), fmt.Sprintf("http://iam.%s/oauth/v2/keys", os.Getenv("EDGE_DOMAIN_ROOT")))
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

// Configure sets up the EMQX
func Configure() {
	client := NewEMQXClient()

	err := client.addBuiltInDBAuthN()
	if err != nil {
		log.Fatalf("Error adding built-in database auth: %v", err)
	}

	err = client.addJWTAuthN()
	if err != nil {
		log.Fatalf("Error adding JWT auth: %v", err)
	}

	err = client.addPostgresAuthN()
	if err != nil {
		log.Fatalf("Error adding PostgreSQL auth client: %v", err)
	}

	err = client.createPublicUser()
	if err != nil {
		log.Fatalf("Error creating public user: %v", err)
	}

	err = client.createSuperuser()
	if err != nil {
		log.Fatalf("Error creating superuser: %v", err)
	}

	err = client.addFileAuthZ()
	if err != nil {
		log.Fatalf("Error adding file authz: %v", err)
		// log.Printf("Warning: Skipping ERROR %s\nManually add rules on the EMQX dashboard. \nUse rules \n\n%s\n\n", err, authzRules)
	}

	fmt.Println("Successfully configured EMQX authentication clients.")
}

// makeRequest sends a POST request to the EMQX API with authorization header
func (c *EMQXClient) makeRequest(endpoint string, payload any, method ...string) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s", c.BaseURL, endpoint)
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Default to POST if no method specified
	httpMethod := "POST"
	if len(method) > 0 {
		httpMethod = method[0]
	}

	req, err := http.NewRequest(httpMethod, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AdminToken))
	return c.Client.Do(req)
}

// addBuiltInDBAuthN adds built-in database authentication to EMQX
func (c *EMQXClient) addBuiltInDBAuthN() error {
	exists, err := c.authMethodExists("password_based:built_in_database")
	if err != nil {
		return err
	}
	if exists {
		fmt.Println("Built-in database auth already exists, skipping...")
		return nil
	}

	payload := map[string]any{
		"user_id_type": "username",
		"password_hash_algorithm": map[string]string{
			"name":          "sha512",
			"salt_position": "suffix",
		},
		"backend":   "built_in_database",
		"mechanism": "password_based",
	}

	res, err := c.makeRequest("authentication", payload)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add built-in database auth: %s", res.Status)
	}

	return nil
}

// addJWTAuthN adds JWT authentication to EMQX
func (c *EMQXClient) addJWTAuthN() error {
	exists, err := c.authMethodExists("jwt")
	if err != nil {
		return err
	}
	if exists {
		fmt.Println("JWT auth already exists, skipping...")
		return nil
	}

	payload := map[string]any{
		"from":     "password",
		"use_jwks": true,
		"verify_claims": map[string]string{
			"sub": "${username}",
		},
		"disconnect_after_expire": true,
		"endpoint":                idPJwksEndpoint,
		"refresh_interval":        30,
		"ssl": map[string]any{
			"enable": true,
			"verify": "verify_none",
		},
		"mechanism": "jwt",
	}

	res, err := c.makeRequest("authentication", payload)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add JWT auth: %s", res.Status)
	}

	return nil
}

// addPostgresAuthN adds PostgreSQL authentication to EMQX
func (c *EMQXClient) addPostgresAuthN() error {
	exists, err := c.authMethodExists("password_based:postgresql")
	if err != nil {
		return err
	}
	if exists {
		fmt.Println("PostgreSQL auth already exists, skipping...")
		return nil
	}

	payload := map[string]any{
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
		"ssl": map[string]any{
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

	res, err := c.makeRequest("authentication", payload)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add PostgreSQL auth client: %s", res.Status)
	}

	return nil
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
func (c *EMQXClient) createUser(user map[string]any) error {
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
	res, err := c.makeRequest("authentication/password_based%3Abuilt_in_database/users", user)
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
	url := c.BaseURL + "/authentication/password_based%3Abuilt_in_database/users"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AdminToken))

	// Send the request
	res, err := c.Client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to check user existence: %s", res.Status)
	}

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

	type authMethod struct {
		ID string `json:"id"`
	}

	var authMethods []authMethod
	err = json.NewDecoder(res.Body).Decode(&authMethods)
	if err != nil {
		return false, err
	}

	for _, method := range authMethods {
		if method.ID == methodId {
			return true, nil
		}
	}
	return false, nil
}

var authzRules = `{allow, {username, {re, "^dashboard\$"}}, subscribe, ["$SYS/#"]}.
{deny, all, subscribe, ["$SYS/#", {eq, "#"}]}.
{allow, all, subscribe, ["/public/#"]}.
{allow, all, subscribe, ["/authn/${username}/#"]}.
{allow, {username, "edge"}, all, ["/#"]}.
{deny, all}.`

func (c *EMQXClient) addFileAuthZ() error {
	policy := map[string]any{
		"enable": true,
		"rules":  authzRules,
		"type":   "file",
	}

	res, err := c.makeRequest("authorization/sources/file", policy, "PUT")
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to add file authz: %s", res.Status)
	}

	return nil
}

func (c *EMQXClient) createSuperuser() error {
	if edgeMqttPassword == "" {
		edgeMqttPassword = rand.NewPassword()
	}

	superuser := map[string]any{
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
	//
	// TODO: should use k8s client-go instead of exec, and when running on Kubernetes
	//
	// cmd := exec.Command("kubectl", "-n", os.Getenv("EF_PROJECT_NS"), "create", "secret", "generic",
	// 	"emqx-mqttuser-edge",
	// 	fmt.Sprintf("--from-literal=password=%s", edgeMqttPassword))

	// // Capture both stdout and stderr
	// var out bytes.Buffer
	// var stderr bytes.Buffer
	// cmd.Stdout = &out
	// cmd.Stderr = &stderr

	// err = cmd.Run()
	// if err != nil {
	// 	return fmt.Errorf("failed to create kubernetes secret: %v\nstdout: %s\nstderr: %s", err, out.String(), stderr.String())
	// }

	return nil
}

func (c *EMQXClient) createPublicUser() error {
	user := map[string]any{
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
