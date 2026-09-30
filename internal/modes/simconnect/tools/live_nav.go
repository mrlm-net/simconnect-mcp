//go:build windows

package tools

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/calc"
	"github.com/mrlm-net/simconnect/pkg/nav"
)

const (
	// maxCrawlRadiusNM bounds an on-demand airway crawl (about 1000 fixes
	// per 250 NM in Europe).
	maxCrawlRadiusNM = 400.0
	planTimeout      = 90 * time.Second
)

// RegisterLiveNavTools registers the weather and navigation tools built on
// pkg/nav: get_weather, get_active_runway, get_atis, get_fix,
// find_airway_route and plan_flight.
func RegisterLiveNavTools(mcp *mcpadapter.Server, src live.Source) {
	registerGetWeather(mcp, src)
	registerGetActiveRunway(mcp, src)
	registerGetATIS(mcp, src)
	registerGetFix(mcp, src)
	registerFindAirwayRoute(mcp, src)
	registerPlanFlight(mcp, src)
}

const weatherNote = "Measured at the user aircraft: the simulator gives the ambient weather there only. " +
	"Gusts, ceiling and dewpoint are not available from SimConnect."

// ── Weather ─────────────────────────────────────────────────────────────────

type weatherOut struct {
	WindDirTrue float64 `json:"wind_dir_true"`
	WindKts     float64 `json:"wind_kts"`
	Calm        bool    `json:"calm"`
	VisibilityM float64 `json:"visibility_m"`
	TempC       float64 `json:"temp_c"`
	QNHhPa      float64 `json:"qnh_hpa"`
	QNHinHg     float64 `json:"qnh_inhg"`
	Precip      string  `json:"precip,omitempty"`
	InCloud     bool    `json:"in_cloud"`
	Icing       bool    `json:"icing_conditions"`
	Note        string  `json:"note"`
}

func weatherJSON(w nav.Weather) weatherOut {
	return weatherOut{
		WindDirTrue: round(w.WindDirTrue, 0), WindKts: round(w.WindKts, 0), Calm: w.IsCalm(),
		VisibilityM: round(w.VisibilityM, 0), TempC: round(w.TempC, 1), QNHhPa: round(w.QNHhPa, 0),
		QNHinHg: round(w.QNHhPa*0.02953, 2), Precip: w.Precip, InCloud: w.InCloud, Icing: nav.IcingConditions(w),
		Note: weatherNote,
	}
}

func registerGetWeather(mcp *mcpadapter.Server, src live.Source) {
	tool := mcpadapter.NewTool("get_weather").
		Description("Return the weather at the user aircraft: wind (degrees true, knots), visibility, temperature, QNH (hPa and " +
			"inHg), precipitation, whether the aircraft is in cloud, and icing conditions (visible moisture at or below " +
			"+10 °C). SimConnect has no gust, ceiling or dewpoint variables.").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, _ map[string]any) (*mcpadapter.CallToolResult, error) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		w, err := src.Weather(ctx)
		if err != nil {
			return sourceError("weather", err), nil
		}
		return mcpadapter.JSONResult(weatherJSON(w))
	})
}

// ── Active runway and ATIS ──────────────────────────────────────────────────

type runwayInUse struct {
	ICAO                 string     `json:"icao"`
	Departure            string     `json:"departure_runway"`
	Arrival              string     `json:"arrival_runway"`
	HeadwindKts          float64    `json:"headwind_kts"`
	CrosswindKts         float64    `json:"crosswind_kts"`
	WithinLimits         bool       `json:"within_wind_limits"`
	ApproachKind         string     `json:"approach_kind"`
	Approach             string     `json:"approach,omitempty"`
	TransitionAltitudeFt float64    `json:"transition_altitude_ft"`
	TransitionLevel      int        `json:"transition_level"`
	PreferredRunways     []string   `json:"preferred_runways,omitempty"`
	Weather              weatherOut `json:"weather"`
}

