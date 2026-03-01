package corpus

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

// ── fixture helpers ───────────────────────────────────────────────────────────

// newBothStore loads the embedded corpus in "both" (merged) mode and returns
// a DocStore backed by it. Tests that depend on specific item counts use this.
func newBothStore(t *testing.T) DocStore {
	t.Helper()
	t.Setenv("DOCS_MSFS_VERSION", "both")

	c, err := LoadEmbedded().Load()
	if err != nil {
		t.Fatalf("LoadEmbedded().Load(): %v", err)
	}
	return NewDocStore(c)
}

// new2020Store loads the embedded corpus in "2020" mode.
func new2020Store(t *testing.T) DocStore {
	t.Helper()
	t.Setenv("DOCS_MSFS_VERSION", "2020")

	c, err := LoadEmbedded().Load()
	if err != nil {
		t.Fatalf("LoadEmbedded().Load(): %v", err)
	}
	return NewDocStore(c)
}

// background returns a non-nil context for test calls.
var background = context.Background()

// ── SimVarCount / EventCount ──────────────────────────────────────────────────

func TestDocStore_SimVarCount_Both(t *testing.T) {
	// "both" merges 2020 and 2024 unique simvar names (387+ shared, 400+ total)
	store := newBothStore(t)
	got := store.SimVarCount()
	if got < 100 {
		t.Errorf("SimVarCount() = %d, want >= 100", got)
	}
}

func TestDocStore_EventCount_Both(t *testing.T) {
	// "both" merges 2020 and 2024 unique event names
	store := newBothStore(t)
	got := store.EventCount()
	if got < 100 {
		t.Errorf("EventCount() = %d, want >= 100", got)
	}
}

func TestDocStore_SimVarCount_2020(t *testing.T) {
	store := new2020Store(t)
	got := store.SimVarCount()
	if got < 100 {
		t.Errorf("SimVarCount() = %d, want >= 100", got)
	}
}

func TestDocStore_EventCount_2020(t *testing.T) {
	store := new2020Store(t)
	got := store.EventCount()
	if got < 100 {
		t.Errorf("EventCount() = %d, want >= 100", got)
	}
}

// ── ListSimVars pagination ────────────────────────────────────────────────────

func TestDocStore_ListSimVars_Page1(t *testing.T) {
	store := newBothStore(t)

	page, err := store.ListSimVars(background, "", 1, 3)
	if err != nil {
		t.Fatalf("ListSimVars page 1: %v", err)
	}
	if page.Page != 1 {
		t.Errorf("Page = %d, want 1", page.Page)
	}
	if len(page.Items) != 3 {
		t.Errorf("len(Items) = %d, want 3", len(page.Items))
	}
	if page.TotalItems < 100 {
		t.Errorf("TotalItems = %d, want >= 100 (real corpus)", page.TotalItems)
	}
	if page.TotalPages < 10 {
		t.Errorf("TotalPages = %d, want >= 10 (ceil(TotalItems/3))", page.TotalPages)
	}
}

func TestDocStore_ListSimVars_Page2(t *testing.T) {
	store := newBothStore(t)

	page, err := store.ListSimVars(background, "", 2, 3)
	if err != nil {
		t.Fatalf("ListSimVars page 2: %v", err)
	}
	if page.Page != 2 {
		t.Errorf("Page = %d, want 2", page.Page)
	}
	if len(page.Items) != 3 {
		t.Errorf("len(Items) = %d, want 3", len(page.Items))
	}
}

func TestDocStore_ListSimVars_LastPage(t *testing.T) {
	store := newBothStore(t)

	// Determine the actual last page dynamically from TotalPages.
	first, err := store.ListSimVars(background, "", 1, 3)
	if err != nil {
		t.Fatalf("ListSimVars first page: %v", err)
	}
	if first.TotalPages == 0 {
		t.Fatal("expected at least one page of simvars")
	}

	page, err := store.ListSimVars(background, "", first.TotalPages, 3)
	if err != nil {
		t.Fatalf("ListSimVars last page: %v", err)
	}
	if len(page.Items) == 0 {
		t.Error("last page must have at least 1 item")
	}
	if len(page.Items) > 3 {
		t.Errorf("last page: len(Items) = %d, want <= 3", len(page.Items))
	}
}

