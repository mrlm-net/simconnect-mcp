//go:build windows

package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/traffic"
)

// RegisterLiveATCTools registers the airborne ATC tools built on the
// library's v0.16 (the live runtime runs a tower per runway and a landing
// sequence per runway end for our traffic): get_landing_sequence,
// approach_instruction, get_atc_log, get_conflicts and separation_minima.
func RegisterLiveATCTools(mcp *mcpadapter.Server, tr live.Traffic) {
	registerGetLandingSequence(mcp, tr)
	registerApproachInstruction(mcp, tr)
	registerGetATCLog(mcp, tr)
	registerGetConflicts(mcp, tr)
	registerSeparationMinima(mcp)
}

func registerGetLandingSequence(mcp *mcpadapter.Server, tr live.Traffic) {
	tool := mcpadapter.NewTool("get_landing_sequence").
		Description("The landing sequence of each runway end with our arrivals, and who uses each runway now. For every " +
			"arrival (ours and other traffic on the final): its number, the one ahead, the spacing it keeps (wake minimum, " +
			"at least 5 NM, more in low visibility or on a contaminated runway), its distance to go, predicted and sequenced " +
			"landing, and the delay it is losing. Our arrivals lose delays by themselves: slower, then a longer downwind, then " +
			"a hold at the STAR fix. The tower list shows who is holding short, lined up, on the runway or on final, and why " +
			"each waits. Our departures and crossings are cleared by the tower unless spawned with hold_for_clearances.").
		StringParam("icao", "Airport ICAO code (default: every airport with our arrivals).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao := ""
		if strArg(args, "icao") != "" {
			var bad *mcpadapter.CallToolResult
			if icao, bad = icaoArg(args, "icao"); bad != nil {
				return bad, nil
			}
		}
		type entry struct {
			Number       int     `json:"number"`
			Callsign     string  `json:"callsign"`
			Wake         string  `json:"wake"`
			Leader       string  `json:"behind,omitempty"`
			SpacingNM    float64 `json:"spacing_nm,omitempty"`
			SpacingWhy   string  `json:"spacing_why,omitempty"`
			DistanceNM   float64 `json:"distance_to_go_nm"`
			Predicted    string  `json:"predicted_landing"`
			Landing      string  `json:"sequenced_landing"`
			DelaySeconds int     `json:"delay_s"`
			Established  bool    `json:"established,omitempty"`
		}
		type runway struct {
			ICAO       string                     `json:"icao"`
			Runway     string                     `json:"runway"`
			Conditions traffic.ApproachConditions `json:"conditions"`
			Arrivals   []entry                    `json:"arrivals"`
		}
		var out []runway
		for _, s := range tr.Sequences(icao) {
			rw := runway{ICAO: s.ICAO, Runway: s.Runway, Conditions: s.Conditions, Arrivals: []entry{}}
			for _, e := range s.Sequence {
				rw.Arrivals = append(rw.Arrivals, entry{e.Number, e.Callsign, fmt.Sprintf("%s/%s", e.Wake.ICAO, e.Wake.Recat), e.Leader,
					e.SpacingNM, e.SpacingWhy, round(e.DistanceToGoNM, 1), e.ETA.UTC().Format("15:04:05Z"), e.Landing.UTC().Format("15:04:05Z"),
					int(e.Delay.Seconds()), e.Fixed})
			}
			out = append(out, rw)
		}
		if out == nil {
			out = []runway{}
		}
		tower := tr.Tower(icao)
		if tower == nil {
			tower = []live.RunwayUser{}
		}
		return mcpadapter.JSONResult(map[string]any{"sequences": out, "tower": tower})
	})
}

var approachActions = []string{"up", "down", "slow", "hold", "release", "direct", "goaround"}