// activeRunway works out the runways in use at icao for the weather at the
// user aircraft, with the airport's limits and preferential runways.
func activeRunway(ctx context.Context, src live.Source, icao string) (*airport.Layout, *airport.Procedures, nav.Weather, nav.RunwayUse, airport.Limits, error) {
	layout, err := src.Layout(ctx, icao)
	if err != nil {
		return nil, nil, nav.Weather{}, nav.RunwayUse{}, airport.Limits{}, err
	}
	procs, _ := src.Procedures(ctx, icao) // optional: limits and the approach
	w, err := src.Weather(ctx)
	if err != nil {
		return nil, nil, nav.Weather{}, nav.RunwayUse{}, airport.Limits{}, err
	}
	lim := airport.LimitsFor(layout, procs)
	return layout, procs, w, nav.ActiveRunways(layout, w, nav.RunwayLimitsFrom(lim)), lim, nil
}

func registerGetActiveRunway(mcp *mcpadapter.Server, src live.Source) {
	tool := mcpadapter.NewTool("get_active_runway").
		Description("Work out the runways in use at an airport as a tower would: the preferential runway if the wind allows, "+
			"else the one with the most headwind, within tailwind and crosswind limits (gusts included). Returns the "+
			"departure and arrival runway, wind components, whether an ILS or visual approach is expected and the best "+
			"published approach, the transition altitude and level. Uses the weather at the user aircraft, so it is "+
			"right for the airport the aircraft is at or near.").
		StringParam("icao", "Airport ICAO code (required).").
		Required("icao").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, bad := icaoArg(args, "icao")
		if bad != nil {
			return bad, nil
		}
		_, procs, w, use, lim, err := activeRunway(ctx, src, icao)
		if err != nil {
			return sourceError("runway in use at "+icao, err), nil
		}
		out := runwayInUse{
			ICAO: icao, Departure: use.Departure.Name, Arrival: use.Arrival.Name,
			HeadwindKts: round(use.HeadwindKts, 0), CrosswindKts: round(use.CrosswindKts, 0),
			WithinLimits: use.WithinLimits, ApproachKind: use.Approach,
			TransitionAltitudeFt: lim.TransitionAltitudeFt,
			TransitionLevel:      nav.TransitionLevel(int(lim.TransitionAltitudeFt), w.QNHhPa),
			PreferredRunways:     lim.PreferredRunways, Weather: weatherJSON(w),
		}
		if procs != nil {
			if a, ok := procs.BestApproach(use.Arrival.Name); ok {
				out.Approach = a.Name
			}
		}
		return mcpadapter.JSONResult(out)
	})
}

func registerGetATIS(mcp *mcpadapter.Server, src live.Source) {
	tool := mcpadapter.NewTool("get_atis").
		Description("Compose the ATIS broadcast of an airport from the simulator's weather and the runway in use: information "+
			"letter, time, runways, approach, wind, visibility, temperature, QNH, transition level. Returns the text as "+
			"written and as spoken (phonetic, for text-to-speech). Uses the weather at the user aircraft.").
		StringParam("icao", "Airport ICAO code (required).").
		StringParam("letter", "Information letter A–Z (default A).").
		Required("icao").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, bad := icaoArg(args, "icao")
		if bad != nil {
			return bad, nil
		}
		letter := byte('A')
		if s := strings.ToUpper(strArg(args, "letter")); s != "" {
			if len(s) != 1 || s[0] < 'A' || s[0] > 'Z' {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: letter must be A–Z"), nil
			}
			letter = s[0]
		}
		layout, procs, w, use, lim, err := activeRunway(ctx, src, icao)
		if err != nil {
			return sourceError("ATIS of "+icao, err), nil
		}
		magVar := 0.0
		if procs != nil {
			magVar = procs.MagVar
		}
		name := layout.Name
		if name == "" {
			name = icao
		}
		a := nav.NewATIS(name, letter, time.Now().UTC(), w, use, int(lim.TransitionAltitudeFt), magVar)
		return mcpadapter.JSONResult(map[string]any{
			"icao": icao, "letter": string(letter), "text": a.Text(), "spoken": a.Spoken(), "note": weatherNote,
		})
	})
}

// ── Fixes and airways ───────────────────────────────────────────────────────

type fixOut struct {
	Key      string   `json:"key"`
	Ident    string   `json:"ident"`
	Region   string   `json:"region"`
	Kind     string   `json:"kind"`
	Name     string   `json:"name,omitempty"`
	Freq     float64  `json:"freq,omitempty"`
	Lat      float64  `json:"lat"`
	Lon      float64  `json:"lon"`
	Terminal bool     `json:"terminal,omitempty"`
	Airways  []string `json:"airways,omitempty"`
}

