//go:build windows

package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// Jetways for our traffic (library pkg/jetway through the traffic engine)
// and which simulator models our traffic uses (model rules and providers).

func registerWorldJetways(mcp *mcpadapter.Server, w live.World) {
	get := mcpadapter.NewTool("get_jetways").
		Description("Jetways for our traffic: on or off, available (needs the helper package), each gate's state, and the door offsets by type.").
		Build()
	mcp.AddTool(get, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		var state, offsets json.RawMessage
		if err := w.Get("/api/jetways", &state); err != nil {
			return worldError("jetways", err), nil
		}
		if err := w.Get("/api/jetways/offsets", &offsets); err != nil {
			return worldError("jetway offsets", err), nil
		}
		return mcpadapter.JSONResult(map[string]any{"jetways": state, "doorOffsets": offsets})
	})

	set := mcpadapter.NewTool("set_jetways").
		Description("Switch jetways for our traffic on or off (off by default); off sends every jetway home.").
		BoolParam("on", "true: dock at gates (required)").
		Required("on").
		Build()
	mcp.AddTool(set, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		on, ok := args["on"].(bool)
		if !ok {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: on (true or false) is required"), nil
		}
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		b, err := w.Do("POST", "/api/jetways", map[string]any{"on": on})
		if err != nil {
			return worldError("jetways", err), nil
		}
		return mcpadapter.TextResult(string(b)), nil
	})

	off := mcpadapter.NewTool("set_jetway_door_offset").
		Description("Where a type's invisible jetway helper goes from our aircraft so the jetway meets the door; overrides the offset from the models' doors.").
		StringParam("type", "ICAO type, e.g. B738 (required)").
		NumberParam("ahead_m", "metres ahead of our aircraft (negative: behind)").
		NumberParam("right_m", "metres to its right (negative: left)").
		Required("type").
		Build()
	mcp.AddTool(off, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		t := strings.TrimSpace(strArg(args, "type"))
		if t == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: type is required"), nil
		}
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		ahead, _ := args["ahead_m"].(float64)
		right, _ := args["right_m"].(float64)
		b, err := w.Do("POST", "/api/jetways/offset", map[string]any{"type": t, "aheadM": ahead, "rightM": right})
		if err != nil {
			return worldError("jetway offset", err), nil
		}
		return mcpadapter.TextResult(string(b)), nil
	})
}

func registerWorldModelChoice(mcp *mcpadapter.Server, w live.World) {
	get := mcpadapter.NewTool("get_model_choice").
		Description("Which simulator models our traffic uses: the model rules (never use) and the providers on or off.").
		Build()
	mcp.AddTool(get, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		var rules, providers json.RawMessage
		if err := w.Get("/api/models/rules", &rules); err != nil {
			return worldError("model rules", err), nil
		}
		if err := w.Get("/api/models/providers", &providers); err != nil {
			return worldError("model providers", err), nil
		}
		return mcpadapter.JSONResult(map[string]any{"rules": rules, "providers": providers})
	})

	rule := mcpadapter.NewTool("set_model_rule").
		Description("Never use (or use again) a simulator model for our traffic; a title ending in * covers every title it starts.").
		StringParam("title", "the model's title, or a prefix with * (required)").
		BoolParam("never", "true: never use it (default true)").
		StringParam("reason", "why, kept with the rule").
		Required("title").
		Build()
	mcp.AddTool(rule, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		title := strings.TrimSpace(strArg(args, "title"))
		if title == "" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: title is required"), nil
		}
		never := true
		if v, ok := args["never"].(bool); ok {
			never = v
		}
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		b, err := w.Do("POST", "/api/models/rules", map[string]any{"title": title, "never": never, "reason": strArg(args, "reason")})
		if err != nil {
			return worldError("model rule", err), nil
		}
		return mcpadapter.TextResult(string(b)), nil
	})

	prov := mcpadapter.NewTool("set_model_provider").
		Description("Switch a traffic model provider (a package's creator, e.g. FSLTL) on or off for our traffic.").
		StringParam("provider", "as get_model_choice lists it (required)").
		BoolParam("on", "true: use its models (required)").
		Required("provider", "on").
		Build()
	mcp.AddTool(prov, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		p := strings.TrimSpace(strArg(args, "provider"))
		on, ok := args["on"].(bool)
		if p == "" || !ok {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: provider and on are required"), nil
		}
		if bad := ensure(ctx, w); bad != nil {
			return bad, nil
		}
		b, err := w.Do("POST", "/api/models/providers", map[string]bool{p: on})
		if err != nil {
			return worldError("model provider", err), nil
		}
		return mcpadapter.TextResult(string(b)), nil
	})
}