func registerApproachInstruction(mcp *mcpadapter.Server, tr live.Traffic) {
	tool := mcpadapter.NewTool("approach_instruction").
		Description("Give one of our arrivals in a landing sequence an approach controller's instruction: \"up\" / \"down\" " +
			"move it a place earlier or later in the landing order (it keeps the place); \"slow\" has it lose another minute " +
			"(slower, then a longer downwind); \"hold\" holds it at its STAR's hold fix, stacked 1000 ft above the others; " +
			"\"release\" takes it out of the hold; \"direct\" sends it straight to the final, leaving out the rest of its STAR; " +
			"\"goaround\" sends it around (the published missed approach, then round to the final, sequenced again). Returns " +
			"what was said. Established arrivals (inside 8 NM) keep their place.").
		StringParam("callsign", "Our arrival's call sign (required).").
		StringParam("instruction", "up, down, slow, hold, release, direct or goaround (required).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		cs := strings.ToUpper(strings.TrimSpace(strArg(args, "callsign")))
		if !callsignRe.MatchString(cs) {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: callsign must be 2–8 letters and digits"), nil
		}
		action := strings.ToLower(strings.TrimSpace(strArg(args, "instruction")))
		ok := false
		for _, a := range approachActions {
			ok = ok || a == action
		}
		if !ok {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: instruction must be one of " + strings.Join(approachActions, ", ")), nil
		}
		said, err := tr.Approach(cs, action)
		switch {
		case errors.Is(err, live.ErrNotSequenced):
			return mcpadapter.ErrorResult(fmt.Sprintf("NOT_FOUND: %v — get_landing_sequence lists the arrivals in sequence", err)), nil
		case errors.Is(err, traffic.ErrEstablished):
			return mcpadapter.ErrorResult(fmt.Sprintf("NOT_APPLICABLE: %s is established on the approach and keeps its place", cs)), nil
		case errors.Is(err, traffic.ErrNotOnProcedure), errors.Is(err, traffic.ErrHolding), errors.Is(err, traffic.ErrNotHolding):
			return mcpadapter.ErrorResult(fmt.Sprintf("NOT_APPLICABLE: %s: %v", cs, err)), nil
		case err != nil:
			return trafficError(cs, err), nil
		}
		return mcpadapter.JSONResult(map[string]any{"callsign": cs, "instruction": action, "said": said})
	})
}

func registerGetATCLog(mcp *mcpadapter.Server, tr live.Traffic) {
	tool := mcpadapter.NewTool("get_atc_log").
		Description("The latest instructions of the runtime's controllers to our traffic, newest last, as ATC says them: " +
			"line-up and take-off clearances, runway crossings, go-arounds, speed and delay instructions, holds.").
		StringParam("icao", "Only this airport (default: all).").
		StringParam("callsign", "Only this aircraft (default: all).").
		NumberParam("limit", "Maximum messages, 1–200 (default 50).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		limit := int(numArg(args, "limit", 50))
		if limit < 1 || limit > 200 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: limit must be 1–200"), nil
		}
		icao := strings.ToUpper(strings.TrimSpace(strArg(args, "icao")))
		cs := strings.ToUpper(strings.TrimSpace(strArg(args, "callsign")))
		var out []live.ATCMessage
		for _, m := range tr.ATCLog(0) {
			if (icao == "" || m.ICAO == icao) && (cs == "" || m.Callsign == cs) {
				out = append(out, m)
			}
		}
		if len(out) > limit {
			out = out[len(out)-limit:]
		}
		if out == nil {
			out = []live.ATCMessage{}
		}
		return mcpadapter.JSONResult(map[string]any{"count": len(out), "messages": out})
	})
}

func registerGetConflicts(mcp *mcpadapter.Server, tr live.Traffic) {
	tool := mcpadapter.NewTool("get_conflicts").
		Description("Predict airborne conflicts: every pair of aircraft in the traffic picture that, flying on as they are " +
			"(track, ground speed, vertical speed), comes closer than 5 NM (3 NM near an airport) and 1000 ft within the " +
			"look-ahead. Departures and arrivals at the same airport near the runway are left to the tower. For each: when " +
			"separation is lost, the closest point, and — where one of the pair is ours — the least disturbing resolution " +
			"(a speed, level or heading change) as advice.").
		StringParam("centre", "Airport ICAO code to centre on (default: the user aircraft).").
		NumberParam("radius_nm", "Radius, NM (default 40, at most 40).").
		NumberParam("lookahead_min", "Look-ahead, minutes, 1–10 (default 5).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		centre := ""
		if strArg(args, "centre") != "" {
			var bad *mcpadapter.CallToolResult
			if centre, bad = icaoArg(args, "centre"); bad != nil {
				return bad, nil
			}
		}
		radius := numArg(args, "radius_nm", 40)
		if radius <= 0 || radius > 40 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: radius_nm must be 1–40"), nil
		}
		look := numArg(args, "lookahead_min", 5)
		if look < 1 || look > 10 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: lookahead_min must be 1–10"), nil
		}
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		list, err := tr.Picture(ctx, centre, radius)
		if err != nil {
			return trafficError("traffic picture", err), nil
		}
		opts := traffic.ConflictOptions{LookAhead: time.Duration(look * float64(time.Minute))}
		type conflict struct {
			Pair           [2]string           `json:"pair"`
			LossInSeconds  int                 `json:"loss_in_s"`
			ClosestNM      float64             `json:"closest_nm"`
			ClosestVertFt  float64             `json:"closest_vertical_ft"`
			ClosestInSecs  int                 `json:"closest_in_s"`
			MinimumNM      float64             `json:"minimum_nm"`
			Resolution     *traffic.Resolution `json:"resolution,omitempty"`
			ResolutionNote string              `json:"resolution_note,omitempty"`
		}
		ours := func(a traffic.TrackedAircraft, _ traffic.ResolutionKind) bool { return a.Ours }
		out := []conflict{}
		for _, c := range traffic.PredictConflicts(list, opts) {
			x := conflict{Pair: [2]string{c.A, c.B}, LossInSeconds: int(c.In.Seconds()), ClosestNM: round(c.ClosestNM, 1),
				ClosestVertFt: round(c.VerticalFt, -1), ClosestInSecs: int(c.ClosestIn.Seconds()), MinimumNM: c.MinNM}
			if r, ok := traffic.ResolveConflict(c, list, ours, opts); ok {
				x.Resolution = &r
			} else {
				x.ResolutionNote = "neither is ours, or nothing tried keeps them apart"
			}
			out = append(out, x)
		}
		return mcpadapter.JSONResult(map[string]any{"count": len(out), "aircraft_checked": len(list), "conflicts": out})
	})
}

