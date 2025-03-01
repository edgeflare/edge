package controller

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	edgev1alpha1 "github.com/edgeflare/edge/api/v1alpha1"
	"github.com/edgeflare/pgo/pkg/pgx/role"
	"github.com/edgeflare/pgo/pkg/util/rand"
	rnd "github.com/edgeflare/pgo/pkg/util/rand"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

// reconcileAuth reconciles the auth component for a project.
// It handles both external and built-in auth providers, with special logic for Zitadel.
// For Zitadel, it ensures required secrets exist before proceeding with the Helm release.
func (r *ProjectReconciler) reconcileAuth(ctx context.Context, project *edgev1alpha1.Project,
	name string, ref *edgev1alpha1.ComponentRef) error {
	logger := log.FromContext(ctx)
	compType := "auth"
	logger.Info("Reconciling auth", "component", name)

	// Handle external auth components
	if ref.IsExternal() {
		if err := r.verifyExternalIdPSecret(ctx, project, name, ref); err != nil {
			logger.Error(err, "External auth secret verification failed")
			_ = r.updateComponentStatus(ctx, project, compType, name, false,
				fmt.Sprintf("Secret error: %v", err), "")
			return err
		}
		return r.handleExternalComponent(ctx, project, compType, name, ref)
	}

	// Handle Zitadel specifically
	if name == "zitadel" {
		if err := r.reconcileZitadel(ctx, project, name, ref); err != nil {
			return err
		}
	}

	// Process the component release
	return r.handleComponentRelease(ctx, project, compType, name, ref)
}

// reconcileZitadel handles the specific requirements for the Zitadel auth component.
// It ensures all required secrets exist and dependencies are ready before proceeding.
func (r *ProjectReconciler) reconcileZitadel(ctx context.Context, project *edgev1alpha1.Project,
	name string, ref *edgev1alpha1.ComponentRef) error {
	logger := log.FromContext(ctx)
	compType := "auth"

	// Load and parse schema for validation
	zitadelSchema, err := os.ReadFile("hack/zitadel.values.schema.json")
	if err != nil {
		logger.Error(err, "Failed to load Zitadel schema")
		return err
	}

	if err = r.ensureZitadelInstanceSecret(ctx, project); err != nil {
		return err
	}

	// Parse the supplied values with schema validation
	var zitadelValues map[string]any
	if ref.Release.ValuesContent != "" {
		zitadelValues, err = r.parseValues(ref.Release.ValuesContent, zitadelSchema)
		if err != nil {
			logger.Error(err, "Failed to parse Zitadel values")
			_ = r.updateComponentStatus(ctx, project, compType, name, false,
				fmt.Sprintf("Values error: %v", err), "")
			return err
		}
	} else {
		zitadelValues = make(map[string]any)
	}

	// Step 1: Ensure or verify masterkey secret
	masterkeySecretName, err := r.reconcileZitadelMasterkey(ctx, project, zitadelValues)
	if err != nil {
		logger.Error(err, "Failed to reconcile Zitadel masterkey")
		_ = r.updateComponentStatus(ctx, project, compType, name, false,
			fmt.Sprintf("Masterkey error: %v", err), "")
		return err
	}

	// Step 2: Wait for PostgreSQL to be ready
	if err := r.waitForPostgreSQLReady(ctx, project); err != nil {
		logger.Error(err, "PostgreSQL is not ready")
		_ = r.updateComponentStatus(ctx, project, compType, name, false,
			fmt.Sprintf("Database error: %v", err), "")
		return err
	}

	// Step 3: Ensure PostgreSQL connection secret for Zitadel
	pgSecretName := fmt.Sprintf("%s-pguser-zitadel", project.Name)
	if err := r.ensureZitadelPostgresSecretAndRole(ctx, project, pgSecretName); err != nil {
		logger.Error(err, "Failed to ensure Zitadel PostgreSQL connection secret")
		_ = r.updateComponentStatus(ctx, project, compType, name, false,
			fmt.Sprintf("Secret error: %v", err), "")
		return err
	}

	// Step 4: Update values to use masterkeySecretName and PostgreSQL connection
	r.updateZitadelValuesWithMasterkey(zitadelValues, masterkeySecretName)
	// r.updateZitadelValuesWithPgSecret(zitadelValues, pgSecretName)

	// Update the reference with the modified values
	updatedValues, err := yaml.Marshal(zitadelValues)
	if err != nil {
		logger.Error(err, "Failed to marshal updated Zitadel values")
		return err
	}

	logger.V(4).Info("Updated Zitadel values", "values", string(updatedValues))
	ref.Release.ValuesContent = string(updatedValues)

	return nil
}

