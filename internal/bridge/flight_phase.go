// Package bridge — flight_phase.go
// Platform-agnostic helpers for deriving track bearing and flight phase
// from raw SimConnect velocity vectors.
// No build tag — compiles and tests on all platforms.
package bridge

import "math"

// computeTrackDeg converts VELOCITY WORLD X (east, fps) and VELOCITY WORLD Z
// (north, fps) into a true track bearing in degrees, 0–359.9, measured
// clockwise from north.  Returns 0 when both components are zero (stationary).
func computeTrackDeg(velX, velZ float64) float64 {
	if velX == 0 && velZ == 0 {
		return 0
	}
	deg := math.Atan2(velX, velZ) * 180.0 / math.Pi
	if deg < 0 {
		deg += 360.0
	}
	return deg
}

// computeFlightPhase infers the phase of flight from on-ground status,
// ground speed (knots), and vertical speed (feet per minute).
//
// Phase table:
//
//	on_ground + GS < 2 kts       → PARKED
//	on_ground + GS ≥ 2 kts       → TAXI
//	airborne + VS > +1000 fpm     → CLIMB
//	airborne + VS +200…+1000 fpm  → CLIMB SHALLOW
//	airborne + VS −200…+200 fpm   → LEVEL
//	airborne + VS −800…−200 fpm   → DESCENT
//	airborne + VS −1500…−800 fpm  → APPROACH
//	airborne + VS < −1500 fpm     → FINAL
func computeFlightPhase(onGround bool, groundSpeedKts, verticalSpeedFPM float64) string {
	if onGround {
		if groundSpeedKts < 2 {
			return "PARKED"
		}
		return "TAXI"
	}
	switch {
	case verticalSpeedFPM > 1000:
		return "CLIMB"
	case verticalSpeedFPM > 200:
		return "CLIMB SHALLOW"
	case verticalSpeedFPM >= -200:
		return "LEVEL"
	case verticalSpeedFPM >= -800:
		return "DESCENT"
	case verticalSpeedFPM >= -1500:
		return "APPROACH"
	default:
		return "FINAL"
	}
}
