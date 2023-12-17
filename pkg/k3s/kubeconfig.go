package k3s

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/edgeflare/edge/pkg/ssh"
)

// DownloadK3sKubeconfig downloads the K3s kubeconfig file
func (s *Service) DownloadK3sKubeconfig() error {
	client, err := ssh.NewSSHClient(s.sshClient.Host, s.sshClient.User, s.sshClient.Password, s.sshClient.Keyfile, s.sshClient.Port, s.sshClient.KeyPassphrase)
	if err != nil {
		return fmt.Errorf("error creating SSH client: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting user home directory: %w", err)
	}

	kubeDir := filepath.Join(home, ".kube")
	if err := ensureDir(kubeDir); err != nil {
		return fmt.Errorf("error ensuring .kube directory exists: %w", err)
	}

	kubeconfigPath := filepath.Join(kubeDir, s.sshClient.Host+".config")

	if err := client.DownloadFile("/etc/rancher/k3s/k3s.yaml", kubeconfigPath, true); err != nil {
		return fmt.Errorf("error downloading file: %w", err)
	}

	if err := modifyKubeconfig(kubeconfigPath, s.sshClient.Host); err != nil {
		return fmt.Errorf("error modifying kubeconfig: %w", err)
	}

	fmt.Println("Kubeconfig saved to", kubeconfigPath)

	return nil
}

func ensureDir(dirName string) error {
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		return os.MkdirAll(dirName, 0755) // Using 0755 permissions for directory
	}
	return nil
}

func modifyKubeconfig(filePath, host string) error {
	fileContents, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	newContents := strings.ReplaceAll(string(fileContents), "server: https://127.0.0.1:6443", "server: https://"+host+":6443")

	return os.WriteFile(filePath, []byte(newContents), 0644)
}

// SetKubeconfig sets the KUBECONFIG environment variable
func (s *Service) SetKubeconfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting user home directory: %w", err)
	}

	kubeDir := filepath.Join(home, ".kube")
	if err := ensureDir(kubeDir); err != nil {
		return fmt.Errorf("error ensuring .kube directory exists: %w", err)
	}

	kubeconfigPath := filepath.Join(kubeDir, s.sshClient.Host+".config")
	os.Setenv("KUBECONFIG", kubeconfigPath)
	fmt.Println("KUBECONFIG set to ", kubeconfigPath)

	return nil
}
