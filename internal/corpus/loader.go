// Package corpus provides the DocLoader interface and its two implementations:
// LoadEmbedded (reads from the compiled-in assets) and LoadFromPath (reads from
// an arbitrary directory on disk). Both implementations delegate to the private
// load function which applies DOCS_MSFS_VERSION merge logic at call time.
package corpus

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"
)

// DocLoader is the interface satisfied by both LoadEmbedded and LoadFromPath.
// Callers invoke Load() to obtain a fully merged Corpus. DOCS_MSFS_VERSION is
// read from the environment at each Load() call, not at construction time.
type DocLoader interface {
	Load() (Corpus, error)
}

// embeddedLoader reads JSON assets from the package-embedded FS.
// version is the SDK version to load; empty string falls back to DOCS_MSFS_VERSION env var.
type embeddedLoader struct{ version string }

// pathLoader reads JSON assets from a directory path on disk.
// version is the SDK version to load; empty string falls back to DOCS_MSFS_VERSION env var.
type pathLoader struct {
	path    string
	version string
}

// LoadEmbedded returns a DocLoader backed by the compiled-in embedded assets.
// The version to serve is read from the DOCS_MSFS_VERSION environment variable
// at each Load() call. Use LoadEmbeddedVersion to supply the version explicitly.
func LoadEmbedded() DocLoader {
	return embeddedLoader{}
}

// LoadEmbeddedVersion returns a DocLoader backed by the embedded assets that
// serves the specified version without reading the environment.
func LoadEmbeddedVersion(version string) DocLoader {
	return embeddedLoader{version: version}
}

// LoadFromPath returns a DocLoader that reads JSON assets from the given
// directory path. os.DirFS constrains reads inside path (no traversal).
// The version is read from DOCS_MSFS_VERSION at each Load() call.
func LoadFromPath(path string) DocLoader {
	return pathLoader{path: path}
}

// LoadFromPathVersion returns a DocLoader that reads from path with an
// explicit version, bypassing the environment variable.
func LoadFromPathVersion(path, version string) DocLoader {
	return pathLoader{path: path, version: version}
}

// Load implements DocLoader for the embedded filesystem.
func (e embeddedLoader) Load() (Corpus, error) {
	return load(assetsFS, e.version)
}

// Load implements DocLoader for the on-disk filesystem.
func (p pathLoader) Load() (Corpus, error) {
	return load(os.DirFS(p.path), p.version)
}

// rawCorpus mirrors the JSON envelope used by every asset file.
// All fields from every file type are present so one type covers all fixtures.
// Fields absent in a given file unmarshal to zero values.
type rawCorpus struct {
	ScrapedAt  string      `json:"scraped_at"`
	SDKVersion string      `json:"sdk_version"`
	SimVars    []SimVar    `json:"simvars"`
	Events     []Event     `json:"events"`
	Functions  []Function  `json:"functions"`
	Structures []Structure `json:"structures"`
	ErrorCodes []ErrorCode `json:"error_codes"`
}

// readFile opens a file within fsys and JSON-decodes it into rawCorpus.
func readFile(fsys fs.FS, name string) (rawCorpus, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return rawCorpus{}, fmt.Errorf("corpus: open %q: %w", name, err)
	}
	defer f.Close() //nolint:errcheck

	var rc rawCorpus
	if err := json.NewDecoder(f).Decode(&rc); err != nil {
		return rawCorpus{}, fmt.Errorf("corpus: decode %q: %w", name, err)
	}
	return rc, nil
}

// parseTimestamp parses an RFC 3339 timestamp string. An empty string returns
// the zero time without an error so that partial fixture files do not fail.
func parseTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("corpus: parse time %q: %w", s, err)
	}
	return t, nil
}

// latestTime parses all given RFC 3339 timestamp strings and returns the most
// recent non-zero time. If all are empty or zero, it returns the zero time.
func latestTime(timestamps ...string) (time.Time, error) {
	var latest time.Time
	for _, ts := range timestamps {
		t, err := parseTimestamp(ts)
		if err != nil {
			return time.Time{}, err
		}
		if t.After(latest) {
			latest = t
		}
	}
	return latest, nil
}

