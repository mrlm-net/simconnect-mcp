// Package corpus provides the DocStore interface and its in-memory
// implementation backed by the corpus types defined in types.go.
//
// The in-memory store is built once from a Corpus value at construction time
// and is read-only at runtime. All query methods are safe for concurrent use.
package corpus

import (
	"context"
	"strings"
	"sync"
)

// DocStore defines the read interface for all SimConnect SDK documentation.
// All List* methods return a Page[T] envelope. All Get* methods return
// ErrNotFound when no item matches. Search returns an empty SearchResults
// (not an error) when the query string is empty or there are no matches.
//
// Implementations must be safe for concurrent read access.
type DocStore interface {
	ListSimVars(ctx context.Context, category string, page, pageSize int) (Page[SimVar], error)
	GetSimVar(ctx context.Context, name string) (SimVar, error)
	ListEvents(ctx context.Context, page, pageSize int) (Page[Event], error)
	GetEvent(ctx context.Context, name string) (Event, error)
	ListFunctions(ctx context.Context, page, pageSize int) (Page[Function], error)
	GetFunction(ctx context.Context, name string) (Function, error)
	ListStructures(ctx context.Context, page, pageSize int) (Page[Structure], error)
	GetStructure(ctx context.Context, name string) (Structure, error)
	ListErrorCodes(ctx context.Context, page, pageSize int) (Page[ErrorCode], error)
	GetErrorCode(ctx context.Context, name string) (ErrorCode, error)
	Search(ctx context.Context, query, type_ string, limit int) (SearchResults, error)
	SimVarCount() int
	EventCount() int
}

// ── constants ─────────────────────────────────────────────────────────────────

const (
	defaultPageSize = 20
	maxPageSize     = 100
	snippetLen      = 200
)

// ── memStore ──────────────────────────────────────────────────────────────────

// memStore is the in-memory DocStore implementation.
// All slices preserve corpus insertion order for stable pagination.
// Map keys are normalised to strings.ToUpper(name) for O(1) lookup.
type memStore struct {
	mu sync.RWMutex

	// ordered slices for pagination
	simVars    []SimVar
	events     []Event
	functions  []Function
	structures []Structure
	errorCodes []ErrorCode

	// O(1) lookup maps; keys are strings.ToUpper(name)
	simVarByName    map[string]SimVar
	eventByName     map[string]Event
	functionByName  map[string]Function
	structureByName map[string]Structure
	errorCodeByName map[string]ErrorCode

	// category index: strings.ToUpper(category) → []SimVar (preserving order)
	simVarsByCategory map[string][]SimVar
}

// NewDocStore constructs an in-memory DocStore from the provided Corpus.
// All internal indexes are built at construction time; subsequent calls are
// read-only and safe for concurrent access.
func NewDocStore(c Corpus) DocStore {
	s := &memStore{
		simVars:           make([]SimVar, len(c.SimVars)),
		events:            make([]Event, len(c.Events)),
		functions:         make([]Function, len(c.Functions)),
		structures:        make([]Structure, len(c.Structures)),
		errorCodes:        make([]ErrorCode, len(c.ErrorCodes)),
		simVarByName:      make(map[string]SimVar, len(c.SimVars)),
		eventByName:       make(map[string]Event, len(c.Events)),
		functionByName:    make(map[string]Function, len(c.Functions)),
		structureByName:   make(map[string]Structure, len(c.Structures)),
		errorCodeByName:   make(map[string]ErrorCode, len(c.ErrorCodes)),
		simVarsByCategory: make(map[string][]SimVar),
	}

	copy(s.simVars, c.SimVars)
	copy(s.events, c.Events)
	copy(s.functions, c.Functions)
	copy(s.structures, c.Structures)
	copy(s.errorCodes, c.ErrorCodes)

	for _, sv := range c.SimVars {
		key := strings.ToUpper(sv.Name)
		s.simVarByName[key] = sv

		catKey := strings.ToUpper(sv.Category)
		s.simVarsByCategory[catKey] = append(s.simVarsByCategory[catKey], sv)
	}
	for _, ev := range c.Events {
		s.eventByName[strings.ToUpper(ev.Name)] = ev
	}
	for _, fn := range c.Functions {
		s.functionByName[strings.ToUpper(fn.Name)] = fn
	}
	for _, st := range c.Structures {
		s.structureByName[strings.ToUpper(st.Name)] = st
	}
	for _, ec := range c.ErrorCodes {
		s.errorCodeByName[strings.ToUpper(ec.Name)] = ec
	}

	return s
}

