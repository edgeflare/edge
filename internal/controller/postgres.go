package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	edgev1alpha1 "github.com/edgeflare/edge/api/v1alpha1"
	"github.com/edgeflare/pgo/pkg/util/rand"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xeipuuv/gojsonschema"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

func (r *ProjectReconciler) reconcileDatabase(ctx context.Context, project *edgev1alpha1.Project,
	name string, ref *edgev1alpha1.ComponentRef) error {
	logger := log.FromContext(ctx)
	compType := "database"
	logger.Info("Reconciling database", "component", name)

	// For external database components, verify the secret exists and update component status
	if ref.IsExternal() {
		if err := r.verifyExternalDatabaseSecret(ctx, project, name, ref); err != nil {
			logger.Error(err, "External database secret verification failed")
			_ = r.updateComponentStatus(ctx, project, compType, name, false,
				fmt.Sprintf("Secret error: %v", err), "")
			return err
		}
		return r.handleExternalComponent(ctx, project, compType, name, ref)
	}

	if name == "postgres" {
		pgSchema, err := os.ReadFile("hack/postgresql.values.schema.json")
		if err != nil {
			logger.Error(err, "Failed to load PostgreSQL schema")
			return err
		}

		// Parse the supplied values
		var pgValues map[string]any
		if ref.Release.ValuesContent != "" {
			pgValues, err = r.parseValues(ref.Release.ValuesContent, pgSchema)
			if err != nil {
				logger.Error(err, "Failed to parse PostgreSQL values")
				_ = r.updateComponentStatus(ctx, project, compType, name, false,
					fmt.Sprintf("Values error: %v", err), "")
				return err
			}
		} else {
			pgValues = make(map[string]any)
		}

		// Check if existingSecret is set
		existingSecret := ""
		if global, ok := pgValues["global"].(map[string]any); ok {
			if postgresql, ok := global["postgresql"].(map[string]any); ok {
				if auth, ok := postgresql["auth"].(map[string]any); ok {
					if es, ok := auth["existingSecret"].(string); ok && es != "" {
						existingSecret = es
					}
				}
			}
		}

		// Generate or verify secret
		secretName := fmt.Sprintf("%s-postgresql", project.Name)
		if existingSecret != "" {
			secretName = existingSecret
			// Verify the existing secret
			if err := r.verifyPostgresSecret(ctx, project.Namespace, secretName); err != nil {
				logger.Error(err, "Existing PostgreSQL secret verification failed")
				_ = r.updateComponentStatus(ctx, project, compType, name, false,
					fmt.Sprintf("Secret error: %v", err), "")
				return err
			}
		} else {
			// Create or update the secret with generated passwords
			if err := r.ensurePostgresSecret(ctx, project, secretName); err != nil {
				logger.Error(err, "Failed to ensure PostgreSQL secret")
				return err
			}

			// Update the values to use created secret
			r.updateValuesWithSecret(pgValues, secretName)
		}

		// Create user connection secret
		if err := r.ensurePostgresUserSecret(ctx, project, secretName); err != nil {
			logger.Error(err, "Failed to ensure PostgreSQL user secret")
			return err
		}

		// Update the values content in the component ref
		updatedValues, err := yaml.Marshal(pgValues)
		if err != nil {
			logger.Error(err, "Failed to marshal updated PostgreSQL values")
			return err
		}

		ref.Release.ValuesContent = string(updatedValues)
	}

	return r.handleComponentRelease(ctx, project, compType, name, ref)
}

// Parse and validate YAML values against the schema
func (r *ProjectReconciler) parseValues(yamlString string, schemaBytes []byte) (map[string]any, error) {
	var yamlData map[string]any
	if err := yaml.Unmarshal([]byte(yamlString), &yamlData); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %v", err)
	}

	jsonData, err := json.Marshal(yamlData)
	if err != nil {
		return nil, fmt.Errorf("error converting to JSON: %v", err)
	}

	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("error during validation: %v", err)
	}

	if !result.Valid() {
		var errMsgs []string
		for _, desc := range result.Errors() {
			errMsgs = append(errMsgs, desc.String())
		}
		return nil, fmt.Errorf("invalid PostgreSQL values: %s", strings.Join(errMsgs, "; "))
	}

	return yamlData, nil
}

// Verify an existing PostgreSQL secret
func (r *ProjectReconciler) verifyPostgresSecret(ctx context.Context, namespace, secretName string) error {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret); err != nil {
		return err
	}

	// Check for required keys
	requiredKeys := []string{"postgres-password", "replication-password"}
	for _, key := range requiredKeys {
		if _, exists := secret.Data[key]; !exists {
			return fmt.Errorf("PostgreSQL secret %s is missing required key: %s", secretName, key)
		}
	}

	return nil
}

