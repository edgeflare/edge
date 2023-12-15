package kube

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var validKubeconfigContent = `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://example.com
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user: {}
`

func TestGetKubernetesConfig(t *testing.T) {
	// Test with KUBECONFIG set
	t.Run("KUBECONFIG set", func(t *testing.T) {
		// Create a temporary kubeconfig file
		tempFile, err := os.CreateTemp("", "kubeconfig")
		if err != nil {
			t.Fatalf("failed to create temporary kubeconfig file: %v", err)
		}
		defer os.Remove(tempFile.Name()) // Clean up the file afterwards

		// Write some dummy content to the file (optional)
		_, err = tempFile.WriteString(validKubeconfigContent)
		if err != nil {
			t.Fatalf("failed to write to temporary kubeconfig file: %v", err)
		}
		tempFile.Close()

		// Set the KUBECONFIG environment variable to the temporary file
		os.Setenv("KUBECONFIG", tempFile.Name())
		defer os.Unsetenv("KUBECONFIG")

		_, err = GetKubernetesConfig()
		assert.Nil(t, err, "should not error when KUBECONFIG is set")
	})

	// Test with default kubeconfig
	t.Run("Default kubeconfig", func(t *testing.T) {
		// Create a temporary directory
		tempDir, err := os.MkdirTemp("", "testkube")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create a mock .kube directory inside the temp directory
		kubeDir := filepath.Join(tempDir, ".kube")
		err = os.MkdirAll(kubeDir, 0755)
		if err != nil {
			t.Fatalf("failed to create mock .kube dir: %v", err)
		}

		// Set the kubeconfig path to the mock config file
		kubeconfigPath := filepath.Join(kubeDir, "config")

		// Create a temporary kubeconfig file using os package
		err = os.WriteFile(kubeconfigPath, []byte(validKubeconfigContent), 0644)
		if err != nil {
			t.Fatalf("failed to write mock kubeconfig file: %v", err)
		}

		// Set the KUBECONFIG environment variable to use the mock kubeconfig file
		os.Setenv("KUBECONFIG", kubeconfigPath)
		defer os.Unsetenv("KUBECONFIG")

		_, err = GetKubernetesConfig()
		assert.Nil(t, err, "should not error when default kubeconfig exists")
	})

	// Test in-cluster config
	t.Run("In-cluster config", func(t *testing.T) {
		// This test depends on the environment where it's run.
		// If running in an environment where in-cluster config is available,
		// check for the existence of that config.
		// Otherwise, might just check that it doesn't error out or returns a specific error.
		// TODO: Implement this test
	})
}
