package corpus

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// ── SimVar ────────────────────────────────────────────────────────────────────

func TestSimVar_JSONRoundTrip(t *testing.T) {
	t.Run("full fields", func(t *testing.T) {
		orig := SimVar{
			Name:             "PLANE ALTITUDE",
			Description:      "The altitude of the aircraft above mean sea level.",
			Units:            []string{"feet", "meters"},
			Settable:         true,
			Category:         "Aircraft Position and Speed",
			Versions:         []string{"2020", "2024"},
			IndexedBy:        "1-indexed engine number",
			Deprecated:       true,
			DeprecatedReason: "Use INDICATED ALTITUDE instead.",
			SourceURL:        "https://docs.flightsimulator.com/simvars/plane-altitude",
		}

		data, err := json.Marshal(orig)
		if err != nil {
			t.Errorf("marshal failed: %v", err)
			return
		}

		var got SimVar
		if err := json.Unmarshal(data, &got); err != nil {
			t.Errorf("unmarshal failed: %v", err)
			return
		}

		assertSimVarEqual(t, orig, got)
	})

	t.Run("without optional fields", func(t *testing.T) {
		orig := SimVar{
			Name:        "ENGINE RPM",
			Description: "Engine RPM.",
			Units:       []string{"rpm"},
			Settable:    false,
			Category:    "Engine",
			Versions:    []string{"2024"},
			SourceURL:   "https://docs.flightsimulator.com/simvars/engine-rpm",
		}

		data, err := json.Marshal(orig)
		if err != nil {
			t.Errorf("marshal failed: %v", err)
			return
		}

		var got SimVar
		if err := json.Unmarshal(data, &got); err != nil {
			t.Errorf("unmarshal failed: %v", err)
			return
		}

		assertSimVarEqual(t, orig, got)
	})
}

func TestSimVar_OmitemptyBehaviour(t *testing.T) {
	t.Run("deprecated=false absent from JSON", func(t *testing.T) {
		v := SimVar{
			Name:      "TEST VAR",
			Units:     []string{"feet"},
			Versions:  []string{"2020"},
			SourceURL: "https://example.com",
		}
		// Deprecated is false (zero value)
		data, err := json.Marshal(v)
		if err != nil {
			t.Errorf("marshal failed: %v", err)
			return
		}
		s := string(data)
		if strings.Contains(s, `"deprecated"`) {
			t.Errorf("expected 'deprecated' key to be absent from JSON when false, got: %s", s)
		}
		if strings.Contains(s, `"indexed_by"`) {
			t.Errorf("expected 'indexed_by' key to be absent from JSON when empty, got: %s", s)
		}
		if strings.Contains(s, `"deprecated_reason"`) {
			t.Errorf("expected 'deprecated_reason' key to be absent from JSON when empty, got: %s", s)
		}
	})

	t.Run("deprecated=true present in JSON", func(t *testing.T) {
		v := SimVar{
			Name:             "OLD VAR",
			Units:            []string{"feet"},
			Versions:         []string{"2020"},
			Deprecated:       true,
			DeprecatedReason: "use NEW VAR",
			SourceURL:        "https://example.com",
		}
		data, err := json.Marshal(v)
		if err != nil {
			t.Errorf("marshal failed: %v", err)
			return
		}
		s := string(data)
		if !strings.Contains(s, `"deprecated":true`) {
			t.Errorf("expected 'deprecated':true in JSON, got: %s", s)
		}
		if !strings.Contains(s, `"deprecated_reason"`) {
			t.Errorf("expected 'deprecated_reason' key in JSON, got: %s", s)
		}
	})
}

func TestSimVar_ZeroValue(t *testing.T) {
	var v SimVar
	data, err := json.Marshal(v)
	if err != nil {
		t.Errorf("marshal zero-value SimVar panicked or failed: %v", err)
	}
	var got SimVar
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal zero-value SimVar failed: %v", err)
	}
}

// ── Event ─────────────────────────────────────────────────────────────────────

