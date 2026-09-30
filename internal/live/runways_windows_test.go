//go:build windows

package live

import (
	"context"
	"testing"

	"github.com/mrlm-net/simconnect/pkg/nav"
)

// A light wind from the other side keeps the runways in use; a real one
// turns them.
func TestRunwayMemoryKeepsRunways(t *testing.T) {
	fx, err := NewFixture("testdata")
	if err != nil {
		t.Fatal(err)
	}
	l, err := fx.Layout(context.Background(), "LKPR")
	if err != nil {
		t.Fatal(err)
	}
	var m RunwayMemory
	use := func(dir, kts float64) nav.RunwayUse {
		return m.Use(l, nav.StaticWeather(dir, kts, 9999, 15, 5, 1013), nav.RunwayLimits{})
	}
	first := use(240, 3)
	for _, wind := range [][2]float64{{60, 3}, {240, 2}, {60, 4}} {
		if u := use(wind[0], wind[1]); u.Departure.Name != first.Departure.Name || u.Arrival.Name != first.Arrival.Name {
			t.Errorf("wind %03.0f/%.0f: %s/%s, want %s/%s kept", wind[0], wind[1], u.Departure.Name, u.Arrival.Name, first.Departure.Name, first.Arrival.Name)
		}
	}
	// Against a real tailwind the runway turns.
	var into nav.RunwayUse
	if first.Departure.Name == "24" {
		into = use(60, 15)
	} else {
		into = use(240, 15)
	}
	if into.Departure.Name == first.Departure.Name {
		t.Errorf("15 kt tailwind: still %s", into.Departure.Name)
	}
	t.Logf("kept %s/%s in light winds, turned to %s/%s", first.Departure.Name, first.Arrival.Name, into.Departure.Name, into.Arrival.Name)
}
