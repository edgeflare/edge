package main

import (
	"os"

	"github.com/edgeflare/edge/cmd"
	"github.com/edgeflare/edge/pkg/db"
	"go.uber.org/zap"
)

func main() {
	var err error
	err = db.InitializeDB()
	if err != nil {
		zap.L().Fatal("Failed to initialize database", zap.Error(err))
	}
	defer func() {
		if closeErr := db.GetDB().Close(); closeErr != nil {
			zap.L().Error("Failed to close database", zap.Error(closeErr))
		}
	}()

	app := cmd.SetupApp()
	err = app.Run(os.Args)
	if err != nil {
		zap.L().Fatal("Failed to run app", zap.Error(err))
	}
}
