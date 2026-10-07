//go:build windows

package live

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/mrlm-net/simconnect/pkg/addons"
	"github.com/mrlm-net/simconnect/pkg/avionics"
	"github.com/mrlm-net/simconnect/pkg/engine"
	"github.com/mrlm-net/simconnect/pkg/manager"
	"github.com/mrlm-net/simconnect/pkg/systems"
	"github.com/mrlm-net/simconnect/pkg/types"
)

// SimConnect IDs of the user aircraft's components, in the Runtime's range.
const (
	aircraftIDBase  = idBase + 4_000
	titleDefID      = aircraftIDBase       // TITLE, ATC TYPE
	titleReqID      = aircraftIDBase + 1   //
	systemsDefID    = aircraftIDBase + 2   //
	systemsReqID    = aircraftIDBase + 3   //
	aircraftPathReq = aircraftIDBase + 4   // AircraftLoaded system state
	radiosBase      = aircraftIDBase + 100 // pkg/avionics events and presses
	controlsBase    = aircraftIDBase + 200 // pkg/systems Controls, a block of 64
	flightDefBase   = aircraftIDBase + 300 // avionics.SetFlight, 2 definitions
	aircraftTimeout = 5 * time.Second
	addonsMaxAge    = 5 * time.Minute // the package scan is a few seconds of disk
)

// AircraftSystems is the user aircraft's systems as its profile reads them.
type AircraftSystems struct {
	Title    string
	ATCType  string
	Path     string          // aircraft.cfg the sim loaded ("" unknown)
	Package  *addons.Package // its package, when found
	Profile  systems.Profile
	Local    []string // local override files that applied
	LocalErr []string // local override files that did not read
	State    systems.State
	Can      map[string]bool // controls and services the profile can operate
}

// Aircraft operates the user aircraft through the library: pkg/systems
// (state, ground controls and services), pkg/avionics (radios, squawk,
// call sign) and pkg/addons (the installed packages).
type Aircraft interface {
	Connected() bool
	// Systems reads the aircraft's systems now.
	Systems(ctx context.Context) (AircraftSystems, error)
	// SetControl sets a control (door, chocks, GPU, parking brake, cabin
	// signs, external power) on or off, or to a value (the no smoking sign).
	SetControl(ctx context.Context, name string, value float64) error
	// PressControl presses a push button (the cabin call).
	PressControl(ctx context.Context, name string) error
	// RequestService asks for one of the sim's ground services.
	RequestService(ctx context.Context, name string) error
	// SetCOM sets COM n's active or standby frequency, in MHz.
	SetCOM(ctx context.Context, n int, standby bool, mhz float64) error
	// SwapCOM swaps COM n's active and standby frequencies.
	SwapCOM(ctx context.Context, n int) error
	// SetSquawk sets the transponder code.
	SetSquawk(ctx context.Context, code string) error
	// SetCallsign sets ATC AIRLINE and ATC FLIGHT NUMBER ("" leaves one).
	SetCallsign(ctx context.Context, airline, number string) error
	// Addons lists the installed packages (cached for a few minutes).
	Addons(refresh bool) (addons.Install, []addons.Package, error)
}

// ErrNoProfileControl: the aircraft's profile gives no way to operate it.
var ErrNoProfileControl = systems.ErrNoControl

// AircraftRuntime implements Aircraft on a manager.
type AircraftRuntime struct {
	mgr         manager.Manager
	profilesDir string

	mu        sync.Mutex
	connected bool
	reader    *systems.Reader
	controls  *systems.Controls
	radios    *avionics.Radios
	path      string
	title     string
	atcType   string
	key       string // what the profile was picked for
	profile   systems.Profile
	local     []string
	localErr  []string
	titleWait []chan struct{}
	pathWait  []chan struct{}
	sysWait   []chan systems.State
	// titleDefined: the TITLE definition is registered on this connection.
	titleDefined bool
	// flightDefined: SetFlight's ATC AIRLINE and ATC FLIGHT NUMBER
	// definitions are registered on this connection.
	flightDefined [2]bool

	addonsMu  sync.Mutex
	install   addons.Install
	pkgs      []addons.Package
	addonsErr error
	addonsAt  time.Time
}