func TestDocStore_ListSimVars_BeyondLastPage(t *testing.T) {
	store := newBothStore(t)

	page, err := store.ListSimVars(background, "", 100000, 10)
	if err != nil {
		t.Fatalf("ListSimVars beyond last page: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0 for out-of-bounds page", len(page.Items))
	}
	if page.TotalItems < 100 {
		t.Errorf("TotalItems = %d, want >= 100", page.TotalItems)
	}
}

func TestDocStore_ListSimVars_ZeroTotal(t *testing.T) {
	// Build a store with an empty corpus.
	store := NewDocStore(Corpus{})

	page, err := store.ListSimVars(background, "", 1, 20)
	if err != nil {
		t.Fatalf("ListSimVars zero total: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(page.Items))
	}
	if page.TotalItems != 0 {
		t.Errorf("TotalItems = %d, want 0", page.TotalItems)
	}
	if page.TotalPages != 0 {
		t.Errorf("TotalPages = %d, want 0", page.TotalPages)
	}
}

func TestDocStore_ListSimVars_CategoryFilter(t *testing.T) {
	store := newBothStore(t)

	// "AIRCRAFT AUTOPILOT/ASSISTANT VARIABLES" is the first scraped category.
	page, err := store.ListSimVars(background, "AIRCRAFT AUTOPILOT/ASSISTANT VARIABLES", 1, 20)
	if err != nil {
		t.Fatalf("ListSimVars category filter: %v", err)
	}
	if page.TotalItems < 1 {
		t.Errorf("expected at least 1 item in category, got %d", page.TotalItems)
	}
	for _, sv := range page.Items {
		if !strings.EqualFold(sv.Category, "AIRCRAFT AUTOPILOT/ASSISTANT VARIABLES") {
			t.Errorf("item %q has unexpected category %q", sv.Name, sv.Category)
		}
	}
}

func TestDocStore_ListSimVars_CategoryFilter_CaseInsensitive(t *testing.T) {
	store := newBothStore(t)

	upper, _ := store.ListSimVars(background, "AIRCRAFT AUTOPILOT/ASSISTANT VARIABLES", 1, 20)
	lower, _ := store.ListSimVars(background, "aircraft autopilot/assistant variables", 1, 20)

	if upper.TotalItems != lower.TotalItems {
		t.Errorf("category filter not case-insensitive: upper=%d lower=%d",
			upper.TotalItems, lower.TotalItems)
	}
}

func TestDocStore_ListSimVars_UnknownCategory_EmptyResult(t *testing.T) {
	store := newBothStore(t)

	page, err := store.ListSimVars(background, "NonExistentCategory", 1, 20)
	if err != nil {
		t.Fatalf("ListSimVars unknown category: %v", err)
	}
	if page.TotalItems != 0 {
		t.Errorf("TotalItems = %d, want 0 for unknown category", page.TotalItems)
	}
	if len(page.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0 for unknown category", len(page.Items))
	}
}

func TestDocStore_ListSimVars_DefaultPageSize(t *testing.T) {
	store := newBothStore(t)

	// pageSize 0 should default to 20
	page, err := store.ListSimVars(background, "", 1, 0)
	if err != nil {
		t.Fatalf("ListSimVars default pageSize: %v", err)
	}
	if page.PageSize != defaultPageSize {
		t.Errorf("PageSize = %d, want %d", page.PageSize, defaultPageSize)
	}
}

func TestDocStore_ListSimVars_MaxPageSizeClamped(t *testing.T) {
	store := newBothStore(t)

	page, err := store.ListSimVars(background, "", 1, 9999)
	if err != nil {
		t.Fatalf("ListSimVars max pageSize: %v", err)
	}
	if page.PageSize != maxPageSize {
		t.Errorf("PageSize = %d, want %d (clamped)", page.PageSize, maxPageSize)
	}
}

// ── GetSimVar ─────────────────────────────────────────────────────────────────

func TestDocStore_GetSimVar_Found(t *testing.T) {
	store := newBothStore(t)

	sv, err := store.GetSimVar(background, "AUTOPILOT AIRSPEED ACQUISITION")
	if err != nil {
		t.Fatalf("GetSimVar found: %v", err)
	}
	if sv.Name != "AUTOPILOT AIRSPEED ACQUISITION" {
		t.Errorf("Name = %q, want %q", sv.Name, "AUTOPILOT AIRSPEED ACQUISITION")
	}
}

func TestDocStore_GetSimVar_CaseInsensitive(t *testing.T) {
	store := newBothStore(t)

	sv, err := store.GetSimVar(background, "autopilot airspeed acquisition")
	if err != nil {
		t.Fatalf("GetSimVar case-insensitive: %v", err)
	}
	if sv.Name != "AUTOPILOT AIRSPEED ACQUISITION" {
		t.Errorf("Name = %q, want %q", sv.Name, "AUTOPILOT AIRSPEED ACQUISITION")
	}
}

func TestDocStore_GetSimVar_NotFound(t *testing.T) {
	store := newBothStore(t)

	_, err := store.GetSimVar(background, "NONEXISTENT VARIABLE")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetSimVar not found: got %v, want ErrNotFound", err)
	}
}

