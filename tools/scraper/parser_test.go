package main

import (
	"os"
	"testing"
)

const (
	testSourceURL  = "https://docs.flightsimulator.com/test"
	testSDKVersion = "2020"
)

// ── ParseSimVarPage ───────────────────────────────────────────────────────────

func TestParseSimVarPage_ReturnsExpectedCount(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_simvars.html")
	simvars, err := ParseSimVarPage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseSimVarPage: unexpected error: %v", err)
	}
	const want = 5
	if len(simvars) != want {
		t.Errorf("got %d simvars, want %d", len(simvars), want)
	}
}

func TestParseSimVarPage_CategoryFromH1(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_simvars.html")
	simvars, err := ParseSimVarPage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseSimVarPage: %v", err)
	}
	want := "Aircraft Position and Speed Variables"
	for _, sv := range simvars {
		if sv.Category != want {
			t.Errorf("simvar %q: Category = %q, want %q", sv.Name, sv.Category, want)
		}
	}
}

func TestParseSimVarPage_SettableTrue(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_simvars.html")
	simvars, err := ParseSimVarPage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseSimVarPage: %v", err)
	}
	var found bool
	for _, sv := range simvars {
		if sv.Name == "PLANE ALTITUDE" {
			found = true
			if !sv.Settable {
				t.Errorf("PLANE ALTITUDE: Settable = false, want true")
			}
			if len(sv.Units) == 0 {
				t.Errorf("PLANE ALTITUDE: Units is empty")
			}
		}
	}
	if !found {
		t.Error("PLANE ALTITUDE not found in parsed simvars")
	}
}

func TestParseSimVarPage_SettableFalse(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_simvars.html")
	simvars, err := ParseSimVarPage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseSimVarPage: %v", err)
	}
	var found bool
	for _, sv := range simvars {
		if sv.Name == "AIRSPEED INDICATED" {
			found = true
			if sv.Settable {
				t.Errorf("AIRSPEED INDICATED: Settable = true, want false")
			}
		}
	}
	if !found {
		t.Error("AIRSPEED INDICATED not found in parsed simvars")
	}
}

func TestParseSimVarPage_IndexedVariable(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_simvars.html")
	simvars, err := ParseSimVarPage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseSimVarPage: %v", err)
	}
	var found bool
	for _, sv := range simvars {
		if sv.Name == "ENGINE RPM" {
			found = true
			if sv.IndexedBy == "" {
				t.Errorf("ENGINE RPM: IndexedBy is empty, want non-empty")
			}
		}
	}
	if !found {
		t.Error("ENGINE RPM not found in parsed simvars (after :index strip)")
	}
}

func TestParseSimVarPage_DeprecatedVariable(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_simvars.html")
	simvars, err := ParseSimVarPage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseSimVarPage: %v", err)
	}
	var found bool
	for _, sv := range simvars {
		if sv.Name == "PLANE HEADING DEGREES MAGNETIC" {
			found = true
			if !sv.Deprecated {
				t.Errorf("PLANE HEADING DEGREES MAGNETIC: Deprecated = false, want true")
			}
			if sv.DeprecatedReason == "" {
				t.Errorf("PLANE HEADING DEGREES MAGNETIC: DeprecatedReason is empty")
			}
		}
	}
	if !found {
		t.Error("PLANE HEADING DEGREES MAGNETIC not found in parsed simvars")
	}
}

func TestParseSimVarPage_SourceURL(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_simvars.html")
	simvars, err := ParseSimVarPage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseSimVarPage: %v", err)
	}
	for _, sv := range simvars {
		if sv.SourceURL != testSourceURL {
			t.Errorf("simvar %q: SourceURL = %q, want %q", sv.Name, sv.SourceURL, testSourceURL)
		}
	}
}

func TestParseSimVarPage_VersionSet(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_simvars.html")
	simvars, err := ParseSimVarPage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseSimVarPage: %v", err)
	}
	for _, sv := range simvars {
		if len(sv.Versions) == 0 {
			t.Errorf("simvar %q: Versions is empty", sv.Name)
			continue
		}
		if sv.Versions[0] != testSDKVersion {
			t.Errorf("simvar %q: Versions[0] = %q, want %q", sv.Name, sv.Versions[0], testSDKVersion)
		}
	}
}

func TestParseSimVarPage_InvalidHTML(t *testing.T) {
	// golang.org/x/net/html is lenient and will not error on malformed HTML —
	// it should return zero simvars (no table found) rather than an error.
	simvars, err := ParseSimVarPage([]byte("<not-valid-html>!!!"), testSDKVersion, testSourceURL)
	if err != nil {
		// An error is acceptable but not required.
		t.Logf("ParseSimVarPage on invalid HTML returned error (acceptable): %v", err)
		return
	}
	// Should produce empty results, not panic.
	if len(simvars) != 0 {
		t.Errorf("invalid HTML: got %d simvars, want 0", len(simvars))
	}
}