// reconcileZitadelMasterkey checks for an existing masterkey and either uses it or creates a new one.
// Returns the secret name of the masterkey to use.
func (r *ProjectReconciler) reconcileZitadelMasterkey(ctx context.Context, project *edgev1alpha1.Project,
	values map[string]any) (string, error) {
	logger := log.FromContext(ctx)

	// Check if masterkeySecretName is already configured in values
	var masterkeySecretName string

	// Access the zitadel section in the values
	zitadelSection, hasZitadelSection := values["zitadel"].(map[string]any)
	if hasZitadelSection {
		// Check for masterkeySecretName
		if msn, ok := zitadelSection["masterkeySecretName"].(string); ok && msn != "" {
			masterkeySecretName = msn
			logger.Info("Found existing masterkeySecretName", "name", masterkeySecretName)
		}
	}

	// If masterkeySecretName is set, verify it exists and contains the required key
	if masterkeySecretName != "" {
		if err := r.verifyZitadelMasterkeySecret(ctx, project.Namespace, masterkeySecretName); err != nil {
			logger.Error(err, "Existing Zitadel masterkey secret verification failed",
				"name", masterkeySecretName)
			return "", err
		}
		logger.Info("Verified existing masterkey secret", "name", masterkeySecretName)
		return masterkeySecretName, nil
	}

	// No masterkeySecretName is set, create one
	masterkeySecretName = fmt.Sprintf("%s-zitadel-masterkey", project.Name)
	if err := r.ensureZitadelMasterkeySecret(ctx, project, masterkeySecretName); err != nil {
		logger.Error(err, "Failed to ensure Zitadel masterkey secret")
		return "", err
	}
	logger.Info("Created masterkey secret", "name", masterkeySecretName)

	return masterkeySecretName, nil
}

// waitForPostgreSQLReady waits for the PostgreSQL database to be available.
// It uses exponential backoff to retry connections until success or timeout.
func (r *ProjectReconciler) waitForPostgreSQLReady(ctx context.Context, project *edgev1alpha1.Project) error {
	logger := log.FromContext(ctx)
	pgSuperuserSecretName := fmt.Sprintf("%s-pguser-postgres", project.Name)

	// Get PostgreSQL admin connection secret
	pgSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      pgSuperuserSecretName,
		Namespace: project.Namespace,
	}, pgSecret); err != nil {
		return fmt.Errorf("failed to get PostgreSQL secret %s: %w", pgSuperuserSecretName, err)
	}

	// Create connection string
	serviceName := fmt.Sprintf("%s-postgres-postgresql-primary.%s.svc.cluster.local",
		project.Name, project.Namespace)
	password := string(pgSecret.Data["PGPASSWORD"])
	connString := fmt.Sprintf("host=%s port=5432 user=postgres password=%s dbname=postgres sslmode=require",
		serviceName, password)

	conn, err := r.connectWithRetry(ctx, connString)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close(ctx)
	}()

	logger.Info("PostgreSQL is ready", "database", "postgres")
	return nil
}

// connectWithRetry attempts to connect to PostgreSQL with exponential backoff.
func (r *ProjectReconciler) connectWithRetry(ctx context.Context, connString string) (*pgx.Conn, error) {
	var conn *pgx.Conn
	var err error

	// Configure retry parameters
	maxRetries := 10
	initialBackoff := time.Second
	maxBackoff := 30 * time.Second
	backoff := initialBackoff

	logger := log.FromContext(ctx)
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Try to connect
		conn, err = pgx.Connect(ctx, connString)
		if err == nil {
			logger.Info("Successfully connected to PostgreSQL", "attempt", attempt)
			return conn, nil
		}

		// If this was our last attempt, return the error
		if attempt == maxRetries {
			logger.Error(err, "Failed to connect to PostgreSQL after maximum retries",
				"maxRetries", maxRetries)
			return nil, fmt.Errorf("failed to connect to PostgreSQL after %d attempts: %w",
				maxRetries, err)
		}

		// Log the failure and prepare to retry
		logger.Info("Failed to connect to PostgreSQL, retrying",
			"attempt", attempt,
			"maxRetries", maxRetries,
			"backoff", backoff.String(),
			"error", err.Error())

		// Wait before next attempt using exponential backoff
		select {
		case <-time.After(backoff):
			// Increase backoff for next attempt, but don't exceed maxBackoff
			backoff = time.Duration(math.Min(float64(backoff*2), float64(maxBackoff)))
		case <-ctx.Done():
			// Context was canceled or timed out
			return nil, fmt.Errorf("context canceled while connecting to PostgreSQL: %w", ctx.Err())
		}
	}

	// This should never be reached due to the return in the loop above, but just in case
	return nil, fmt.Errorf("failed to connect to PostgreSQL: unexpected exit from retry loop")
}

