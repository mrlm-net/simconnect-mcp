//go:build windows

package tools

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/nav"
	"github.com/mrlm-net/simconnect/pkg/traffic"
)

var callsignRe = regexp.MustCompile(`^[A-Z0-9]{2,8}$`)

const spawnTimeout = 60 * time.Second

// RegisterLiveTrafficTools registers the AI traffic tools built on
// pkg/traffic: list_aircraft_models, spawn_departure, spawn_arrival,
// list_our_traffic, atc_clearance, get_traffic_picture and
// generate_schedule.
func RegisterLiveTrafficTools(mcp *mcpadapter.Server, src live.Source, tr live.Traffic) {
	registerListAircraftModels(mcp, tr)
	registerSpawnDeparture(mcp, src, tr)
	registerSpawnArrival(mcp, src, tr)
	registerListOurTraffic(mcp, tr)
	registerATCClearance(mcp, tr)
	registerGetTrafficPicture(mcp, tr)
	registerGenerateSchedule(mcp)
}

func trafficError(what string, err error) *mcpadapter.CallToolResult {
	if errors.Is(err, live.ErrUnknownFlight) {
		return mcpadapter.ErrorResult(fmt.Sprintf("NOT_FOUND: %s: %v — list_our_traffic lists ours", what, err))
	}
	if errors.Is(err, live.ErrNotConnected) || errors.Is(err, live.ErrNotFound) || errors.Is(err, live.ErrTimeout) {
		return sourceError(what, err)
	}
	return mcpadapter.ErrorResult(fmt.Sprintf("TRAFFIC_ERROR: %s: %v", what, err))
}

func registerListAircraftModels(mcp *mcpadapter.Server, tr live.Traffic) {
	tool := mcpadapter.NewTool("list_aircraft_models").
		Description("List the aircraft installed in the simulator that AI traffic can use: \"title\" or \"title|livery\" as " +
			"spawn_departure and spawn_arrival take them in model. Filter with words that must all appear (e.g. \"A320 " +
			"Lufthansa\"). Without a model the spawn tools pick one of the aircraft type in the call sign's airline livery.").
		StringParam("filter", "Words that must all appear in the title (case-insensitive).").
		NumberParam("limit", "Maximum titles, 1–500 (default 100).").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		limit := int(numArg(args, "limit", 100))
		if limit < 1 || limit > 500 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: limit must be 1–500"), nil
		}
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		models, err := tr.Models(ctx)
		if err != nil {
			return trafficError("aircraft models", err), nil
		}
		words := strings.Fields(strings.ToLower(strArg(args, "filter")))
		var out []string
		total := 0
	next:
		for _, m := range models {
			lm := strings.ToLower(m)
			for _, w := range words {
				if !strings.Contains(lm, w) {
					continue next
				}
			}
			total++
			if len(out) < limit {
				out = append(out, m)
			}
		}
		return mcpadapter.JSONResult(map[string]any{"total": total, "count": len(out), "models": out})
	})
}

// spawnCommon reads the arguments both spawn tools share.
func spawnCommon(ctx context.Context, src live.Source, args map[string]any, arrival bool) (icao, callsign, runway string, g *airport.Graph, procs *airport.Procedures, lim airport.Limits, bad *mcpadapter.CallToolResult) {
	icao, bad = icaoArg(args, "icao")
	if bad != nil {
		return
	}
	callsign = strings.ToUpper(strArg(args, "callsign"))
	if !callsignRe.MatchString(callsign) {
		bad = mcpadapter.ErrorResult("INVALID_ARGUMENT: callsign must be 2–8 letters or digits, e.g. \"CSA123\" (the first three letters pick the airline)")
		return
	}
	var err error
	if g, err = src.Graph(ctx, icao); err != nil {
		bad = sourceError("taxi graph of "+icao, err)
		return
	}
	procs, _ = src.Procedures(ctx, icao) // optional: SIDs/STARs and the limits
	lim = airport.LimitsFor(g.Layout, procs)
	runway = strings.ToUpper(strArg(args, "runway"))
	if runway == "" {
		w, err := src.Weather(ctx)
		if err != nil {
			bad = sourceError("weather for the runway in use (give runway)", err)
			return
		}
		use := nav.ActiveRunways(g.Layout, w, nav.RunwayLimitsFrom(lim))
		if runway = use.Departure.Name; arrival {
			runway = use.Arrival.Name
		}
	}
	if _, _, ok := g.Layout.RunwayEnd(runway); !ok {
		bad = mcpadapter.ErrorResult(fmt.Sprintf("INVALID_ARGUMENT: %s has no runway %s", icao, runway))
	}
	return
}

