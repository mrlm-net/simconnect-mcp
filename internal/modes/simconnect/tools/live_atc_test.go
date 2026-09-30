//go:build windows

package tools

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mrlm-net/simconnect-mcp/internal/live"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/calc"
	"github.com/mrlm-net/simconnect/pkg/traffic"
)

func TestLiveATCTools(t *testing.T) {
	now := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	// Two of ours converging head-on at FL200, 40 NM apart.
	lat, lon := calc.DisplaceByHeading(50.1, 14.26, 90, 40*1852)
	ft := &live.FixtureTraffic{
		SequenceList: []live.RunwaySequence{{ICAO: "LKPR", Runway: "24", Sequence: []traffic.SequenceEntry{
			{Callsign: "CSA1", Number: 1, DistanceToGoNM: 12.34, ETA: now, Landing: now},
			{Callsign: "DLH2", Number: 2, Leader: "CSA1", SpacingNM: 5, DistanceToGoNM: 20, ETA: now, Landing: now.Add(2 * time.Minute), Delay: 90 * time.Second},
		}}},
		TowerList:    []live.RunwayUser{{Runway: "06/24", Callsign: "TVS3", Phase: "holding short", Ours: true, Waiting: "CSA1 on a 3.0 NM final"}},
		Log:          []live.ATCMessage{{At: now, ICAO: "LKPR", Callsign: "TVS3", Text: "TVS3, runway 24, line up and wait"}, {At: now, ICAO: "EDDM", Callsign: "X1", Text: "X1, go around"}},
		ApproachSaid: "DLH2, number 2, lose a minute: 210 kt",
		PictureList: []traffic.TrackedAircraft{
			{Observation: traffic.Observation{ObjectID: 1, Tail: "CSA7", Position: airport.LatLon{Lat: 50.1, Lon: 14.26}, AltFt: 20000, AGLFt: 19000, GroundKts: 450, Heading: 90}, Ours: true},
			{Observation: traffic.Observation{ObjectID: 2, Tail: "DLH8", Position: airport.LatLon{Lat: lat, Lon: lon}, AltFt: 20000, AGLFt: 19000, GroundKts: 450, Heading: 270}},
		},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mcp := mcpadapter.NewServer("test", "1.0.0")
	RegisterLiveATCTools(mcp, ft)
	mcp.MountStreamableHTTP(r, "/mcp")
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	call := func(tool string, args map[string]any) (map[string]any, bool) {
		t.Helper()
		resp := callToolEvent(t, srv.URL, tool, args)
		isErr := false
		if r, ok := resp["result"].(map[string]any); ok {
			isErr, _ = r["isError"].(bool)
		}
		var got map[string]any
		if json.Unmarshal([]byte(contentTextEvent(t, resp)), &got) != nil {
			got = map[string]any{"text": contentTextEvent(t, resp)}
		}
		return got, isErr
	}

	got, isErr := call("get_landing_sequence", map[string]any{"icao": "lkpr"})
	seqs, _ := got["sequences"].([]any)
	if isErr || len(seqs) != 1 {
		t.Fatalf("sequence: %v", got)
	}
	arr := seqs[0].(map[string]any)["arrivals"].([]any)
	second := arr[1].(map[string]any)
	if len(arr) != 2 || second["behind"] != "CSA1" || second["delay_s"] != 90.0 || arr[0].(map[string]any)["distance_to_go_nm"] != 12.3 {
		t.Errorf("arrivals: %v", arr)
	}
	if tw, _ := got["tower"].([]any); len(tw) != 1 {
		t.Errorf("tower: %v", got["tower"])
	}

	got, isErr = call("approach_instruction", map[string]any{"callsign": "dlh2", "instruction": "slow"})
	if isErr || got["said"] != ft.ApproachSaid || len(ft.Instructions) != 1 || ft.Instructions[0] != "DLH2 slow" {
		t.Errorf("instruction: %v %v", got, ft.Instructions)
	}
	if _, isErr = call("approach_instruction", map[string]any{"callsign": "DLH2", "instruction": "barrel roll"}); !isErr {
		t.Error("unknown instruction accepted")
	}
	ft.ApproachErr = live.ErrNotSequenced
	if got, isErr = call("approach_instruction", map[string]any{"callsign": "NOPE1", "instruction": "up"}); !isErr {
		t.Errorf("not sequenced: %v", got)
	}

	got, isErr = call("get_atc_log", map[string]any{"icao": "LKPR"})
	if isErr || got["count"] != 1.0 {
		t.Errorf("log: %v", got)
	}

	got, isErr = call("get_conflicts", nil)
	cs, _ := got["conflicts"].([]any)
	if isErr || len(cs) != 1 {
		t.Fatalf("conflicts: %v", got)
	}
	if res, _ := cs[0].(map[string]any)["resolution"].(map[string]any); res == nil || res["callsign"] != "CSA7" {
		t.Errorf("resolution: %v (only ours is steered)", cs[0])
	}

	got, isErr = call("separation_minima", map[string]any{"leader": "A388", "follower": "A320"})
	if isErr || got["wake_minimum_nm"].(float64) < 6 || got["departure_interval_s"].(float64) < 120 {
		t.Errorf("A388 then A320: %v", got)
	}
	got, _ = call("separation_minima", map[string]any{"leader": "A320", "follower": "A320", "visibility_m": 400})
	if got["spacing_on_final_nm"].(float64) < 6 {
		t.Errorf("low visibility: %v", got)
	}
	if _, isErr = call("separation_minima", map[string]any{"leader": "A320"}); !isErr {
		t.Error("missing follower accepted")
	}
}
