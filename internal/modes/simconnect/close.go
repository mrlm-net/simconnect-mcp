//go:build windows

package simconnect

import (
	"log"
	"time"

	"github.com/mrlm-net/simconnect-mcp/internal/bridge"
)

// removeWait gives the simulator time to act on the removal requests before
// the connection closes.
const removeWait = 500 * time.Millisecond

// Close implements modes.Closer: it takes our AI aircraft out of the
// simulator, then closes the SimConnect connection.
func (m *simconnectMode) Close() error {
	return Shutdown(m.cleanup, m.bridge)
}

// Shutdown runs cleanup (see tools.RegisterAll) and closes b; both may be
// nil. Shared by simconnect and both modes.
func Shutdown(cleanup func() int, b bridge.Bridge) error {
	if cleanup != nil {
		if n := cleanup(); n > 0 {
			log.Printf("[simconnect] removed %d AI aircraft of ours from the simulator", n)
			time.Sleep(removeWait)
		}
	}
	if b == nil {
		return nil
	}
	return b.Close()
}