// pickProcedure returns the named procedure, the first for the runway with
// "" or "auto", or none with "none".
func pickProcedure(list []airport.Procedure, name, kind, runway string) (airport.Procedure, bool, error) {
	switch strings.ToLower(name) {
	case "none":
		return airport.Procedure{}, false, nil
	case "", "auto":
		if len(list) == 0 {
			return airport.Procedure{}, false, nil
		}
		return list[0], true, nil
	}
	for _, p := range list {
		if strings.EqualFold(p.Name, name) {
			return p, true, nil
		}
	}
	var names []string
	for _, p := range list {
		names = append(names, p.Name)
	}
	return airport.Procedure{}, false, fmt.Errorf("no %s %q for runway %s (%s)", kind, name, runway, strings.Join(names, ", "))
}

// departureEnd is where a SID from runway starts: the far threshold.
func departureEnd(l *airport.Layout, runway string) (airport.LatLon, float64, bool) {
	r, end, ok := l.RunwayEnd(runway)
	if !ok {
		return airport.LatLon{}, 0, false
	}
	if end.Name == r.Primary.Name {
		return r.Secondary.Threshold, r.Altitude, true
	}
	return r.Primary.Threshold, r.Altitude, true
}

func registerSpawnDeparture(mcp *mcpadapter.Server, src live.Source, tr live.Traffic) {
	tool := mcpadapter.NewTool("spawn_departure").
		Description("Put an AI departure under our control on a stand at an airport in the simulator (this adds an aircraft to " +
			"the sim). It pushes back, taxis the planned route to the runway, lines up and takes off, then flies the SID. " +
			"With hold_for_clearances (default true) it waits at every step for atc_clearance: pushback, taxi, (cross), " +
			"lineup, takeoff; otherwise it goes by itself. Stand, runway (in use for the weather), SID and model (the type " +
			"in the call sign's airline livery) are chosen when not given. Returns the flight: stand, runway, SID, taxi " +
			"route, state and the clearances it takes now. Follow it with list_our_traffic.").
		StringParam("icao", "Airport ICAO code (required); the simulator must have it loaded around the user aircraft.").
		StringParam("callsign", "Call sign, e.g. \"CSA123\" (required).").
		StringParam("stand", "Stand label, e.g. \"C22\" (default: a free stand that fits).").
		StringParam("runway", "Departure runway (default: in use).").
		StringParam("entry", "Runway entry taxiway for an intersection departure, e.g. \"B\" (default: full length).").
		StringParam("sid", "SID name, \"auto\" (default: one for the runway) or \"none\" (climb straight ahead).").
		StringParam("model", "Aircraft title from list_aircraft_models (\"title\" or \"title|livery\").").
		StringParam("aircraft_type", "ICAO type to pick a model by, e.g. \"A20N\", \"B738\" (default A320).").
		StringParam("via", "Taxiways to follow in order, e.g. \"F, L\".").
		BoolParam("hold_for_clearances", "Wait at every step for atc_clearance (default true).").
		Required("icao", "callsign").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		ctx, cancel := context.WithTimeout(ctx, spawnTimeout)
		defer cancel()
		icao, callsign, runway, g, procs, lim, bad := spawnCommon(ctx, src, args, false)
		if bad != nil {
			return bad, nil
		}
		s := live.DepartureSpec{Graph: g, Limits: &lim, Callsign: callsign, Stand: strings.ToUpper(strArg(args, "stand")),
			Runway: runway, Entry: strings.ToUpper(strArg(args, "entry")), Model: strArg(args, "model"),
			Type: strings.ToUpper(strArg(args, "aircraft_type")), Taxiways: listArg(args, "via"), HoldForClearances: true}
		if b, ok := args["hold_for_clearances"].(bool); ok {
			s.HoldForClearances = b
		}
		if procs != nil {
			sid, ok, err := pickProcedure(procs.SIDsFor(runway), strArg(args, "sid"), "SID", runway)
			if err != nil {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: " + err.Error()), nil
			}
			if ok {
				der, elev, _ := departureEnd(g.Layout, runway)
				pts, err := procs.ResolveSID(sid.Name, runway, "", der, elev)
				if err != nil {
					return mcpadapter.ErrorResult(fmt.Sprintf("INVALID_ARGUMENT: SID %s: %v", sid.Name, err)), nil
				}
				s.SID, s.Departure = sid.Name, pts
			}
		}
		v, err := tr.SpawnDeparture(ctx, s)
		if err != nil {
			return trafficError("departure "+callsign+" at "+icao, err), nil
		}
		return mcpadapter.JSONResult(v)
	})
}

