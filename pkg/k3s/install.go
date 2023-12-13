package k3s

import (
	"fmt"
	"strings"
	"time"

	"github.com/edgeflare/edge/pkg/db"
	"github.com/edgeflare/edge/pkg/ssh"
	"go.uber.org/zap"
)

// InstallCluster installs a K3s cluster on the remote host
func (s *Service) InstallK3s(output ssh.OutputWriter, initCluster bool, tlsSan, k3sArgs, version string) (string, error) {
	// Fetch the latest version if not provided
	if version == "" {
		versions, err := GetLatestK3sVersions()
		if err != nil || len(versions) == 0 {
			zap.L().Error("error fetching K3s versions", zap.Error(err))
			return "", err
		}
		version = versions[0] // Use the latest stable version
	}

	cmdStr := buildInstallCommand(initCluster, tlsSan, s.sshClient.Host, k3sArgs, version)
	err := s.Exec(output, cmdStr)
	if err != nil {
		zap.L().Error("error installing K3s", zap.Error(err))
		return "", err
	}

	clusterID := db.GenerateID() // Function to generate a unique cluster ID
	newCluster := db.K3sCluster{ID: clusterID, Status: "Running", Version: version, IsHA: initCluster, CreatedAt: time.Now(), UpdatedAt: time.Now(), Apiserver: s.sshClient.Host}
	if err := db.InsertCluster(s.DB, newCluster); err != nil {
		zap.L().Error("error inserting cluster", zap.Error(err))
		return "", err
	}

	newNode := db.K3sNode{ID: db.GenerateID(), ClusterID: clusterID, IP: s.sshClient.Host, Role: "server", Status: "Running", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := db.InsertNode(s.DB, newNode); err != nil {
		zap.L().Error("error inserting node", zap.Error(err))
		return "", err
	}

	zap.L().Info("K3s installed successfully", zap.String("version", version))

	return clusterID, nil
}

// BuildInstallCommand builds the command to install k3s on the remote machine.
func buildInstallCommand(initCluster bool, tlsSan, nodeExternalIP, k3sArgs, version string) string {
	var flags []string

	if initCluster {
		flags = append(flags, "--cluster-init")
	}
	if tlsSan != "" {
		flags = append(flags, fmt.Sprintf("--tls-san=%s", tlsSan))
	}
	if nodeExternalIP != "" {
		flags = append(flags, fmt.Sprintf("--node-external-ip=%s", nodeExternalIP))
	}
	if k3sArgs != "" {
		flags = append(flags, k3sArgs)
	}

	installCmd := "INSTALL_K3S_EXEC=\"server %s\" bash -"
	if version != "" {
		installCmd = fmt.Sprintf("INSTALL_K3S_VERSION=\"%s\" %s", version, installCmd)
	}

	return fmt.Sprintf(`curl -sfL https://get.k3s.io | `+installCmd, strings.Join(flags, " "))
}