func registerSeparationMinima(mcp *mcpadapter.Server) {
	tool := mcpadapter.NewTool("separation_minima").
		Description("Wake turbulence and runway separation for a pair of aircraft types (ICAO designators, e.g. A320, B77W, " +
			"A388): their wake categories (ICAO and RECAT-EU), the spacing the follower keeps behind the leader on final " +
			"— by the scheme, and in the given conditions (low visibility, runway surface, reduced separation) — the " +
			"departure interval behind the leader, and the runway occupancy of each. Pure calculation.").
		StringParam("leader", "Leading aircraft type (required).").
		StringParam("follower", "Following aircraft type (required).").
		StringParam("scheme", "icao or recat (default icao).").
		NumberParam("visibility_m", "Visibility on final, m (default: good).").
		StringParam("surface", "Runway surface: dry, wet or contaminated (default dry).").
		NumberParam("same_route", "1: departures on the same SID (a longer interval); default 0.").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		lt, ft := strings.ToUpper(strings.TrimSpace(strArg(args, "leader"))), strings.ToUpper(strings.TrimSpace(strArg(args, "follower")))
		if lt == "" || ft == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: leader and follower types are required"), nil
		}
		scheme := traffic.SchemeICAO
		switch strings.ToLower(strArg(args, "scheme")) {
		case "", "icao":
		case "recat", "recat-eu":
			scheme = traffic.SchemeRecat
		default:
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: scheme must be icao or recat"), nil
		}
		surface := traffic.RunwayDry
		switch strings.ToLower(strArg(args, "surface")) {
		case "", "dry":
		case "wet":
			surface = traffic.RunwayWet
		case "contaminated":
			surface = traffic.RunwayContaminated
		default:
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: surface must be dry, wet or contaminated"), nil
		}
		lw, fw := traffic.WakeFor(lt), traffic.WakeFor(ft)
		cond := traffic.ApproachConditions{VisibilityM: numArg(args, "visibility_m", 0), Surface: surface}
		spacing, why := traffic.ArrivalSpacing(lw, fw, scheme, cond, true)
		return mcpadapter.JSONResult(map[string]any{
			"leader":                      map[string]any{"type": lt, "wake": lw, "landing_occupancy_s": int(traffic.RunwayOccupancyIn(lw, true, surface).Seconds())},
			"follower":                    map[string]any{"type": ft, "wake": fw, "landing_occupancy_s": int(traffic.RunwayOccupancyIn(fw, true, surface).Seconds())},
			"wake_minimum_nm":             traffic.ArrivalSeparationNM(lw, fw, scheme),
			"spacing_on_final_nm":         spacing,
			"spacing_why":                 why,
			"departure_interval_s":        int(traffic.DepartureInterval(lw, fw, numArg(args, "same_route", 0) == 1).Seconds()),
			"follower_takeoff_occupancy_s": int(traffic.RunwayOccupancyIn(fw, false, surface).Seconds()),
		})
	})
}
