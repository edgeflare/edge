package cmd

import (
	"fmt"

	"github.com/edgeflare/edge/pkg/config"
	"github.com/edgeflare/edge/pkg/server"
	"github.com/edgeflare/edge/ui"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func startServer(config *config.Config) error {
	e := echo.New()

	// middlewares
	// HTTP request logging
	if config.HTTP.EnableLog {
		e.Use(middleware.Logger())
	}
	// CORS
	if config.HTTP.CORS.Enabled {
		corsConfig := middleware.DefaultCORSConfig
		corsConfig.AllowOrigins = config.HTTP.CORS.AllowOrigins
		e.Use(middleware.CORSWithConfig(corsConfig))
	}
	// Recover from panics
	e.Use(middleware.Recover())

	// Static files
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Filesystem: ui.GetFileSystem(false),
		HTML5:      true,
	}))

	// Register API handlers
	server.RegisterHandlers(e, config)

	// Start the server
	if err := e.Start(fmt.Sprintf(":%d", config.HTTP.Port)); err != nil {
		zap.L().Fatal("Failed to start server", zap.Error(err))
		return err
	}

	return nil
}

func serverCommand() *cli.Command {
	return &cli.Command{
		Name:    "server",
		Aliases: []string{"s"},
		Usage:   "start the server for REST API and web UI",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "http-port",
				Aliases: []string{"p"},
				Usage:   "HTTP port to listen on",
				Value:   8080,
				EnvVars: []string{"EDGE_HTTP_PORT"},
			},
			&cli.BoolFlag{
				Name:    "http-enableLog",
				Aliases: []string{"l"},
				Usage:   "whether to log HTTP requests",
				Value:   false,
				EnvVars: []string{"EDGE_HTTP_ENABLELOG"},
			},
			&cli.BoolFlag{
				Name:    "http-cors-enabled",
				Aliases: []string{"c"},
				Usage:   "whether to allow CORS",
				Value:   false,
				EnvVars: []string{"EDGE_HTTP_CORS_ENABLED"},
			},
			&cli.StringSliceFlag{
				Name:    "http-cors-allowOrigins",
				Aliases: []string{"o"},
				Usage:   "allowed origins for CORS",
				EnvVars: []string{"EDGE_HTTP_CORS_ALLOWORIGINS"},
				Value:   cli.NewStringSlice("*"),
			},
			&cli.StringFlag{
				Name:    "http-auth-type",
				Aliases: []string{"t"},
				Usage:   "HTTP Auth type",
				EnvVars: []string{"EDGE_HTTP_AUTH_TYPE"},
				Value:   "none",
			},
			&cli.StringFlag{
				Name:    "http-auth-basic-username",
				Aliases: []string{"u"},
				Usage:   "HTTP Basic Auth username",
				EnvVars: []string{"EDGE_HTTP_AUTH_BASIC_USERNAME"},
				Value:   "edge",
			},
			&cli.StringFlag{
				Name:    "http-auth-basic-password",
				Aliases: []string{"w"},
				Usage:   "HTTP Basic Auth password",
				EnvVars: []string{"EDGE_HTTP_AUTH_BASIC_PASSWORD"},
				Value:   "edge",
			},
			&cli.StringFlag{
				Name:    "http-auth-jwt-issuer",
				Aliases: []string{"i"},
				Usage:   "HTTP JWT Auth issuer",
				EnvVars: []string{"EDGE_HTTP_AUTH_JWT_ISSUER"},
			},
			&cli.StringFlag{
				Name:    "http-auth-jwt-clientId",
				Aliases: []string{"a"},
				Usage:   "HTTP JWT Auth clientId",
				EnvVars: []string{"EDGE_HTTP_AUTH_JWT_CLIENTID"},
			},
			&cli.StringFlag{
				Name:    "http-auth-jwt-jwkEndpoint",
				Aliases: []string{"j"},
				Usage:   "HTTP JWT Auth JWK endpoint",
				EnvVars: []string{"EDGE_HTTP_AUTH_JWT_JWKENDPOINT"},
			},
		},
		Action: func(c *cli.Context) error {
			config, err := config.Load(c)
			if err != nil {
				zap.L().Fatal("Failed to load config", zap.Error(err))
				return err
			}

			if err = startServer(config); err != nil {
				zap.L().Fatal("Failed to start server", zap.Error(err))
				return err
			}
			return nil
		},
	}
}