// ensureZitadelMasterkeySecret creates or updates the Zitadel masterkey secret.
func (r *ProjectReconciler) ensureZitadelMasterkeySecret(ctx context.Context,
	project *edgev1alpha1.Project, secretName string) error {
	// Check if secret exists
	existingSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: project.Namespace}, existingSecret)

	// Generate masterkey if it doesn't exist
	masterkey := rand.NewPassword(32)

	if err == nil {
		// Secret exists, check if masterkey is present
		if mk, exists := existingSecret.Data["masterkey"]; exists && len(mk) > 0 {
			return nil // Masterkey already exists, no need to update
		}

		// Update the secret with the masterkey
		if existingSecret.Data == nil {
			existingSecret.Data = make(map[string][]byte)
		}
		existingSecret.Data["masterkey"] = []byte(masterkey)
		return r.Update(ctx, existingSecret)
	} else if errors.IsNotFound(err) {
		// Create a new secret
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: project.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: project.APIVersion,
						Kind:       project.Kind,
						Name:       project.Name,
						UID:        project.UID,
						Controller: ptr.To(true),
					},
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"masterkey": []byte(masterkey),
			},
		}

		return r.Create(ctx, secret)
	}

	return err
}

// verifyZitadelMasterkeySecret verifies that a Zitadel masterkey secret exists and has the required key.
func (r *ProjectReconciler) verifyZitadelMasterkeySecret(ctx context.Context,
	namespace, secretName string) error {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret); err != nil {
		return fmt.Errorf("masterkey secret %s not found: %w", secretName, err)
	}

	// Check for required key
	if masterkey, exists := secret.Data["masterkey"]; !exists || len(masterkey) == 0 {
		return fmt.Errorf("Zitadel masterkey secret %s is missing required key: masterkey", secretName)
	}

	return nil
}

// updateZitadelValuesWithMasterkey updates the Zitadel values to reference the masterkey secret.
func (r *ProjectReconciler) updateZitadelValuesWithMasterkey(values map[string]any, secretName string) {
	// Ensure the zitadel section exists
	zitadel, ok := values["zitadel"].(map[string]any)
	if !ok {
		zitadel = make(map[string]any)
		values["zitadel"] = zitadel
	}

	// Set the masterkey secret name
	zitadel["masterkeySecretName"] = secretName

	// Ensure masterkey is not set as it would cause conflict
	delete(zitadel, "masterkey")
}

// verifyExternalIdPSecret verifies that an external IdP secret exists and has the required keys.
func (r *ProjectReconciler) verifyExternalIdPSecret(ctx context.Context,
	project *edgev1alpha1.Project, name string, ref *edgev1alpha1.ComponentRef) error {
	// External IdP secret verification would be implemented here
	return nil
}

func (r *ProjectReconciler) ensureZitadelInstanceSecret(ctx context.Context, project *edgev1alpha1.Project) error {
	firstInstanceSecretName := fmt.Sprintf("%s-zitadel-firstinstance", project.Name)

	// Check if the secret already exists
	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: firstInstanceSecretName, Namespace: project.Namespace}, existing)

	if errors.IsNotFound(err) {
		// Secret doesn't exist, create it
		firstInstanceSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      firstInstanceSecretName,
				Namespace: project.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: project.APIVersion,
						Kind:       project.Kind,
						Name:       project.Name,
						UID:        project.UID,
						Controller: ptr.To(true),
					},
				},
			},
			Type: corev1.SecretTypeOpaque,
			StringData: map[string]string{
				"ZITADEL_FIRSTINSTANCE_ORG_HUMAN_PASSWORD": rnd.NewPassword(16),
			},
		}
		return r.Create(ctx, firstInstanceSecret)
	} else if err != nil {
		// Handle other errors
		return err
	}

	// Secret already exists, no need to create
	return nil
}

