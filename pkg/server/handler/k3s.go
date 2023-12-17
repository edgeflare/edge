package handler

import (
	"net/http"

	"github.com/edgeflare/edge/pkg/k3s"
	"github.com/edgeflare/edge/pkg/ssh"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// K3sInstallRequest is the request payload for K3s install
type K3sInstallRequest struct {
	SSHClient          *ssh.Client `json:"ssh" validate:"required"`
	TLSSAN             string      `json:"tls_san,omitempty" validate:"omitempty,hostname"`
	K3sArgs            string      `json:"k3s_args,omitempty"`
	Version            string      `json:"version,omitempty"`
	Cluster            bool        `json:"cluster,omitempty"`
	DownloadKubeconfig bool        `json:"KUBECONFIG,omitempty"`
	// NodeExternalIP string      `json:"node_external_ip" validate:"omitempty,ip"`
}

// K3sUninstallRequest is the request payload for K3s uninstall
type K3sUninstallRequest struct {
	SSHClient *ssh.Client `json:"ssh" validate:"required"`
	Agent     bool        `json:"agent,omitempty"`
}

type K3sJoinRequest struct {
	SSHClient *ssh.Client `json:"ssh" validate:"required"`
	Server    string      `json:"server,omitempty"`
	Token     string      `json:"token,omitempty"`
	Master    bool        `json:"master,omitempty"`
}

// EchoOutputHandler is a struct to handle the output from SSH
type EchoOutputHandler struct {
	ctx echo.Context
}

// Write method to write the output
func (e EchoOutputHandler) Write(p []byte) (n int, err error) {
	return e.ctx.Response().Write(p) // Stream to Echo's response writer
}

// Flush method to ensure all data is sent
func (e EchoOutputHandler) Flush() {
	e.ctx.Response().Flush()
}

// K3sInstall installs K3s
func K3sInstall(c echo.Context) error {
	setCommonHeaders(c)

	req := new(K3sInstallRequest)
	if err := bindAndValidateRequest(c, req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	sshClient, err := createSSHClient(req.SSHClient)
	if err != nil {
		zap.L().Error("error creating SSH client", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	k3sService := k3s.NewK3sService(sshClient)

	clusterID, err := k3sService.InstallK3s(&EchoOutputHandler{ctx: c}, req.Cluster, req.TLSSAN, req.K3sArgs, req.Version)
	if err != nil {
		zap.L().Error("error installing cluster", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := k3sService.DownloadK3sKubeconfig(); err != nil {
		zap.L().Error("error downloading kubeconfig", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if req.DownloadKubeconfig {
		if err := k3sService.SetKubeconfig(); err != nil {
			zap.L().Error("error setting kubeconfig", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"cluster_id": clusterID})
}

// K3sUninstall uninstalls K3s
func K3sUninstall(c echo.Context) error {
	setCommonHeaders(c)

	req := new(K3sUninstallRequest)
	if err := bindAndValidateRequest(c, req); err != nil {
		return err
	}

	sshClient, err := createSSHClient(req.SSHClient)
	if err != nil {
		zap.L().Error("error creating SSH client", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	k3sService := k3s.NewK3sService(sshClient)

	nodeID, err := k3sService.UninstallK3s(&EchoOutputHandler{ctx: c}, req.Agent)
	if err != nil {
		zap.L().Error("error downloading kubeconfig", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"uninstalled": nodeID})
}

// K3sJoin joins a node to a K3s cluster
func K3sJoin(c echo.Context) error {
	setCommonHeaders(c)

	req := new(K3sJoinRequest)
	if err := bindAndValidateRequest(c, req); err != nil {
		return err
	}

	sshClient, err := createSSHClient(req.SSHClient)
	if err != nil {
		zap.L().Error("error creating SSH client", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	k3sService := k3s.NewK3sService(sshClient)
	nodeID, err := k3sService.JoinK3s(&EchoOutputHandler{ctx: c}, req.Server, req.Master, req.Token)

	if err != nil {
		zap.L().Error("error building join command", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"joined": nodeID})
}

// Set common HTTP headers
func setCommonHeaders(c echo.Context) {
	c.Response().Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Response().Header().Set("Transfer-Encoding", "chunked")
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
}

// Bind and validate request
func bindAndValidateRequest(c echo.Context, req interface{}) error {
	if err := c.Bind(req); err != nil {
		zap.L().Error("Invalid request payload", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := validate.Struct(req); err != nil {
		zap.L().Error("request payload validation failed", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return nil
}

// Create SSH client
func createSSHClient(sshClient *ssh.Client) (*ssh.Client, error) {
	return ssh.NewSSHClient(
		sshClient.Host,
		sshClient.User,
		sshClient.Password,
		sshClient.Keyfile,
		sshClient.Port,
		sshClient.KeyPassphrase,
	)
}

// ListClusters lists K3s clusters
func ListClusters(c echo.Context) error {
	k3sService := k3s.NewK3sService(nil)

	clusters, err := k3sService.ListClusters()
	if err != nil {
		zap.L().Error("error listing clusters", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Return clusters as JSON
	return c.JSON(http.StatusOK, clusters)
}

// ListNodes lists K3s nodes in a cluster
func ListNodes(c echo.Context) error {
	k3sService := k3s.NewK3sService(nil)
	clusterID := c.Param("clusterId") // Or get it from path parameter

	nodes, err := k3sService.ListNodesByCluster(clusterID)
	if err != nil {
		zap.L().Error("error listing nodes", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Return nodes as JSON
	return c.JSON(http.StatusOK, nodes)
}

// K3sStableVersions lists K3s stable versions
func K3sStableVersions(c echo.Context) error {
	versions, err := k3s.GetLatestK3sVersions()
	if err != nil {
		zap.L().Error("error fetching K3s versions", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Return versions as JSON
	return c.JSON(http.StatusOK, versions)
}