// ── helpers ───────────────────────────────────────────────────────────────────

// clampPageSize enforces min 1, max maxPageSize, default defaultPageSize.
func clampPageSize(ps int) int {
	if ps <= 0 {
		return defaultPageSize
	}
	if ps > maxPageSize {
		return maxPageSize
	}
	return ps
}

// paginate slices items for the requested 1-indexed page.
// Returns an empty slice for an out-of-bounds page (not an error).
func paginate[T any](items []T, page, pageSize int) Page[T] {
	total := len(items)
	ps := clampPageSize(pageSize)

	totalPages := 0
	if total > 0 {
		totalPages = (total + ps - 1) / ps
	}

	if page < 1 || total == 0 {
		return Page[T]{
			Items:      []T{},
			Page:       page,
			PageSize:   ps,
			TotalItems: total,
			TotalPages: totalPages,
		}
	}

	start := (page - 1) * ps
	if start >= total {
		return Page[T]{
			Items:      []T{},
			Page:       page,
			PageSize:   ps,
			TotalItems: total,
			TotalPages: totalPages,
		}
	}

	end := start + ps
	if end > total {
		end = total
	}

	return Page[T]{
		Items:      items[start:end],
		Page:       page,
		PageSize:   ps,
		TotalItems: total,
		TotalPages: totalPages,
	}
}

// excerpt returns the first n characters of s, trimming trailing whitespace.
func excerpt(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.TrimRight(s[:n], " \t")
}

// ── SimVar ────────────────────────────────────────────────────────────────────

func (s *memStore) ListSimVars(_ context.Context, category string, page, pageSize int) (Page[SimVar], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if category == "" {
		return paginate(s.simVars, page, pageSize), nil
	}

	catKey := strings.ToUpper(category)
	filtered := s.simVarsByCategory[catKey] // nil if not found → paginate handles zero len
	return paginate(filtered, page, pageSize), nil
}

func (s *memStore) GetSimVar(_ context.Context, name string) (SimVar, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sv, ok := s.simVarByName[strings.ToUpper(name)]
	if !ok {
		return SimVar{}, ErrNotFound
	}
	return sv, nil
}

// ── Event ─────────────────────────────────────────────────────────────────────

func (s *memStore) ListEvents(_ context.Context, page, pageSize int) (Page[Event], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return paginate(s.events, page, pageSize), nil
}

func (s *memStore) GetEvent(_ context.Context, name string) (Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ev, ok := s.eventByName[strings.ToUpper(name)]
	if !ok {
		return Event{}, ErrNotFound
	}
	return ev, nil
}

// ── Function ──────────────────────────────────────────────────────────────────

func (s *memStore) ListFunctions(_ context.Context, page, pageSize int) (Page[Function], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return paginate(s.functions, page, pageSize), nil
}

func (s *memStore) GetFunction(_ context.Context, name string) (Function, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fn, ok := s.functionByName[strings.ToUpper(name)]
	if !ok {
		return Function{}, ErrNotFound
	}
	return fn, nil
}

// ── Structure ─────────────────────────────────────────────────────────────────

func (s *memStore) ListStructures(_ context.Context, page, pageSize int) (Page[Structure], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return paginate(s.structures, page, pageSize), nil
}

func (s *memStore) GetStructure(_ context.Context, name string) (Structure, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st, ok := s.structureByName[strings.ToUpper(name)]
	if !ok {
		return Structure{}, ErrNotFound
	}
	return st, nil
}

