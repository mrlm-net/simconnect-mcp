//go:build windows

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// The add-on camera (MSFS 2024) through the traffic engine: its director
// films our traffic, follows an aircraft, holds a view, plays a scripted
// scene or one shot, or hands the camera back to the simulator.

var (
	cameraActions  = []string{"off", "auto", "follow", "view", "scene", "sim", "look"}
	cameraViewList = []string{"chase", "cockpit", "wing", "front", "top", "tower"}
	simCameraList  = []string{"cockpit", "chase", "drone", "fixed", "environment"}
)

func registerWorldCamera(mcp *mcpadapter.Server, w live.World) {
	get := mcpadapter.NewTool("get_camera").
		Description("The add-on camera now (mode, subject, shot, acquired) and the scripted scenes it can play.").
		Build()
	mcp.AddTool(get, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		var state, scenes json.RawMessage
		if err := w.Get("/api/camera", &state); err != nil {
			return worldError("camera", err), nil
		}
		if err := w.Get("/api/camera/scenes", &scenes); err != nil {
			return worldError("camera scenes", err), nil
		}
		return mcpadapter.JSONResult(map[string]any{"camera": state, "scenes": scenes})
	})

	set := mcpadapter.NewTool("set_camera").
		Description("Drive the MSFS 2024 add-on camera: auto (films our traffic), follow or view an aircraft, play a scene, "+
			"the sim's own cameras, or off (back to the simulator).").
		StringParam("action", strings.Join(cameraActions, ", ")+" (required)").
		StringParam("callsign", "follow, view: one of ours; view: \"me\" for the user aircraft").
		StringParam("view", "view: "+strings.Join(cameraViewList, ", ")).
		StringParam("icao", "scene, view tower: the airport").
		StringParam("scene", "scene: a key from get_camera").
		StringParam("sim", "sim: "+strings.Join(simCameraList, ", ")+"; none: step the view").
		NumberParam("step", "sim: +1 or -1, next or previous view").
		NumberParam("yaw", "look: tower bearing, degrees").
		NumberParam("tilt", "look: degrees").
		NumberParam("fov", "look: field of view, degrees").
		BoolParam("swing", "look: the tower looks round by itself").
		Required("action").
		Build()
	mcp.AddTool(set, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		action := strings.ToLower(strArg(args, "action"))
		if !contains(cameraActions, action) {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: action must be one of " + strings.Join(cameraActions, ", ")), nil
		}
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		var b []byte
		var err error
		switch action {
		case "off", "auto":
			b, err = w.Do("POST", "/api/camera", map[string]any{"mode": action})
		case "follow":
			a, ferr := findOurs(w, strArg(args, "callsign"))
			if ferr != nil {
				return worldError("follow", ferr), nil
			}
			b, err = w.Do("POST", "/api/camera", map[string]any{"mode": "follow", "id": a.ID})
		case "view":
			view := strings.ToLower(strArg(args, "view"))
			if !contains(cameraViewList, view) {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: view must be one of " + strings.Join(cameraViewList, ", ")), nil
			}
			body := map[string]any{"mode": "view", "view": view, "icao": strings.ToUpper(strArg(args, "icao"))}
			if view == "tower" {
				if body["icao"] == "" {
					return mcpadapter.ErrorResult("INVALID_ARGUMENT: view tower needs icao"), nil
				}
			} else if cs := strArg(args, "callsign"); strings.EqualFold(cs, "me") || cs == "" {
				body["id"] = -1
			} else {
				a, ferr := findOurs(w, cs)
				if ferr != nil {
					return worldError("view", ferr), nil
				}
				body["id"] = a.ID
			}
			b, err = w.Do("POST", "/api/camera", body)
		case "scene":
			icao, bad := icaoArg(args, "icao")
			if bad != nil {
				return bad, nil
			}
			name := strArg(args, "scene")
			if name == "" {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: scene needs a scene key from get_camera"), nil
			}
			if lerr := loadAirport(w, icao); lerr != nil {
				return worldError("loading "+icao, lerr), nil
			}
			b, err = w.Do("POST", "/api/camera/scene?icao="+url.QueryEscape(icao)+"&name="+url.QueryEscape(name), nil)
		case "sim":
			sim := strings.ToLower(strArg(args, "sim"))
			step := int(numArg(args, "step", 0))
			if sim != "" && !contains(simCameraList, sim) {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: sim must be one of " + strings.Join(simCameraList, ", ")), nil
			}
			if sim == "" && step == 0 {
				return mcpadapter.ErrorResult("INVALID_ARGUMENT: sim needs a camera or a step"), nil
			}
			b, err = w.Do("POST", "/api/camera", map[string]any{"mode": "sim", "sim": sim, "step": step})
		case "look":
			body := map[string]any{"mode": "look", "yaw": numArg(args, "yaw", 0), "tilt": numArg(args, "tilt", 0),
				"fov": numArg(args, "fov", 0)}
			if s, ok := args["swing"].(bool); ok {
				body["swing"] = s
			}
			b, err = w.Do("POST", "/api/camera", body)
		}
		if err != nil {
			return worldError(fmt.Sprintf("camera %s", action), err), nil
		}
		return rawResult(b), nil
	})

	registerWorldCameraShots(mcp, w)
}

