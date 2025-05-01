package main

import (
	"fmt"
	"net"
	"time"

	"github.com/edgeflare/edge/internal/stack/emqx"
	"github.com/edgeflare/edge/internal/stack/zitadel"
)

func main() {
	go func() {
		// Wait for envoy ports to be open
		for !isPortOpen("envoy", int(envoyHttpPort)) || !isPortOpen("envoy", int(envoyHttpsPort)) {
			fmt.Println("Waiting for Envoy to be ready...")
			time.Sleep(time.Second * 2)
		}
		zitadel.Configure() // should return an error if configuration fails
		for _, a := range addons {
			if a == "emqx" {
				emqx.Configure()
			}
		}
	}()
	Main()
}

// isPortOpen checks if a port is open
func isPortOpen(host string, port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%v", host, port), time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
