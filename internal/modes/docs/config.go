package docs

import "os"

// Config holds all configuration for the docs operating mode.
// All env var access is centralised in ConfigFromEnv; no other function
// in this package reads environment variables directly.
type Config struct {
	MSFSVersion  string // "2020", "2024", or "both" — default "2024"
	OverridePath string // filesystem path to JSON assets; empty = use embedded
	ListenAddr   string // e.g. ":8080"
}

// ConfigFromEnv reads docs-mode configuration from environment variables.
//
//   - DOCS_MSFS_VERSION → MSFSVersion (default "2024")
//   - DOCS_OVERRIDE_PATH → OverridePath (default "")
//   - PORT → ListenAddr as ":"+port (default ":8080")
func ConfigFromEnv() Config {
	version := os.Getenv("DOCS_MSFS_VERSION")
	if version == "" {
		version = "2024"
	}

	overridePath := os.Getenv("DOCS_OVERRIDE_PATH")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		MSFSVersion:  version,
		OverridePath: overridePath,
		ListenAddr:   ":" + port,
	}
}