func TestEvent_JSONRoundTrip(t *testing.T) {
	t.Run("with parameters", func(t *testing.T) {
		orig := Event{
			Name:        "LANDING_LIGHTS_TOGGLE",
			Description: "Toggle landing lights.",
			Parameters: []EventParam{
				{Index: 0, Name: "lightIndex", Description: "Which light to toggle.", Type: "DWORD"},
			},
			Versions:  []string{"2020", "2024"},
			SourceURL: "https://docs.flightsimulator.com/events/landing-lights-toggle",
		}

		data, err := json.Marshal(orig)
		if err != nil {
			t.Errorf("marshal failed: %v", err)
			return
		}
		var got Event
		if err := json.Unmarshal(data, &got); err != nil {
			t.Errorf("unmarshal failed: %v", err)
			return
		}

		assertEventEqual(t, orig, got)
	})

	t.Run("with deprecated", func(t *testing.T) {
		orig := Event{
			Name:             "THROTTLE_SET",
			Description:      "Sets throttle.",
			Versions:         []string{"2020"},
			Deprecated:       true,
			DeprecatedReason: "Use AXIS_THROTTLE_SET.",
			SourceURL:        "https://docs.flightsimulator.com/events/throttle-set",
		}

		data, err := json.Marshal(orig)
		if err != nil {
			t.Errorf("marshal failed: %v", err)
			return
		}
		var got Event
		if err := json.Unmarshal(data, &got); err != nil {
			t.Errorf("unmarshal failed: %v", err)
			return
		}

		assertEventEqual(t, orig, got)
	})
}

func TestEvent_OmitemptyBehaviour(t *testing.T) {
	t.Run("parameters omitted when empty slice", func(t *testing.T) {
		v := Event{
			Name:      "BRAKES",
			Versions:  []string{"2020"},
			SourceURL: "https://example.com",
		}
		data, err := json.Marshal(v)
		if err != nil {
			t.Errorf("marshal failed: %v", err)
			return
		}
		if strings.Contains(string(data), `"parameters"`) {
			t.Errorf("expected 'parameters' key absent when nil/empty, got: %s", string(data))
		}
	})

	t.Run("deprecated=false absent from JSON", func(t *testing.T) {
		v := Event{
			Name:      "BRAKES",
			Versions:  []string{"2020"},
			SourceURL: "https://example.com",
		}
		data, err := json.Marshal(v)
		if err != nil {
			t.Errorf("marshal failed: %v", err)
			return
		}
		if strings.Contains(string(data), `"deprecated"`) {
			t.Errorf("expected 'deprecated' absent when false, got: %s", string(data))
		}
	})
}

func TestEvent_ZeroValue(t *testing.T) {
	var v Event
	data, err := json.Marshal(v)
	if err != nil {
		t.Errorf("marshal zero-value Event failed: %v", err)
	}
	var got Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal zero-value Event failed: %v", err)
	}
}

// ── EventParam ────────────────────────────────────────────────────────────────

func TestEventParam_JSONRoundTrip(t *testing.T) {
	orig := EventParam{
		Index:       1,
		Name:        "speed",
		Description: "Speed in knots.",
		Type:        "float",
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Errorf("marshal failed: %v", err)
		return
	}
	var got EventParam
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal failed: %v", err)
		return
	}

	if got.Index != orig.Index {
		t.Errorf("Index: got %d, want %d", got.Index, orig.Index)
	}
	if got.Name != orig.Name {
		t.Errorf("Name: got %q, want %q", got.Name, orig.Name)
	}
	if got.Description != orig.Description {
		t.Errorf("Description: got %q, want %q", got.Description, orig.Description)
	}
	if got.Type != orig.Type {
		t.Errorf("Type: got %q, want %q", got.Type, orig.Type)
	}
}

func TestEventParam_ZeroValue(t *testing.T) {
	var v EventParam
	data, err := json.Marshal(v)
	if err != nil {
		t.Errorf("marshal zero-value EventParam failed: %v", err)
	}
	var got EventParam
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal zero-value EventParam failed: %v", err)
	}
}

// ── Function ──────────────────────────────────────────────────────────────────