// NewAircraft attaches an AircraftRuntime to mgr. profilesDir holds local
// profile overrides (*.json, pkg/systems' format) that win per value over
// the shipped ones; "" none.
func NewAircraft(mgr manager.Manager, profilesDir string) *AircraftRuntime {
	a := &AircraftRuntime{mgr: mgr, profilesDir: profilesDir}
	a.resetLocked()
	a.connected = mgr.ConnectionState() == manager.StateConnected || mgr.ConnectionState() == manager.StateAvailable
	mgr.OnOpen(func(types.ConnectionOpenData) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.resetLocked()
		a.connected = true
	})
	mgr.OnConnectionStateChange(func(_, state manager.ConnectionState) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.connected = state == manager.StateConnected || state == manager.StateAvailable
	})
	mgr.OnAircraftLoaded(func(path string) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.path = path
	})
	mgr.OnMessage(a.handle)
	return a
}

// resetLocked makes the components anew for a new connection: the old
// definitions and event mappings are gone with it.
func (a *AircraftRuntime) resetLocked() {
	a.reader = systems.NewReader(a.mgr, systemsDefID, systemsReqID)
	a.controls = systems.NewControls(a.mgr, controlsBase)
	a.radios = avionics.New(a.mgr, radiosBase)
	a.key, a.title, a.atcType = "", "", ""
	a.titleDefined = false
	a.flightDefined = [2]bool{}
}

// Connected implements Aircraft.
func (a *AircraftRuntime) Connected() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.connected
}

// titleData is the TITLE, ATC TYPE block.
type titleData struct {
	Title   [256]byte
	ATCType [64]byte
}

// handle runs in the manager's dispatch goroutine for every message.
func (a *AircraftRuntime) handle(msg engine.Message) {
	if msg.Err != nil || msg.SIMCONNECT_RECV == nil {
		return
	}
	switch types.SIMCONNECT_RECV_ID(msg.DwID) {
	case types.SIMCONNECT_RECV_ID_SIMOBJECT_DATA:
		a.mu.Lock()
		r := a.reader
		a.mu.Unlock()
		if st, ok := r.Handle(msg); ok {
			a.mu.Lock()
			for _, ch := range a.sysWait {
				ch <- st
			}
			a.sysWait = nil
			a.mu.Unlock()
			return
		}
		d := msg.AsSimObjectData()
		if uint32(d.DwRequestID) != titleReqID {
			return
		}
		t := engine.CastDataAs[titleData](&d.DwData)
		a.mu.Lock()
		a.title, a.atcType = engine.BytesToString(t.Title[:]), engine.BytesToString(t.ATCType[:])
		// Many aircraft give a localization key ("ATCCOM.ATC_NAME AIRBUS.0.text")
		// the sim does not translate for SimConnect: no type.
		if strings.HasPrefix(a.atcType, "ATCCOM.") {
			a.atcType = ""
		}
		for _, ch := range a.titleWait {
			close(ch)
		}
		a.titleWait = nil
		a.mu.Unlock()
	case types.SIMCONNECT_RECV_ID_SYSTEM_STATE:
		st := (*types.SIMCONNECT_RECV_SYSTEM_STATE)(unsafe.Pointer(msg.SIMCONNECT_RECV))
		if uint32(st.DwRequestID) != aircraftPathReq {
			return
		}
		a.mu.Lock()
		a.path = engine.BytesToString(st.SzString[:])
		for _, ch := range a.pathWait {
			close(ch)
		}
		a.pathWait = nil
		a.mu.Unlock()
	}
}

