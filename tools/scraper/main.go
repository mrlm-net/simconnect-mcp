package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mrlm-net/simconnect-mcp/internal/corpus"
)

// ── URL catalogs ──────────────────────────────────────────────────────────────
//
// These are the canonical docs.flightsimulator.com pages for each corpus type
// and SDK version. Each entry maps to a scrape+parse operation. Only the pages
// listed here are fetched — the scraper does NOT crawl links.

type pageSpec struct {
	url      string
	category string
}

var simvarPages2020 = []pageSpec{
	{
		url:      "https://docs.flightsimulator.com/html/Programming_Tools/SimVars/Aircraft_SimVars/Aircraft_Position_And_Speed_Variables.htm",
		category: "Aircraft Position and Speed",
	},
	{
		url:      "https://docs.flightsimulator.com/html/Programming_Tools/SimVars/Aircraft_SimVars/Aircraft_Engine_Variables.htm",
		category: "Aircraft Engine",
	},
	{
		url:      "https://docs.flightsimulator.com/html/Programming_Tools/SimVars/Miscellaneous_Variables.htm",
		category: "Miscellaneous",
	},
}

var simvarPages2024 = []pageSpec{
	{
		url:      "https://docs.flightsimulator.com/msfs2024/html/Programming_Tools/SimVars/Aircraft_SimVars/Aircraft_Position_And_Speed_Variables.htm",
		category: "Aircraft Position and Speed",
	},
	{
		url:      "https://docs.flightsimulator.com/msfs2024/html/Programming_Tools/SimVars/Aircraft_SimVars/Aircraft_Engine_Variables.htm",
		category: "Aircraft Engine",
	},
}

var eventPages2020 = []pageSpec{
	{url: "https://docs.flightsimulator.com/html/Programming_Tools/Event_IDs/Aircraft_Autopilot_Flight_Assist_Events.htm"},
	{url: "https://docs.flightsimulator.com/html/Programming_Tools/Event_IDs/Aircraft_Lighting_Events.htm"},
}

var eventPages2024 = []pageSpec{
	{url: "https://docs.flightsimulator.com/msfs2024/html/Programming_Tools/Event_IDs/Aircraft_Autopilot_Flight_Assist_Events.htm"},
}

var functionPages = []string{
	"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/General/SimConnect_Open.htm",
	"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/General/SimConnect_Close.htm",
	"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/Simulation_Variables/SimConnect_RequestDataOnSimObject.htm",
	"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/Simulation_Variables/SimConnect_AddToDataDefinition.htm",
}

var structurePages = []string{
	"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV.htm",
	"https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/Structures_And_Enumerations/SIMCONNECT_DATA_INITPOSITION.htm",
}

var errorCodePage = "https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/API_Reference/Structures_And_Enumerations/SIMCONNECT_EXCEPTION.htm"

// ── scrapeResult bundles a file write outcome ─────────────────────────────────

type scrapeResult struct {
	filename string
	itemType string
	count    int
	err      error
}