// ensureZitadelPostgresSecretAndRole ensures a Zitadel PostgreSQL user exists and creates/updates the corresponding secret
func (r *ProjectReconciler) ensureZitadelPostgresSecretAndRole(ctx context.Context, project *edgev1alpha1.Project, secretName string) error {
	logger := log.FromContext(ctx)
	// Create a new context with timeout for database operations
	dbCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 1. Get the PostgreSQL superuser secret
	pgSuperuserSecret := &corev1.Secret{}
	pgSuperuserSecretName := fmt.Sprintf("%s-pguser-postgres", project.Name)
	if err := r.Get(ctx, types.NamespacedName{
		Name:      pgSuperuserSecretName,
		Namespace: project.Namespace}, pgSuperuserSecret); err != nil {
		return fmt.Errorf("failed to get PostgreSQL superuser secret: %w", err)
	}

	// Validate required secret data exists
	requiredFields := []string{"PGHOST", "PGPORT", "PGPASSWORD"}
	for _, field := range requiredFields {
		if _, exists := pgSuperuserSecret.Data[field]; !exists {
			return fmt.Errorf("PostgreSQL superuser secret missing required field: %s", field)
		}
	}

	// 2. Extract connection info
	pgHost := string(pgSuperuserSecret.Data["PGHOST"])
	pgPort := string(pgSuperuserSecret.Data["PGPORT"])
	pgSuperPassword := string(pgSuperuserSecret.Data["PGPASSWORD"])

	// Define Zitadel user credentials
	zitadelUser := "zitadel"
	zitadelPassword := newAlphaNumericPassword(16)
	zitadelDatabase := "main"

	// 3. Build connection strings
	serviceName := fmt.Sprintf("%s-postgres-postgresql-primary.%s.svc.cluster.local",
		project.Name, project.Namespace)

	// Connection string for superuser
	superUserConnString := fmt.Sprintf("host=%s port=%s user=postgres password=%s dbname=postgres sslmode=require",
		serviceName, pgPort, pgSuperPassword)

	// Connection string for Zitadel user
	zitadelConnString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		pgHost, pgPort, zitadelUser, zitadelPassword, zitadelDatabase)

	// 4. Prepare secret data
	secretData := map[string][]byte{
		"PGDATABASE":  []byte(zitadelDatabase),
		"PGHOST":      []byte(pgHost),
		"PGPASSWORD":  []byte(zitadelPassword),
		"PGPORT":      []byte(pgPort),
		"PGSSLMODE":   []byte("require"),
		"PGUSER":      []byte(zitadelUser),
		"conn-string": []byte(zitadelConnString),
	}

	// 5. Check if the Zitadel PostgreSQL secret already exists and handle accordingly
	zitadelPgSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: project.Namespace}, zitadelPgSecret)

	// Handle secret creation/update
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("error getting Zitadel PostgreSQL secret: %w", err)
		}

		// Secret doesn't exist, create it
		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: project.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: project.APIVersion,
						Kind:       project.Kind,
						Name:       project.Name,
						UID:        project.UID,
						Controller: ptr.To(true),
					},
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: secretData,
		}
		if err := r.Create(ctx, newSecret); err != nil {
			return fmt.Errorf("failed to create Zitadel PostgreSQL secret: %w", err)
		}
		logger.Info("Created Zitadel PostgreSQL secret", "name", secretName, "namespace", project.Namespace)
	} else {
		// Secret exists, update it if needed
		zitadelPgSecret.Data = secretData
		if err := r.Update(ctx, zitadelPgSecret); err != nil {
			return fmt.Errorf("failed to update Zitadel PostgreSQL secret: %w", err)
		}
		logger.Info("Updated Zitadel PostgreSQL secret", "name", secretName, "namespace", project.Namespace)
	}

	// 6. Connect to PostgreSQL with retry
	pool, err := pgConnectWithRetry(dbCtx, superUserConnString, 5, 2*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer pool.Close()

	// 7. Create or update the PostgreSQL role
	zitadelPGRole := role.Role{
		Name:      zitadelUser,
		CanLogin:  true,
		Password:  zitadelPassword,
		CreateDB:  true, // Allow to create its own database
		Inherit:   true,
		ConnLimit: 100, // Set a reasonable connection limit
	}

	existingRole, err := role.Get(dbCtx, pool, zitadelPGRole.Name)
	if err != nil {
		if err == role.ErrRoleNotFound {
			// Role doesn't exist, create it
			if err := role.Create(dbCtx, pool, zitadelPGRole); err != nil {
				return fmt.Errorf("failed to create Zitadel PostgreSQL role: %w", err)
			}
			logger.Info("Created Zitadel PostgreSQL role", "name", zitadelUser)
		} else {
			return fmt.Errorf("error checking for existing PostgreSQL role: %w", err)
		}
	} else {
		// Role exists, update it
		logger.Info("Existing role found, updating", "name", existingRole.Name, "oid", existingRole.OID)
		if err := role.Update(dbCtx, pool, zitadelPGRole); err != nil {
			return fmt.Errorf("failed to update Zitadel PostgreSQL role: %w", err)
		}
		logger.Info("Updated Zitadel PostgreSQL role", "name", zitadelUser)
	}

	// 8. Ensure database exists
	if _, err := pool.Exec(dbCtx, "CREATE DATABASE main;"); err != nil {
		// Check for specific PostgreSQL error code for "database already exists" (42P04)
		pgErr, isPgError := err.(*pgconn.PgError)
		if isPgError && pgErr.Code == "42P04" {
			logger.Info("Database already exists", "database", "main")
		} else {
			return fmt.Errorf("failed to create database: %w", err)
		}
	} else {
		logger.Info("Created database", "database", "main")
	}

	return nil
}

