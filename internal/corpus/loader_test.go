package corpus

import (
	"os"
	"testing"
	"time"
)

// ── LoadEmbedded ──────────────────────────────────────────────────────────────

func TestLoadEmbedded_DefaultVersion_NonEmptyCorpus(t *testing.T) {
	// DOCS_MSFS_VERSION unset → both versions merged.
	t.Setenv("DOCS_MSFS_VERSION", "")

	c, err := LoadEmbedded().Load()
	if err != nil {
		t.Fatalf("LoadEmbedded().Load() error: %v", err)
	}
	if len(c.SimVars) == 0 {
		t.Error("expected non-empty SimVars")
	}
	if len(c.Events) == 0 {
		t.Error("expected non-empty Events")
	}
	if len(c.Functions) == 0 {
		t.Error("expected non-empty Functions")
	}
	if len(c.Structures) == 0 {
		t.Error("expected non-empty Structures")
	}
	if len(c.ErrorCodes) == 0 {
		t.Error("expected non-empty ErrorCodes")
	}
}

func TestLoadEmbedded_DefaultVersion_ScrapedAtNonZero(t *testing.T) {
	t.Setenv("DOCS_MSFS_VERSION", "")

	c, err := LoadEmbedded().Load()
	if err != nil {
		t.Fatalf("LoadEmbedded().Load() error: %v", err)
	}
	if c.ScrapedAt.IsZero() {
		t.Error("expected non-zero ScrapedAt")
	}
}

func TestLoadEmbedded_Version2020_AllSimVarVersions(t *testing.T) {
	t.Setenv("DOCS_MSFS_VERSION", "2020")

	c, err := LoadEmbedded().Load()
	if err != nil {
		t.Fatalf("LoadEmbedded().Load() error: %v", err)
	}
	if len(c.SimVars) == 0 {
		t.Fatal("expected non-empty SimVars for version 2020")
	}
	for i, sv := range c.SimVars {
		if len(sv.Versions) != 1 || sv.Versions[0] != "2020" {
			t.Errorf("SimVars[%d] (%q): Versions = %v, want [\"2020\"]", i, sv.Name, sv.Versions)
		}
	}
}

func TestLoadEmbedded_Version2020_ScrapedAtNonZero(t *testing.T) {
	t.Setenv("DOCS_MSFS_VERSION", "2020")

	c, err := LoadEmbedded().Load()
	if err != nil {
		t.Fatalf("LoadEmbedded().Load() error: %v", err)
	}
	if c.ScrapedAt.IsZero() {
		t.Error("expected non-zero ScrapedAt for version 2020")
	}
}

func TestLoadEmbedded_Version2020_AllEventVersions(t *testing.T) {
	t.Setenv("DOCS_MSFS_VERSION", "2020")

	c, err := LoadEmbedded().Load()
	if err != nil {
		t.Fatalf("LoadEmbedded().Load() error: %v", err)
	}
	for i, ev := range c.Events {
		if len(ev.Versions) != 1 || ev.Versions[0] != "2020" {
			t.Errorf("Events[%d] (%q): Versions = %v, want [\"2020\"]", i, ev.Name, ev.Versions)
		}
	}
}

func TestLoadEmbedded_Version2024_AllSimVarVersions(t *testing.T) {
	t.Setenv("DOCS_MSFS_VERSION", "2024")

	c, err := LoadEmbedded().Load()
	if err != nil {
		t.Fatalf("LoadEmbedded().Load() error: %v", err)
	}
	if len(c.SimVars) == 0 {
		t.Fatal("expected non-empty SimVars for version 2024")
	}
	for i, sv := range c.SimVars {
		if len(sv.Versions) != 1 || sv.Versions[0] != "2024" {
			t.Errorf("SimVars[%d] (%q): Versions = %v, want [\"2024\"]", i, sv.Name, sv.Versions)
		}
	}
}

func TestLoadEmbedded_Version2024_ScrapedAtNonZero(t *testing.T) {
	t.Setenv("DOCS_MSFS_VERSION", "2024")

	c, err := LoadEmbedded().Load()
	if err != nil {
		t.Fatalf("LoadEmbedded().Load() error: %v", err)
	}
	if c.ScrapedAt.IsZero() {
		t.Error("expected non-zero ScrapedAt for version 2024")
	}
}

func TestLoadEmbedded_VersionBoth_SharedSimVarsHaveTwoVersions(t *testing.T) {
	t.Setenv("DOCS_MSFS_VERSION", "both")

	c, err := LoadEmbedded().Load()
	if err != nil {
		t.Fatalf("LoadEmbedded().Load() error: %v", err)
	}

	// In "both" mode, simvars that appear in both 2020 and 2024 docs must have
	// Versions: ["2020","2024"]. The real corpus has 387+ such shared simvars.
	sharedCount := 0
	for _, sv := range c.SimVars {
		if len(sv.Versions) == 2 && sv.Versions[0] == "2020" && sv.Versions[1] == "2024" {
			sharedCount++
		}
	}

	if sharedCount < 50 {
		t.Errorf("expected at least 50 shared SimVars with versions [\"2020\",\"2024\"], found %d", sharedCount)
	}
}