var kindNames = map[nav.FixKind]string{nav.KindWaypoint: "waypoint", nav.KindVOR: "VOR", nav.KindNDB: "NDB"}

func fixJSON(r nav.NavResult) fixOut {
	f := r.Fix
	out := fixOut{Key: nav.Key(f.Ident, f.Region, f.Kind).String(), Ident: f.Ident, Region: f.Region, Kind: kindNames[f.Kind], Name: f.Name,
		Freq: f.Freq, Lat: round(f.Position.Lat, 6), Lon: round(f.Position.Lon, 6), Terminal: f.Terminal}
	for _, l := range r.Routes {
		s := l.Airway
		if l.Prev != nil {
			s = l.Prev.Key.Ident + " " + s
		}
		if l.Next != nil {
			s += " " + l.Next.Key.Ident
		}
		out.Airways = append(out.Airways, s)
	}
	return out
}

func registerGetFix(mcp *mcpadapter.Server, src live.Source) {
	tool := mcpadapter.NewTool("get_fix").
		Description("Look up an enroute fix in the simulator's navdata: a waypoint, VOR or NDB with its position, frequency "+
			"(MHz for a VOR, kHz for an NDB), name, and the airways through it (previous fix, airway, next fix). "+
			"Identifiers repeat between waypoints, VORs and NDBs; without kind all three are tried.").
		StringParam("ident", "Fix identifier, e.g. \"VOZ\", \"GOLOP\" (required).").
		StringParam("region", "ICAO region, e.g. \"LK\" (recommended: identifiers repeat worldwide).").
		StringParam("kind", "W (waypoint), V (VOR) or N (NDB). Default: all.").
		Required("ident").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		ident, region := strings.ToUpper(strArg(args, "ident")), strings.ToUpper(strArg(args, "region"))
		if !icaoRe.MatchString(ident) || !regionRe.MatchString(region) {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: ident must be 1–9 and region 0–4 uppercase letters or digits"), nil
		}
		kinds := []nav.FixKind{nav.KindWaypoint, nav.KindVOR, nav.KindNDB}
		if k := strings.ToUpper(strArg(args, "kind")); k != "" {
			key, err := fixKeyArg(ident + "." + region + "." + k)
			if err != nil {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: " + err.Error()), nil
			}
			kinds = []nav.FixKind{key.Kind}
		}
		type res struct {
			r   nav.NavResult
			err error
		}
		results := make([]res, len(kinds))
		done := make(chan struct{})
		for i, k := range kinds {
			go func() {
				r, err := src.Fix(ctx, nav.Key(ident, region, k))
				results[i] = res{r, err}
				done <- struct{}{}
			}()
		}
		for range kinds {
			<-done
		}
		// A VOR or NDB on the airway network also answers as a waypoint, and a
		// navaid request finds that waypoint record even when the navaid is of
		// the other kind (VOZ answers as an NDB without frequency). Keep real
		// navaids, and waypoints not standing for one of them.
		var navaids, waypoints []nav.NavResult
		for _, r := range results {
			switch {
			case r.err != nil && !errors.Is(r.err, live.ErrNotFound):
				return sourceError("fix "+ident, r.err), nil
			case r.err != nil:
			case r.r.Fix.Kind == nav.KindWaypoint:
				waypoints = append(waypoints, r.r)
			case r.r.Fix.Freq > 0:
				navaids = append(navaids, r.r)
			}
		}
		var found []fixOut
		for _, r := range navaids {
			found = append(found, fixJSON(r))
		}
		for _, w := range waypoints {
			dup := false
			for _, n := range navaids {
				dup = dup || n.Fix.Position == w.Fix.Position
			}
			if !dup {
				found = append(found, fixJSON(w))
			}
		}
		if len(found) == 0 {
			return mcpadapter.ErrorResult(fmt.Sprintf("NOT_FOUND: no fix %s %s", ident, region)), nil
		}
		return mcpadapter.JSONResult(map[string]any{"count": len(found), "fixes": found})
	})
}

// crawlArea is the circle an airway crawl between two points covers.
func crawlArea(a, b airport.LatLon, marginNM float64) (airport.LatLon, float64) {
	d := calc.HaversineNM(a.Lat, a.Lon, b.Lat, b.Lon)
	lat, lon := midpoint(a, b)
	return airport.LatLon{Lat: lat, Lon: lon}, math.Min(d/2+marginNM, maxCrawlRadiusNM)
}

