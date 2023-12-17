package ssh

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

// OutputWriter is an interface that wraps the Write method.
type OutputWriter interface {
	Write([]byte) (int, error)
}

// Client is a struct to hold the SSH client details
type Client struct {
	Config        *ssh.ClientConfig `json:"-"` // Internal use only
	connection    *ssh.Client       `json:"-"` // Internal use only
	Host          string            `json:"host" validate:"required,hostname|ipaddress"`
	User          string            `json:"user" validate:"required"`
	Password      string            `json:"password,omitempty"`
	Keyfile       string            `json:"keyfile,omitempty"`
	KeyPassphrase string            `json:"keypassphrase,omitempty"`
	Port          int               `json:"port,omitempty" validate:"omitempty,min=1,max=65535"`
}

// Command is a struct to hold the SSH command details
type Command struct {
	Stdin  io.Reader
	Stdout OutputWriter
	Stderr OutputWriter
	Path   string
	Env    []string
}

// Connect connects to the SSH server
func (c *Client) Connect() error {
	var err error
	c.connection, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), c.Config)
	return err
}

// Close closes the SSH connection
func (c *Client) Close() error {
	if c.connection != nil {
		return c.connection.Close()
	}
	return nil
}

// NewSSHClient returns a new SSH client instance
func NewSSHClient(host, user, password, keyfile string, port int, keypassphrase string) (*Client, error) {
	// Create the SSH client configuration
	config, err := NewSSHClientConfig(user, password, keyfile, keypassphrase)
	if err != nil {
		return nil, err
	}

	// Check if port is provided, if not, use the default SSH port (22)
	if port == 0 {
		port = 22
	}

	// Return a new Client instance with the provided details
	return &Client{
		Config:        config,
		Host:          host,
		User:          user,
		Password:      password,
		Keyfile:       keyfile,
		Port:          port,
		KeyPassphrase: keypassphrase,
	}, nil
}

// NewSSHClientConfig returns a new SSH client configuration
func NewSSHClientConfig(user, password, keyfile, passphrase string) (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("unable to get user home directory: %w", err)
	}

	// If a keyfile is provided, use it
	if keyfile == "" {
		keyfile = fmt.Sprintf("%s/.ssh/id_rsa", home)
	}

	key, err := os.ReadFile(keyfile)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %w", err)
	}

	var signer ssh.Signer
	// Check if passphrase is provided
	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("unable to parse private key with passphrase: %w", err)
		}
	} else {
		signer, err = ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("unable to parse private key: %w", err)
		}
	}

	authMethods = append(authMethods, ssh.PublicKeys(signer))

	// If a password is provided, use it as well
	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	return &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Replace with proper host key verification for production
		// HostKeyCallback: KnownHostsFile(filepath.Join(home, ".ssh", "known_hosts")), // Not yet functional
	}, nil
}

// KnownHostsFile returns a host key callback function that checks the provided file for the host key
func KnownHostsFile(knownHostsFile string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		file, err := os.Open(knownHostsFile)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var hostKey ssh.PublicKey
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) != 3 {
				continue
			}
			if fields[0] == hostname {
				var err error
				hostKey, _, _, _, err = ssh.ParseAuthorizedKey(scanner.Bytes())
				if err != nil {
					return err
				}
				break
			}
		}

		if hostKey == nil {
			return fmt.Errorf("no hostkey found for %s", hostname)
		}

		if hostKey.Type() != key.Type() || string(hostKey.Marshal()) != string(key.Marshal()) {
			return fmt.Errorf("hostkey mismatch for %s", hostname)
		}

		return nil
	}
}