func TestFunction_JSONRoundTrip(t *testing.T) {
	t.Run("with remarks", func(t *testing.T) {
		orig := Function{
			Name:        "SimConnect_Open",
			Description: "Opens a connection to the simulator.",
			Signature:   "SIMCONNECTAPI SimConnect_Open(HANDLE * phSimConnect, LPCSTR szName, HWND hWnd, DWORD UserEventWin32, HANDLE hEventHandle, DWORD ConfigIndex)",
			Parameters: []FunctionParam{
				{Name: "phSimConnect", Type: "HANDLE *", Direction: "out", Description: "Handle to the connection."},
				{Name: "szName", Type: "const char *", Direction: "in", Description: "Application name."},
				{Name: "ConfigIndex", Type: "DWORD", Direction: "in", Description: "Config index.", Optional: true},
			},
			ReturnType: "HRESULT",
			Remarks:    "Must be called before any other SimConnect function.",
			SourceURL:  "https://docs.flightsimulator.com/functions/simconnect-open",
		}

		data, err := json.Marshal(orig)
		if err != nil {
			t.Errorf("marshal failed: %v", err)
			return
		}
		var got Function
		if err := json.Unmarshal(data, &got); err != nil {
			t.Errorf("unmarshal failed: %v", err)
			return
		}

		assertFunctionEqual(t, orig, got)
	})

	t.Run("empty parameters slice", func(t *testing.T) {
		orig := Function{
			Name:        "SimConnect_Close",
			Description: "Closes the connection.",
			Signature:   "SIMCONNECTAPI SimConnect_Close(HANDLE hSimConnect)",
			Parameters:  []FunctionParam{},
			ReturnType:  "HRESULT",
			SourceURL:   "https://docs.flightsimulator.com/functions/simconnect-close",
		}

		data, err := json.Marshal(orig)
		if err != nil {
			t.Errorf("marshal failed: %v", err)
			return
		}
		var got Function
		if err := json.Unmarshal(data, &got); err != nil {
			t.Errorf("unmarshal failed: %v", err)
			return
		}

		if got.Name != orig.Name {
			t.Errorf("Name: got %q, want %q", got.Name, orig.Name)
		}
		if len(got.Parameters) != 0 {
			t.Errorf("Parameters: got %d, want 0", len(got.Parameters))
		}
	})
}

func TestFunction_RemarksOmitempty(t *testing.T) {
	v := Function{
		Name:      "SimConnect_Close",
		Signature: "SIMCONNECTAPI SimConnect_Close(HANDLE h)",
		SourceURL: "https://example.com",
	}
	data, err := json.Marshal(v)
	if err != nil {
		t.Errorf("marshal failed: %v", err)
		return
	}
	if strings.Contains(string(data), `"remarks"`) {
		t.Errorf("expected 'remarks' absent when empty, got: %s", string(data))
	}
}

func TestFunction_ZeroValue(t *testing.T) {
	var v Function
	data, err := json.Marshal(v)
	if err != nil {
		t.Errorf("marshal zero-value Function failed: %v", err)
	}
	var got Function
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal zero-value Function failed: %v", err)
	}
}

// ── FunctionParam ─────────────────────────────────────────────────────────────

func TestFunctionParam_JSONRoundTrip(t *testing.T) {
	t.Run("optional=true present", func(t *testing.T) {
		orig := FunctionParam{
			Name:        "ConfigIndex",
			Type:        "DWORD",
			Direction:   "in",
			Description: "Optional config index.",
			Optional:    true,
		}

		data, err := json.Marshal(orig)
		if err != nil {
			t.Errorf("marshal failed: %v", err)
			return
		}

		s := string(data)
		if !strings.Contains(s, `"optional":true`) {
			t.Errorf("expected 'optional':true in JSON, got: %s", s)
		}

		var got FunctionParam
		if err := json.Unmarshal(data, &got); err != nil {
			t.Errorf("unmarshal failed: %v", err)
			return
		}

		assertFunctionParamEqual(t, orig, got)
	})

	t.Run("optional=false absent from JSON", func(t *testing.T) {
		orig := FunctionParam{
			Name:        "hSimConnect",
			Type:        "HANDLE",
			Direction:   "in",
			Description: "Connection handle.",
			Optional:    false,
		}

		data, err := json.Marshal(orig)
		if err != nil {
			t.Errorf("marshal failed: %v", err)
			return
		}

		if strings.Contains(string(data), `"optional"`) {
			t.Errorf("expected 'optional' absent when false, got: %s", string(data))
		}

		var got FunctionParam
		if err := json.Unmarshal(data, &got); err != nil {
			t.Errorf("unmarshal failed: %v", err)
			return
		}

		assertFunctionParamEqual(t, orig, got)
	})
}

func TestFunctionParam_ZeroValue(t *testing.T) {
	var v FunctionParam
	data, err := json.Marshal(v)
	if err != nil {
		t.Errorf("marshal zero-value FunctionParam failed: %v", err)
	}
	var got FunctionParam
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal zero-value FunctionParam failed: %v", err)
	}
}

// ── Structure ─────────────────────────────────────────────────────────────────

