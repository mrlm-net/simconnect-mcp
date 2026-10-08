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
// scene, or hands the camera back to the simulator.

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
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