// Generate or update PostgreSQL secret
func (r *ProjectReconciler) ensurePostgresSecret(ctx context.Context, project *edgev1alpha1.Project, secretName string) error {
	// Check if secret exists
	existingSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: project.Namespace}, existingSecret)

	// Generate passwords if they don't exist
	pgPassword := rand.NewPassword(16)
	replPassword := rand.NewPassword(16)

	if err == nil {
		// Secret exists, check if passwords are present
		if pgPass, exists := existingSecret.Data["postgres-password"]; exists {
			pgPassword = string(pgPass)
		}
		if replPass, exists := existingSecret.Data["replication-password"]; exists {
			replPassword = string(replPass)
		}

		// Update the secret with any missing data
		existingSecret.Data["postgres-password"] = []byte(pgPassword)
		existingSecret.Data["replication-password"] = []byte(replPassword)

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
				"postgres-password":    []byte(pgPassword),
				"replication-password": []byte(replPassword),
			},
		}

		return r.Create(ctx, secret)
	}

	return err
}

// Create or update the user connection secret
func (r *ProjectReconciler) ensurePostgresUserSecret(ctx context.Context, project *edgev1alpha1.Project, authSecretName string) error {
	// First get the auth secret to extract the postgres password
	authSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: authSecretName, Namespace: project.Namespace}, authSecret); err != nil {
		return err
	}

	pgPassword := string(authSecret.Data["postgres-password"])
	if pgPassword == "" {
		return fmt.Errorf("postgres-password not found in secret %s", authSecretName)
	}

	// TODO: improve instead of hack
	serviceName := fmt.Sprintf("%s-postgres-postgresql-primary.%s.svc.cluster.local", project.Name, project.Namespace)
	connString := fmt.Sprintf("host=%s port=5432 user=postgres password=%s dbname=postgres sslmode=prefer",
		serviceName, pgPassword)

	// Create or update the user secret
	userSecretName := fmt.Sprintf("%s-pguser-postgres", project.Name)
	userSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: userSecretName, Namespace: project.Namespace}, userSecret)

	secretData := map[string][]byte{
		"PGDATABASE":  []byte("postgres"),
		"PGHOST":      []byte(serviceName),
		"PGPASSWORD":  []byte(pgPassword),
		"PGPORT":      []byte("5432"),
		"PGSSLMODE":   []byte("prefer"),
		"PGUSER":      []byte("postgres"),
		"conn-string": []byte(connString),
	}

	if err == nil {
		// Update existing secret
		userSecret.Data = secretData
		return r.Update(ctx, userSecret)
	} else if errors.IsNotFound(err) {
		// Create new secret
		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userSecretName,
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

		return r.Create(ctx, newSecret)
	}

	return err
}

// Update the values to use our secret
func (r *ProjectReconciler) updateValuesWithSecret(values map[string]any, secretName string) {
	// Ensure the global.postgresql.auth.existingSecret is set
	global, ok := values["global"].(map[string]any)
	if !ok {
		global = make(map[string]any)
		values["global"] = global
	}

	postgresql, ok := global["postgresql"].(map[string]any)
	if !ok {
		postgresql = make(map[string]any)
		global["postgresql"] = postgresql
	}

	auth, ok := postgresql["auth"].(map[string]any)
	if !ok {
		auth = make(map[string]any)
		postgresql["auth"] = auth
	}

	auth["existingSecret"] = secretName
}

func (r *ProjectReconciler) verifyExternalDatabaseSecret(ctx context.Context, project *edgev1alpha1.Project,
	name string, ref *edgev1alpha1.ComponentRef) error {
	_ = name

	// Skip if secret name is not provided
	secretName := ref.GetSecretName()
	if secretName == "" {
		return nil
	}

	// Check if secret exists
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: project.Namespace,
	}, secret)

	if errors.IsNotFound(err) {
		return fmt.Errorf("required external database secret %s not found", secretName)
	}

	if err != nil {
		return fmt.Errorf("failed to get external database secret: %w", err)
	}

	// Verify secret contains required libpq environment variables
	requiredFields := []string{"PGPASSWORD", "PGHOST", "PGPORT", "PGUSER", "PGDATABASE"}
	missingFields := []string{}

	for _, field := range requiredFields {
		if _, exists := secret.Data[field]; !exists {
			missingFields = append(missingFields, field)
		}
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("external database secret %s is missing required fields: %v",
			secretName, strings.Join(missingFields, ", "))
	}

	return nil
}

func pgConnectWithRetry(ctx context.Context, connString string, maxRetries int, retryInterval time.Duration) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		pool, err = pgxpool.New(ctx, connString)
		if err == nil {
			// Test the connection with a simple query
			if err = pool.Ping(ctx); err == nil {
				return pool, nil
			}
			// Close the pool if ping failed
			pool.Close()
		}

		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context canceled while connecting to PostgreSQL: %w", ctx.Err())
			case <-time.After(retryInterval):
				// Log retry attempt
				fmt.Println("Retrying PostgreSQL connection",
					"attempt", attempt,
					"maxRetries", maxRetries,
					"error", err.Error())
			}
		}
	}

	return nil, fmt.Errorf("failed to connect to PostgreSQL after %d attempts: %w", maxRetries, err)
}
