package corpus

// assets_test.go validates the 7 hand-authored JSON fixture files in
// internal/corpus/assets/ against the acceptance criteria defined in
// GitHub Issue #6 (feat(corpus): seed fixture JSON corpus assets).
//
// Tests are organised as t.Run subtests, one per fixture file, plus
// cross-file overlap checks at the bottom of the file.
//
// All file I/O uses os.ReadFile with paths relative to the package
// directory — Go's test runner sets cwd to the package directory, so
// "assets/<file>.json" resolves correctly without runtime.Caller tricks.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readFixture reads a JSON fixture file from the assets directory and
// unmarshals it into a Corpus value. The test is failed immediately if
// the file cannot be read or the JSON is malformed.
func readFixture(t *testing.T, filename string) Corpus {
	t.Helper()

	path := filepath.Join("assets", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readFixture: os.ReadFile(%q) failed: %v", path, err)
	}

	var c Corpus
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("readFixture: json.Unmarshal(%q) failed: %v", path, err)
	}

	return c
}

// ── simvars_2020.json ─────────────────────────────────────────────────────────

func TestAssets_SimVars2020(t *testing.T) {
	t.Run("file loads and is valid JSON", func(t *testing.T) {
		c := readFixture(t, "simvars_2020.json")

		// AC-1: file exists and is valid JSON (readFixture would fatal if not).
		// AC-2: only simvars populated; all other collections empty.
		if len(c.SimVars) == 0 {
			t.Fatal("SimVars must not be empty")
		}
		if len(c.Events) != 0 {
			t.Errorf("Events must be empty, got %d", len(c.Events))
		}
		if len(c.Functions) != 0 {
			t.Errorf("Functions must be empty, got %d", len(c.Functions))
		}
		if len(c.Structures) != 0 {
			t.Errorf("Structures must be empty, got %d", len(c.Structures))
		}
		if len(c.ErrorCodes) != 0 {
			t.Errorf("ErrorCodes must be empty, got %d", len(c.ErrorCodes))
		}
	})

	t.Run("sdk_version is 0.21.1", func(t *testing.T) {
		c := readFixture(t, "simvars_2020.json")
		if c.SDKVersion != "0.21.1" {
			t.Errorf("SDKVersion: got %q, want %q", c.SDKVersion, "0.21.1")
		}
	})

	t.Run("has exactly 5 simvar entries", func(t *testing.T) {
		c := readFixture(t, "simvars_2020.json")
		if len(c.SimVars) != 5 {
			t.Errorf("SimVars count: got %d, want 5", len(c.SimVars))
		}
	})

	t.Run("at least one entry has indexed_by populated", func(t *testing.T) {
		c := readFixture(t, "simvars_2020.json")
		found := false
		for _, sv := range c.SimVars {
			if sv.IndexedBy != "" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected at least one SimVar with non-empty indexed_by")
		}
	})

	t.Run("at least one entry has deprecated=true", func(t *testing.T) {
		c := readFixture(t, "simvars_2020.json")
		found := false
		for _, sv := range c.SimVars {
			if sv.Deprecated {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected at least one SimVar with deprecated=true")
		}
	})

	t.Run("all entries have non-empty source_url", func(t *testing.T) {
		c := readFixture(t, "simvars_2020.json")
		for i, sv := range c.SimVars {
			if strings.TrimSpace(sv.SourceURL) == "" {
				t.Errorf("SimVars[%d] (%q) has empty source_url", i, sv.Name)
			}
		}
	})

	t.Run("contains PLANE ALTITUDE, GROUND ALTITUDE, ENGINE RPM", func(t *testing.T) {
		c := readFixture(t, "simvars_2020.json")
		required := map[string]bool{
			"PLANE ALTITUDE":  false,
			"GROUND ALTITUDE": false,
			"ENGINE RPM":      false,
		}
		for _, sv := range c.SimVars {
			if _, ok := required[sv.Name]; ok {
				required[sv.Name] = true
			}
		}
		for name, found := range required {
			if !found {
				t.Errorf("expected SimVar %q to be present in simvars_2020.json", name)
			}
		}
	})
}

// ── simvars_2024.json ─────────────────────────────────────────────────────────

func TestAssets_SimVars2024(t *testing.T) {
	t.Run("file loads and is valid JSON", func(t *testing.T) {
		c := readFixture(t, "simvars_2024.json")

		if len(c.SimVars) == 0 {
			t.Fatal("SimVars must not be empty")
		}
		if len(c.Events) != 0 {
			t.Errorf("Events must be empty, got %d", len(c.Events))
		}
		if len(c.Functions) != 0 {
			t.Errorf("Functions must be empty, got %d", len(c.Functions))
		}
		if len(c.Structures) != 0 {
			t.Errorf("Structures must be empty, got %d", len(c.Structures))
		}
		if len(c.ErrorCodes) != 0 {
			t.Errorf("ErrorCodes must be empty, got %d", len(c.ErrorCodes))
		}
	})

	t.Run("sdk_version is 1.2.0", func(t *testing.T) {
		c := readFixture(t, "simvars_2024.json")
		if c.SDKVersion != "1.2.0" {
			t.Errorf("SDKVersion: got %q, want %q", c.SDKVersion, "1.2.0")
		}
	})

	t.Run("has exactly 5 simvar entries", func(t *testing.T) {
		c := readFixture(t, "simvars_2024.json")
		if len(c.SimVars) != 5 {
			t.Errorf("SimVars count: got %d, want 5", len(c.SimVars))
		}
	})

	t.Run("contains 3 names shared with simvars_2020", func(t *testing.T) {
		// AC-4: PLANE ALTITUDE, GROUND ALTITUDE, ENGINE RPM must appear in both.
		shared := []string{"PLANE ALTITUDE", "GROUND ALTITUDE", "ENGINE RPM"}
		c := readFixture(t, "simvars_2024.json")

		names := make(map[string]bool, len(c.SimVars))
		for _, sv := range c.SimVars {
			names[sv.Name] = true
		}

		for _, name := range shared {
			if !names[name] {
				t.Errorf("expected shared SimVar %q to be present in simvars_2024.json", name)
			}
		}
	})

	t.Run("all entries have non-empty source_url", func(t *testing.T) {
		c := readFixture(t, "simvars_2024.json")
		for i, sv := range c.SimVars {
			if strings.TrimSpace(sv.SourceURL) == "" {
				t.Errorf("SimVars[%d] (%q) has empty source_url", i, sv.Name)
			}
		}
	})
}

// ── events_2020.json ──────────────────────────────────────────────────────────

func TestAssets_Events2020(t *testing.T) {
	t.Run("file loads and is valid JSON", func(t *testing.T) {
		c := readFixture(t, "events_2020.json")

		if len(c.Events) == 0 {
			t.Fatal("Events must not be empty")
		}
		if len(c.SimVars) != 0 {
			t.Errorf("SimVars must be empty, got %d", len(c.SimVars))
		}
		if len(c.Functions) != 0 {
			t.Errorf("Functions must be empty, got %d", len(c.Functions))
		}
		if len(c.Structures) != 0 {
			t.Errorf("Structures must be empty, got %d", len(c.Structures))
		}
		if len(c.ErrorCodes) != 0 {
			t.Errorf("ErrorCodes must be empty, got %d", len(c.ErrorCodes))
		}
	})

	t.Run("has exactly 5 event entries", func(t *testing.T) {
		c := readFixture(t, "events_2020.json")
		if len(c.Events) != 5 {
			t.Errorf("Events count: got %d, want 5", len(c.Events))
		}
	})

	t.Run("THROTTLE_SET has at least one parameter", func(t *testing.T) {
		c := readFixture(t, "events_2020.json")
		found := false
		for _, ev := range c.Events {
			if ev.Name == "THROTTLE_SET" {
				found = true
				if len(ev.Parameters) == 0 {
					t.Error("THROTTLE_SET must have at least one parameter")
				}
				break
			}
		}
		if !found {
			t.Error("THROTTLE_SET event not found in events_2020.json")
		}
	})

	t.Run("all entries have non-empty source_url", func(t *testing.T) {
		c := readFixture(t, "events_2020.json")
		for i, ev := range c.Events {
			if strings.TrimSpace(ev.SourceURL) == "" {
				t.Errorf("Events[%d] (%q) has empty source_url", i, ev.Name)
			}
		}
	})
}

// ── events_2024.json ──────────────────────────────────────────────────────────

func TestAssets_Events2024(t *testing.T) {
	t.Run("file loads and is valid JSON", func(t *testing.T) {
		c := readFixture(t, "events_2024.json")

		if len(c.Events) == 0 {
			t.Fatal("Events must not be empty")
		}
		if len(c.SimVars) != 0 {
			t.Errorf("SimVars must be empty, got %d", len(c.SimVars))
		}
		if len(c.Functions) != 0 {
			t.Errorf("Functions must be empty, got %d", len(c.Functions))
		}
		if len(c.Structures) != 0 {
			t.Errorf("Structures must be empty, got %d", len(c.Structures))
		}
		if len(c.ErrorCodes) != 0 {
			t.Errorf("ErrorCodes must be empty, got %d", len(c.ErrorCodes))
		}
	})

	t.Run("has exactly 5 event entries", func(t *testing.T) {
		c := readFixture(t, "events_2024.json")
		if len(c.Events) != 5 {
			t.Errorf("Events count: got %d, want 5", len(c.Events))
		}
	})

	t.Run("AXIS_THROTTLE_SET has at least one parameter", func(t *testing.T) {
		c := readFixture(t, "events_2024.json")
		found := false
		for _, ev := range c.Events {
			if ev.Name == "AXIS_THROTTLE_SET" {
				found = true
				if len(ev.Parameters) == 0 {
					t.Error("AXIS_THROTTLE_SET must have at least one parameter")
				}
				break
			}
		}
		if !found {
			t.Error("AXIS_THROTTLE_SET event not found in events_2024.json")
		}
	})

	t.Run("at least 2 event names overlap with events_2020", func(t *testing.T) {
		// AC-8: 2 events overlap between events_2020 and events_2024.
		c2020 := readFixture(t, "events_2020.json")
		c2024 := readFixture(t, "events_2024.json")

		names2020 := make(map[string]bool, len(c2020.Events))
		for _, ev := range c2020.Events {
			names2020[ev.Name] = true
		}

		overlap := 0
		for _, ev := range c2024.Events {
			if names2020[ev.Name] {
				overlap++
			}
		}

		if overlap < 2 {
			t.Errorf("expected at least 2 overlapping event names between 2020 and 2024, got %d", overlap)
		}
	})

	t.Run("all entries have non-empty source_url", func(t *testing.T) {
		c := readFixture(t, "events_2024.json")
		for i, ev := range c.Events {
			if strings.TrimSpace(ev.SourceURL) == "" {
				t.Errorf("Events[%d] (%q) has empty source_url", i, ev.Name)
			}
		}
	})
}

// ── functions.json ────────────────────────────────────────────────────────────

func TestAssets_Functions(t *testing.T) {
	t.Run("file loads and is valid JSON", func(t *testing.T) {
		c := readFixture(t, "functions.json")

		if len(c.Functions) == 0 {
			t.Fatal("Functions must not be empty")
		}
		if len(c.SimVars) != 0 {
			t.Errorf("SimVars must be empty, got %d", len(c.SimVars))
		}
		if len(c.Events) != 0 {
			t.Errorf("Events must be empty, got %d", len(c.Events))
		}
		if len(c.Structures) != 0 {
			t.Errorf("Structures must be empty, got %d", len(c.Structures))
		}
		if len(c.ErrorCodes) != 0 {
			t.Errorf("ErrorCodes must be empty, got %d", len(c.ErrorCodes))
		}
	})

	t.Run("has exactly 3 function entries", func(t *testing.T) {
		c := readFixture(t, "functions.json")
		if len(c.Functions) != 3 {
			t.Errorf("Functions count: got %d, want 3", len(c.Functions))
		}
	})

	t.Run("SimConnect_Open has exactly 6 parameters", func(t *testing.T) {
		c := readFixture(t, "functions.json")
		found := false
		for _, fn := range c.Functions {
			if fn.Name == "SimConnect_Open" {
				found = true
				if len(fn.Parameters) != 6 {
					t.Errorf("SimConnect_Open: Parameters count: got %d, want 6", len(fn.Parameters))
				}
				break
			}
		}
		if !found {
			t.Error("SimConnect_Open not found in functions.json")
		}
	})

	t.Run("SimConnect_Open ConfigIndex parameter is optional", func(t *testing.T) {
		c := readFixture(t, "functions.json")
		for _, fn := range c.Functions {
			if fn.Name != "SimConnect_Open" {
				continue
			}
			foundConfigIndex := false
			for _, p := range fn.Parameters {
				if p.Name == "ConfigIndex" {
					foundConfigIndex = true
					if !p.Optional {
						t.Error("SimConnect_Open ConfigIndex parameter must have optional=true")
					}
					break
				}
			}
			if !foundConfigIndex {
				t.Error("SimConnect_Open must have a parameter named ConfigIndex")
			}
			break
		}
	})

	t.Run("all entries have non-empty source_url", func(t *testing.T) {
		c := readFixture(t, "functions.json")
		for i, fn := range c.Functions {
			if strings.TrimSpace(fn.SourceURL) == "" {
				t.Errorf("Functions[%d] (%q) has empty source_url", i, fn.Name)
			}
		}
	})
}

// ── structures.json ───────────────────────────────────────────────────────────

func TestAssets_Structures(t *testing.T) {
	t.Run("file loads and is valid JSON", func(t *testing.T) {
		c := readFixture(t, "structures.json")

		if len(c.Structures) == 0 {
			t.Fatal("Structures must not be empty")
		}
		if len(c.SimVars) != 0 {
			t.Errorf("SimVars must be empty, got %d", len(c.SimVars))
		}
		if len(c.Events) != 0 {
			t.Errorf("Events must be empty, got %d", len(c.Events))
		}
		if len(c.Functions) != 0 {
			t.Errorf("Functions must be empty, got %d", len(c.Functions))
		}
		if len(c.ErrorCodes) != 0 {
			t.Errorf("ErrorCodes must be empty, got %d", len(c.ErrorCodes))
		}
	})

	t.Run("has exactly 2 structure entries", func(t *testing.T) {
		c := readFixture(t, "structures.json")
		if len(c.Structures) != 2 {
			t.Errorf("Structures count: got %d, want 2", len(c.Structures))
		}
	})

	t.Run("all entries have non-empty source_url", func(t *testing.T) {
		c := readFixture(t, "structures.json")
		for i, st := range c.Structures {
			if strings.TrimSpace(st.SourceURL) == "" {
				t.Errorf("Structures[%d] (%q) has empty source_url", i, st.Name)
			}
		}
	})

	t.Run("each structure has at least one field", func(t *testing.T) {
		c := readFixture(t, "structures.json")
		for i, st := range c.Structures {
			if len(st.Fields) == 0 {
				t.Errorf("Structures[%d] (%q) has no fields", i, st.Name)
			}
		}
	})
}

// ── error_codes.json ──────────────────────────────────────────────────────────

func TestAssets_ErrorCodes(t *testing.T) {
	t.Run("file loads and is valid JSON", func(t *testing.T) {
		c := readFixture(t, "error_codes.json")

		if len(c.ErrorCodes) == 0 {
			t.Fatal("ErrorCodes must not be empty")
		}
		if len(c.SimVars) != 0 {
			t.Errorf("SimVars must be empty, got %d", len(c.SimVars))
		}
		if len(c.Events) != 0 {
			t.Errorf("Events must be empty, got %d", len(c.Events))
		}
		if len(c.Functions) != 0 {
			t.Errorf("Functions must be empty, got %d", len(c.Functions))
		}
		if len(c.Structures) != 0 {
			t.Errorf("Structures must be empty, got %d", len(c.Structures))
		}
	})

	t.Run("has exactly 5 error code entries", func(t *testing.T) {
		c := readFixture(t, "error_codes.json")
		if len(c.ErrorCodes) != 5 {
			t.Errorf("ErrorCodes count: got %d, want 5", len(c.ErrorCodes))
		}
	})

	t.Run("values span 0 through 4", func(t *testing.T) {
		// AC-10: values 0-4 present.
		c := readFixture(t, "error_codes.json")
		present := make(map[int]bool, 5)
		for _, ec := range c.ErrorCodes {
			present[ec.Value] = true
		}
		for v := 0; v <= 4; v++ {
			if !present[v] {
				t.Errorf("expected error code with value %d to be present", v)
			}
		}
	})

	t.Run("all entries have non-empty source_url", func(t *testing.T) {
		c := readFixture(t, "error_codes.json")
		for i, ec := range c.ErrorCodes {
			if strings.TrimSpace(ec.SourceURL) == "" {
				t.Errorf("ErrorCodes[%d] (%q) has empty source_url", i, ec.Name)
			}
		}
	})

	t.Run("all entries have non-empty name", func(t *testing.T) {
		c := readFixture(t, "error_codes.json")
		for i, ec := range c.ErrorCodes {
			if strings.TrimSpace(ec.Name) == "" {
				t.Errorf("ErrorCodes[%d] has empty name", i)
			}
		}
	})
}

// ── cross-file: simvars overlap ───────────────────────────────────────────────

func TestAssets_SimVars_SharedNames(t *testing.T) {
	t.Run("PLANE ALTITUDE, GROUND ALTITUDE, ENGINE RPM exist in both files", func(t *testing.T) {
		// AC-4: exactly these 3 names must be shared between simvars_2020 and simvars_2024.
		c2020 := readFixture(t, "simvars_2020.json")
		c2024 := readFixture(t, "simvars_2024.json")

		names2020 := make(map[string]bool, len(c2020.SimVars))
		for _, sv := range c2020.SimVars {
			names2020[sv.Name] = true
		}

		required := []string{"PLANE ALTITUDE", "GROUND ALTITUDE", "ENGINE RPM"}
		for _, name := range required {
			if !names2020[name] {
				t.Errorf("simvars_2020: missing required SimVar %q", name)
			}
		}

		names2024 := make(map[string]bool, len(c2024.SimVars))
		for _, sv := range c2024.SimVars {
			names2024[sv.Name] = true
		}

		for _, name := range required {
			if !names2024[name] {
				t.Errorf("simvars_2024: missing required shared SimVar %q", name)
			}
		}
	})
}