// load is the shared implementation for both DocLoader variants.
// It reads DOCS_MSFS_VERSION from the environment at call time and applies
// version-selection and merge logic.
//
// Version semantics:
//
//	"2020"     — simvars_2020.json + events_2020.json; Versions: ["2020"]
//	"2024"     — simvars_2024.json + events_2024.json; Versions: ["2024"]
//	"both"/"" — all four versioned files, deduplicated by canonical name;
//	             shared entries get Versions: ["2020","2024"]
//	anything else — error
func load(fsys fs.FS, version string) (Corpus, error) {
	if version == "" {
		version = os.Getenv("DOCS_MSFS_VERSION")
	}

	switch version {
	case "2020", "2024":
		return loadSingleVersion(fsys, version)
	case "both", "":
		return loadBothVersions(fsys)
	default:
		return Corpus{}, fmt.Errorf(
			"corpus: unknown DOCS_MSFS_VERSION %q: must be \"2020\", \"2024\", \"both\", or unset",
			version,
		)
	}
}

// loadSingleVersion loads the versioned SimVar and Event files alongside the
// three version-agnostic files (functions, structures, error_codes).
// The Versions field on every SimVar and Event is forced to [version].
func loadSingleVersion(fsys fs.FS, version string) (Corpus, error) {
	svRaw, err := readFile(fsys, "assets/simvars_"+version+".json")
	if err != nil {
		return Corpus{}, err
	}
	evRaw, err := readFile(fsys, "assets/events_"+version+".json")
	if err != nil {
		return Corpus{}, err
	}
	fnRaw, err := readFile(fsys, "assets/functions.json")
	if err != nil {
		return Corpus{}, err
	}
	stRaw, err := readFile(fsys, "assets/structures.json")
	if err != nil {
		return Corpus{}, err
	}
	ecRaw, err := readFile(fsys, "assets/error_codes.json")
	if err != nil {
		return Corpus{}, err
	}

	// Enforce the Versions annotation regardless of what the fixture contains.
	simVars := make([]SimVar, len(svRaw.SimVars))
	for i, sv := range svRaw.SimVars {
		sv.Versions = []string{version}
		simVars[i] = sv
	}
	events := make([]Event, len(evRaw.Events))
	for i, ev := range evRaw.Events {
		ev.Versions = []string{version}
		events[i] = ev
	}

	scrapedAt, err := latestTime(
		svRaw.ScrapedAt, evRaw.ScrapedAt,
		fnRaw.ScrapedAt, stRaw.ScrapedAt, ecRaw.ScrapedAt,
	)
	if err != nil {
		return Corpus{}, err
	}

	return Corpus{
		ScrapedAt:  scrapedAt,
		SDKVersion: svRaw.SDKVersion,
		SimVars:    simVars,
		Events:     events,
		Functions:  fnRaw.Functions,
		Structures: stRaw.Structures,
		ErrorCodes: ecRaw.ErrorCodes,
	}, nil
}