func midpoint(a, b airport.LatLon) (float64, float64) {
	d := calc.HaversineNM(a.Lat, a.Lon, b.Lat, b.Lon)
	lat, lon := calc.DisplaceByHeading(a.Lat, a.Lon, calc.BearingDegrees(a.Lat, a.Lon, b.Lat, b.Lon), d/2*1852)
	return lat, lon
}

func registerFindAirwayRoute(mcp *mcpadapter.Server, src live.Source) {
	tool := mcpadapter.NewTool("find_airway_route").
		Description("Find the airway route between two enroute fixes over the simulator's airway network (A*, with a penalty "+
			"for every change of airway). The network is crawled from the simulator on demand around the two fixes (a few "+
			"seconds; cached). Returns the ICAO route string (\"VOZ M725 OKF\"), each step with its airway and leg distance, "+
			"the route and direct distances. When the fixes are not connected or the airways are over max_stretch times the "+
			"direct distance, the route is direct (DCT).").
		StringParam("from", "Start fix as IDENT, IDENT.REGION or IDENT.REGION.KIND, e.g. \"VOZ.LK.V\" (required).").
		StringParam("to", "End fix, same format (required).").
		NumberParam("max_stretch", "Fly direct when the airways are longer than this times the direct distance (default 1.5; 0 = always airways).").
		Required("from", "to").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		from, err := fixKeyArg(strArg(args, "from"))
		if err != nil {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: " + err.Error()), nil
		}
		to, err := fixKeyArg(strArg(args, "to"))
		if err != nil {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: " + err.Error()), nil
		}
		ctx, cancel := context.WithTimeout(ctx, planTimeout)
		defer cancel()
		a, err := src.Fix(ctx, from)
		if err != nil {
			return sourceError("fix "+from.String(), err), nil
		}
		b, err := src.Fix(ctx, to)
		if err != nil {
			return sourceError("fix "+to.String(), err), nil
		}
		center, radius := crawlArea(a.Fix.Position, b.Fix.Position, 60)
		g, err := src.Airways(ctx, center, radius, []nav.FixKey{a.Key, b.Key})
		if err != nil {
			return sourceError("airways", err), nil
		}
		// The loaded fixes carry the simulator's region; use them as the keys.
		fk, tk := nav.Key(a.Fix.Ident, a.Fix.Region, a.Fix.Kind), nav.Key(b.Fix.Ident, b.Fix.Region, b.Fix.Kind)
		stretch := numArg(args, "max_stretch", 1.5)
		var steps []nav.RouteStep
		if stretch <= 0 {
			steps, err = g.Route(fk, tk)
		} else {
			steps, err = g.RouteOrDirect(fk, tk, stretch)
		}
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("NO_ROUTE: %v", err)), nil
		}
		type step struct {
			Airway     string  `json:"airway,omitempty"`
			Fix        string  `json:"fix"`
			Lat        float64 `json:"lat"`
			Lon        float64 `json:"lon"`
			DistanceNM float64 `json:"distance_nm"`
		}
		out := make([]step, 0, len(steps))
		for _, s := range steps {
			out = append(out, step{s.Airway, s.Fix.String(), round(s.Position.Lat, 6), round(s.Position.Lon, 6), round(s.DistanceNM, 1)})
		}
		return mcpadapter.JSONResult(map[string]any{
			"route":       nav.FormatRoute(steps),
			"distance_nm": round(nav.RouteDistanceNM(steps), 1),
			"direct_nm":   round(calc.HaversineNM(a.Fix.Position.Lat, a.Fix.Position.Lon, b.Fix.Position.Lat, b.Fix.Position.Lon), 1),
			"steps":       out,
			"graph":       map[string]any{"fixes_crawled_within_nm": round(radius, 0), "segments": g.SegmentCount()},
		})
	})
}

// ── Flight plans ────────────────────────────────────────────────────────────