// ready reads the aircraft's title, type and path, and picks its profile
// when any of them (or the local override files) changed.
func (a *AircraftRuntime) ready(ctx context.Context) error {
	a.mu.Lock()
	if !a.connected {
		a.mu.Unlock()
		return ErrNotConnected
	}
	titleCh, pathCh := make(chan struct{}), make(chan struct{})
	a.titleWait = append(a.titleWait, titleCh)
	a.pathWait = append(a.pathWait, pathCh)
	define := !a.titleDefined
	a.titleDefined = true
	a.mu.Unlock()

	if define {
		if err := a.mgr.AddToDataDefinition(titleDefID, "TITLE", "", types.SIMCONNECT_DATATYPE_STRING256, 0, 0); err != nil {
			return fmt.Errorf("define TITLE: %w", err)
		}
		if err := a.mgr.AddToDataDefinition(titleDefID, "ATC TYPE", "", types.SIMCONNECT_DATATYPE_STRING64, 0, 1); err != nil {
			return fmt.Errorf("define ATC TYPE: %w", err)
		}
	}
	if err := a.mgr.RequestDataOnSimObject(titleReqID, titleDefID, types.SIMCONNECT_OBJECT_ID_USER,
		types.SIMCONNECT_PERIOD_ONCE, 0, 0, 0, 0); err != nil {
		return fmt.Errorf("request TITLE: %w", err)
	}
	if err := a.mgr.RequestSystemState(aircraftPathReq, types.SIMCONNECT_SYSTEM_STATE_AIRCRAFT_LOADED); err != nil {
		return fmt.Errorf("request AircraftLoaded: %w", err)
	}
	for _, ch := range []chan struct{}{titleCh, pathCh} {
		select {
		case <-ch:
		case <-time.After(aircraftTimeout):
			return ErrTimeout
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	locals, names, errs := a.localProfiles()
	a.mu.Lock()
	path, title, atcType := a.path, a.title, a.atcType
	a.mu.Unlock()
	pkg := a.aircraftPackage(path)
	ac := systems.Aircraft{Title: title, ATCType: atcType}
	if pkg != nil {
		ac.Package = pkg.Folder
	}
	key := ac.Package + "|" + ac.Title + "|" + ac.ATCType + "|" + strings.Join(names, ",")

	a.mu.Lock()
	defer a.mu.Unlock()
	if key == a.key {
		return nil
	}
	p := systems.For(ac, locals...)
	a.reader.Use(p)
	a.controls.Use(p)
	a.radios.Use(p.Actions)
	var applied []string
	for i, l := range locals {
		if l.Matches(ac) || l.Name == p.Name {
			applied = append(applied, names[i])
		}
	}
	a.key, a.profile, a.local, a.localErr = key, p, applied, errs
	return nil
}

// localProfiles reads the local override files: the profiles, their file
// names, and why each that did not read did not.
func (a *AircraftRuntime) localProfiles() ([]systems.Profile, []string, []string) {
	if a.profilesDir == "" {
		return nil, nil, nil
	}
	files, _ := filepath.Glob(filepath.Join(a.profilesDir, "*.json"))
	sort.Strings(files)
	var out []systems.Profile
	var names, errs []string
	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			errs = append(errs, filepath.Base(f)+": "+err.Error())
			continue
		}
		p, err := systems.ReadProfile(fh)
		fh.Close()
		if err != nil {
			errs = append(errs, filepath.Base(f)+": "+err.Error())
			continue
		}
		out, names = append(out, p), append(names, filepath.Base(f))
	}
	return out, names, errs
}

// aircraftPackage is the package holding the aircraft at path; nil when
// the add-ons can't be read or none holds it.
func (a *AircraftRuntime) aircraftPackage(path string) *addons.Package {
	if path == "" {
		return nil
	}
	_, pkgs, err := a.Addons(false)
	if err != nil {
		return nil
	}
	if p, ok := addons.AircraftPackage(pkgs, path); ok {
		return &p
	}
	return nil
}

// Addons implements Aircraft.
func (a *AircraftRuntime) Addons(refresh bool) (addons.Install, []addons.Package, error) {
	a.addonsMu.Lock()
	defer a.addonsMu.Unlock()
	if !refresh && !a.addonsAt.IsZero() && time.Since(a.addonsAt) < addonsMaxAge {
		return a.install, a.pkgs, a.addonsErr
	}
	a.addonsAt = time.Now()
	a.install, a.pkgs, a.addonsErr = addons.Install{}, nil, nil
	in, err := addons.Find()
	if err != nil {
		a.addonsErr = err
		return a.install, nil, err
	}
	pkgs, err := addons.Scan(in.Packages)
	a.install, a.pkgs, a.addonsErr = in, pkgs, err
	return in, pkgs, err
}

// Systems implements Aircraft.
func (a *AircraftRuntime) Systems(ctx context.Context) (AircraftSystems, error) {
	st, err := a.state(ctx)
	if err != nil {
		return AircraftSystems{}, err
	}
	a.mu.Lock()
	out := AircraftSystems{Title: a.title, ATCType: a.atcType, Path: a.path, Profile: a.profile,
		Local: a.local, LocalErr: a.localErr, State: st, Can: map[string]bool{}}
	ctl := a.controls
	a.mu.Unlock()
	out.Package = a.aircraftPackage(out.Path)
	for _, n := range ControlNames(st) {
		out.Can[n] = ctl.Can(n)
	}
	for _, n := range ServiceNames {
		out.Can[n] = ctl.Can(n)
	}
	return out, nil
}