// loadBothVersions loads all four versioned files, merges SimVars and Events
// by canonical uppercase name, and annotates shared entries with both versions.
// Functions, Structures, and ErrorCodes are version-agnostic singletons sourced
// from the non-versioned fixture files.
func loadBothVersions(fsys fs.FS) (Corpus, error) {
	sv2020, err := readFile(fsys, "assets/simvars_2020.json")
	if err != nil {
		return Corpus{}, err
	}
	sv2024, err := readFile(fsys, "assets/simvars_2024.json")
	if err != nil {
		return Corpus{}, err
	}
	ev2020, err := readFile(fsys, "assets/events_2020.json")
	if err != nil {
		return Corpus{}, err
	}
	ev2024, err := readFile(fsys, "assets/events_2024.json")
	if err != nil {
		return Corpus{}, err
	}
	fnRaw, err := readFile(fsys, "assets/functions.json")
	if err != nil {
		return Corpus{}, err
	}
	stRaw, err := readFile(fsys, "assets/structures.json")
	if err != nil {
		return Corpus{}, err
	}
	ecRaw, err := readFile(fsys, "assets/error_codes.json")
	if err != nil {
		return Corpus{}, err
	}

	simVars := mergeSimVars(sv2020.SimVars, sv2024.SimVars)
	events := mergeEvents(ev2020.Events, ev2024.Events)

	scrapedAt, err := latestTime(
		sv2020.ScrapedAt, sv2024.ScrapedAt,
		ev2020.ScrapedAt, ev2024.ScrapedAt,
		fnRaw.ScrapedAt, stRaw.ScrapedAt, ecRaw.ScrapedAt,
	)
	if err != nil {
		return Corpus{}, err
	}

	return Corpus{
		ScrapedAt:  scrapedAt,
		SDKVersion: sv2024.SDKVersion,
		SimVars:    simVars,
		Events:     events,
		Functions:  fnRaw.Functions,
		Structures: stRaw.Structures,
		ErrorCodes: ecRaw.ErrorCodes,
	}, nil
}

// mergeSimVars combines two SimVar slices, deduplicating by
// strings.ToUpper(Name). Items appearing in both slices receive
// Versions: ["2020","2024"]; items in only one slice retain their version.
// Insertion order: 2020 items first, then 2024-only items appended.
func mergeSimVars(a, b []SimVar) []SimVar {
	type entry struct {
		sv     SimVar
		in2020 bool
		in2024 bool
	}

	keys := make([]string, 0, len(a)+len(b))
	index := make(map[string]*entry, len(a)+len(b))

	for _, sv := range a {
		key := strings.ToUpper(sv.Name)
		if _, exists := index[key]; !exists {
			keys = append(keys, key)
			index[key] = &entry{sv: sv, in2020: true}
		} else {
			index[key].in2020 = true
		}
	}

	for _, sv := range b {
		key := strings.ToUpper(sv.Name)
		if e, exists := index[key]; exists {
			e.in2024 = true
		} else {
			keys = append(keys, key)
			index[key] = &entry{sv: sv, in2024: true}
		}
	}

	result := make([]SimVar, 0, len(keys))
	for _, key := range keys {
		e := index[key]
		sv := e.sv
		switch {
		case e.in2020 && e.in2024:
			sv.Versions = []string{"2020", "2024"}
		case e.in2020:
			sv.Versions = []string{"2020"}
		default:
			sv.Versions = []string{"2024"}
		}
		result = append(result, sv)
	}
	return result
}

// mergeEvents combines two Event slices, deduplicating by
// strings.ToUpper(Name). Merging semantics mirror mergeSimVars.
func mergeEvents(a, b []Event) []Event {
	type entry struct {
		ev     Event
		in2020 bool
		in2024 bool
	}

	keys := make([]string, 0, len(a)+len(b))
	index := make(map[string]*entry, len(a)+len(b))

	for _, ev := range a {
		key := strings.ToUpper(ev.Name)
		if _, exists := index[key]; !exists {
			keys = append(keys, key)
			index[key] = &entry{ev: ev, in2020: true}
		} else {
			index[key].in2020 = true
		}
	}

	for _, ev := range b {
		key := strings.ToUpper(ev.Name)
		if e, exists := index[key]; exists {
			e.in2024 = true
		} else {
			keys = append(keys, key)
			index[key] = &entry{ev: ev, in2024: true}
		}
	}

	result := make([]Event, 0, len(keys))
	for _, key := range keys {
		e := index[key]
		ev := e.ev
		switch {
		case e.in2020 && e.in2024:
			ev.Versions = []string{"2020", "2024"}
		case e.in2020:
			ev.Versions = []string{"2020"}
		default:
			ev.Versions = []string{"2024"}
		}
		result = append(result, ev)
	}
	return result
}