func TestStructure_JSONRoundTrip(t *testing.T) {
	orig := Structure{
		Name:        "SIMCONNECT_RECV",
		Description: "Base receive packet.",
		Fields: []StructField{
			{Name: "dwSize", Type: "DWORD", Description: "Size of the structure in bytes."},
			{Name: "dwVersion", Type: "DWORD", Description: "Protocol version."},
			{Name: "dwID", Type: "DWORD", Description: "Message ID."},
		},
		Remarks:   "All receive packets begin with this header.",
		SourceURL: "https://docs.flightsimulator.com/structures/simconnect-recv",
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Errorf("marshal failed: %v", err)
		return
	}
	var got Structure
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal failed: %v", err)
		return
	}

	assertStructureEqual(t, orig, got)
}

func TestStructure_RemarksOmitempty(t *testing.T) {
	v := Structure{
		Name:      "SIMCONNECT_RECV",
		Fields:    []StructField{},
		SourceURL: "https://example.com",
	}
	data, err := json.Marshal(v)
	if err != nil {
		t.Errorf("marshal failed: %v", err)
		return
	}
	if strings.Contains(string(data), `"remarks"`) {
		t.Errorf("expected 'remarks' absent when empty, got: %s", string(data))
	}
}

func TestStructure_ZeroValue(t *testing.T) {
	var v Structure
	data, err := json.Marshal(v)
	if err != nil {
		t.Errorf("marshal zero-value Structure failed: %v", err)
	}
	var got Structure
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal zero-value Structure failed: %v", err)
	}
}

// ── StructField ───────────────────────────────────────────────────────────────

func TestStructField_JSONRoundTrip(t *testing.T) {
	orig := StructField{
		Name:        "dwSize",
		Type:        "DWORD",
		Description: "Size of the structure in bytes.",
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Errorf("marshal failed: %v", err)
		return
	}
	var got StructField
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal failed: %v", err)
		return
	}

	if got.Name != orig.Name {
		t.Errorf("Name: got %q, want %q", got.Name, orig.Name)
	}
	if got.Type != orig.Type {
		t.Errorf("Type: got %q, want %q", got.Type, orig.Type)
	}
	if got.Description != orig.Description {
		t.Errorf("Description: got %q, want %q", got.Description, orig.Description)
	}
}

func TestStructField_ZeroValue(t *testing.T) {
	var v StructField
	data, err := json.Marshal(v)
	if err != nil {
		t.Errorf("marshal zero-value StructField failed: %v", err)
	}
	var got StructField
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal zero-value StructField failed: %v", err)
	}
}

// ── ErrorCode ─────────────────────────────────────────────────────────────────

func TestErrorCode_JSONRoundTrip(t *testing.T) {
	orig := ErrorCode{
		Name:        "SIMCONNECT_EXCEPTION_NONE",
		Value:       0,
		Description: "No error occurred.",
		SourceURL:   "https://docs.flightsimulator.com/errors/simconnect-exception-none",
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Errorf("marshal failed: %v", err)
		return
	}
	var got ErrorCode
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal failed: %v", err)
		return
	}

	if got.Name != orig.Name {
		t.Errorf("Name: got %q, want %q", got.Name, orig.Name)
	}
	if got.Value != orig.Value {
		t.Errorf("Value: got %d, want %d", got.Value, orig.Value)
	}
	if got.Description != orig.Description {
		t.Errorf("Description: got %q, want %q", got.Description, orig.Description)
	}
	if got.SourceURL != orig.SourceURL {
		t.Errorf("SourceURL: got %q, want %q", got.SourceURL, orig.SourceURL)
	}
}

func TestErrorCode_ZeroValue(t *testing.T) {
	var v ErrorCode
	data, err := json.Marshal(v)
	if err != nil {
		t.Errorf("marshal zero-value ErrorCode failed: %v", err)
	}
	var got ErrorCode
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal zero-value ErrorCode failed: %v", err)
	}
}

// ── Corpus ────────────────────────────────────────────────────────────────────

