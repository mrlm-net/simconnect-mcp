// Package bridge — flight_phase_test.go
// Unit tests for computeFlightPhase and computeTrackDeg.
// No build tag — runs on all platforms.
package bridge

import (
	"math"
	"testing"
)

func TestComputeFlightPhase(t *testing.T) {
	cases := []struct {
		name     string
		onGround bool
		gs, vs   float64
		want     string
	}{
		{"parked_zero_gs", true, 0, 0, "PARKED"},
		{"parked_low_gs", true, 1.9, 0, "PARKED"},
		{"taxi", true, 2.0, 0, "TAXI"},
		{"taxi_fast", true, 20, 0, "TAXI"},
		{"climb", false, 250, 1001, "CLIMB"},
		// CLIMB: VS > 1000 (exclusive)
		{"climb_boundary_above", false, 250, 1001, "CLIMB"},
		// CLIMB SHALLOW: VS > 200 and <= 1000 (so 1000 itself is CLIMB SHALLOW)
		{"climb_shallow_at_1000", false, 250, 1000, "CLIMB SHALLOW"},
		{"climb_shallow", false, 250, 600, "CLIMB SHALLOW"},
		{"climb_shallow_boundary_low", false, 250, 201, "CLIMB SHALLOW"},
		// LEVEL: VS >= -200 and <= 200 (so -200 is LEVEL, 200 is LEVEL)
		{"level_at_200", false, 250, 200, "LEVEL"},
		{"level_zero", false, 250, 0, "LEVEL"},
		{"level_at_neg200", false, 250, -200, "LEVEL"},
		// DESCENT: VS >= -800 and < -200 (so -800 is DESCENT)
		{"descent", false, 250, -500, "DESCENT"},
		{"descent_at_neg800", false, 250, -800, "DESCENT"},
		// APPROACH: VS >= -1500 and < -800 (so -800 threshold is already DESCENT)
		{"approach", false, 180, -1200, "APPROACH"},
		{"approach_at_neg801", false, 180, -801, "APPROACH"},
		// FINAL: VS < -1500 (so -1500 itself is APPROACH)
		{"approach_at_neg1500", false, 140, -1500, "APPROACH"},
		{"final", false, 140, -1501, "FINAL"},
		{"final_steep", false, 140, -3000, "FINAL"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeFlightPhase(tc.onGround, tc.gs, tc.vs)
			if got != tc.want {
				t.Errorf("computeFlightPhase(onGround=%v, gs=%.1f, vs=%.1f) = %q, want %q",
					tc.onGround, tc.gs, tc.vs, got, tc.want)
			}
		})
	}
}

func TestComputeTrackDeg(t *testing.T) {
	const tol = 0.001

	cases := []struct {
		name     string
		velX     float64 // east component
		velZ     float64 // north component
		want     float64
	}{
		{"stationary", 0, 0, 0},
		{"north", 0, 100, 0},
		{"east", 100, 0, 90},
		{"south", 0, -100, 180},
		{"west", -100, 0, 270},
		{"northeast_45", 100, 100, 45},
		{"southeast_135", 100, -100, 135},
		{"southwest_225", -100, -100, 225},
		{"northwest_315", -100, 100, 315},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeTrackDeg(tc.velX, tc.velZ)
			if math.Abs(got-tc.want) > tol {
				t.Errorf("computeTrackDeg(velX=%.1f, velZ=%.1f) = %.4f, want %.4f",
					tc.velX, tc.velZ, got, tc.want)
			}
		})
	}
}