func main() {
	outDir := flag.String("out", "internal/corpus/assets/", "output directory for JSON files")
	version := flag.String("version", "both", "SDK version to scrape: 2020, 2024, or both")
	dryRun := flag.Bool("dry-run", false, "parse only, do not write output files")
	delay := flag.Duration("delay", time.Second, "minimum delay between HTTP requests")
	flag.Parse()

	if *version != "2020" && *version != "2024" && *version != "both" {
		log.Fatalf("-version must be 2020, 2024, or both; got %q", *version)
	}

	start := time.Now()
	s := NewScraper(*delay)

	var results []scrapeResult
	var failCount int

	// ── SimVars ───────────────────────────────────────────────────────────────

	if *version == "2020" || *version == "both" {
		r := scrapeSimVars(s, simvarPages2020, "2020", *outDir, *dryRun)
		results = append(results, r)
		if r.err != nil {
			failCount++
		}
	}
	if *version == "2024" || *version == "both" {
		r := scrapeSimVars(s, simvarPages2024, "2024", *outDir, *dryRun)
		results = append(results, r)
		if r.err != nil {
			failCount++
		}
	}

	// ── Events ────────────────────────────────────────────────────────────────

	if *version == "2020" || *version == "both" {
		r := scrapeEvents(s, eventPages2020, "2020", *outDir, *dryRun)
		results = append(results, r)
		if r.err != nil {
			failCount++
		}
	}
	if *version == "2024" || *version == "both" {
		r := scrapeEvents(s, eventPages2024, "2024", *outDir, *dryRun)
		results = append(results, r)
		if r.err != nil {
			failCount++
		}
	}

	// ── Functions (version-agnostic) ──────────────────────────────────────────

	r := scrapeFunctions(s, functionPages, *outDir, *dryRun)
	results = append(results, r)
	if r.err != nil {
		failCount++
	}

	// ── Structures (version-agnostic) ─────────────────────────────────────────

	r = scrapeStructures(s, structurePages, *outDir, *dryRun)
	results = append(results, r)
	if r.err != nil {
		failCount++
	}

	// ── Error Codes (version-agnostic) ───────────────────────────────────────

	r = scrapeErrorCodes(s, errorCodePage, *outDir, *dryRun)
	results = append(results, r)
	if r.err != nil {
		failCount++
	}

	// ── Summary ───────────────────────────────────────────────────────────────

	elapsed := time.Since(start)
	fmt.Printf("\nScrape summary (%s)\n", elapsed.Round(time.Millisecond))
	fmt.Printf("%-30s  %8s  %8s\n", "File", "Items", "Status")
	fmt.Printf("%s\n", repeatStr("-", 52))

	filesWritten := 0
	for _, res := range results {
		status := "ok"
		if res.err != nil {
			status = fmt.Sprintf("FAIL: %v", res.err)
		} else if !*dryRun {
			filesWritten++
		}
		fmt.Printf("%-30s  %8d  %s\n", res.filename, res.count, status)
	}

	if *dryRun {
		fmt.Println("\n[dry-run] No files written.")
	} else {
		fmt.Printf("\nFiles written: %d  Failures: %d\n", filesWritten, failCount)
	}

	if failCount > 0 {
		os.Exit(1)
	}
}

// ── Per-type scrape functions ─────────────────────────────────────────────────

func scrapeSimVars(s *Scraper, pages []pageSpec, ver, outDir string, dryRun bool) scrapeResult {
	filename := fmt.Sprintf("simvars_%s.json", ver)
	var allVars []corpus.SimVar

	for _, p := range pages {
		htmlBytes, err := s.FetchPage(p.url)
		if err != nil {
			log.Printf("simvars %s: fetch %s: %v", ver, p.url, err)
			continue
		}
		vars, err := ParseSimVarPage(htmlBytes, ver, p.url)
		if err != nil {
			log.Printf("simvars %s: parse %s: %v", ver, p.url, err)
			continue
		}
		allVars = append(allVars, vars...)
	}

	if len(allVars) == 0 {
		return scrapeResult{filename: filename, itemType: "simvar", err: fmt.Errorf("no simvars scraped for version %s", ver)}
	}

	c := buildCorpus(ver)
	c.SimVars = allVars

	err := writeCorpusFile(c, outDir, filename, dryRun)
	return scrapeResult{filename: filename, itemType: "simvar", count: len(allVars), err: err}
}

func scrapeEvents(s *Scraper, pages []pageSpec, ver, outDir string, dryRun bool) scrapeResult {
	filename := fmt.Sprintf("events_%s.json", ver)
	var allEvents []corpus.Event

	for _, p := range pages {
		htmlBytes, err := s.FetchPage(p.url)
		if err != nil {
			log.Printf("events %s: fetch %s: %v", ver, p.url, err)
			continue
		}
		events, err := ParseEventPage(htmlBytes, ver, p.url)
		if err != nil {
			log.Printf("events %s: parse %s: %v", ver, p.url, err)
			continue
		}
		allEvents = append(allEvents, events...)
	}

	if len(allEvents) == 0 {
		return scrapeResult{filename: filename, itemType: "event", err: fmt.Errorf("no events scraped for version %s", ver)}
	}

	c := buildCorpus(ver)
	c.Events = allEvents

	err := writeCorpusFile(c, outDir, filename, dryRun)
	return scrapeResult{filename: filename, itemType: "event", count: len(allEvents), err: err}
}