func registerSpawnArrival(mcp *mcpadapter.Server, src live.Source, tr live.Traffic) {
	tool := mcpadapter.NewTool("spawn_arrival").
		Description("Put an AI arrival under our control into the simulator (this adds an aircraft to the sim): at the STAR's " +
			"first fix (or spawn_nm out on final with star=\"none\"), flying the STAR and the best approach, landing, vacating " +
			"and taxiing to a stand. With hold_for_clearance (default true) it waits clear of the runway for atc_clearance " +
			"taxi and before runway crossings; goaround sends it around on final. Runway (in use for the weather), stand, " +
			"STAR and model are chosen when not given. Returns the flight; follow it with list_our_traffic.").
		StringParam("icao", "Airport ICAO code (required).").
		StringParam("callsign", "Call sign, e.g. \"DLH4AB\" (required).").
		StringParam("runway", "Landing runway (default: in use).").
		StringParam("stand", "Stand label (default: a free stand that fits, near the runway).").
		StringParam("star", "STAR name, \"auto\" (default: one for the runway) or \"none\" (straight in on final).").
		NumberParam("spawn_nm", "With star=\"none\": distance out on final to start, NM (default 5).").
		StringParam("model", "Aircraft title from list_aircraft_models.").
		StringParam("aircraft_type", "ICAO type to pick a model by (default A320).").
		StringParam("via", "Taxiways to follow to the stand, in order.").
		BoolParam("hold_for_clearance", "Wait for the taxi clearance and at runway crossings (default true).").
		Required("icao", "callsign").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		ctx, cancel := context.WithTimeout(ctx, spawnTimeout)
		defer cancel()
		icao, callsign, runway, g, procs, lim, bad := spawnCommon(ctx, src, args, true)
		if bad != nil {
			return bad, nil
		}
		s := live.ArrivalSpec{Graph: g, Limits: &lim, Callsign: callsign, Stand: strings.ToUpper(strArg(args, "stand")),
			Runway: runway, Model: strArg(args, "model"), Type: strings.ToUpper(strArg(args, "aircraft_type")),
			SpawnNM: numArg(args, "spawn_nm", 0), Taxiways: listArg(args, "via"), HoldForClearance: true}
		if b, ok := args["hold_for_clearance"].(bool); ok {
			s.HoldForClearance = b
		}
		if procs != nil {
			star, ok, err := pickProcedure(procs.STARsFor(runway), strArg(args, "star"), "STAR", runway)
			if err != nil {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: " + err.Error()), nil
			}
			if ok {
				pts, err := procs.Arrival(runway, starEntry(star, runway))
				if err != nil {
					return mcpadapter.ErrorResult(fmt.Sprintf("INVALID_ARGUMENT: STAR %s: %v", star.Name, err)), nil
				}
				s.STAR, s.Procedure = star.Name, pts
				if a, ok := procs.BestApproach(runway); ok {
					s.STAR += " → " + a.Name
				}
			}
		}
		v, err := tr.SpawnArrival(ctx, s)
		if err != nil {
			return trafficError("arrival "+callsign+" at "+icao, err), nil
		}
		return mcpadapter.JSONResult(v)
	})
}