// ── ParseEventPage ────────────────────────────────────────────────────────────

func TestParseEventPage_ReturnsExpectedCount(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_events.html")
	events, err := ParseEventPage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseEventPage: unexpected error: %v", err)
	}
	const want = 4
	if len(events) != want {
		t.Errorf("got %d events, want %d", len(events), want)
	}
}

func TestParseEventPage_Names(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_events.html")
	events, err := ParseEventPage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseEventPage: %v", err)
	}
	wantNames := map[string]bool{
		"LANDING_LIGHTS_TOGGLE": true,
		"AUTOPILOT_ON":          true,
		"GEAR_TOGGLE":           true,
		"FLAPS_UP":              true,
	}
	for _, ev := range events {
		if !wantNames[ev.Name] {
			t.Errorf("unexpected event name %q", ev.Name)
		}
		delete(wantNames, ev.Name)
	}
	for name := range wantNames {
		t.Errorf("expected event %q not found", name)
	}
}

func TestParseEventPage_DeprecatedEvent(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_events.html")
	events, err := ParseEventPage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseEventPage: %v", err)
	}
	var found bool
	for _, ev := range events {
		if ev.Name == "FLAPS_UP" {
			found = true
			if !ev.Deprecated {
				t.Errorf("FLAPS_UP: Deprecated = false, want true")
			}
			if ev.DeprecatedReason == "" {
				t.Errorf("FLAPS_UP: DeprecatedReason is empty")
			}
		}
	}
	if !found {
		t.Error("FLAPS_UP not found in parsed events")
	}
}

func TestParseEventPage_SourceURL(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_events.html")
	events, err := ParseEventPage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseEventPage: %v", err)
	}
	for _, ev := range events {
		if ev.SourceURL != testSourceURL {
			t.Errorf("event %q: SourceURL = %q, want %q", ev.Name, ev.SourceURL, testSourceURL)
		}
	}
}

// ── ParseFunctionPage ─────────────────────────────────────────────────────────

func TestParseFunctionPage_ReturnsOneFunction(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_functions.html")
	fns, err := ParseFunctionPage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseFunctionPage: %v", err)
	}
	if len(fns) != 1 {
		t.Fatalf("got %d functions, want 1", len(fns))
	}
}

func TestParseFunctionPage_Name(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_functions.html")
	fns, _ := ParseFunctionPage(html, testSDKVersion, testSourceURL)
	if len(fns) == 0 {
		t.Fatal("no functions parsed")
	}
	want := "SimConnect_Open"
	if fns[0].Name != want {
		t.Errorf("Name = %q, want %q", fns[0].Name, want)
	}
}

func TestParseFunctionPage_Signature(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_functions.html")
	fns, _ := ParseFunctionPage(html, testSDKVersion, testSourceURL)
	if len(fns) == 0 {
		t.Fatal("no functions parsed")
	}
	if fns[0].Signature == "" {
		t.Error("Signature is empty")
	}
	if !contains(fns[0].Signature, "SimConnect_Open") {
		t.Errorf("Signature %q does not contain 'SimConnect_Open'", fns[0].Signature)
	}
}

func TestParseFunctionPage_Parameters(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_functions.html")
	fns, _ := ParseFunctionPage(html, testSDKVersion, testSourceURL)
	if len(fns) == 0 {
		t.Fatal("no functions parsed")
	}
	fn := fns[0]
	if len(fn.Parameters) == 0 {
		t.Fatal("Parameters is empty")
	}
	// phSimConnect should be the first parameter.
	if fn.Parameters[0].Name != "phSimConnect" {
		t.Errorf("Parameters[0].Name = %q, want %q", fn.Parameters[0].Name, "phSimConnect")
	}
}

func TestParseFunctionPage_ReturnType(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_functions.html")
	fns, _ := ParseFunctionPage(html, testSDKVersion, testSourceURL)
	if len(fns) == 0 {
		t.Fatal("no functions parsed")
	}
	if fns[0].ReturnType == "" {
		t.Error("ReturnType is empty")
	}
}

func TestParseFunctionPage_SourceURL(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_functions.html")
	fns, _ := ParseFunctionPage(html, testSDKVersion, testSourceURL)
	if len(fns) == 0 {
		t.Fatal("no functions parsed")
	}
	if fns[0].SourceURL != testSourceURL {
		t.Errorf("SourceURL = %q, want %q", fns[0].SourceURL, testSourceURL)
	}
}

// ── ParseStructurePage ────────────────────────────────────────────────────────

