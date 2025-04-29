package main

import (
	"time"

	"github.com/edgeflare/edge/internal/stack/emqx"
	"github.com/edgeflare/edge/internal/stack/zitadel"
)

func main() {
	go func() {
		// for zitadel and envoyproxy to start
		time.Sleep(time.Second * 5)
		zitadel.Configure()

		for _, a := range addons {
			if a == "emqx" {
				emqx.Configure()
			}
		}
	}()

	Main()
}
