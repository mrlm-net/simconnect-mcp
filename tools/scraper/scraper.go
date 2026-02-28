// Package main implements a rate-limited CLI scraper for the SimConnect SDK
// documentation hosted at docs.flightsimulator.com.
//
// The scraper is a standalone tool — it is NOT imported by the main server
// binary and MUST NOT be referenced from cmd/ or internal/. Build and run it
// directly with:
//
//	go run ./tools/scraper/ [flags]
//	go build ./tools/scraper/
package main

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// userAgent is sent with every HTTP request per ADR-006 attribution requirements.
const userAgent = "simconnect-mcp-scraper/1.0 (github.com/mrlm-net/simconnect-mcp)"

// cacheDir is the subdirectory used for local page caches, relative to the
// working directory at runtime.
const cacheDir = ".scraper-cache"

// cacheTTL is the maximum age of a cached page before it is re-fetched.
const cacheTTL = 24 * time.Hour

// maxRetries is the maximum number of fetch attempts before giving up.
const maxRetries = 3

// ErrRobotsDisallowed is returned by CheckRobots when the target path is
// disallowed by the site's robots.txt.
var ErrRobotsDisallowed = errors.New("path disallowed by robots.txt")

// robotsEntry holds the parsed Disallow rules for a single base URL.
type robotsEntry struct {
	disallowedPaths []string
}

// Scraper is the central HTTP client with integrated rate limiting, robots.txt
// enforcement, and local disk caching. All fields must be set via NewScraper —
// do not construct Scraper directly.
type Scraper struct {
	client      *http.Client
	delay       time.Duration
	lastRequest time.Time
	mu          sync.Mutex // guards lastRequest

	robotsMu    sync.Mutex // guards robotsCache
	robotsCache map[string]*robotsEntry
}

// NewScraper creates a Scraper with the given minimum delay between requests.
// A delay of 0 disables rate limiting (useful for testing against a mock
// server). The HTTP client uses a 30-second timeout.
func NewScraper(delay time.Duration) *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		delay:       delay,
		robotsCache: make(map[string]*robotsEntry),
	}
}

// FetchPage fetches the given URL, enforcing:
//  1. robots.txt compliance — returns ErrRobotsDisallowed if the path is
//     disallowed (the robots.txt for the base URL is fetched once and cached
//     in memory for the lifetime of the Scraper).
//  2. Disk cache — if a cache file for the URL exists and is younger than
//     cacheTTL, its content is returned without making an HTTP request.
//  3. Rate limiting — at least s.delay elapses between consecutive HTTP
//     requests.
//  4. Retry with exponential backoff — up to maxRetries attempts with delays
//     of 1s, 2s, 4s on transient HTTP errors (5xx) or network failures.
//
// The User-Agent header is always set to userAgent per ADR-006.
func (s *Scraper) FetchPage(rawURL string) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse url %q: %w", rawURL, err)
	}

	baseURL := parsed.Scheme + "://" + parsed.Host
	if err := s.CheckRobots(baseURL, parsed.Path); err != nil {
		return nil, err
	}

	// Check local disk cache.
	cacheFile := cacheFilePath(rawURL)
	if cached, ok := readCache(cacheFile); ok {
		return cached, nil
	}

	// Fetch with rate limiting and retry.
	var body []byte
	backoff := time.Second
	for attempt := 1; attempt <= maxRetries; attempt++ {
		s.waitForRateLimit()

		body, err = s.doGet(rawURL)
		if err == nil {
			break
		}

		if attempt < maxRetries {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	if err != nil {
		return nil, fmt.Errorf("fetch %q after %d attempts: %w", rawURL, maxRetries, err)
	}

	// Write to cache (best-effort; cache write failure is non-fatal).
	_ = writeCache(cacheFile, body)

	return body, nil
}

// CheckRobots returns ErrRobotsDisallowed if targetPath is disallowed for the
// simconnect-mcp-scraper user agent by the robots.txt at baseURL. The robots.txt
// is fetched once per baseURL and the result is cached in memory. A missing or
// unreadable robots.txt is treated as fully permissive.
func (s *Scraper) CheckRobots(baseURL, targetPath string) error {
	s.robotsMu.Lock()
	entry, cached := s.robotsCache[baseURL]
	s.robotsMu.Unlock()

	if !cached {
		entry = s.fetchRobots(baseURL)
		s.robotsMu.Lock()
		s.robotsCache[baseURL] = entry
		s.robotsMu.Unlock()
	}

	for _, disallowed := range entry.disallowedPaths {
		if strings.HasPrefix(targetPath, disallowed) {
			return fmt.Errorf("%w: %s", ErrRobotsDisallowed, targetPath)
		}
	}
	return nil
}

// fetchRobots downloads and parses robots.txt for baseURL, extracting Disallow
// rules that apply to all agents (*) or to our specific user agent name.
// On any error it returns an empty (fully-permissive) entry.
func (s *Scraper) fetchRobots(baseURL string) *robotsEntry {
	robotsURL := strings.TrimRight(baseURL, "/") + "/robots.txt"

	s.waitForRateLimit()
	body, err := s.doGet(robotsURL)
	if err != nil {
		// Treat unreadable robots.txt as fully permissive per ADR-006 guidance.
		return &robotsEntry{}
	}

	return parseRobots(body)
}

// parseRobots parses a robots.txt body and returns the Disallow rules that
// apply to all agents (*) or to "simconnect-mcp-scraper".
func parseRobots(body []byte) *robotsEntry {
	entry := &robotsEntry{}
	scanner := bufio.NewScanner(strings.NewReader(string(body)))

	// Track whether the current User-agent block applies to us.
	applies := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and blank lines.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}

		field := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])

		switch strings.ToLower(field) {
		case "user-agent":
			applies = value == "*" || strings.HasPrefix(strings.ToLower(value), "simconnect-mcp-scraper")
		case "disallow":
			if applies && value != "" {
				entry.disallowedPaths = append(entry.disallowedPaths, value)
			}
		}
	}

	return entry
}

// waitForRateLimit blocks until at least s.delay has elapsed since the last
// request. When s.delay is 0 it returns immediately.
func (s *Scraper) waitForRateLimit() {
	if s.delay == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	since := time.Since(s.lastRequest)
	if since < s.delay {
		time.Sleep(s.delay - since)
	}
	s.lastRequest = time.Now()
}

// doGet performs a single GET request and returns the response body.
// It sets the User-Agent header and treats any non-2xx status as an error.
func (s *Scraper) doGet(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status %d for %s", resp.StatusCode, rawURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return body, nil
}

// cacheFilePath returns the filesystem path for the cached version of a URL.
// The path is <cacheDir>/<sha256-of-url>.html.
func cacheFilePath(rawURL string) string {
	hash := sha256.Sum256([]byte(rawURL))
	name := fmt.Sprintf("%x.html", hash)
	return filepath.Join(cacheDir, name)
}

// readCache returns the cached content for path if it exists and is younger
// than cacheTTL. The second return value is false on any error or cache miss.
func readCache(path string) ([]byte, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if time.Since(info.ModTime()) > cacheTTL {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

// writeCache writes body to path atomically: it writes to a temp file first
// then renames. The cache directory is created if it does not exist.
func writeCache(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp) // clean up on rename failure
		return fmt.Errorf("rename cache file: %w", err)
	}
	return nil
}