// ── ListEvents pagination ─────────────────────────────────────────────────────

func TestDocStore_ListEvents_Page1(t *testing.T) {
	store := newBothStore(t)

	page, err := store.ListEvents(background, 1, 5)
	if err != nil {
		t.Fatalf("ListEvents page 1: %v", err)
	}
	if page.TotalItems < 100 {
		t.Errorf("TotalItems = %d, want >= 100", page.TotalItems)
	}
	if len(page.Items) != 5 {
		t.Errorf("len(Items) = %d, want 5", len(page.Items))
	}
}

func TestDocStore_ListEvents_BeyondLastPage(t *testing.T) {
	store := newBothStore(t)

	page, err := store.ListEvents(background, 999, 20)
	if err != nil {
		t.Fatalf("ListEvents beyond last page: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(page.Items))
	}
}

// ── GetEvent ──────────────────────────────────────────────────────────────────

func TestDocStore_GetEvent_Found(t *testing.T) {
	store := newBothStore(t)

	ev, err := store.GetEvent(background, "AP_AIRSPEED_HOLD")
	if err != nil {
		t.Fatalf("GetEvent found: %v", err)
	}
	if ev.Name != "AP_AIRSPEED_HOLD" {
		t.Errorf("Name = %q, want %q", ev.Name, "AP_AIRSPEED_HOLD")
	}
}

func TestDocStore_GetEvent_NotFound(t *testing.T) {
	store := newBothStore(t)

	_, err := store.GetEvent(background, "NONEXISTENT_EVENT")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetEvent not found: got %v, want ErrNotFound", err)
	}
}

// ── ListFunctions ─────────────────────────────────────────────────────────────

func TestDocStore_ListFunctions_Page1(t *testing.T) {
	store := newBothStore(t)

	page, err := store.ListFunctions(background, 1, 20)
	if err != nil {
		t.Fatalf("ListFunctions page 1: %v", err)
	}
	// real corpus has 7 functions scraped
	if page.TotalItems < 3 {
		t.Errorf("TotalItems = %d, want >= 3", page.TotalItems)
	}
	if len(page.Items) < 3 {
		t.Errorf("len(Items) = %d, want >= 3", len(page.Items))
	}
}

func TestDocStore_ListFunctions_BeyondLastPage(t *testing.T) {
	store := newBothStore(t)

	page, err := store.ListFunctions(background, 999, 20)
	if err != nil {
		t.Fatalf("ListFunctions beyond last page: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(page.Items))
	}
}

// ── GetFunction ───────────────────────────────────────────────────────────────

func TestDocStore_GetFunction_Found(t *testing.T) {
	store := newBothStore(t)

	fn, err := store.GetFunction(background, "SimConnect_Open")
	if err != nil {
		t.Fatalf("GetFunction found: %v", err)
	}
	if fn.Name != "SimConnect_Open" {
		t.Errorf("Name = %q, want %q", fn.Name, "SimConnect_Open")
	}
}

func TestDocStore_GetFunction_NotFound(t *testing.T) {
	store := newBothStore(t)

	_, err := store.GetFunction(background, "SimConnect_Nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetFunction not found: got %v, want ErrNotFound", err)
	}
}

