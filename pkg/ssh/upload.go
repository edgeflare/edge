package ssh

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// UploadFile uploads a file to the remote server
func (c *Client) UploadFile(localFilePath, remoteFilePath string, sudo bool) error {
	// Connect to the SSH server
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), c.Config)
	if err != nil {
		return fmt.Errorf("failed to dial SSH: %w", err)
	}
	defer client.Close()

	// Start an SFTP session
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("failed to start SFTP session: %w", err)
	}
	defer sftpClient.Close()

	// Open the local file
	localFile, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()

	var tempPath string
	if sudo {
		// Use a temporary path if sudo is required
		tempPath = "/tmp/tempfile"
	} else {
		// Use the original file path if no sudo is required
		tempPath = remoteFilePath
	}

	// Create the remote file
	remoteFile, err := sftpClient.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}
	defer remoteFile.Close()

	// Copy the file contents from local to remote
	if _, err := io.Copy(remoteFile, localFile); err != nil {
		return fmt.Errorf("failed to copy file from local to remote: %w", err)
	}

	if sudo {
		// Move the file to the desired location with sudo
		moveCmd := fmt.Sprintf("sudo mv %s %s", tempPath, remoteFilePath)
		if err := c.Exec(&Command{Path: moveCmd}); err != nil {
			return fmt.Errorf("failed to move file with sudo: %w", err)
		}
	}

	return nil
}
