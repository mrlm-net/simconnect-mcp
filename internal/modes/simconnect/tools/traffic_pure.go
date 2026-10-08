//go:build windows

package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/traffic"
)

// The traffic tools that only compute: no simulator, no engine.

func registerGenerateSchedule(mcp *mcpadapter.Server) {
	tool := mcpadapter.NewTool("generate_schedule").
		Description("Generate a realistic airline schedule for airports (nothing is spawned).").
		StringParam("airports", "e.g. \"LKPR, EDDM\"").
		NumberParam("hours", "1–24 (default 2)").
		StringParam("start", "RFC 3339 (default now)").
		NumberParam("density", "0.1–3 (default 1)").
		NumberParam("seed", "Default 1").
		NumberParam("limit", "1–500 (default 100)").
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

func registerSeparationMinima(mcp *mcpadapter.Server) {
	tool := mcpadapter.NewTool("separation_minima").
		Description("Wake categories, final spacing, departure interval and runway occupancy for a leader/follower type "+
			"pair.").
		StringParam("leader", "ICAO type").
		StringParam("follower", "ICAO type").
		StringParam("scheme", "icao (default) or recat").
		NumberParam("visibility_m", "m (default good)").
		StringParam("surface", "dry (default), wet or contaminated").
		NumberParam("same_route", "1: same SID (default 0)").
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
			"leader":                       map[string]any{"type": lt, "wake": lw, "landing_occupancy_s": int(traffic.RunwayOccupancyIn(lw, true, surface).Seconds())},
			"follower":                     map[string]any{"type": ft, "wake": fw, "landing_occupancy_s": int(traffic.RunwayOccupancyIn(fw, true, surface).Seconds())},
			"wake_minimum_nm":              traffic.ArrivalSeparationNM(lw, fw, scheme),
			"spacing_on_final_nm":          spacing,
			"spacing_why":                  why,
			"departure_interval_s":         int(traffic.DepartureInterval(lw, fw, numArg(args, "same_route", 0) == 1).Seconds()),
			"follower_takeoff_occupancy_s": int(traffic.RunwayOccupancyIn(fw, false, surface).Seconds()),
		})
	})
}
