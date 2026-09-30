//go:build windows

package live

import (
	"slices"
	"sync"

	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/nav"
)

// RunwayMemory keeps each airport's runways in use: the ends chosen last
// are preferred while the wind allows them (nav.ActiveRunways takes the
// first preferred end within the limits), so a calm or variable wind does
// not swap the runway between one aircraft and the next.
type RunwayMemory struct {
	mu       sync.Mutex
	dep, arr map[string]string // by ICAO
}

// Use returns the runways in use at l for the weather, and remembers them.
func (m *RunwayMemory) Use(l *airport.Layout, w nav.Weather, lim nav.RunwayLimits) nav.RunwayUse {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.dep == nil {
		m.dep, m.arr = map[string]string{}, map[string]string{}
	}
	arrPref := lim.PreferredArrival
	if len(arrPref) == 0 {
		arrPref = lim.Preferred
	}
	if d := m.dep[l.ICAO]; d != "" {
		lim.Preferred = append([]string{d}, slices.DeleteFunc(slices.Clone(lim.Preferred), func(s string) bool { return s == d })...)
	}
	if a := m.arr[l.ICAO]; a != "" {
		arrPref = append([]string{a}, slices.DeleteFunc(slices.Clone(arrPref), func(s string) bool { return s == a })...)
	}
	lim.PreferredArrival = arrPref
	u := nav.ActiveRunways(l, w, lim)
	m.dep[l.ICAO], m.arr[l.ICAO] = u.Departure.Name, u.Arrival.Name
	return u
}