// starEntry is the first fix of a STAR flown to runway: on its common route,
// else on the runway's transition.
func starEntry(star airport.Procedure, runway string) string {
	legs := slices.Clone(star.Legs)
	for _, t := range star.RunwayTransitions {
		if strings.TrimLeft(t.Runway, "0") == strings.TrimLeft(runway, "0") || t.Runway == "ALL" {
			legs = append(legs, t.Legs...)
			break
		}
	}
	for _, l := range legs {
		if l.HasFix() {
			return l.Fix
		}
	}
	return ""
}

func registerListOurTraffic(mcp *mcpadapter.Server, tr live.Traffic) {
	tool := mcpadapter.NewTool("list_our_traffic").
		Description("List the AI aircraft under our control (spawn_departure, spawn_arrival): state (e.g. awaiting pushback, " +
			"taxiing, holding short, lined up, departing, approaching, rollout, parked), position, speed, current taxiway, " +
			"what it is holding short of, errors, and actions — the clearances atc_clearance takes now.").
		Build()

	mcp.AddTool(tool, func(context.Context, map[string]any) (*mcpadapter.CallToolResult, error) {
		fl := tr.Flights()
		return mcpadapter.JSONResult(map[string]any{"count": len(fl), "flights": fl})
	})
}

func registerATCClearance(mcp *mcpadapter.Server, tr live.Traffic) {
	tool := mcpadapter.NewTool("atc_clearance").
		Description("Give one of our AI aircraft a clearance or instruction. Departures: pushback, taxi (to the holding point), " +
			"cross (a runway on the way), lineup (line up and wait), takeoff, hold (hold position), abort (reject the " +
			"take-off before V1). Arrivals: goaround (on final), taxi (to the stand), cross, hold. Both: remove (take it " +
			"out of the simulator). A clearance given early means no stop there (takeoff while taxiing: rolling take-off). " +
			"The flight's actions list what fits now.").
		StringParam("callsign", "Call sign of one of ours (required).").
		StringParam("action", "pushback, taxi, cross, lineup, takeoff, hold, abort, goaround or remove (required).").
		Required("callsign", "action").
		Build()

	mcp.AddTool(tool, func(_ context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		callsign := strings.ToUpper(strArg(args, "callsign"))
		action := strings.ToLower(strArg(args, "action"))
		v, err := tr.Clear(callsign, action)
		if err != nil {
			return trafficError(action+" "+callsign, err), nil
		}
		return mcpadapter.JSONResult(v)
	})
}

func registerGetTrafficPicture(mcp *mcpadapter.Server, tr live.Traffic) {
	tool := mcpadapter.NewTool("get_traffic_picture").
		Description("The traffic picture: every aircraft the simulator has around the user aircraft (or an airport) with call " +
			"sign, aircraft title, position, altitude, ground speed, heading, vertical speed, and its phase — parked, " +
			"taxiing, runway, departing, enroute or arriving — and the airport it belongs to; ours are marked. The first " +
			"call starts a scan and takes a few seconds.").
		StringParam("centre", "Airport ICAO code to centre on (default: the user aircraft).").
		NumberParam("radius_nm", "Radius, NM (default 40, at most 40: the scan reaches about 43 NM from the user aircraft).").
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
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		list, err := tr.Picture(ctx, centre, radius)
		if err != nil {
			return trafficError("traffic picture", err), nil
		}
		type aircraft struct {
			Callsign  string  `json:"callsign,omitempty"`
			Title     string  `json:"title,omitempty"`
			Phase     string  `json:"phase"`
			Airport   string  `json:"airport,omitempty"`
			Lat       float64 `json:"lat"`
			Lon       float64 `json:"lon"`
			AltFt     float64 `json:"alt_ft"`
			AGLFt     float64 `json:"agl_ft"`
			GroundKts float64 `json:"ground_speed_kts"`
			Heading   float64 `json:"heading_true"`
			VSFpm     float64 `json:"vs_fpm"`
			User      bool    `json:"user,omitempty"`
			Ours      bool    `json:"ours,omitempty"`
		}
		out := make([]aircraft, 0, len(list))
		for _, a := range list {
			out = append(out, aircraft{a.Tail, a.Title, string(a.Phase), a.Airport, round(a.Position.Lat, 5), round(a.Position.Lon, 5),
				round(a.AltFt, -1), round(a.AGLFt, 0), round(a.GroundKts, 0), round(a.Heading, 0), round(a.VSFpm, -1), a.User, a.Ours})
		}
		return mcpadapter.JSONResult(map[string]any{"centre": centre, "radius_nm": radius, "count": len(out), "aircraft": out})
	})
}