func TestCorpus_JSONRoundTrip(t *testing.T) {
	// Use UTC and truncate to second to avoid sub-second precision loss in JSON.
	scrapedAt := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	orig := Corpus{
		ScrapedAt:  scrapedAt,
		SDKVersion: "0.21.1",
		SimVars: []SimVar{
			{Name: "PLANE ALTITUDE", Units: []string{"feet"}, Versions: []string{"2020"}, SourceURL: "https://example.com/sv"},
		},
		Events: []Event{
			{Name: "BRAKES", Versions: []string{"2020"}, SourceURL: "https://example.com/ev"},
		},
		Functions: []Function{
			{Name: "SimConnect_Open", Parameters: []FunctionParam{}, SourceURL: "https://example.com/fn"},
		},
		Structures: []Structure{
			{Name: "SIMCONNECT_RECV", Fields: []StructField{}, SourceURL: "https://example.com/st"},
		},
		ErrorCodes: []ErrorCode{
			{Name: "SIMCONNECT_EXCEPTION_NONE", Value: 0, SourceURL: "https://example.com/ec"},
		},
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Errorf("marshal failed: %v", err)
		return
	}
	var got Corpus
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal failed: %v", err)
		return
	}

	// Use time.Equal for ScrapedAt — avoids monotonic clock and timezone mismatch.
	if !got.ScrapedAt.Equal(orig.ScrapedAt) {
		t.Errorf("ScrapedAt: got %v, want %v", got.ScrapedAt, orig.ScrapedAt)
	}
	if got.SDKVersion != orig.SDKVersion {
		t.Errorf("SDKVersion: got %q, want %q", got.SDKVersion, orig.SDKVersion)
	}
	if len(got.SimVars) != len(orig.SimVars) {
		t.Errorf("SimVars len: got %d, want %d", len(got.SimVars), len(orig.SimVars))
	}
	if len(got.Events) != len(orig.Events) {
		t.Errorf("Events len: got %d, want %d", len(got.Events), len(orig.Events))
	}
	if len(got.Functions) != len(orig.Functions) {
		t.Errorf("Functions len: got %d, want %d", len(got.Functions), len(orig.Functions))
	}
	if len(got.Structures) != len(orig.Structures) {
		t.Errorf("Structures len: got %d, want %d", len(got.Structures), len(orig.Structures))
	}
	if len(got.ErrorCodes) != len(orig.ErrorCodes) {
		t.Errorf("ErrorCodes len: got %d, want %d", len(got.ErrorCodes), len(orig.ErrorCodes))
	}
}

func TestCorpus_ZeroValue(t *testing.T) {
	var v Corpus
	data, err := json.Marshal(v)
	if err != nil {
		t.Errorf("marshal zero-value Corpus failed: %v", err)
	}
	var got Corpus
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal zero-value Corpus failed: %v", err)
	}
}

// ── SearchResult ──────────────────────────────────────────────────────────────

func TestSearchResult_JSONRoundTrip(t *testing.T) {
	orig := SearchResult{
		Type:      "simvar",
		Name:      "PLANE ALTITUDE",
		Excerpt:   "The altitude of the aircraft above mean sea level.",
		SourceURL: "https://docs.flightsimulator.com/simvars/plane-altitude",
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Errorf("marshal failed: %v", err)
		return
	}
	var got SearchResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal failed: %v", err)
		return
	}

	if got.Type != orig.Type {
		t.Errorf("Type: got %q, want %q", got.Type, orig.Type)
	}
	if got.Name != orig.Name {
		t.Errorf("Name: got %q, want %q", got.Name, orig.Name)
	}
	if got.Excerpt != orig.Excerpt {
		t.Errorf("Excerpt: got %q, want %q", got.Excerpt, orig.Excerpt)
	}
	if got.SourceURL != orig.SourceURL {
		t.Errorf("SourceURL: got %q, want %q", got.SourceURL, orig.SourceURL)
	}
}

func TestSearchResult_ZeroValue(t *testing.T) {
	var v SearchResult
	data, err := json.Marshal(v)
	if err != nil {
		t.Errorf("marshal zero-value SearchResult failed: %v", err)
	}
	var got SearchResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal zero-value SearchResult failed: %v", err)
	}
}

// ── SearchResults ─────────────────────────────────────────────────────────────

