package ssh

import (
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// Exec executes a given SSH command on the remote machine.
func (c *Client) Exec(cmd *Command) error {
	return c.runCommand(cmd)
}

func (c *Client) runCommand(cmd *Command) error {
	var (
		session *ssh.Session
		err     error
	)

	if session, err = c.newSession(); err != nil {
		return err
	}
	defer session.Close()

	if err = c.prepareCommand(session, cmd); err != nil {
		return err
	}

	err = session.Run(cmd.Path)
	return err
}

func (c *Client) prepareCommand(session *ssh.Session, cmd *Command) error {
	for _, env := range cmd.Env {
		variable := strings.Split(env, "=")
		if len(variable) != 2 {
			continue
		}

		if err := session.Setenv(variable[0], variable[1]); err != nil {
			return err
		}
	}

	if cmd.Stdin != nil {
		stdin, err := session.StdinPipe()
		if err != nil {
			return fmt.Errorf("unable to setup stdin for session: %v", err)
		}
		go copyAndLogError(stdin, cmd.Stdin)
	}

	if cmd.Stdout != nil {
		stdout, err := session.StdoutPipe()
		if err != nil {
			return fmt.Errorf("unable to setup stdout for session: %v", err)
		}
		go copyAndLogError(cmd.Stdout, stdout)
	}

	if cmd.Stderr != nil {
		stderr, err := session.StderrPipe()
		if err != nil {
			return fmt.Errorf("unable to setup stderr for session: %v", err)
		}
		go copyAndLogError(cmd.Stderr, stderr)
	}

	return nil
}

func (c *Client) newSession() (*ssh.Session, error) {
	if c.Config == nil {
		c.Config = &ssh.ClientConfig{
			User:            "admin",
			Auth:            []ssh.AuthMethod{ /* your auth methods */ },
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Not recommended for production
		}
	}

	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), c.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %s", err)
	}

	session, err := connection.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %s", err)
	}

	modes := ssh.TerminalModes{
		// ssh.ECHO:          0,  // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		session.Close()
		return nil, fmt.Errorf("request for pseudo terminal failed: %s", err)
	}

	return session, nil
}

// copyAndLogError wraps io.Copy in a goroutine and logs any errors that occur.
func copyAndLogError(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		zap.L().Error("io.Copy error", zap.Error(err))
	}
}