// Composition first: a shot is one move, or a slow glide between two
// frames picked from stills (<move>_dist<back>_t<moment>); a capture reads
// the simulator's own camera to compose new ones.
func registerWorldCameraShots(mcp *mcpadapter.Server, w live.World) {
	shot := mcpadapter.NewTool("play_camera_shot").
		Description("Play one camera shot on one of our aircraft, then give the camera back: a move, a glide by number "+
			"(get_camera_glides), or a glide from one frame to another (to \"dolly\" slides along the aircraft). Returns what plays.").
		StringParam("icao", "The airport (required)").
		StringParam("callsign", "One of ours (required)").
		StringParam("move", "A move from get_camera_moves").
		NumberParam("glide", "A glide's number from get_camera_glides, from 1").
		StringParam("from", "A frame, e.g. sideDolly_dist1_t50; with to").
		StringParam("to", "A frame, or dolly").
		NumberParam("side", "move: +1 right, -1 left").
		NumberParam("sec", "Length, seconds (a glide: the longest)").
		NumberParam("back", "Distance scale, 1 as written, 2 twice as far").
		NumberParam("fov", "Field of view scale, 1 as written").
		NumberParam("at", "Hold the shot still at this point of it, 0 its start to 1 its end").
		Required("icao", "callsign").
		Build()
	mcp.AddTool(shot, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, bad := icaoArg(args, "icao")
		if bad != nil {
			return bad, nil
		}
		move, from, to, glide := strArg(args, "move"), strArg(args, "from"), strArg(args, "to"), int(numArg(args, "glide", 0))
		kinds := 0
		for _, set := range []bool{move != "", glide > 0, from != "" || to != ""} {
			if set {
				kinds++
			}
		}
		if kinds != 1 {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: give one of move, glide, or from and to"), nil
		}
		if (from == "") != (to == "") {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: a glide needs both from and to"), nil
		}
		if !w.Running() {
			return mcpadapter.ErrorResult("NOT_FOUND: no traffic of ours: the engine has not started"), nil
		}
		a, err := findOurs(w, strArg(args, "callsign"))
		if err != nil {
			return worldError("shot", err), nil
		}
		if lerr := loadAirport(w, icao); lerr != nil {
			return worldError("loading "+icao, lerr), nil
		}
		body := map[string]any{"icao": icao, "tail": a.Tail, "move": move, "from": from, "to": to, "glide": glide,
			"side": numArg(args, "side", 0), "sec": numArg(args, "sec", 0), "back": numArg(args, "back", 0), "fov": numArg(args, "fov", 0)}
		if at, ok := args["at"].(float64); ok {
			body["at"] = at
		}
		b, err := w.Do("POST", "/api/camera/shot", body)
		if err != nil {
			return worldError("shot on "+a.Tail, err), nil
		}
		return rawResult(b), nil
	})

	glides := mcpadapter.NewTool("get_camera_glides").
		Description("The glides between picked frames, each [from, to], numbered from 1 for play_camera_shot.").
		Build()
	mcp.AddTool(glides, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		var list [][2]string
		if err := w.Get("/api/camera/glides", &list); err != nil {
			return worldError("camera glides", err), nil
		}
		out := make([]map[string]any, len(list))
		for i, g := range list {
			out[i] = map[string]any{"glide": i + 1, "from": g[0], "to": g[1]}
		}
		return mcpadapter.JSONResult(map[string]any{"glides": out})
	})

	moves := mcpadapter.NewTool("get_camera_moves").
		Description("The moves a camera shot takes (play_camera_shot move, a scene shot's move).").
		Build()
	mcp.AddTool(moves, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		var list []string
		if err := w.Get("/api/camera/moves", &list); err != nil {
			return worldError("camera moves", err), nil
		}
		return mcpadapter.JSONResult(map[string]any{"moves": list})
	})

	capture := mcpadapter.NewTool("capture_camera").
		Description("Read the simulator's own camera without taking it: its position, heading, pitch and field of view, "+
			"its offset from the aircraft filmed, and where it is on the airport (nearest stand and taxiway, along and beside the runway).").
		StringParam("icao", "The airport (required)").
		StringParam("name", "A name for the composition").
		StringParam("callsign", "The aircraft filmed (default: ours it aims nearest at, within 2 km)").
		Required("icao").
		Build()
	mcp.AddTool(capture, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		icao, bad := icaoArg(args, "icao")
		if bad != nil {
			return bad, nil
		}
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		if lerr := loadAirport(w, icao); lerr != nil {
			return worldError("loading "+icao, lerr), nil
		}
		b, err := w.Do("POST", "/api/camera/capture", map[string]any{"icao": icao, "name": strArg(args, "name"),
			"tail": strings.ToUpper(strArg(args, "callsign"))})
		if err != nil {
			return worldError("camera capture", err), nil
		}
		return rawResult(b), nil
	})
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
