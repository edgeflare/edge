package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/edgeflare/edge/pkg/ssh"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"
)

// Config is the configuration for the app
type Config struct {
	HTTP           *HTTPConfig       `mapstructure:"http"`
	SSHClient      *ssh.Client       `mapstructure:"ssh"`
	ExtraHelmRepos *[]HelmRepository `mapstructure:"extraHelmRepos"`
	LogLevel       string            `mapstructure:"logLevel"`
	Kubeconfig     string            `mapstructure:"kubeconfig"`
}

// HTTPCorsConfig is the configuration for CORS
type HTTPCorsConfig struct {
	AllowOrigins []string `mapstructure:"allowOrigins"`
	Enabled      bool     `mapstructure:"enabled"`
}

// HTTPConfig is the configuration for the HTTP server
type HTTPConfig struct {
	CORS      *HTTPCorsConfig `mapstructure:"cors"`
	Auth      *HTTPAuthConfig `mapstructure:"auth"`
	Port      int             `mapstructure:"port"`
	EnableLog bool            `mapstructure:"enableLog"`
}

// HTTPAuthConfig is the configuration for HTTP authentication
type HTTPAuthConfig struct {
	Basic *HTTPBasicAuthConfig `mapstructure:"basic" json:"basic"`
	JWT   *HTTPJWTAuthConfig   `mapstructure:"jwt" json:"jwt"`
	Type  string               `mapstructure:"type" json:"type"`
}

// HTTPBasicAuthConfig is the configuration for HTTP basic authentication
type HTTPBasicAuthConfig struct {
	Username string `mapstructure:"username" json:"username"`
	Password string `mapstructure:"password" json:"password"`
}

// HTTPJWTAuthConfig is the configuration for HTTP JWT authentication
type HTTPJWTAuthConfig struct {
	Issuer      string `mapstructure:"issuer" json:"issuer"`
	ClientID    string `mapstructure:"clientId" json:"clientId"`
	JWKEndpoint string `mapstructure:"jwkEndpoint" json:"jwkEndpoint"`
}

func Load(c *cli.Context) (*Config, error) {
	v := initializeViper()

	if err := setDefaults(v); err != nil {
		return nil, err
	}

	if err := loadConfigFile(v, c); err != nil {
		return nil, err
	}

	overrideConfigWithCLI(v, c)

	if err := addExtraHelmRepos(v, c); err != nil {
		return nil, err
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func initializeViper() *viper.Viper {
	v := viper.New()
	v.SetEnvPrefix("EDGE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	return v
}

func setDefaults(v *viper.Viper) error {
	// Set default values for the log level
	v.SetDefault("logLevel", "info")

	// Default values for HTTP configuration
	v.SetDefault("http.port", 8080)
	v.SetDefault("http.enableLog", false)
	v.SetDefault("http.cors.enabled", false)
	v.SetDefault("http.cors.allowOrigins", []string{"*"})
	v.SetDefault("http.auth.type", "none") // options: none, basic, jwt
	v.SetDefault("http.auth.basic.username", "edge")
	v.SetDefault("http.auth.basic.password", "edge")

	// other default values as required by your application

	return nil
}

func loadConfigFile(v *viper.Viper, c *cli.Context) error {
	var configPath string

	if c.IsSet("config") {
		// Use the configuration file specified by the --config CLI flag
		configPath = c.String("config")
	} else {
		// Use the default configuration file path
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("unable to get user home directory: %w", err)
		}
		configPath = filepath.Join(home, ".edge", "edge.yaml")
	}

	// Check if the configuration file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Configuration file does not exist; no need to load
		return nil
	}

	// Set the path and attempt to read the configuration file
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read the configuration file: %w", err)
	}

	return nil
}

func overrideConfigWithCLI(v *viper.Viper, c *cli.Context) {
	// Override log level if set via CLI
	if c.IsSet("log-level") {
		v.Set("logLevel", c.String("log-level"))
	}

	// Override kubeconfig if set via CLI
	if c.IsSet("kubeconfig") {
		v.Set("kubeconfig", c.String("kubeconfig"))
		os.Setenv("KUBECONFIG", c.String("kubeconfig"))
	}

	// HTTP server configuration overrides
	if c.IsSet("http-port") {
		v.Set("http.port", c.Int("http-port"))
	}
	if c.IsSet("http-enableLog") {
		v.Set("http.enableLog", c.Bool("http-enableLog"))
	}

	// HTTP CORS configuration overrides
	if c.IsSet("http-cors-enabled") {
		v.Set("http.cors.enabled", c.Bool("http-cors-enabled"))
	}
	if c.IsSet("http-cors-allowOrigins") {
		v.Set("http.cors.allowOrigins", c.StringSlice("http-cors-allowOrigins"))
	}

	// HTTP authentication configuration overrides
	if c.IsSet("http-auth-type") {
		v.Set("http.auth.type", c.String("http-auth-type"))
	}
	if c.IsSet("http-auth-basic-username") {
		v.Set("http.auth.basic.username", c.String("http-auth-basic-username"))
	}
	if c.IsSet("http-auth-basic-password") {
		v.Set("http.auth.basic.password", c.String("http-auth-basic-password"))
	}

	// HTTP JWT Auth configuration overrides
	if c.IsSet("http-auth-jwt-issuer") {
		v.Set("http.auth.jwt.issuer", c.String("http-auth-jwt-issuer"))
	}
	if c.IsSet("http-auth-jwt-clientId") {
		v.Set("http.auth.jwt.clientId", c.String("http-auth-jwt-clientId"))
	}
	if c.IsSet("http-auth-jwt-jwkEndpoint") {
		v.Set("http.auth.jwt.jwkEndpoint", c.String("http-auth-jwt-jwkEndpoint"))
	}
}

func addExtraHelmRepos(v *viper.Viper, c *cli.Context) error {
	if c.IsSet("extraHelmRepos") {
		extraRepos := c.StringSlice("extraHelmRepos")
		var repos []HelmRepository
		for _, repo := range extraRepos {
			split := strings.Split(repo, "=")
			if len(split) != 2 {
				return fmt.Errorf("invalid extra-repos format. should be in the form of 'name=url'")
			}
			repos = append(repos, HelmRepository{
				Name: split[0],
				URL:  split[1],
			})
		}
		v.Set("extraHelmRepos", repos)
	}

	return nil
}
