//go:build windows

package tools

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/nav"
	"github.com/mrlm-net/simconnect/pkg/types"
)

const metersToFeet = 3.28084

var airportICAORe = regexp.MustCompile(`^[A-Z0-9]{3,8}$`)

// strArg returns a trimmed string argument.
func strArg(args map[string]any, key string) string {
	s, _ := args[key].(string)
	return strings.TrimSpace(s)
}

// numArg returns a number argument, or def when absent.
func numArg(args map[string]any, key string, def float64) float64 {
	if n, ok := args[key].(float64); ok {
		return n
	}
	return def
}

// listArg splits a comma or space separated argument ("F, L" or "F L").
func listArg(args map[string]any, key string) []string {
	return strings.FieldsFunc(strArg(args, key), func(r rune) bool { return r == ',' || r == ' ' })
}

// icaoArg returns the upper-cased airport code in args[key], or an error result.
func icaoArg(args map[string]any, key string) (string, *mcpadapter.CallToolResult) {
	icao := strings.ToUpper(strArg(args, key))
	if !airportICAORe.MatchString(icao) {
		return "", mcpadapter.ErrorResult(fmt.Sprintf("INVALID_ARGUMENT: %s must be an airport ICAO code, e.g. \"LKPR\"", key))
	}
	return icao, nil
}

// sourceError turns a live.Source error into a tool error result.
func sourceError(what string, err error) *mcpadapter.CallToolResult {
	code := "SIM_ERROR"
	switch {
	case errors.Is(err, live.ErrNotConnected):
		code = "BRIDGE_DISCONNECTED"
	case errors.Is(err, live.ErrNotFound):
		code = "NOT_FOUND"
	case errors.Is(err, live.ErrTimeout), errors.Is(err, context.DeadlineExceeded):
		code = "TIMEOUT"
	}
	return mcpadapter.ErrorResult(fmt.Sprintf("%s: %s: %v", code, what, err))
}

// round rounds to the given decimals, for compact JSON.
func round(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}

type latLon struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func ll(p airport.LatLon) latLon { return latLon{round(p.Lat, 6), round(p.Lon, 6)} }

// fixKeyArg parses "VOZ", "VOZ.LK" or "VOZ.LK.V"; the kind defaults to a
// waypoint.
func fixKeyArg(s string) (nav.FixKey, error) {
	parts := strings.Split(strings.ToUpper(strings.TrimSpace(s)), ".")
	if parts[0] == "" || len(parts) > 3 {
		return nav.FixKey{}, fmt.Errorf("fix %q: use IDENT, IDENT.REGION or IDENT.REGION.KIND (W, V or N)", s)
	}
	k := nav.FixKey{Ident: parts[0], Kind: nav.KindWaypoint}
	if len(parts) > 1 {
		k.Region = parts[1]
	}
	if len(parts) > 2 {
		switch parts[2] {
		case "W", "V", "N":
			k.Kind = nav.FixKind(parts[2][0])
		default:
			return nav.FixKey{}, fmt.Errorf("fix %q: kind must be W (waypoint), V (VOR) or N (NDB)", s)
		}
	}
	return k, nil
}

var parkingTypeNames = map[types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE]string{
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_NONE:            "NONE",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_RAMP_GA:         "RAMP_GA",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_RAMP_GA_SMALL:   "RAMP_GA_SMALL",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_RAMP_GA_MEDIUM:  "RAMP_GA_MEDIUM",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_RAMP_GA_LARGE:   "RAMP_GA_LARGE",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_RAMP_CARGO:      "RAMP_CARGO",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_RAMP_MIL_CARGO:  "RAMP_MIL_CARGO",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_RAMP_MIL_COMBAT: "RAMP_MIL_COMBAT",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_GATE_SMALL:      "GATE_SMALL",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_GATE_MEDIUM:     "GATE_MEDIUM",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_GATE_HEAVY:      "GATE_HEAVY",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_DOCK_GA:         "DOCK_GA",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_FUEL:            "FUEL",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_VEHICLE:         "VEHICLE",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_RAMP_GA_EXTRA:   "RAMP_GA_EXTRA",
	types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE_GATE_EXTRA:      "GATE_EXTRA",
}

func parkingTypeName(t types.SIMCONNECT_FACILITY_TAXI_PARKING_TYPE) string {
	if s, ok := parkingTypeNames[t]; ok {
		return s
	}
	return fmt.Sprintf("TYPE_%d", t)
}
