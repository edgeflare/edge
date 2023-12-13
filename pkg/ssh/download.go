package ssh

import (
	"fmt"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// DownloadFile downloads a file from the remote server
func (c *Client) DownloadFile(remoteFilePath, localFilePath string, sudo bool) error {
	var tempPath string
	var err error

	if sudo {
		// Use sudo to create a temporary copy with read permissions
		tempPath = "/tmp/tempfile"
		copyCmd := fmt.Sprintf("sudo cp %s %s && sudo chmod 644 %s", remoteFilePath, tempPath, tempPath)
		if err = c.Exec(&Command{Path: copyCmd}); err != nil {
			return fmt.Errorf("failed to create temporary copy of file with sudo: %w", err)
		}
	} else {
		// Use the original file path if no sudo is required
		tempPath = remoteFilePath
	}

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

	// Open the remote file
	remoteFile, err := sftpClient.Open(tempPath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer remoteFile.Close()

	// Create the local file
	localFile, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	// Copy the file contents from remote to local
	_, err = remoteFile.WriteTo(localFile) // Use existing 'err' variable, not ':='
	if err != nil {
		return fmt.Errorf("failed to copy file from remote to local: %w", err)
	}

	if sudo {
		// Optionally, delete the temporary file after downloading
		deleteCmd := fmt.Sprintf("sudo rm %s", tempPath)
		if err = c.Exec(&Command{Path: deleteCmd}); err != nil {
			return fmt.Errorf("failed to delete temporary file: %w", err)
		}
	}

	return nil
}