// state reads the systems once, with the aircraft's profile.
func (a *AircraftRuntime) state(ctx context.Context) (systems.State, error) {
	if err := a.ready(ctx); err != nil {
		return systems.State{}, err
	}
	ch := make(chan systems.State, 1)
	a.mu.Lock()
	a.sysWait = append(a.sysWait, ch)
	r := a.reader
	a.mu.Unlock()
	if err := r.Request(types.SIMCONNECT_PERIOD_ONCE); err != nil {
		return systems.State{}, fmt.Errorf("request systems: %w", err)
	}
	select {
	case st := <-ch:
		return st, nil
	case <-time.After(aircraftTimeout):
		return systems.State{}, ErrTimeout
	case <-ctx.Done():
		return systems.State{}, ctx.Err()
	}
}

// ServiceNames are the sim's ground services Controls.Request knows.
var ServiceNames = []string{systems.Jetway, systems.Stairs, systems.Baggage, systems.Catering,
	systems.PowerSupply, systems.FuelTruck, systems.Pushback}

// ControlNames are the controls SetControl and PressControl know for an
// aircraft in state st: its doors, then the ground and cabin controls.
func ControlNames(st systems.State) []string {
	var out []string
	for i := range st.DoorNames {
		out = append(out, systems.Door(i))
	}
	return append(out, systems.Chocks, systems.GPU, systems.ParkingBrake, systems.Seatbelts,
		systems.NoSmoking, systems.ExtPower, systems.CabinCall)
}

// SetControl implements Aircraft.
func (a *AircraftRuntime) SetControl(ctx context.Context, name string, value float64) error {
	st, err := a.state(ctx)
	if err != nil {
		return err
	}
	a.mu.Lock()
	ctl := a.controls
	a.mu.Unlock()
	if name == systems.NoSmoking {
		return ctl.SetValue(name, value, st)
	}
	return ctl.Set(name, value != 0, st)
}

// PressControl implements Aircraft.
func (a *AircraftRuntime) PressControl(ctx context.Context, name string) error {
	st, err := a.state(ctx)
	if err != nil {
		return err
	}
	a.mu.Lock()
	ctl := a.controls
	a.mu.Unlock()
	return ctl.Press(name, st)
}

// RequestService implements Aircraft.
func (a *AircraftRuntime) RequestService(ctx context.Context, name string) error {
	if err := a.ready(ctx); err != nil {
		return err
	}
	a.mu.Lock()
	ctl := a.controls
	a.mu.Unlock()
	return ctl.Request(name)
}

// radiosReady picks the profile (the model's own radio actions) and
// returns the radios.
func (a *AircraftRuntime) radiosReady(ctx context.Context) (*avionics.Radios, error) {
	if err := a.ready(ctx); err != nil {
		return nil, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.radios, nil
}

// SetCOM implements Aircraft.
func (a *AircraftRuntime) SetCOM(ctx context.Context, n int, standby bool, mhz float64) error {
	r, err := a.radiosReady(ctx)
	if err != nil {
		return err
	}
	if standby {
		return r.SetCOMStandby(n, mhz)
	}
	return r.SetCOMActive(n, mhz)
}

// SwapCOM implements Aircraft.
func (a *AircraftRuntime) SwapCOM(ctx context.Context, n int) error {
	r, err := a.radiosReady(ctx)
	if err != nil {
		return err
	}
	return r.SwapCOM(n)
}

// SetSquawk implements Aircraft.
func (a *AircraftRuntime) SetSquawk(ctx context.Context, code string) error {
	r, err := a.radiosReady(ctx)
	if err != nil {
		return err
	}
	return r.SetSquawk(code)
}

// SetCallsign implements Aircraft. SetFlight adds its definitions on every
// call; they are cleared first so a second call doesn't define them twice.
func (a *AircraftRuntime) SetCallsign(ctx context.Context, airline, number string) error {
	a.mu.Lock()
	up := a.connected
	a.mu.Unlock()
	if !up {
		return ErrNotConnected
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Clearing a definition never made raises an UNRECOGNIZED_ID exception.
	a.mu.Lock()
	for i, used := range []bool{airline != "", number != ""} {
		if used && a.flightDefined[i] {
			_ = a.mgr.ClearDataDefinition(flightDefBase + uint32(i))
		}
		a.flightDefined[i] = a.flightDefined[i] || used
	}
	a.mu.Unlock()
	return avionics.SetFlight(a.mgr, flightDefBase, airline, number)
}

// IsInputError reports whether err is the library refusing an argument
// (a bad radio, frequency or squawk) before anything was sent.
func IsInputError(err error) bool {
	return errors.Is(err, avionics.ErrBadRadio) || errors.Is(err, avionics.ErrBadFrequency) ||
		errors.Is(err, avionics.ErrBadSquawk)
}
