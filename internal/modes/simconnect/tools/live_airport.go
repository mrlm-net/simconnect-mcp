//go:build windows

package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/airport"
)

// RegisterLiveAirportTools registers the airport tools built on pkg/airport:
// get_airport_procedures, plan_taxi_route, get_runway_entries_exits and
// find_stands.
func RegisterLiveAirportTools(mcp *mcpadapter.Server, src live.Source) {
	registerGetAirportProcedures(mcp, src)
	registerPlanTaxiRoute(mcp, src)
	registerGetRunwayEntriesExits(mcp, src)
	registerFindStands(mcp, src)
}

// ── Procedures ──────────────────────────────────────────────────────────────

type procedureSummary struct {
	Name               string   `json:"name"`
	Runways            []string `json:"runways,omitempty"`
	EnrouteTransitions []string `json:"enroute_transitions,omitempty"`
}

type approachSummary struct {
	Name        string   `json:"name"`
	Runway      string   `json:"runway"`
	Transitions []string `json:"transitions,omitempty"`
}

type procedurePoint struct {
	Ident       string  `json:"ident,omitempty"`
	Kind        string  `json:"kind,omitempty"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Leg         string  `json:"leg"`
	CourseTrue  float64 `json:"course_true,omitempty"`
	AltMinFt    float64 `json:"alt_min_ft,omitempty"`
	AltMaxFt    float64 `json:"alt_max_ft,omitempty"`
	SpeedMaxKts float64 `json:"speed_max_kts,omitempty"`
	FlyOver     bool    `json:"fly_over,omitempty"`
	IAF         bool    `json:"iaf,omitempty"`
	FAF         bool    `json:"faf,omitempty"`
	MAP         bool    `json:"map,omitempty"`
	Vectors     bool    `json:"vectors,omitempty"`
}

func procedurePoints(pts []airport.NavPoint) []procedurePoint {
	out := make([]procedurePoint, 0, len(pts))
	for _, p := range pts {
		out = append(out, procedurePoint{
			Ident: p.Ident, Kind: p.Kind, Lat: round(p.Position.Lat, 6), Lon: round(p.Position.Lon, 6),
			Leg: p.LegType.String(), CourseTrue: round(p.Course, 1),
			AltMinFt: round(p.AltMin*metersToFeet, -1), AltMaxFt: round(p.AltMax*metersToFeet, -1),
			SpeedMaxKts: p.SpeedMax, FlyOver: p.FlyOver, IAF: p.IAF, FAF: p.FAF, MAP: p.MAP, Vectors: p.Vectors,
		})
	}
	return out
}

func transitionNames(ts []airport.Transition) []string {
	var out []string
	for _, t := range ts {
		if t.Name != "" {
			out = append(out, t.Name)
		}
	}
	return out
}

func summarise(ps []airport.Procedure) []procedureSummary {
	out := make([]procedureSummary, 0, len(ps))
	for _, p := range ps {
		out = append(out, procedureSummary{Name: p.Name, Runways: p.Runways(), EnrouteTransitions: transitionNames(p.EnrouteTransitions)})
	}
	return out
}

func registerGetAirportProcedures(mcp *mcpadapter.Server, src live.Source) {
	tool := mcpadapter.NewTool("get_airport_procedures").
		Description("List an airport's SIDs, STARs and instrument approaches from the simulator's navdata, or resolve one into "+
			"the points it flies. Without name: every procedure (optionally only those of runway) with its runways and "+
			"transitions. With name: the SID, STAR or approach as points in order — ident, position, ARINC 424 leg type, "+
			"true course, altitude window (ft) and speed limit, IAF/FAF/MAP — for the runway and transition given. "+
			"Also returns the magnetic variation. Unknown airports end in NOT_FOUND after about 30 s.").
		StringParam("icao", "Airport ICAO code, e.g. \"LKPR\" (required).").
		StringParam("runway", "Runway end, e.g. \"24\" or \"06L\". Filters the list; required to resolve a SID or STAR serving several runways.").
		StringParam("name", "Procedure to resolve: a SID or STAR name (\"VOZ5M\") or an approach name (\"ILS 24\", \"RNAV 06 Z\").").
		StringParam("transition", "Enroute transition of a SID/STAR, or the approach transition (IAF) of an approach.").
		Required("icao").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, bad := icaoArg(args, "icao")
		if bad != nil {
			return bad, nil
		}
		runway := strings.ToUpper(strArg(args, "runway"))
		name := strings.ToUpper(strArg(args, "name"))
		transition := strings.ToUpper(strArg(args, "transition"))

		procs, err := src.Procedures(ctx, icao)
		if err != nil {
			return sourceError("procedures of "+icao, err), nil
		}

		if name == "" {
			sids, stars := procs.Departures, procs.Arrivals
			approaches := procs.Approaches
			if runway != "" {
				sids, stars, approaches = procs.SIDsFor(runway), procs.STARsFor(runway), procs.ApproachesFor(runway)
			}
			appr := make([]approachSummary, 0, len(approaches))
			for _, a := range approaches {
				appr = append(appr, approachSummary{Name: a.Name, Runway: a.Runway, Transitions: transitionNames(a.Transitions)})
			}
			return mcpadapter.JSONResult(map[string]any{
				"icao": icao, "runway": runway, "magvar": round(procs.MagVar, 1),
				"sids": summarise(sids), "stars": summarise(stars), "approaches": appr,
			})
		}

		kind, points, missed, err := resolveProcedure(ctx, src, icao, procs, name, runway, transition)
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("INVALID_ARGUMENT: %v", err)), nil
		}
		out := map[string]any{
			"icao": icao, "kind": kind, "name": name, "runway": runway, "transition": transition,
			"magvar": round(procs.MagVar, 1), "points": procedurePoints(points),
		}
		if missed != nil {
			out["missed_approach"] = procedurePoints(missed)
		}
		return mcpadapter.JSONResult(out)
	})
}

// resolveProcedure finds name among the SIDs, STARs and approaches and
// resolves it into points.
func resolveProcedure(ctx context.Context, src live.Source, icao string, procs *airport.Procedures, name, runway, transition string) (string, []airport.NavPoint, []airport.NavPoint, error) {
	pick := func(ps []airport.Procedure) (airport.Procedure, bool) {
		for _, p := range ps {
			if strings.EqualFold(p.Name, name) {
				return p, true
			}
		}
		return airport.Procedure{}, false
	}
	needRunway := func(p airport.Procedure) (string, error) {
		if runway != "" {
			return runway, nil
		}
		if rw := p.Runways(); len(rw) == 1 {
			return rw[0], nil
		} else if len(rw) > 1 {
			return "", fmt.Errorf("%s serves runways %s: give runway", p.Name, strings.Join(rw, ", "))
		}
		return "", nil
	}

	if p, ok := pick(procs.Departures); ok {
		rw, err := needRunway(p)
		if err != nil {
			return "", nil, nil, err
		}
		// A SID starts at the departure end of the runway: the far threshold.
		layout, err := src.Layout(ctx, icao)
		if err != nil {
			return "", nil, nil, err
		}
		r, end, ok := layout.RunwayEnd(rw)
		if !ok {
			return "", nil, nil, fmt.Errorf("%s has no runway %s", icao, rw)
		}
		der := r.Primary.Threshold
		if end.Name == r.Primary.Name {
			der = r.Secondary.Threshold
		}
		pts, err := procs.ResolveSID(p.Name, rw, transition, der, r.Altitude)
		return "SID", pts, nil, err
	}
	if p, ok := pick(procs.Arrivals); ok {
		rw, err := needRunway(p)
		if err != nil {
			return "", nil, nil, err
		}
		pts, err := procs.ResolveSTAR(p.Name, transition, rw)
		return "STAR", pts, nil, err
	}
	for _, a := range procs.Approaches {
		if strings.EqualFold(a.Name, name) {
			pts, err := procs.ResolveApproach(a.Name, transition)
			if err != nil {
				return "", nil, nil, err
			}
			missed, _ := procs.MissedApproach(a.Name)
			return "APPROACH", pts, missed, nil
		}
	}
	return "", nil, nil, fmt.Errorf("%s has no SID, STAR or approach %q — list them without name", icao, name)
}

// ── Taxi routes ─────────────────────────────────────────────────────────────

type taxiRoute struct {
	ICAO            string   `json:"icao"`
	Direction       string   `json:"direction"`
	Parking         string   `json:"parking"`
	Runway          string   `json:"runway"`
	Entry           string   `json:"entry,omitempty"`
	Exit            string   `json:"exit,omitempty"`
	Instruction     string   `json:"instruction"`
	LengthM         float64  `json:"length_m"`
	Taxiways        []string `json:"taxiways"`
	RunwayCrossings []string `json:"runway_crossings"`
	HoldShort       string   `json:"hold_short,omitempty"`
	Tight           bool     `json:"tight,omitempty"`
	Points          []latLon `json:"points,omitempty"`
}

func registerPlanTaxiRoute(mcp *mcpadapter.Server, src live.Source) {
	tool := mcpadapter.NewTool("plan_taxi_route").
		Description("Plan a taxi route at an airport on the simulator's taxi network, the way ATC would give it: fewer turns, "+
			"no needless runway crossings, taxiways the aircraft fits. direction=departure: from the parking stand to the "+
			"holding point of the runway (full length, or at entry). direction=arrival: from a runway exit (exit, or the one "+
			"reached after rollout_m) to the stand. Returns the taxi instruction, length, taxiways in order, runway "+
			"crossings, the holding point, and optionally the points. Use get_runway_entries_exits and find_stands for names.").
		StringParam("icao", "Airport ICAO code (required).").
		StringParam("parking", "Stand label, e.g. \"C22\", \"S22A\" (required).").
		StringParam("runway", "Runway end, e.g. \"24\" (required).").
		StringParam("direction", "\"departure\" (default) or \"arrival\".").
		StringParam("entry", "Departure: the taxiway to enter the runway by (intersection departure), e.g. \"B\".").
		StringParam("exit", "Arrival: the taxiway to vacate the runway by, e.g. \"D\".").
		NumberParam("rollout_m", "Arrival without exit: landing roll in meters before vacating (default 1500).").
		StringParam("via", "Taxiways to follow in order, e.g. \"F, L\".").
		NumberParam("wingspan_m", "Aircraft wing span in meters: keeps to taxiways it fits (e.g. 35.8 for an A320, 64.8 for a 777-300ER).").
		BoolParam("include_points", "Include the route's points (lat/lon). Default false.").
		Required("icao", "parking", "runway").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, bad := icaoArg(args, "icao")
		if bad != nil {
			return bad, nil
		}
		parking, runway := strings.ToUpper(strArg(args, "parking")), strings.ToUpper(strArg(args, "runway"))
		if parking == "" || runway == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: parking and runway are required"), nil
		}
		direction := strings.ToLower(strArg(args, "direction"))
		if direction == "" {
			direction = "departure"
		}
		if direction != "departure" && direction != "arrival" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: direction must be departure or arrival"), nil
		}

		layout, err := src.Layout(ctx, icao)
		if err != nil {
			return sourceError("layout of "+icao, err), nil
		}
		g, err := src.Graph(ctx, icao)
		if err != nil {
			return sourceError("taxi graph of "+icao, err), nil
		}
		idx, err := layout.ParkingIndex(parking)
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("INVALID_ARGUMENT: parking %s: %v — use find_stands", parking, err)), nil
		}
		opts := airport.RouteOptions{Taxiways: listArg(args, "via"), HalfSpan: numArg(args, "wingspan_m", 0) / 2}

		var route *airport.Route
		out := taxiRoute{ICAO: icao, Direction: direction, Parking: layout.Parking[idx].Label(), Runway: runway}
		if direction == "departure" {
			out.Entry = strings.ToUpper(strArg(args, "entry"))
			route, err = g.RouteToRunwayEntry(idx, runway, out.Entry, opts)
		} else {
			var exit airport.RunwayExit
			exit, err = findExit(g, runway, strings.ToUpper(strArg(args, "exit")), numArg(args, "rollout_m", 1500))
			if err == nil {
				out.Exit = exit.Taxiway
				route, err = g.RouteFromRunway(exit, idx, opts)
			}
		}
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("NO_ROUTE: %v", err)), nil
		}

		out.LengthM = round(route.Length, 0)
		out.Taxiways, out.RunwayCrossings, out.Tight = route.Taxiways, route.RunwayCrossings, route.Tight
		if out.RunwayCrossings == nil {
			out.RunwayCrossings = []string{}
		}
		if route.Entry != "" {
			out.Entry = route.Entry
		}
		if route.HoldShort != nil {
			out.HoldShort = route.HoldShort.Name
			if route.HoldShort.ILS {
				out.HoldShort += " (ILS hold)"
			}
		}
		out.Instruction = taxiInstruction(out)
		if b, _ := args["include_points"].(bool); b {
			for _, p := range route.Points {
				out.Points = append(out.Points, ll(p))
			}
		}
		return mcpadapter.JSONResult(out)
	})
}

// findExit picks the named exit, or the one ExitFor gives after rollout.
func findExit(g *airport.Graph, runway, name string, rollout float64) (airport.RunwayExit, error) {
	if name == "" {
		return g.ExitFor(runway, rollout)
	}
	exits, err := g.RunwayExits(runway)
	if err != nil {
		return airport.RunwayExit{}, err
	}
	var names []string
	for _, e := range exits {
		if strings.EqualFold(e.Taxiway, name) {
			return e, nil
		}
		names = append(names, e.Taxiway)
	}
	return airport.RunwayExit{}, fmt.Errorf("runway %s has no exit %s (exits: %s)", runway, name, strings.Join(names, ", "))
}

func taxiInstruction(r taxiRoute) string {
	via := strings.Join(r.Taxiways, " ")
	var s string
	if r.Direction == "departure" {
		s = "taxi to holding point runway " + r.Runway
		if r.Entry != "" {
			s += " at " + r.Entry
		}
	} else {
		s = "vacate via " + r.Exit + ", taxi to stand " + r.Parking
	}
	if via != "" {
		s += " via " + via
	}
	for _, c := range r.RunwayCrossings {
		s += ", cross runway " + c
	}
	return s
}

// ── Runway entries and exits ───────────────────────────────────────────────

func registerGetRunwayEntriesExits(mcp *mcpadapter.Server, src live.Source) {
	tool := mcpadapter.NewTool("get_runway_entries_exits").
		Description("List the taxiways onto a runway end for departures (nearest the threshold first, with the runway length "+
			"remaining ahead) and the exits for landings on it (distance from the threshold, angle, high-speed, side). "+
			"Names feed plan_taxi_route's entry and exit.").
		StringParam("icao", "Airport ICAO code (required).").
		StringParam("runway", "Runway end, e.g. \"24\" (required).").
		Required("icao", "runway").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, bad := icaoArg(args, "icao")
		if bad != nil {
			return bad, nil
		}
		runway := strings.ToUpper(strArg(args, "runway"))
		g, err := src.Graph(ctx, icao)
		if err != nil {
			return sourceError("taxi graph of "+icao, err), nil
		}
		entries, err := g.RunwayEntries(runway)
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("INVALID_ARGUMENT: %v", err)), nil
		}
		exits, err := g.RunwayExits(runway)
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("INVALID_ARGUMENT: %v", err)), nil
		}
		type entry struct {
			Taxiway       string  `json:"taxiway"`
			FromThreshold float64 `json:"from_threshold_m"`
			RemainingM    float64 `json:"remaining_m"`
			Angle         float64 `json:"angle"`
		}
		type exit struct {
			Taxiway   string  `json:"taxiway"`
			AlongM    float64 `json:"from_threshold_m"`
			Angle     float64 `json:"angle"`
			HighSpeed bool    `json:"high_speed,omitempty"`
			Side      string  `json:"side"`
		}
		ens := make([]entry, 0, len(entries))
		for _, e := range entries {
			ens = append(ens, entry{e.Taxiway, round(e.FromThreshold, 0), round(e.Remaining, 0), round(e.Angle, 0)})
		}
		exs := make([]exit, 0, len(exits))
		for _, e := range exits {
			exs = append(exs, exit{e.Taxiway, round(e.Along, 0), round(e.Angle, 0), e.HighSpeed, e.Side.String()})
		}
		return mcpadapter.JSONResult(map[string]any{"icao": icao, "runway": runway, "entries": ens, "exits": exs})
	})
}

// ── Stands ──────────────────────────────────────────────────────────────────

func registerFindStands(mcp *mcpadapter.Server, src live.Source) {
	tool := mcpadapter.NewTool("find_stands").
		Description("Find parking stands at an airport that fit an aircraft: stand label, type, size class (small/medium/heavy), "+
			"radius, heading, the airlines the scenery assigns, and the stands it overlaps (split or alternate stands). "+
			"Filter by wing span, airline (ICAO code, e.g. \"DLH\"; stands without airlines serve any) and gates only.").
		StringParam("icao", "Airport ICAO code (required).").
		NumberParam("wingspan_m", "Aircraft wing span in meters (default 0: any stand).").
		StringParam("airline", "Airline ICAO code the stand must serve.").
		BoolParam("gates_only", "Only gates (no ramps). Default false.").
		NumberParam("limit", "Maximum stands, 1–500 (default 100).").
		Required("icao").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, bad := icaoArg(args, "icao")
		if bad != nil {
			return bad, nil
		}
		limit := int(numArg(args, "limit", 100))
		if limit < 1 || limit > 500 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: limit must be 1–500"), nil
		}
		airline := strings.ToUpper(strArg(args, "airline"))
		gates, _ := args["gates_only"].(bool)
		layout, err := src.Layout(ctx, icao)
		if err != nil {
			return sourceError("layout of "+icao, err), nil
		}

		type stand struct {
			Label     string   `json:"label"`
			Type      string   `json:"type"`
			Size      string   `json:"size"`
			RadiusM   float64  `json:"radius_m"`
			Heading   float64  `json:"heading_true"`
			Airlines  []string `json:"airlines,omitempty"`
			Conflicts []string `json:"overlaps,omitempty"`
			Lat       float64  `json:"lat"`
			Lon       float64  `json:"lon"`
		}
		// RADIUS is half the space a stand offers: half the span plus a margin.
		minRadius := 0.0
		if span := numArg(args, "wingspan_m", 0); span > 0 {
			minRadius = span/2 + 1
		}
		var out []stand
		total := 0
		for _, i := range layout.SuitableStands(minRadius) {
			p := layout.Parking[i]
			if gates && !p.IsGate() || airline != "" && !p.ServesAirline(airline) {
				continue
			}
			total++
			if len(out) >= limit {
				continue
			}
			var conflicts []string
			for _, j := range layout.ParkingConflicts(i) {
				conflicts = append(conflicts, layout.Parking[j].Label())
			}
			out = append(out, stand{p.Label(), parkingTypeName(p.Type), p.Size().String(), round(p.Radius, 1),
				round(p.Heading, 0), p.Airlines, conflicts, round(p.Position.Lat, 6), round(p.Position.Lon, 6)})
		}
		return mcpadapter.JSONResult(map[string]any{"icao": icao, "total": total, "count": len(out), "stands": out})
	})
}
