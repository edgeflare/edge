package k3s

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

func getNodeStatus(host string, port int) string {
	// Define the host, port, and retry settings
	maxRetries := 3
	retryDelay := 2 * time.Second // Delay between retries

	// Set up a TLS configuration with InsecureSkipVerify
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	// Combine host and port
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	// Retry loop
	for attempt := 1; attempt <= maxRetries; attempt++ {
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", address, tlsConfig)
		if err != nil {
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return "Unreachable"
		}
		conn.Close()
		break
	}

	return "Running"
}