func TestSearchResults_JSONRoundTrip(t *testing.T) {
	orig := SearchResults{
		Query: "altitude",
		Total: 2,
		Results: []SearchResult{
			{Type: "simvar", Name: "PLANE ALTITUDE", Excerpt: "MSL altitude.", SourceURL: "https://example.com/sv1"},
			{Type: "simvar", Name: "GROUND ALTITUDE", Excerpt: "Ground elevation.", SourceURL: "https://example.com/sv2"},
		},
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Errorf("marshal failed: %v", err)
		return
	}
	var got SearchResults
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal failed: %v", err)
		return
	}

	if got.Query != orig.Query {
		t.Errorf("Query: got %q, want %q", got.Query, orig.Query)
	}
	if got.Total != orig.Total {
		t.Errorf("Total: got %d, want %d", got.Total, orig.Total)
	}
	if len(got.Results) != len(orig.Results) {
		t.Errorf("Results len: got %d, want %d", len(got.Results), len(orig.Results))
	}
	for i := range orig.Results {
		if got.Results[i].Name != orig.Results[i].Name {
			t.Errorf("Results[%d].Name: got %q, want %q", i, got.Results[i].Name, orig.Results[i].Name)
		}
		if got.Results[i].SourceURL != orig.Results[i].SourceURL {
			t.Errorf("Results[%d].SourceURL: got %q, want %q", i, got.Results[i].SourceURL, orig.Results[i].SourceURL)
		}
	}
}

func TestSearchResults_ZeroValue(t *testing.T) {
	var v SearchResults
	data, err := json.Marshal(v)
	if err != nil {
		t.Errorf("marshal zero-value SearchResults failed: %v", err)
	}
	var got SearchResults
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal zero-value SearchResults failed: %v", err)
	}
}

// ── Page[T] ───────────────────────────────────────────────────────────────────

func TestPage_SimVar_JSONRoundTrip(t *testing.T) {
	orig := Page[SimVar]{
		Items: []SimVar{
			{Name: "PLANE ALTITUDE", Units: []string{"feet"}, Versions: []string{"2020"}, SourceURL: "https://example.com/sv1"},
			{Name: "ENGINE RPM", Units: []string{"rpm"}, Versions: []string{"2024"}, SourceURL: "https://example.com/sv2"},
		},
		Page:       1,
		PageSize:   50,
		TotalItems: 2,
		TotalPages: 1,
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Errorf("marshal Page[SimVar] failed: %v", err)
		return
	}
	var got Page[SimVar]
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal Page[SimVar] failed: %v", err)
		return
	}

	assertPageFields(t, orig.Page, orig.PageSize, orig.TotalItems, orig.TotalPages,
		got.Page, got.PageSize, got.TotalItems, got.TotalPages)

	if len(got.Items) != len(orig.Items) {
		t.Errorf("Items len: got %d, want %d", len(got.Items), len(orig.Items))
		return
	}
	for i := range orig.Items {
		if got.Items[i].Name != orig.Items[i].Name {
			t.Errorf("Items[%d].Name: got %q, want %q", i, got.Items[i].Name, orig.Items[i].Name)
		}
	}
}

func TestPage_ErrorCode_JSONRoundTrip(t *testing.T) {
	orig := Page[ErrorCode]{
		Items: []ErrorCode{
			{Name: "SIMCONNECT_EXCEPTION_NONE", Value: 0, SourceURL: "https://example.com/ec0"},
			{Name: "SIMCONNECT_EXCEPTION_ERROR", Value: 1, SourceURL: "https://example.com/ec1"},
			{Name: "SIMCONNECT_EXCEPTION_SIZE_MISMATCH", Value: 2, SourceURL: "https://example.com/ec2"},
		},
		Page:       2,
		PageSize:   3,
		TotalItems: 9,
		TotalPages: 3,
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Errorf("marshal Page[ErrorCode] failed: %v", err)
		return
	}
	var got Page[ErrorCode]
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("unmarshal Page[ErrorCode] failed: %v", err)
		return
	}

	assertPageFields(t, orig.Page, orig.PageSize, orig.TotalItems, orig.TotalPages,
		got.Page, got.PageSize, got.TotalItems, got.TotalPages)

	if len(got.Items) != len(orig.Items) {
		t.Errorf("Items len: got %d, want %d", len(got.Items), len(orig.Items))
		return
	}
	for i := range orig.Items {
		if got.Items[i].Name != orig.Items[i].Name {
			t.Errorf("Items[%d].Name: got %q, want %q", i, got.Items[i].Name, orig.Items[i].Name)
		}
		if got.Items[i].Value != orig.Items[i].Value {
			t.Errorf("Items[%d].Value: got %d, want %d", i, got.Items[i].Value, orig.Items[i].Value)
		}
	}
}

