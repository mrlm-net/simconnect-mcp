//go:build windows

package live

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mrlm-net/simconnect/pkg/engine"
	"github.com/mrlm-net/simconnect/pkg/manager"
	"github.com/mrlm-net/simconnect/pkg/traffic/world"
	"github.com/mrlm-net/simconnect/pkg/types"
)

// The traffic engine's library helpers sit off their default IDs, in the
// Runtime's range (IDBase moves the loaders, injector, airway crawl and
// enroute creations; docs/traffic-world.md#simconnect-ids). Its fixed IDs
// (2000–2021, 8200+, 10010, 20000–31279) clash with none of the bridge's
// (1,000,002+) nor the Runtime's (900,000,000+).
const (
	worldIDBase    uint32 = idBase + 50_000_000
	worldQueue            = 32768 // room for the first burst (airway crawl, facility lists)
	worldUpTimeout        = 30 * time.Second
	worldReapply          = 5 * time.Minute // how long a reconnect waits to set the traffic again
)

// World is the library's traffic engine (pkg/traffic/world) as the tools
// use it: its HTTP API in process, started on first use.
type World interface {
	// Ensure runs the engine on the connection, waiting until it is up.
	Ensure(ctx context.Context) error
	// Running reports whether it runs now (without starting it).
	Running() bool
	// Do and Get call its HTTP API (world.World.Do and Get).
	Do(method, path string, body any) ([]byte, error)
	Get(path string, v any) error
	// SetSchedule, SetCorridor and SetRealTraffic set the traffic and keep
	// the setting: the engine forgets it with the connection, and it is
	// set again on the next one.
	SetSchedule(s world.ScheduleSettings) error
	SetCorridor(c world.CorridorSettings) error
	SetRealTraffic(on bool, icao string) error
	// Snapshot is the engine's state (world.World.Snapshot).
	Snapshot() world.Snapshot
	// Error is why the engine last failed to start or set the traffic.
	Error() string
}

// TrafficWorld runs a world.World on a manager's connection: it feeds it
// every message while it runs, runs it again on each new connection, and
// sets the schedule, corridor and real traffic there again.
type TrafficWorld struct {
	mgr manager.Manager
	w   *world.World

	mu       sync.Mutex
	want     bool               // a tool started it: run on each connection
	cancel   context.CancelFunc // the running RunOn's
	gen      int                // which RunOn is the current one
	schedule *world.ScheduleSettings
	corridor *world.CorridorSettings
	real     *realSetting
	err      string
}

type realSetting struct {
	on   bool
	icao string
}

// NewTrafficWorld attaches a TrafficWorld to mgr. dataDir is where the
// engine keeps what it learns (each airport's airways, de-icing pads); ""
// a folder in the user cache.
func NewTrafficWorld(mgr manager.Manager, dataDir string) *TrafficWorld {
	if dataDir == "" {
		if d, err := os.UserCacheDir(); err == nil {
			dataDir = filepath.Join(d, "simconnect-mcp", "traffic")
		}
	}
	if dataDir != "" {
		_ = os.MkdirAll(dataDir, 0o755)
	}
	t := &TrafficWorld{mgr: mgr}
	t.w = world.New(world.Options{IDBase: worldIDBase, QueueSize: worldQueue, DataDir: dataDir})
	mgr.OnMessage(func(msg engine.Message) {
		t.mu.Lock()
		running := t.cancel != nil
		t.mu.Unlock()
		if running {
			// A copy: the engine reads it later, and the manager reuses the
			// buffer once this callback returns.
			t.w.Feed(msg.Detach())
		}
	})
	mgr.OnOpen(func(types.ConnectionOpenData) {
		t.mu.Lock()
		want := t.want
		t.mu.Unlock()
		if want {
			go t.restart()
		}
	})
	mgr.OnConnectionStateChange(func(_, state manager.ConnectionState) {
		if state != manager.StateConnected && state != manager.StateAvailable {
			t.stop()
		}
	})
	return t
}

// connected reports whether the manager's connection is up.
func (t *TrafficWorld) connected() bool {
	s := t.mgr.ConnectionState()
	return s == manager.StateConnected || s == manager.StateAvailable
}