// ── ListStructures ────────────────────────────────────────────────────────────

func TestDocStore_ListStructures_Page1(t *testing.T) {
	store := newBothStore(t)

	page, err := store.ListStructures(background, 1, 20)
	if err != nil {
		t.Fatalf("ListStructures page 1: %v", err)
	}
	// real corpus has 5 structures scraped
	if page.TotalItems < 2 {
		t.Errorf("TotalItems = %d, want >= 2", page.TotalItems)
	}
}

func TestDocStore_ListStructures_BeyondLastPage(t *testing.T) {
	store := newBothStore(t)

	page, err := store.ListStructures(background, 999, 20)
	if err != nil {
		t.Fatalf("ListStructures beyond last page: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(page.Items))
	}
}

// ── GetStructure ──────────────────────────────────────────────────────────────

func TestDocStore_GetStructure_Found(t *testing.T) {
	store := newBothStore(t)

	st, err := store.GetStructure(background, "SIMCONNECT_RECV")
	if err != nil {
		t.Fatalf("GetStructure found: %v", err)
	}
	if st.Name != "SIMCONNECT_RECV" {
		t.Errorf("Name = %q, want %q", st.Name, "SIMCONNECT_RECV")
	}
}

func TestDocStore_GetStructure_NotFound(t *testing.T) {
	store := newBothStore(t)

	_, err := store.GetStructure(background, "NONEXISTENT_STRUCT")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetStructure not found: got %v, want ErrNotFound", err)
	}
}

// ── ListErrorCodes ────────────────────────────────────────────────────────────

func TestDocStore_ListErrorCodes_Page1(t *testing.T) {
	store := newBothStore(t)

	page, err := store.ListErrorCodes(background, 1, 20)
	if err != nil {
		t.Fatalf("ListErrorCodes page 1: %v", err)
	}
	// real corpus has 44 error codes scraped
	if page.TotalItems < 10 {
		t.Errorf("TotalItems = %d, want >= 10", page.TotalItems)
	}
}

func TestDocStore_ListErrorCodes_BeyondLastPage(t *testing.T) {
	store := newBothStore(t)

	page, err := store.ListErrorCodes(background, 999, 20)
	if err != nil {
		t.Fatalf("ListErrorCodes beyond last page: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(page.Items))
	}
}

// ── GetErrorCode ──────────────────────────────────────────────────────────────

func TestDocStore_GetErrorCode_Found(t *testing.T) {
	store := newBothStore(t)

	ec, err := store.GetErrorCode(background, "SIMCONNECT_EXCEPTION_NONE")
	if err != nil {
		t.Fatalf("GetErrorCode found: %v", err)
	}
	if ec.Name != "SIMCONNECT_EXCEPTION_NONE" {
		t.Errorf("Name = %q, want %q", ec.Name, "SIMCONNECT_EXCEPTION_NONE")
	}
	if ec.Value != 0 {
		t.Errorf("Value = %d, want 0", ec.Value)
	}
}

func TestDocStore_GetErrorCode_NotFound(t *testing.T) {
	store := newBothStore(t)

	_, err := store.GetErrorCode(background, "SIMCONNECT_EXCEPTION_NONEXISTENT")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetErrorCode not found: got %v, want ErrNotFound", err)
	}
}

// ── Search ────────────────────────────────────────────────────────────────────

func TestDocStore_Search_EmptyQuery(t *testing.T) {
	store := newBothStore(t)

	res, err := store.Search(background, "", "all", 20)
	if err != nil {
		t.Fatalf("Search empty query: %v", err)
	}
	if res.Total != 0 {
		t.Errorf("Total = %d, want 0 for empty query", res.Total)
	}
	if len(res.Results) != 0 {
		t.Errorf("len(Results) = %d, want 0 for empty query", len(res.Results))
	}
}

