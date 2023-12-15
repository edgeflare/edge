package k3s

import (
	"database/sql"

	"github.com/edgeflare/edge/pkg/db"
	"github.com/edgeflare/edge/pkg/ssh"
	"go.uber.org/zap"
)

// Service is the service for managing k3s cluster nodes using SSH.
type Service struct {
	sshClient *ssh.Client
	DB        *sql.DB
}

// NewK3sService provides functionalities for managing k3s cluster nodes using SSH.
func NewK3sService(client *ssh.Client) *Service {
	return &Service{sshClient: client, DB: db.GetDB()}
}

// Exec executes a given SSH command on the remote machine.
// It takes an OutputWriter to handle the output of the SSH command, and a cmdStr
// which is the string representation of the command to be executed.
func (s *Service) Exec(output ssh.OutputWriter, cmdStr string) error {
	cmd := &ssh.Command{
		Path:   cmdStr,
		Stdout: output,
		Stderr: output,
	}
	return s.sshClient.Exec(cmd)
}

// ListClusters returns a list of all clusters in the database.
func (s *Service) ListClusters() ([]db.K3sCluster, error) {
	// This method encapsulates the logic to fetch all clusters
	// And update the status of each cluster
	clusters, err := db.SelectClusters(s.DB)
	if err != nil {
		return nil, err
	}

	for i := range clusters {
		status := getNodeStatus(clusters[i].Apiserver, 6443)

		if clusters[i].Status == status {
			continue
		}

		clusters[i].Status = status
		err := db.UpdateCluster(s.DB, clusters[i])
		if err != nil {
			zap.L().Error("error updating cluster status", zap.Error(err))
			return nil, err
		}
	}

	return clusters, nil
}

// ListNodesByCluster returns a list of all nodes in a cluster.
func (s *Service) ListNodesByCluster(clusterID string) ([]db.K3sNode, error) {
	// This method encapsulates the logic to fetch nodes by cluster ID
	nodes, err := db.SelectNodesByCluster(s.DB, clusterID)
	if err != nil {
		return nil, err
	}

	for i := range nodes {
		status := getNodeStatus(nodes[i].IP, 10250)

		if nodes[i].Status == status {
			continue
		}

		nodes[i].Status = status
		err := db.UpdateNode(s.DB, nodes[i])
		if err != nil {
			zap.L().Error("error updating node status", zap.Error(err))
			return nil, err
		}
	}

	return nodes, nil
}
