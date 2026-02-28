package main

import (
	"fmt"
	"sort"

	"github.com/mrlm-net/simconnect-mcp/internal/modes"
)

// modeFactory is a function that constructs a configured Mode and returns its listen address.
type modeFactory func() (modes.Mode, string, error)

// modeRegistry maps MCP_MODE values to factory functions.
// Entries are added by platform-specific files via registerMode.
var modeRegistry = map[string]modeFactory{}

// registerMode adds a factory to the registry. Panics on duplicate registration.
func registerMode(name string, f modeFactory) {
	if _, exists := modeRegistry[name]; exists {
		panic(fmt.Sprintf("mode %q already registered", name))
	}
	modeRegistry[name] = f
}

// availableModes returns a sorted list of registered mode names.
func availableModes() []string {
	names := make([]string, 0, len(modeRegistry))
	for k := range modeRegistry {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