func TestDocStore_Search_TypeSimVar(t *testing.T) {
	store := newBothStore(t)

	// "altitude" matches autopilot altitude vars (AUTOPILOT ALTITUDE LOCK VAR, etc.)
	res, err := store.Search(background, "altitude", "simvar", 20)
	if err != nil {
		t.Fatalf("Search type=simvar: %v", err)
	}
	if res.Total == 0 {
		t.Error("expected at least one simvar result for 'altitude'")
	}
	for _, r := range res.Results {
		if r.Type != "simvar" {
			t.Errorf("result type = %q, want simvar", r.Type)
		}
	}
}

func TestDocStore_Search_TypeEvent(t *testing.T) {
	store := newBothStore(t)

	// "airspeed" matches AP_AIRSPEED_HOLD, AP_AIRSPEED_ON, AP_AIRSPEED_OFF, etc.
	res, err := store.Search(background, "airspeed", "event", 20)
	if err != nil {
		t.Fatalf("Search type=event: %v", err)
	}
	if res.Total == 0 {
		t.Error("expected at least one event result for 'airspeed'")
	}
	for _, r := range res.Results {
		if r.Type != "event" {
			t.Errorf("result type = %q, want event", r.Type)
		}
	}
}

func TestDocStore_Search_TypeAll_MatchesMultipleCategories(t *testing.T) {
	store := newBothStore(t)

	// "simconnect" appears in function names, structure names, and error code names.
	// Use a high limit (200) because all ~52 functions match "simconnect" and the
	// search processes types in order — error_codes only appear after functions.
	res, err := store.Search(background, "simconnect", "all", 200)
	if err != nil {
		t.Fatalf("Search type=all: %v", err)
	}

	typesSeen := make(map[string]bool)
	for _, r := range res.Results {
		typesSeen[r.Type] = true
	}

	// Should find at least functions and error_codes
	if !typesSeen["function"] {
		t.Error("expected function results for 'simconnect'")
	}
	if !typesSeen["error_code"] {
		t.Error("expected error_code results for 'simconnect'")
	}
}

func TestDocStore_Search_CaseInsensitive(t *testing.T) {
	store := newBothStore(t)

	upper, err := store.Search(background, "ALTITUDE", "simvar", 20)
	if err != nil {
		t.Fatalf("Search ALTITUDE: %v", err)
	}
	lower, err := store.Search(background, "altitude", "simvar", 20)
	if err != nil {
		t.Fatalf("Search altitude: %v", err)
	}

	if upper.Total != lower.Total {
		t.Errorf("case-insensitive mismatch: ALTITUDE=%d altitude=%d", upper.Total, lower.Total)
	}
}

func TestDocStore_Search_SnippetMaxLen(t *testing.T) {
	store := newBothStore(t)

	res, err := store.Search(background, "altitude", "all", 20)
	if err != nil {
		t.Fatalf("Search snippet length: %v", err)
	}
	for _, r := range res.Results {
		if len(r.Excerpt) > snippetLen {
			t.Errorf("result %q excerpt length %d exceeds %d", r.Name, len(r.Excerpt), snippetLen)
		}
	}
}

func TestDocStore_Search_LimitRespected(t *testing.T) {
	store := newBothStore(t)

	// "the" is common enough to appear in all descriptions
	res, err := store.Search(background, "the", "all", 3)
	if err != nil {
		t.Fatalf("Search limit: %v", err)
	}
	if res.Total > 3 {
		t.Errorf("Total = %d exceeds limit 3", res.Total)
	}
	if len(res.Results) > 3 {
		t.Errorf("len(Results) = %d exceeds limit 3", len(res.Results))
	}
}

func TestDocStore_Search_NoMatch_EmptyResults(t *testing.T) {
	store := newBothStore(t)

	res, err := store.Search(background, "xyzzy_no_such_thing", "all", 20)
	if err != nil {
		t.Fatalf("Search no match: %v", err)
	}
	if res.Total != 0 {
		t.Errorf("Total = %d, want 0 for unmatched query", res.Total)
	}
}