func TestPage_ZeroValue(t *testing.T) {
	t.Run("Page[SimVar]", func(t *testing.T) {
		var v Page[SimVar]
		data, err := json.Marshal(v)
		if err != nil {
			t.Errorf("marshal zero-value Page[SimVar] failed: %v", err)
		}
		var got Page[SimVar]
		if err := json.Unmarshal(data, &got); err != nil {
			t.Errorf("unmarshal zero-value Page[SimVar] failed: %v", err)
		}
	})

	t.Run("Page[ErrorCode]", func(t *testing.T) {
		var v Page[ErrorCode]
		data, err := json.Marshal(v)
		if err != nil {
			t.Errorf("marshal zero-value Page[ErrorCode] failed: %v", err)
		}
		var got Page[ErrorCode]
		if err := json.Unmarshal(data, &got); err != nil {
			t.Errorf("unmarshal zero-value Page[ErrorCode] failed: %v", err)
		}
	})
}

// ── ErrNotFound ───────────────────────────────────────────────────────────────

func TestErrNotFound_Sentinel(t *testing.T) {
	t.Run("errors.Is same instance", func(t *testing.T) {
		if !errors.Is(ErrNotFound, ErrNotFound) {
			t.Errorf("errors.Is(ErrNotFound, ErrNotFound) must be true")
		}
	})

	t.Run("errors.Is returns true when wrapping ErrNotFound", func(t *testing.T) {
		wrapped := errors.New("not found")
		// A freshly created error with the same message is NOT the sentinel.
		if errors.Is(wrapped, ErrNotFound) {
			t.Errorf("errors.Is(errors.New(\"not found\"), ErrNotFound) must be false — different instance")
		}
	})

	t.Run("error message matches sentinel text", func(t *testing.T) {
		if ErrNotFound.Error() != "not found" {
			t.Errorf("ErrNotFound.Error(): got %q, want %q", ErrNotFound.Error(), "not found")
		}
	})
}

// ── assertion helpers ─────────────────────────────────────────────────────────

func assertSimVarEqual(t *testing.T, want, got SimVar) {
	t.Helper()
	if got.Name != want.Name {
		t.Errorf("Name: got %q, want %q", got.Name, want.Name)
	}
	if got.Description != want.Description {
		t.Errorf("Description: got %q, want %q", got.Description, want.Description)
	}
	if len(got.Units) != len(want.Units) {
		t.Errorf("Units len: got %d, want %d", len(got.Units), len(want.Units))
	} else {
		for i := range want.Units {
			if got.Units[i] != want.Units[i] {
				t.Errorf("Units[%d]: got %q, want %q", i, got.Units[i], want.Units[i])
			}
		}
	}
	if got.Settable != want.Settable {
		t.Errorf("Settable: got %v, want %v", got.Settable, want.Settable)
	}
	if got.Category != want.Category {
		t.Errorf("Category: got %q, want %q", got.Category, want.Category)
	}
	if len(got.Versions) != len(want.Versions) {
		t.Errorf("Versions len: got %d, want %d", len(got.Versions), len(want.Versions))
	} else {
		for i := range want.Versions {
			if got.Versions[i] != want.Versions[i] {
				t.Errorf("Versions[%d]: got %q, want %q", i, got.Versions[i], want.Versions[i])
			}
		}
	}
	if got.IndexedBy != want.IndexedBy {
		t.Errorf("IndexedBy: got %q, want %q", got.IndexedBy, want.IndexedBy)
	}
	if got.Deprecated != want.Deprecated {
		t.Errorf("Deprecated: got %v, want %v", got.Deprecated, want.Deprecated)
	}
	if got.DeprecatedReason != want.DeprecatedReason {
		t.Errorf("DeprecatedReason: got %q, want %q", got.DeprecatedReason, want.DeprecatedReason)
	}
	if got.SourceURL != want.SourceURL {
		t.Errorf("SourceURL: got %q, want %q", got.SourceURL, want.SourceURL)
	}
}

func assertEventEqual(t *testing.T, want, got Event) {
	t.Helper()
	if got.Name != want.Name {
		t.Errorf("Name: got %q, want %q", got.Name, want.Name)
	}
	if got.Description != want.Description {
		t.Errorf("Description: got %q, want %q", got.Description, want.Description)
	}
	if len(got.Parameters) != len(want.Parameters) {
		t.Errorf("Parameters len: got %d, want %d", len(got.Parameters), len(want.Parameters))
	}
	if len(got.Versions) != len(want.Versions) {
		t.Errorf("Versions len: got %d, want %d", len(got.Versions), len(want.Versions))
	}
	if got.Deprecated != want.Deprecated {
		t.Errorf("Deprecated: got %v, want %v", got.Deprecated, want.Deprecated)
	}
	if got.DeprecatedReason != want.DeprecatedReason {
		t.Errorf("DeprecatedReason: got %q, want %q", got.DeprecatedReason, want.DeprecatedReason)
	}
	if got.SourceURL != want.SourceURL {
		t.Errorf("SourceURL: got %q, want %q", got.SourceURL, want.SourceURL)
	}
}

