package docs

import (
	"os"
	"strconv"
)

// Config holds all configuration for the docs operating mode.
// All env var access is centralised in ConfigFromEnv; no other function
// in this package reads environment variables directly.
type Config struct {
	MSFSVersion  string // "2020", "2024", or "both" — default "2024"
	OverridePath string // filesystem path to JSON assets; empty = use embedded
	LiveScrape   bool   // DOCS_LIVE_SCRAPE=true: require confirm_live_scraping per tool call
	ListenAddr   string // e.g. ":8080"
}

// ConfigFromEnv reads docs-mode configuration from environment variables.
//
//   - DOCS_MSFS_VERSION → MSFSVersion (default "2024")
//   - DOCS_OVERRIDE_PATH → OverridePath (default "")
//   - DOCS_LIVE_SCRAPE → LiveScrape (default false)
//   - PORT → ListenAddr as ":"+port (default ":8080")
func ConfigFromEnv() Config {
	version := os.Getenv("DOCS_MSFS_VERSION")
	if version == "" {
		version = "2024"
	}

	overridePath := os.Getenv("DOCS_OVERRIDE_PATH")
	liveScrape := os.Getenv("DOCS_LIVE_SCRAPE") == "true"

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	} else if n, err := strconv.Atoi(port); err != nil || n < 1025 || n > 65535 {
		port = "8080"
	}

	return Config{
		MSFSVersion:  version,
		OverridePath: overridePath,
		LiveScrape:   liveScrape,
		ListenAddr:   ":" + port,
	}
}