func TestDocStore_Search_SourceURLPresent(t *testing.T) {
	store := newBothStore(t)

	res, err := store.Search(background, "altitude", "simvar", 20)
	if err != nil {
		t.Fatalf("Search source URL: %v", err)
	}
	for _, r := range res.Results {
		if r.SourceURL == "" {
			t.Errorf("result %q has empty SourceURL", r.Name)
		}
	}
}

// ── Concurrent reads ──────────────────────────────────────────────────────────

func TestDocStore_ConcurrentReads_NoRace(t *testing.T) {
	store := newBothStore(t)

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			// Mix of List and Get operations using real scraped data names.
			_, _ = store.ListSimVars(background, "", 1, 5)
			_, _ = store.GetSimVar(background, "AUTOPILOT AIRSPEED ACQUISITION")
			_, _ = store.ListEvents(background, 1, 5)
			_, _ = store.GetEvent(background, "AP_AIRSPEED_HOLD")
			_, _ = store.ListFunctions(background, 1, 10)
			_, _ = store.GetFunction(background, "SimConnect_Open")
			_, _ = store.ListStructures(background, 1, 10)
			_, _ = store.GetStructure(background, "SIMCONNECT_RECV")
			_, _ = store.ListErrorCodes(background, 1, 10)
			_, _ = store.GetErrorCode(background, "SIMCONNECT_EXCEPTION_NONE")
			_, _ = store.Search(background, "altitude", "all", 10)
			_ = store.SimVarCount()
			_ = store.EventCount()
		}()
	}

	wg.Wait()
}

// ── paginate helper ───────────────────────────────────────────────────────────

func TestPaginate_FirstPage(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	p := paginate(items, 1, 2)
	if p.TotalItems != 5 {
		t.Errorf("TotalItems = %d, want 5", p.TotalItems)
	}
	if p.TotalPages != 3 {
		t.Errorf("TotalPages = %d, want 3", p.TotalPages)
	}
	if len(p.Items) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(p.Items))
	}
	if p.Items[0] != 1 || p.Items[1] != 2 {
		t.Errorf("Items = %v, want [1 2]", p.Items)
	}
}

func TestPaginate_LastPagePartial(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	p := paginate(items, 3, 2)
	if len(p.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(p.Items))
	}
	if p.Items[0] != 5 {
		t.Errorf("Items[0] = %d, want 5", p.Items[0])
	}
}

func TestPaginate_BeyondLastPage(t *testing.T) {
	items := []int{1, 2, 3}

	p := paginate(items, 99, 2)
	if len(p.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(p.Items))
	}
}

func TestPaginate_EmptySlice(t *testing.T) {
	p := paginate([]int{}, 1, 20)
	if p.TotalItems != 0 || p.TotalPages != 0 || len(p.Items) != 0 {
		t.Errorf("paginate empty: got TotalItems=%d TotalPages=%d len=%d",
			p.TotalItems, p.TotalPages, len(p.Items))
	}
}

func TestPaginate_PageZero_TreatedAsOutOfBounds(t *testing.T) {
	items := []int{1, 2, 3}

	p := paginate(items, 0, 20)
	if len(p.Items) != 0 {
		t.Errorf("page 0: len(Items) = %d, want 0", len(p.Items))
	}
}

// ── excerpt helper ────────────────────────────────────────────────────────────

func TestExcerpt_ShorterThanLimit(t *testing.T) {
	s := "short"
	got := excerpt(s, 200)
	if got != s {
		t.Errorf("excerpt = %q, want %q", got, s)
	}
}

func TestExcerpt_ExactlyLimit(t *testing.T) {
	s := strings.Repeat("a", 200)
	got := excerpt(s, 200)
	if got != s {
		t.Errorf("excerpt length = %d, want 200", len(got))
	}
}

func TestExcerpt_LongerThanLimit(t *testing.T) {
	s := strings.Repeat("a", 300)
	got := excerpt(s, 200)
	if len(got) != 200 {
		t.Errorf("excerpt length = %d, want 200", len(got))
	}
}
