//go:build windows

package live

import (
	"context"
	"testing"

	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/calc"
)

// An aircraft is on a final when it is within 20 NM of a threshold, on its
// extended centreline and heading for it; beside it or flying away it is not.
func TestFinalOfAndOnRunway(t *testing.T) {
	fx, err := NewFixture("testdata")
	if err != nil {
		t.Fatal(err)
	}
	l, err := fx.Layout(context.Background(), "LKPR")
	if err != nil {
		t.Fatal(err)
	}
	_, e24, ok := l.RunwayEnd("24")
	if !ok {
		t.Fatal("no runway 24")
	}
	out := e24.Heading + 180
	at := func(nm, side float64) airport.LatLon {
		lat, lon := calc.DisplaceByHeading(e24.Threshold.Lat, e24.Threshold.Lon, out, nm*1852)
		lat, lon = calc.DisplaceByHeading(lat, lon, out+90, side*1852)
		return airport.LatLon{Lat: lat, Lon: lon}
	}
	if end, d, ok := finalOf(l, at(8, 0.3), e24.Heading); !ok || end.Name != "24" || d < 7.5 || d > 8.5 {
		t.Errorf("on the 24 final at 8 NM: %v %q %.1f", ok, end.Name, d)
	}
	for name, c := range map[string]struct {
		p   airport.LatLon
		hdg float64
	}{
		"beside it":   {at(8, 3), e24.Heading},
		"flying away": {at(8, 0), out},
		"too far":     {at(25, 0), e24.Heading},
	} {
		if end, _, ok := finalOf(l, c.p, c.hdg); ok && end.Name == "24" {
			t.Errorf("%s: on the 24 final", name)
		}
	}
	rw, ok := runwayNamed(l, "24")
	if !ok || rw.Name() != "06/24" {
		t.Fatalf("runway of 24: %q %v", rw.Name(), ok)
	}
	if !onRunway(rw, e24.Threshold) || onRunway(rw, at(1, 0)) {
		t.Error("on the runway: threshold yes, 1 NM out no")
	}
}
