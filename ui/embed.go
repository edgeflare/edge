package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"os"

	"go.uber.org/zap"
)

//go:embed dist/edge-ui/*
var embeddedFiles embed.FS

// GetFileSystem returns the http.FileSystem to use for the UI
// If useOS is true, it will use the live filesystem
// Otherwise, it will use the embedded filesystem
func GetFileSystem(useOS bool) http.FileSystem {
	if useOS {
		zap.L().Info("using live/filesystem mode")
		return http.FS(os.DirFS("ui/dist/edge-ui/browser"))
	}

	zap.L().Info("using embed mode")
	fsys, err := fs.Sub(embeddedFiles, "dist/edge-ui/browser")
	if err != nil {
		panic(err)
	}

	return http.FS(fsys)
}
