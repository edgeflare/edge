package main

import (
	"time"

	"github.com/edgeflare/edge/internal/stack/zitadel"
)

func main() {
	go func() {
		// for zitadel and envoyproxy to start
		time.Sleep(time.Second * 5)
		zitadel.Configure()
	}()

	Main()
}