// start runs the engine on the connection now; false when it already runs
// or there is no connection.
func (t *TrafficWorld) start() bool {
	client := t.mgr.Client()
	if client == nil || !t.connected() {
		return false
	}
	t.mu.Lock()
	if t.cancel != nil {
		t.mu.Unlock()
		return false
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	t.gen++
	gen := t.gen
	t.mu.Unlock()
	go func() {
		err := t.w.RunOn(ctx, client)
		t.mu.Lock()
		if t.gen == gen {
			t.cancel = nil
		}
		if err != nil {
			t.err = err.Error()
		}
		t.mu.Unlock()
	}()
	return true
}

// stop ends the engine's run (the connection is gone).
func (t *TrafficWorld) stop() {
	t.mu.Lock()
	c := t.cancel
	t.cancel = nil
	t.mu.Unlock()
	if c != nil {
		c()
	}
}

// restart runs the engine on a new connection and sets its traffic again
// once it is up.
func (t *TrafficWorld) restart() {
	t.stop()
	if !t.start() {
		return
	}
	deadline := time.Now().Add(worldReapply)
	for time.Now().Before(deadline) {
		if t.w.Snapshot().Connected {
			t.reapply()
			return
		}
		time.Sleep(time.Second)
	}
	t.setErr("the traffic engine did not come up on the new connection")
}

// reapply sets the kept schedule, corridor and real traffic again.
func (t *TrafficWorld) reapply() {
	t.mu.Lock()
	s, c, r := t.schedule, t.corridor, t.real
	t.mu.Unlock()
	if s != nil {
		if err := t.w.SetSchedule(*s); err != nil {
			t.setErr("schedule after reconnect: " + err.Error())
		}
	}
	if r != nil {
		if err := t.w.SetRealTraffic(r.on, r.icao); err != nil {
			t.setErr("real traffic after reconnect: " + err.Error())
		}
	}
	if c != nil {
		if err := t.w.SetCorridor(*c); err != nil {
			t.setErr("corridor after reconnect: " + err.Error())
		}
	}
}

func (t *TrafficWorld) setErr(s string) {
	t.mu.Lock()
	t.err = s
	t.mu.Unlock()
}

// Ensure implements World.
func (t *TrafficWorld) Ensure(ctx context.Context) error {
	if !t.connected() {
		return ErrNotConnected
	}
	t.mu.Lock()
	t.want = true
	t.mu.Unlock()
	t.start()
	deadline := time.NewTimer(worldUpTimeout)
	defer deadline.Stop()
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for !t.w.Snapshot().Connected {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("traffic engine: %w", ErrTimeout)
		case <-tick.C:
		}
	}
	return nil
}

// Running implements World.
func (t *TrafficWorld) Running() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cancel != nil
}

// Do implements World.
func (t *TrafficWorld) Do(method, path string, body any) ([]byte, error) {
	return t.w.Do(method, path, body)
}

// Get implements World.
func (t *TrafficWorld) Get(path string, v any) error { return t.w.Get(path, v) }

// Snapshot implements World.
func (t *TrafficWorld) Snapshot() world.Snapshot { return t.w.Snapshot() }

// Error implements World.
func (t *TrafficWorld) Error() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.err
}

// SetSchedule implements World.
func (t *TrafficWorld) SetSchedule(s world.ScheduleSettings) error {
	if err := t.w.SetSchedule(s); err != nil {
		return err
	}
	t.mu.Lock()
	t.schedule = &s
	if !s.Enabled {
		t.schedule = nil
	}
	t.mu.Unlock()
	return nil
}

// SetCorridor implements World.
func (t *TrafficWorld) SetCorridor(c world.CorridorSettings) error {
	if err := t.w.SetCorridor(c); err != nil {
		return err
	}
	t.mu.Lock()
	t.corridor = &c
	if !c.Enabled {
		t.corridor = nil
	}
	t.mu.Unlock()
	return nil
}

// SetRealTraffic implements World.
func (t *TrafficWorld) SetRealTraffic(on bool, icao string) error {
	if err := t.w.SetRealTraffic(on, icao); err != nil {
		return err
	}
	t.mu.Lock()
	t.real = &realSetting{on, icao}
	if !on {
		t.real = nil
	}
	t.mu.Unlock()
	return nil
}

// Close takes our traffic out of the simulator, which keeps AI objects
// after their client disconnects, and stops the engine; it returns how
// many aircraft it removed. The schedule and corridor stop first so
// nothing new appears. En route aircraft the schedule created are not in
// the engine's control list and stay until the connection closes.
func (t *TrafficWorld) Close() int {
	t.mu.Lock()
	t.want = false
	running := t.cancel != nil
	t.mu.Unlock()
	if !running {
		return 0
	}
	_, _ = t.w.Do("POST", "/api/schedule", map[string]any{"enabled": false})
	_, _ = t.w.Do("POST", "/api/corridor", map[string]any{"enabled": false})
	var list []world.ControlView
	n := 0
	if err := t.w.Get("/api/control", &list); err == nil {
		for _, a := range list {
			if _, err := t.w.Do("POST", fmt.Sprintf("/api/control/%d/remove", a.ID), nil); err == nil {
				n++
			}
		}
	}
	t.stop()
	return n
}

// DecodeJSON decodes a World API answer into v; an empty answer leaves it.
func DecodeJSON(b []byte, v any) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, v)
}