// ── ErrorCode ─────────────────────────────────────────────────────────────────

func (s *memStore) ListErrorCodes(_ context.Context, page, pageSize int) (Page[ErrorCode], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return paginate(s.errorCodes, page, pageSize), nil
}

func (s *memStore) GetErrorCode(_ context.Context, name string) (ErrorCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ec, ok := s.errorCodeByName[strings.ToUpper(name)]
	if !ok {
		return ErrorCode{}, ErrNotFound
	}
	return ec, nil
}

// ── Search ────────────────────────────────────────────────────────────────────

// Search performs a case-insensitive substring match on Name and Description
// across all corpus categories (or a specific one when type_ is set).
// An empty query returns SearchResults{Total: 0} without error.
// type_ valid values: "simvar", "event", "function", "structure", "error_code", "all" (or "").
// Results are capped at limit (0 means defaultPageSize).
func (s *memStore) Search(_ context.Context, query, type_ string, limit int) (SearchResults, error) {
	if query == "" {
		return SearchResults{Query: query, Total: 0, Results: []SearchResult{}}, nil
	}

	if limit <= 0 {
		limit = defaultPageSize
	}

	q := strings.ToLower(query)

	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []SearchResult

	includeAll := type_ == "" || type_ == "all"

	if includeAll || type_ == "simvar" {
		for _, sv := range s.simVars {
			if len(results) >= limit {
				break
			}
			if matchItem(q, sv.Name, sv.Description) {
				results = append(results, SearchResult{
					Type:      "simvar",
					Name:      sv.Name,
					Excerpt:   excerpt(sv.Description, snippetLen),
					SourceURL: sv.SourceURL,
				})
			}
		}
	}

	if (includeAll || type_ == "event") && len(results) < limit {
		for _, ev := range s.events {
			if len(results) >= limit {
				break
			}
			if matchItem(q, ev.Name, ev.Description) {
				results = append(results, SearchResult{
					Type:      "event",
					Name:      ev.Name,
					Excerpt:   excerpt(ev.Description, snippetLen),
					SourceURL: ev.SourceURL,
				})
			}
		}
	}

	if (includeAll || type_ == "function") && len(results) < limit {
		for _, fn := range s.functions {
			if len(results) >= limit {
				break
			}
			if matchItem(q, fn.Name, fn.Description) {
				results = append(results, SearchResult{
					Type:      "function",
					Name:      fn.Name,
					Excerpt:   excerpt(fn.Description, snippetLen),
					SourceURL: fn.SourceURL,
				})
			}
		}
	}

	if (includeAll || type_ == "structure") && len(results) < limit {
		for _, st := range s.structures {
			if len(results) >= limit {
				break
			}
			if matchItem(q, st.Name, st.Description) {
				results = append(results, SearchResult{
					Type:      "structure",
					Name:      st.Name,
					Excerpt:   excerpt(st.Description, snippetLen),
					SourceURL: st.SourceURL,
				})
			}
		}
	}

	if (includeAll || type_ == "error_code") && len(results) < limit {
		for _, ec := range s.errorCodes {
			if len(results) >= limit {
				break
			}
			if matchItem(q, ec.Name, ec.Description) {
				results = append(results, SearchResult{
					Type:      "error_code",
					Name:      ec.Name,
					Excerpt:   excerpt(ec.Description, snippetLen),
					SourceURL: ec.SourceURL,
				})
			}
		}
	}

	if results == nil {
		results = []SearchResult{}
	}

	return SearchResults{
		Query:   query,
		Total:   len(results),
		Results: results,
	}, nil
}

// matchItem returns true when query q (already lowercased) appears in
// the lowercased name or description.
func matchItem(q, name, description string) bool {
	return strings.Contains(strings.ToLower(name), q) ||
		strings.Contains(strings.ToLower(description), q)
}

// ── Counts ────────────────────────────────────────────────────────────────────

func (s *memStore) SimVarCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.simVars)
}

func (s *memStore) EventCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}