func TestParseStructurePage_ReturnsOneStructure(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_structures.html")
	structs, err := ParseStructurePage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseStructurePage: %v", err)
	}
	if len(structs) != 1 {
		t.Fatalf("got %d structures, want 1", len(structs))
	}
}

func TestParseStructurePage_Name(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_structures.html")
	structs, _ := ParseStructurePage(html, testSDKVersion, testSourceURL)
	if len(structs) == 0 {
		t.Fatal("no structures parsed")
	}
	want := "SIMCONNECT_DATA_INITPOSITION"
	if structs[0].Name != want {
		t.Errorf("Name = %q, want %q", structs[0].Name, want)
	}
}

func TestParseStructurePage_Fields(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_structures.html")
	structs, _ := ParseStructurePage(html, testSDKVersion, testSourceURL)
	if len(structs) == 0 {
		t.Fatal("no structures parsed")
	}
	st := structs[0]
	const wantFields = 8
	if len(st.Fields) != wantFields {
		t.Errorf("got %d fields, want %d", len(st.Fields), wantFields)
	}
}

func TestParseStructurePage_FirstField(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_structures.html")
	structs, _ := ParseStructurePage(html, testSDKVersion, testSourceURL)
	if len(structs) == 0 || len(structs[0].Fields) == 0 {
		t.Fatal("no fields parsed")
	}
	f := structs[0].Fields[0]
	if f.Name != "Latitude" {
		t.Errorf("Fields[0].Name = %q, want %q", f.Name, "Latitude")
	}
	if f.Type != "double" {
		t.Errorf("Fields[0].Type = %q, want %q", f.Type, "double")
	}
}

func TestParseStructurePage_Remarks(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_structures.html")
	structs, _ := ParseStructurePage(html, testSDKVersion, testSourceURL)
	if len(structs) == 0 {
		t.Fatal("no structures parsed")
	}
	if structs[0].Remarks == "" {
		t.Error("Remarks is empty")
	}
}

func TestParseStructurePage_SourceURL(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_structures.html")
	structs, _ := ParseStructurePage(html, testSDKVersion, testSourceURL)
	if len(structs) == 0 {
		t.Fatal("no structures parsed")
	}
	if structs[0].SourceURL != testSourceURL {
		t.Errorf("SourceURL = %q, want %q", structs[0].SourceURL, testSourceURL)
	}
}

// ── ParseErrorCodePage ────────────────────────────────────────────────────────

func TestParseErrorCodePage_ReturnsExpectedCount(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_errorcodes.html")
	codes, err := ParseErrorCodePage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseErrorCodePage: %v", err)
	}
	const want = 5
	if len(codes) != want {
		t.Errorf("got %d error codes, want %d", len(codes), want)
	}
}

func TestParseErrorCodePage_FirstCode(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_errorcodes.html")
	codes, err := ParseErrorCodePage(html, testSDKVersion, testSourceURL)
	if err != nil {
		t.Fatalf("ParseErrorCodePage: %v", err)
	}
	if len(codes) == 0 {
		t.Fatal("no error codes parsed")
	}
	ec := codes[0]
	if ec.Name != "SIMCONNECT_EXCEPTION_NONE" {
		t.Errorf("codes[0].Name = %q, want %q", ec.Name, "SIMCONNECT_EXCEPTION_NONE")
	}
	if ec.Value != 0 {
		t.Errorf("codes[0].Value = %d, want 0", ec.Value)
	}
}

func TestParseErrorCodePage_Values(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_errorcodes.html")
	codes, _ := ParseErrorCodePage(html, testSDKVersion, testSourceURL)
	for i, ec := range codes {
		if ec.Value != i {
			t.Errorf("codes[%d].Value = %d, want %d", i, ec.Value, i)
		}
	}
}

func TestParseErrorCodePage_SourceURL(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_errorcodes.html")
	codes, _ := ParseErrorCodePage(html, testSDKVersion, testSourceURL)
	for _, ec := range codes {
		if ec.SourceURL != testSourceURL {
			t.Errorf("errorcode %q: SourceURL = %q, want %q", ec.Name, ec.SourceURL, testSourceURL)
		}
	}
}

func TestParseErrorCodePage_Descriptions(t *testing.T) {
	html := mustReadFixture(t, "testdata/sample_errorcodes.html")
	codes, _ := ParseErrorCodePage(html, testSDKVersion, testSourceURL)
	for _, ec := range codes {
		if ec.Description == "" {
			t.Errorf("errorcode %q: Description is empty", ec.Name)
		}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// mustReadFixture reads a test fixture file, failing the test if the file
// cannot be read. No network calls are made.
func mustReadFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return data
}

// contains is a helper for simple substring checks in tests.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