func assertFunctionEqual(t *testing.T, want, got Function) {
	t.Helper()
	if got.Name != want.Name {
		t.Errorf("Name: got %q, want %q", got.Name, want.Name)
	}
	if got.Description != want.Description {
		t.Errorf("Description: got %q, want %q", got.Description, want.Description)
	}
	if got.Signature != want.Signature {
		t.Errorf("Signature: got %q, want %q", got.Signature, want.Signature)
	}
	if len(got.Parameters) != len(want.Parameters) {
		t.Errorf("Parameters len: got %d, want %d", len(got.Parameters), len(want.Parameters))
	} else {
		for i := range want.Parameters {
			assertFunctionParamEqual(t, want.Parameters[i], got.Parameters[i])
		}
	}
	if got.ReturnType != want.ReturnType {
		t.Errorf("ReturnType: got %q, want %q", got.ReturnType, want.ReturnType)
	}
	if got.Remarks != want.Remarks {
		t.Errorf("Remarks: got %q, want %q", got.Remarks, want.Remarks)
	}
	if got.SourceURL != want.SourceURL {
		t.Errorf("SourceURL: got %q, want %q", got.SourceURL, want.SourceURL)
	}
}

func assertFunctionParamEqual(t *testing.T, want, got FunctionParam) {
	t.Helper()
	if got.Name != want.Name {
		t.Errorf("Name: got %q, want %q", got.Name, want.Name)
	}
	if got.Type != want.Type {
		t.Errorf("Type: got %q, want %q", got.Type, want.Type)
	}
	if got.Direction != want.Direction {
		t.Errorf("Direction: got %q, want %q", got.Direction, want.Direction)
	}
	if got.Description != want.Description {
		t.Errorf("Description: got %q, want %q", got.Description, want.Description)
	}
	if got.Optional != want.Optional {
		t.Errorf("Optional: got %v, want %v", got.Optional, want.Optional)
	}
}

func assertStructureEqual(t *testing.T, want, got Structure) {
	t.Helper()
	if got.Name != want.Name {
		t.Errorf("Name: got %q, want %q", got.Name, want.Name)
	}
	if got.Description != want.Description {
		t.Errorf("Description: got %q, want %q", got.Description, want.Description)
	}
	if len(got.Fields) != len(want.Fields) {
		t.Errorf("Fields len: got %d, want %d", len(got.Fields), len(want.Fields))
	} else {
		for i := range want.Fields {
			if got.Fields[i].Name != want.Fields[i].Name {
				t.Errorf("Fields[%d].Name: got %q, want %q", i, got.Fields[i].Name, want.Fields[i].Name)
			}
			if got.Fields[i].Type != want.Fields[i].Type {
				t.Errorf("Fields[%d].Type: got %q, want %q", i, got.Fields[i].Type, want.Fields[i].Type)
			}
		}
	}
	if got.Remarks != want.Remarks {
		t.Errorf("Remarks: got %q, want %q", got.Remarks, want.Remarks)
	}
	if got.SourceURL != want.SourceURL {
		t.Errorf("SourceURL: got %q, want %q", got.SourceURL, want.SourceURL)
	}
}

func assertPageFields(t *testing.T, wantPage, wantPageSize, wantTotalItems, wantTotalPages,
	gotPage, gotPageSize, gotTotalItems, gotTotalPages int) {
	t.Helper()
	if gotPage != wantPage {
		t.Errorf("Page: got %d, want %d", gotPage, wantPage)
	}
	if gotPageSize != wantPageSize {
		t.Errorf("PageSize: got %d, want %d", gotPageSize, wantPageSize)
	}
	if gotTotalItems != wantTotalItems {
		t.Errorf("TotalItems: got %d, want %d", gotTotalItems, wantTotalItems)
	}
	if gotTotalPages != wantTotalPages {
		t.Errorf("TotalPages: got %d, want %d", gotTotalPages, wantTotalPages)
	}
}