func registerGenerateSchedule(mcp *mcpadapter.Server) {
	tool := mcpadapter.NewTool("generate_schedule").
		Description("Generate a realistic airline schedule for airports: flights with call sign, airline, aircraft type, origin, " +
			"destination, STD/STA (UTC) and distance, following time-of-day waves and each airline's bases and fleet (17 " +
			"European airlines, about 80 airports). Deterministic for a seed. Pure computation — nothing is spawned; use it to " +
			"pick flights for spawn_departure and spawn_arrival.").
		StringParam("airports", "Airports to schedule, e.g. \"LKPR\" or \"LKPR, EDDM\" (required).").
		NumberParam("hours", "Hours from start, 1–24 (default 2).").
		StringParam("start", "Start time, RFC 3339 (default: now, UTC).").
		NumberParam("density", "Traffic density, 0.1–3 (default 1).").
		NumberParam("seed", "Random seed (default 1).").
		NumberParam("limit", "Maximum flights, 1–500 (default 100).").
		Required("airports").
		Build()

	mcp.AddTool(tool, func(_ context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		var focus []string
		for _, a := range listArg(args, "airports") {
			a = strings.ToUpper(a)
			if !airportICAORe.MatchString(a) {
				return mcpadapter.ErrorResult(fmt.Sprintf("INVALID_ARGUMENT: %q is not an airport ICAO code", a)), nil
			}
			focus = append(focus, a)
		}
		if len(focus) == 0 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: airports is required"), nil
		}
		hours, density := numArg(args, "hours", 2), numArg(args, "density", 1)
		limit := int(numArg(args, "limit", 100))
		if hours < 1 || hours > 24 || density < 0.1 || density > 3 || limit < 1 || limit > 500 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: hours 1–24, density 0.1–3, limit 1–500"), nil
		}
		from := time.Now().UTC().Truncate(time.Minute)
		if s := strArg(args, "start"); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: start must be RFC 3339, e.g. 2026-10-01T06:00:00Z"), nil
			}
			from = t.UTC()
		}
		cfg := traffic.DefaultScheduleConfig()
		known := map[string]bool{}
		for _, a := range cfg.Airports {
			known[a.ICAO] = true
		}
		var unknown []string
		for _, a := range focus {
			if !known[a] {
				unknown = append(unknown, a)
			}
		}
		flights := traffic.Schedule(cfg, traffic.ScheduleOptions{Focus: focus, Density: density, Seed: uint64(numArg(args, "seed", 1))},
			from, from.Add(time.Duration(hours*float64(time.Hour))))
		total := len(flights)
		if len(flights) > limit {
			flights = flights[:limit]
		}
		out := map[string]any{"from": from, "hours": hours, "total": total, "count": len(flights), "flights": flights}
		if len(unknown) > 0 {
			out["warning"] = "not in the schedule's airport list (few or no flights): " + strings.Join(unknown, ", ")
		}
		return mcpadapter.JSONResult(out)
	})
}