// Add this helper function to your controller package
func newAlphaNumericPassword(length int) string {
	// Generate a password with 3x the length to ensure we have enough characters after filtering
	rawPassword := rand.NewPassword(length * 3)

	// Filter to only alphanumeric characters
	var filtered []rune
	for _, char := range rawPassword {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') {
			filtered = append(filtered, char)
		}
	}

	// Ensure we have enough characters after filtering
	if len(filtered) < length {
		// Fallback to a simple alphanumeric generation if we don't have enough characters
		return newAlphaNumericPassword(length)
	}

	// Return only the requested length
	return string(filtered[:length])
}

// // updateZitadelValuesWithPgSecret updates the Zitadel values to reference the PostgreSQL secret.
// func (r *ProjectReconciler) updateZitadelValuesWithPgSecret(values map[string]any, pgSecretName string) {
// 	// Ensure env array exists
// 	env, ok := values["env"].([]any)
// 	if !ok {
// 		env = make([]any, 0)
// 	}

// 	// Add PostgreSQL connection environment variables
// 	pgEnvVars := []map[string]any{
// 		{
// 			"name": "ZITADEL_DATABASE_POSTGRES_HOST",
// 			"valueFrom": map[string]any{
// 				"secretKeyRef": map[string]any{
// 					"name": pgSecretName,
// 					"key":  "PGHOST",
// 				},
// 			},
// 		},
// 		{
// 			"name": "ZITADEL_DATABASE_POSTGRES_PORT",
// 			"valueFrom": map[string]any{
// 				"secretKeyRef": map[string]any{
// 					"name": pgSecretName,
// 					"key":  "PGPORT",
// 				},
// 			},
// 		},
// 		{
// 			"name": "ZITADEL_DATABASE_POSTGRES_USER",
// 			"valueFrom": map[string]any{
// 				"secretKeyRef": map[string]any{
// 					"name": pgSecretName,
// 					"key":  "PGUSER",
// 				},
// 			},
// 		},
// 		{
// 			"name": "ZITADEL_DATABASE_POSTGRES_PASSWORD",
// 			"valueFrom": map[string]any{
// 				"secretKeyRef": map[string]any{
// 					"name": pgSecretName,
// 					"key":  "PGPASSWORD",
// 				},
// 			},
// 		},
// 		{
// 			"name": "ZITADEL_DATABASE_POSTGRES_DATABASE",
// 			"valueFrom": map[string]any{
// 				"secretKeyRef": map[string]any{
// 					"name": pgSecretName,
// 					"key":  "PGDATABASE",
// 				},
// 			},
// 		},
// 		{
// 			"name": "ZITADEL_DATABASE_POSTGRES_SSL_MODE",
// 			"valueFrom": map[string]any{
// 				"secretKeyRef": map[string]any{
// 					"name": pgSecretName,
// 					"key":  "PGSSLMODE",
// 				},
// 			},
// 		},
// 	}

// 	// Add the PostgreSQL environment variables
// 	for _, pgEnvVar := range pgEnvVars {
// 		// Check if the environment variable already exists
// 		exists := false
// 		for i, existingEnvVar := range env {
// 			if existingVar, ok := existingEnvVar.(map[string]any); ok &&
// 				existingVar["name"] == pgEnvVar["name"] {
// 				// Update the existing environment variable
// 				env[i] = pgEnvVar
// 				exists = true
// 				break
// 			}
// 		}

// 		// Add the environment variable if it doesn't exist
// 		if !exists {
// 			env = append(env, pgEnvVar)
// 		}
// 	}

// 	// Update the env in the values
// 	values["env"] = env
// }
