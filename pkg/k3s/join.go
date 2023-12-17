package k3s

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/edgeflare/edge/pkg/db"
	"github.com/edgeflare/edge/pkg/ssh"
	"go.uber.org/zap"
)

// JoinK3s joins a node to a k3s cluster.
func (s *Service) JoinK3s(output ssh.OutputWriter, server string, master bool, token string) (string, error) {
	var err error

	// Download the token if it's not provided
	if token == "" {
		token, err = s.downloadToken(server)
		if err != nil {
			zap.L().Error("error downloading token", zap.String("server", server), zap.Error(err))
			return "", fmt.Errorf("error downloading token: %w", err)
		}
	}

	joinCmdStr := fmt.Sprintf(`curl -sfL https://get.k3s.io | K3S_URL="https://%s:6443" K3S_TOKEN="%s" sh -s - --node-external-ip "%s"`, server, token, s.sshClient.Host)

	if err = s.Exec(output, joinCmdStr); err != nil {
		zap.L().Error("error executing join command", zap.String("command", joinCmdStr), zap.Error(err))
		return "", err
	}

	// Node and cluster logic here...
	node, err := db.SelectNodeByIP(s.DB, server)
	if err != nil {
		return "", err
	}

	cluster, err := db.SelectCluster(s.DB, node.ClusterID)
	if err != nil {
		return "", err
	}
	cluster.UpdatedAt = time.Now()
	if err = db.UpdateCluster(s.DB, *cluster); err != nil {
		return "", err
	}

	nodeRole := "agent"
	if master {
		nodeRole = "server"
	}
	newNode := db.K3sNode{ID: db.GenerateID(), ClusterID: cluster.ID, IP: s.sshClient.Host, Role: nodeRole, Status: "Running", CreatedAt: node.CreatedAt, UpdatedAt: node.UpdatedAt}

	if err := db.InsertNode(s.DB, newNode); err != nil {
		zap.L().Error("error inserting node", zap.Error(err))
		return "", err
	}

	zap.L().Info("node joined successfully", zap.String("node_id", newNode.ID))

	return newNode.ID, nil
}

// DownloadToken downloads the token from the given server.
func (s *Service) downloadToken(server string) (string, error) {
	tokenPath := "/var/lib/rancher/k3s/server/node-token"
	localPath := server + "-node-token"

	sshClient, err := ssh.NewSSHClient(server, s.sshClient.User, s.sshClient.Password, s.sshClient.Keyfile, s.sshClient.Port, s.sshClient.KeyPassphrase)
	if err != nil {
		return "", fmt.Errorf("error creating SSH client: %w", err)
	}

	err = sshClient.DownloadFile(tokenPath, localPath, true)
	if err != nil {
		return "", fmt.Errorf("error downloading token file: %w", err)
	}

	// Read the token from the local file
	token, err := os.ReadFile(localPath)
	if err != nil {
		return "", fmt.Errorf("error reading token file: %w", err)
	}

	// Trim any whitespace from the token
	trimmedToken := strings.TrimSpace(string(token))

	if err := os.Remove(localPath); err != nil {
		return "", fmt.Errorf("error removing token file: %w", err)
	}

	return trimmedToken, nil
}
