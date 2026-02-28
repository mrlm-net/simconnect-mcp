//go:build windows

package simconnect

import (
	"fmt"
	"os"
	"time"
)

// Config holds configuration for the simconnect operating mode.
type Config struct {
	AppName           string
	ReconnectInterval time.Duration
	ListenAddr        string
}

// ConfigFromEnv reads Config from environment variables.
//
//   - SIMCONNECT_APP_NAME           → AppName (default "simconnect-mcp")
//   - SIMCONNECT_RECONNECT_INTERVAL → ReconnectInterval as a Go duration (default "5s")
//   - PORT                          → ListenAddr as ":"+port (default ":8080")
func ConfigFromEnv() Config {
	appName := os.Getenv("SIMCONNECT_APP_NAME")
	if appName == "" {
		appName = "simconnect-mcp"
	}

	reconnectInterval := 5 * time.Second
	if raw := os.Getenv("SIMCONNECT_RECONNECT_INTERVAL"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil {
			reconnectInterval = d
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		AppName:           appName,
		ReconnectInterval: reconnectInterval,
		ListenAddr:        fmt.Sprintf(":%s", port),
	}
}