func scrapeFunctions(s *Scraper, pages []string, outDir string, dryRun bool) scrapeResult {
	filename := "functions.json"
	var allFuncs []corpus.Function

	for _, u := range pages {
		htmlBytes, err := s.FetchPage(u)
		if err != nil {
			log.Printf("functions: fetch %s: %v", u, err)
			continue
		}
		fns, err := ParseFunctionPage(htmlBytes, "", u)
		if err != nil {
			log.Printf("functions: parse %s: %v", u, err)
			continue
		}
		allFuncs = append(allFuncs, fns...)
	}

	if len(allFuncs) == 0 {
		return scrapeResult{filename: filename, itemType: "function", err: fmt.Errorf("no functions scraped")}
	}

	c := buildCorpus("")
	c.Functions = allFuncs

	err := writeCorpusFile(c, outDir, filename, dryRun)
	return scrapeResult{filename: filename, itemType: "function", count: len(allFuncs), err: err}
}

func scrapeStructures(s *Scraper, pages []string, outDir string, dryRun bool) scrapeResult {
	filename := "structures.json"
	var allStructs []corpus.Structure

	for _, u := range pages {
		htmlBytes, err := s.FetchPage(u)
		if err != nil {
			log.Printf("structures: fetch %s: %v", u, err)
			continue
		}
		structs, err := ParseStructurePage(htmlBytes, "", u)
		if err != nil {
			log.Printf("structures: parse %s: %v", u, err)
			continue
		}
		allStructs = append(allStructs, structs...)
	}

	if len(allStructs) == 0 {
		return scrapeResult{filename: filename, itemType: "structure", err: fmt.Errorf("no structures scraped")}
	}

	c := buildCorpus("")
	c.Structures = allStructs

	err := writeCorpusFile(c, outDir, filename, dryRun)
	return scrapeResult{filename: filename, itemType: "structure", count: len(allStructs), err: err}
}

func scrapeErrorCodes(s *Scraper, pageURL, outDir string, dryRun bool) scrapeResult {
	filename := "error_codes.json"

	htmlBytes, err := s.FetchPage(pageURL)
	if err != nil {
		return scrapeResult{filename: filename, itemType: "errorcode", err: fmt.Errorf("fetch %s: %w", pageURL, err)}
	}

	codes, err := ParseErrorCodePage(htmlBytes, "", pageURL)
	if err != nil {
		return scrapeResult{filename: filename, itemType: "errorcode", err: err}
	}

	if len(codes) == 0 {
		return scrapeResult{filename: filename, itemType: "errorcode", err: fmt.Errorf("no error codes scraped")}
	}

	c := buildCorpus("")
	c.ErrorCodes = codes

	writeErr := writeCorpusFile(c, outDir, filename, dryRun)
	return scrapeResult{filename: filename, itemType: "errorcode", count: len(codes), err: writeErr}
}

// ── Output helpers ────────────────────────────────────────────────────────────

// buildCorpus creates a baseline Corpus with ScrapedAt set to now and
// SDKVersion set to ver (may be empty for version-agnostic types).
func buildCorpus(sdkVersion string) corpus.Corpus {
	return corpus.Corpus{
		ScrapedAt:  time.Now().UTC(),
		SDKVersion: sdkVersion,
		SimVars:    []corpus.SimVar{},
		Events:     []corpus.Event{},
		Functions:  []corpus.Function{},
		Structures: []corpus.Structure{},
		ErrorCodes: []corpus.ErrorCode{},
	}
}

// writeCorpusFile serialises c to JSON and writes it atomically to outDir/filename.
// When dryRun is true the file is not written and the function returns nil.
//
// Atomic write strategy:
//  1. Write to a temp file in the same directory.
//  2. Rename the temp file over the destination — this is atomic on POSIX and
//     best-effort on Windows (rename over existing is atomic on NTFS).
//
// If an existing file is present and the write fails, the existing file is
// preserved (the temp file is removed and the error is returned).
func writeCorpusFile(c corpus.Corpus, outDir, filename string, dryRun bool) error {
	if dryRun {
		return nil
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", filename, err)
	}

	dest := filepath.Join(outDir, filename)
	tmp := dest + ".tmp"

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write tmp for %s: %w", filename, err)
	}

	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, dest, err)
	}

	return nil
}

// repeatStr returns a string consisting of s repeated n times.
func repeatStr(s string, n int) string {
	result := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		result = append(result, s...)
	}
	return string(result)
}
