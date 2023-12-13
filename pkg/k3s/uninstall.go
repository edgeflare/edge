package k3s

import (
	"github.com/edgeflare/edge/pkg/db"
	"github.com/edgeflare/edge/pkg/ssh"
)

// BuildUninstallCommand builds the command to uninstall k3s from the remote machine.
func buildUninstallCommand(agent bool) string {
	if agent {
		return "/usr/local/bin/k3s-agent-uninstall.sh"
	}
	return "/usr/local/bin/k3s-uninstall.sh"
}

// UninstallK3s uninstalls K3s from host
func (s *Service) UninstallK3s(output ssh.OutputWriter, agent bool) (string, error) {
	uninstallCmd := buildUninstallCommand(agent)

	if err := s.Exec(output, uninstallCmd); err != nil {
		return "", err
	}

	node, err := db.SelectNodeByIP(s.DB, s.sshClient.Host)

	if err != nil {
		return "", err
	}

	cluster, err := db.SelectCluster(s.DB, node.ClusterID)

	if err != nil {
		return "", err
	}

	if err = db.DeleteNode(s.DB, node.ID); err != nil {
		return "", err
	}

	clusterNodes, err := db.SelectNodesByCluster(s.DB, cluster.ID)
	if err != nil {
		return "", err
	}

	if len(clusterNodes) == 0 {
		if err := db.DeleteCluster(s.DB, cluster.ID); err != nil {
			return "", err
		}
	}

	return node.ID, nil

}