// procedureSeeds are the enroute fixes where an airport's SIDs end and its
// STARs begin: the airway crawl starts there.
func procedureSeeds(p *airport.Procedures) []nav.FixKey {
	if p == nil {
		return nil
	}
	seen := map[nav.FixKey]bool{}
	var out []nav.FixKey
	add := func(l airport.Leg) {
		// Without a kind (data captured before FIX_TYPE was read) try it as a waypoint.
		kind := nav.KindWaypoint
		if len(l.FixKind) == 1 {
			kind = nav.FixKind(l.FixKind[0])
		}
		if l.Fix == "" || kind != nav.KindWaypoint && kind != nav.KindVOR && kind != nav.KindNDB {
			return
		}
		k := nav.Key(l.Fix, l.Region, kind)
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	last := func(ls []airport.Leg) {
		if len(ls) > 0 {
			add(ls[len(ls)-1])
		}
	}
	first := func(ls []airport.Leg) {
		if len(ls) > 0 {
			add(ls[0])
		}
	}
	// A SID is flown runway transition → legs → enroute transition, a STAR
	// the other way round; any part may be empty (LKPR's SIDs are all
	// runway transitions), so every part's end counts.
	for _, s := range p.Departures {
		last(s.Legs)
		for _, t := range append(s.RunwayTransitions, s.EnrouteTransitions...) {
			last(t.Legs)
		}
	}
	for _, s := range p.Arrivals {
		first(s.Legs)
		for _, t := range append(s.RunwayTransitions, s.EnrouteTransitions...) {
			first(t.Legs)
		}
	}
	return out
}

type planWaypoint struct {
	Ident      string  `json:"ident,omitempty"`
	Airway     string  `json:"via,omitempty"`
	Phase      string  `json:"phase"`
	AltFt      float64 `json:"alt_ft"`
	DistanceNM float64 `json:"dist_nm"`
	Constraint string  `json:"constraint,omitempty"`
	Lat        float64 `json:"lat"`
	Lon        float64 `json:"lon"`
}

func registerPlanFlight(mcp *mcpadapter.Server, src live.Source) {
	tool := mcpadapter.NewTool("plan_flight").
		Description("Plan an IFR flight between two airports from the simulator's navdata: runways in use (weather at the user "+
			"aircraft for the departure), SID, airways (crawled on demand; direct where they detour), STAR and best "+
			"approach, semicircular cruise level, vertical profile with TOC/TOD, distance, time and fuel for the aircraft "+
			"type. Returns the ICAO route and the waypoints; optionally the MSFS .pln file, and load_into_sim=true loads it "+
			"as the user aircraft's flight plan (changes the simulator's flight plan). Takes up to a minute the first time.").
		StringParam("departure", "Departure airport ICAO (required).").
		StringParam("arrival", "Arrival airport ICAO (required).").
		StringParam("aircraft_type", "ICAO type designator, e.g. \"A20N\", \"B738\", \"B77W\" (default A320: planning speeds, fuel flow).").
		NumberParam("cruise_fl", "Cruise flight level, e.g. 340 (default: chosen by direction and distance).").
		StringParam("departure_runway", "Departure runway (default: in use for the weather).").
		StringParam("arrival_runway", "Arrival runway (default: in use for calm wind, the longest).").
		BoolParam("airways", "Route over airways (default true); false plans direct between SID and STAR.").
		NumberParam("alternate_fuel_kg", "Alternate fuel to add, kg (default 0).").
		BoolParam("include_pln", "Include the .pln file text in the result (default false).").
		BoolParam("load_into_sim", "Load the plan into the simulator as the user aircraft's flight plan (default false).").
		Required("departure", "arrival").
		Build()

	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		dep, bad := icaoArg(args, "departure")
		if bad != nil {
			return bad, nil
		}
		arr, bad := icaoArg(args, "arrival")
		if bad != nil {
			return bad, nil
		}
		ctx, cancel := context.WithTimeout(ctx, planTimeout)
		defer cancel()

		var warnings []string
		info := func(icao string) (nav.AirportInfo, *airport.Procedures, error) {
			l, err := src.Layout(ctx, icao)
			if err != nil {
				return nav.AirportInfo{}, nil, err
			}
			p, err := src.Procedures(ctx, icao)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("%s: no procedures (%v); joins the airways direct", icao, err))
				p = nil
			}
			return nav.AirportInfo{ICAO: icao, Name: l.Name, Layout: l, Procedures: p}, p, nil
		}
		di, dp, err := info(dep)
		if err != nil {
			return sourceError("departure "+dep, err), nil
		}
		ai, ap, err := info(arr)
		if err != nil {
			return sourceError("arrival "+arr, err), nil
		}

		req := nav.FlightPlanRequest{
			Departure: di, Arrival: ai,
			Type:            strings.ToUpper(strArg(args, "aircraft_type")),
			CruiseFL:        int(numArg(args, "cruise_fl", 0)),
			DepartureRunway: strings.ToUpper(strArg(args, "departure_runway")),
			ArrivalRunway:   strings.ToUpper(strArg(args, "arrival_runway")),
			DepLimits:       nav.RunwayLimitsFrom(airport.LimitsFor(di.Layout, dp)),
			ArrLimits:       nav.RunwayLimitsFrom(airport.LimitsFor(ai.Layout, ap)),
			AlternateFuelKg: numArg(args, "alternate_fuel_kg", 0),
		}
		if req.DepartureRunway == "" {
			if w, err := src.Weather(ctx); err == nil {
				req.DepWeather = &w
			} else {
				warnings = append(warnings, "no weather; the departure runway is chosen for calm wind")
			}
		}

		var graph *nav.AirwayGraph
		if b, ok := args["airways"].(bool); !ok || b {
			a, b := airport.LatLon{Lat: di.Layout.Latitude, Lon: di.Layout.Longitude}, airport.LatLon{Lat: ai.Layout.Latitude, Lon: ai.Layout.Longitude}
			center, radius := crawlArea(a, b, 100)
			seeds := append(procedureSeeds(dp), procedureSeeds(ap)...)
			if len(seeds) == 0 {
				warnings = append(warnings, "no SID or STAR fixes to start the airway crawl from; direct")
			} else if g, err := src.Airways(ctx, center, radius, seeds); err == nil {
				graph = g
			} else {
				warnings = append(warnings, fmt.Sprintf("airways unavailable (%v); direct", err))
			}
			if calc.HaversineNM(a.Lat, a.Lon, b.Lat, b.Lon)/2+100 > maxCrawlRadiusNM {
				warnings = append(warnings, fmt.Sprintf("airways crawled within %.0f NM of the midpoint only; direct beyond", maxCrawlRadiusNM))
			}
		}

		fp, err := nav.Plan(req, graph)
		if err != nil {
			return mcpadapter.ErrorResult(fmt.Sprintf("PLAN_ERROR: %v", err)), nil
		}

		wps := make([]planWaypoint, 0, len(fp.Waypoints))
		for _, w := range fp.Waypoints {
			wps = append(wps, planWaypoint{Ident: w.Ident, Airway: w.Airway, Phase: string(w.Phase), AltFt: round(w.AltFt, -1),
				DistanceNM: round(w.DistanceNM, 1), Constraint: w.Constraint(), Lat: round(w.Position.Lat, 6), Lon: round(w.Position.Lon, 6)})
		}
		out := map[string]any{
			"departure": dep, "arrival": arr, "aircraft_type": fp.Performance.Type,
			"departure_runway": fp.DepartureRunway, "arrival_runway": fp.ArrivalRunway,
			"sid": fp.SID, "sid_transition": fp.SIDTransition, "star": fp.STAR, "star_transition": fp.STARTransition,
			"approach": fp.Approach, "approach_transition": fp.ApproachTransition,
			"route": fp.Route, "cruise_fl": fp.CruiseFL, "magnetic_track": round(fp.MagneticTrack, 0),
			"distance_nm": round(fp.DistanceNM, 0), "toc_nm": round(fp.TOCNM, 0), "tod_nm": round(fp.TODNM, 0),
			"ete_minutes": round(fp.ETE.Minutes(), 0), "fuel_kg": fp.Fuel, "waypoints": wps,
		}
		if fp.Performance.Type == "" {
			warnings = append(warnings, "unknown aircraft type: planned as a generic medium jet")
		}

		include, _ := args["include_pln"].(bool)
		load, _ := args["load_into_sim"].(bool)
		if include || load {
			pln, err := fp.PLN()
			if err != nil {
				return mcpadapter.ErrorResult(fmt.Sprintf("PLAN_ERROR: .pln: %v", err)), nil
			}
			if include {
				out["pln"] = string(pln)
			}
			if load {
				if err := src.LoadFlightPlan(ctx, pln); err != nil {
					return sourceError("loading the flight plan", err), nil
				}
				out["loaded_into_sim"] = true
			}
		}
		if len(warnings) > 0 {
			out["warnings"] = warnings
		}
		return mcpadapter.JSONResult(out)
	})
}