func TestLoadEmbedded_VersionBoth_ScrapedAtNonZero(t *testing.T) {
	t.Setenv("DOCS_MSFS_VERSION", "both")

	c, err := LoadEmbedded().Load()
	if err != nil {
		t.Fatalf("LoadEmbedded().Load() error: %v", err)
	}
	if c.ScrapedAt.IsZero() {
		t.Error("expected non-zero ScrapedAt for version both")
	}
}

func TestLoadEmbedded_UnknownVersion_ReturnsError(t *testing.T) {
	t.Setenv("DOCS_MSFS_VERSION", "invalid")

	_, err := LoadEmbedded().Load()
	if err == nil {
		t.Fatal("expected error for unknown DOCS_MSFS_VERSION, got nil")
	}
}

func TestLoadEmbedded_UnknownVersionValues_ReturnError(t *testing.T) {
	cases := []string{"2019", "2025", "msfs", "BOTH", "Both", "1", "0"}
	for _, v := range cases {
		t.Run(v, func(t *testing.T) {
			t.Setenv("DOCS_MSFS_VERSION", v)
			_, err := LoadEmbedded().Load()
			if err == nil {
				t.Errorf("DOCS_MSFS_VERSION=%q: expected error, got nil", v)
			}
		})
	}
}

// ── LoadFromPath ──────────────────────────────────────────────────────────────

func TestLoadFromPath_SameSimVarCountAsEmbedded_2020(t *testing.T) {
	t.Setenv("DOCS_MSFS_VERSION", "2020")

	// LoadFromPath receives the corpus package dir; readFile constructs
	// "assets/<file>" paths relative to it.
	pkgDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	embedded, err := LoadEmbedded().Load()
	if err != nil {
		t.Fatalf("LoadEmbedded().Load(): %v", err)
	}

	fromPath, err := LoadFromPath(pkgDir).Load()
	if err != nil {
		t.Fatalf("LoadFromPath(%q).Load(): %v", pkgDir, err)
	}

	if len(fromPath.SimVars) != len(embedded.SimVars) {
		t.Errorf("SimVars count: LoadFromPath=%d, LoadEmbedded=%d", len(fromPath.SimVars), len(embedded.SimVars))
	}
}

func TestLoadFromPath_SameSimVarCountAsEmbedded_Both(t *testing.T) {
	t.Setenv("DOCS_MSFS_VERSION", "both")

	pkgDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	embedded, err := LoadEmbedded().Load()
	if err != nil {
		t.Fatalf("LoadEmbedded().Load(): %v", err)
	}

	fromPath, err := LoadFromPath(pkgDir).Load()
	if err != nil {
		t.Fatalf("LoadFromPath(%q).Load(): %v", pkgDir, err)
	}

	if len(fromPath.SimVars) != len(embedded.SimVars) {
		t.Errorf("SimVars count: LoadFromPath=%d, LoadEmbedded=%d", len(fromPath.SimVars), len(embedded.SimVars))
	}
}

func TestLoadFromPath_ScrapedAtNonZero(t *testing.T) {
	pkgDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	for _, version := range []string{"2020", "2024", "both", ""} {
		t.Run("version="+version, func(t *testing.T) {
			t.Setenv("DOCS_MSFS_VERSION", version)

			c, err := LoadFromPath(pkgDir).Load()
			if err != nil {
				t.Fatalf("LoadFromPath(%q).Load(): %v", pkgDir, err)
			}
			if c.ScrapedAt.IsZero() {
				t.Error("expected non-zero ScrapedAt")
			}
		})
	}
}

// ── latestTime helper ─────────────────────────────────────────────────────────

func TestLatestTime_ReturnsLatest(t *testing.T) {
	got, err := latestTime("2025-01-01T00:00:00Z", "2025-06-15T12:00:00Z", "2024-12-31T23:59:59Z")
	if err != nil {
		t.Fatalf("latestTime: %v", err)
	}
	want := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("latestTime: got %v, want %v", got, want)
	}
}

func TestLatestTime_EmptyStringsReturnZero(t *testing.T) {
	got, err := latestTime("", "", "")
	if err != nil {
		t.Fatalf("latestTime: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("latestTime with all empty strings: got %v, want zero time", got)
	}
}

func TestLatestTime_InvalidTimestampReturnsError(t *testing.T) {
	_, err := latestTime("not-a-time")
	if err == nil {
		t.Error("expected error for invalid timestamp, got nil")
	}
}
