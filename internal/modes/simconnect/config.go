//go:build windows

package simconnect

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds configuration for the simconnect operating mode.
type Config struct {
	AppName    string
	ListenAddr string
}

// ConfigFromEnv reads Config from environment variables.
//
//   - SIMCONNECT_APP_NAME → AppName (default "simconnect-mcp")
//   - PORT                → ListenAddr as ":"+port (default ":8080")
//
// Reconnect behaviour is managed internally by the mrlm-net/simconnect Manager.
func ConfigFromEnv() Config {
	appName := os.Getenv("SIMCONNECT_APP_NAME")
	if appName == "" {
		appName = "simconnect-mcp"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	} else if n, err := strconv.Atoi(port); err != nil || n < 1025 || n > 65535 {
		port = "8080"
	}

	return Config{
		AppName:    appName,
		ListenAddr: fmt.Sprintf(":%s", port),
	}
}
