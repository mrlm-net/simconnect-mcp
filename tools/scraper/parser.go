package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/mrlm-net/simconnect-mcp/internal/corpus"
	"golang.org/x/net/html"
)

// ── Public parser functions ───────────────────────────────────────────────────
//
// Each ParseXxxPage function accepts raw HTML bytes along with the SDK version
// string and the source URL (as required by ADR-006 for per-item attribution).
// Malformed table rows are skipped with a log warning; partial results are
// returned alongside any non-nil error only in catastrophic cases.

// ParseSimVarPage parses a SimVar category page from docs.flightsimulator.com.
// The page is expected to contain an HTML table with at least the columns:
// "Variable name", "Description", "Units", and "Settable".
//
// Indexed variables are identified by a ":index" suffix in the name column.
// Deprecated variables are identified by a <span class="deprecated"> element
// in the description cell.
func ParseSimVarPage(htmlBytes []byte, sdkVersion, sourceURL string) ([]corpus.SimVar, error) {
	doc, err := html.Parse(strings.NewReader(string(htmlBytes)))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	category := extractH1(doc)
	rows := extractTableRows(doc)

	var simvars []corpus.SimVar

	// rows[0] is the header row — skip it.
	for i, row := range rows {
		if i == 0 {
			continue
		}
		cells := extractCells(row)
		if len(cells) < 3 {
			log.Printf("simvar row %d: expected >=3 cells, got %d — skipping", i, len(cells))
			continue
		}

		name := cleanText(cellText(cells[0]))
		if name == "" {
			log.Printf("simvar row %d: empty name — skipping", i)
			continue
		}

		descCell := cells[1]
		description := cleanText(cellText(descCell))
		deprecated, deprecatedReason := extractDeprecated(descCell, description)

		units := parseUnitList(cleanText(cellText(cells[2])))

		settable := false
		if len(cells) >= 4 {
			settable = strings.EqualFold(strings.TrimSpace(cellText(cells[3])), "yes")
		}

		// Indexed variables have ":index" in the name.
		indexedBy := ""
		if strings.Contains(strings.ToLower(name), ":index") {
			name = strings.Replace(name, ":index", "", 1)
			name = strings.TrimRight(name, " ")
			indexedBy = "1-indexed"
		}

		sv := corpus.SimVar{
			Name:             name,
			Description:      description,
			Units:            units,
			Settable:         settable,
			Category:         category,
			Versions:         []string{sdkVersion},
			IndexedBy:        indexedBy,
			Deprecated:       deprecated,
			DeprecatedReason: deprecatedReason,
			SourceURL:        sourceURL,
		}
		simvars = append(simvars, sv)
	}

	return simvars, nil
}

// ParseGPSVarPage parses a GPS variable subpage (Airports.htm, Intersections.htm,
// NDB_VOR.htm, Flightplans.htm, Miscellaneous.htm) from docs.flightsimulator.com.
//
// GPS pages use a 3-column format — "GPS Variable", "Units", "Settable" — with no
// Description column. Each page may contain multiple tables (one per subsection).
//
// pageDeprecated and pageDeprecatedReason are applied to every variable when set.
// In MSFS 2024 all GPS variables carry a page-level deprecation notice.
func ParseGPSVarPage(htmlBytes []byte, sdkVersion, sourceURL string, pageDeprecated bool, pageDeprecatedReason string) ([]corpus.SimVar, error) {
	doc, err := html.Parse(strings.NewReader(string(htmlBytes)))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	category := extractH1(doc)
	tables := findAllTables(doc)

	var simvars []corpus.SimVar

	for _, table := range tables {
		rows := rowsFromTable(table)
		for i, row := range rows {
			if i == 0 {
				continue // skip header row
			}
			cells := extractCells(row)
			if len(cells) < 2 {
				log.Printf("gps simvar row %d: expected >=2 cells, got %d — skipping", i, len(cells))
				continue
			}

			name := cleanText(cellText(cells[0]))
			if name == "" {
				log.Printf("gps simvar row %d: empty name — skipping", i)
				continue
			}

			units := parseUnitList(cleanText(cellText(cells[1])))

			settable := false
			if len(cells) >= 3 {
				settable = strings.EqualFold(strings.TrimSpace(cellText(cells[2])), "yes")
			}

			sv := corpus.SimVar{
				Name:             name,
				Units:            units,
				Settable:         settable,
				Category:         category,
				Versions:         []string{sdkVersion},
				Deprecated:       pageDeprecated,
				DeprecatedReason: pageDeprecatedReason,
				SourceURL:        sourceURL,
			}
			simvars = append(simvars, sv)
		}
	}

	return simvars, nil
}

// ParseEventPage parses an Event (Key Event ID) page from docs.flightsimulator.com.
// The page is expected to contain an HTML table with at least the columns:
// "Event name" and "Description".
func ParseEventPage(htmlBytes []byte, sdkVersion, sourceURL string) ([]corpus.Event, error) {
	doc, err := html.Parse(strings.NewReader(string(htmlBytes)))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	rows := extractTableRows(doc)

	var events []corpus.Event

	for i, row := range rows {
		if i == 0 {
			continue
		}
		cells := extractCells(row)
		if len(cells) < 2 {
			log.Printf("event row %d: expected >=2 cells, got %d — skipping", i, len(cells))
			continue
		}

		name := cleanText(cellText(cells[0]))
		if name == "" {
			log.Printf("event row %d: empty name — skipping", i)
			continue
		}

		descCell := cells[1]
		description := cleanText(cellText(descCell))
		deprecated, deprecatedReason := extractDeprecated(descCell, description)

		ev := corpus.Event{
			Name:             name,
			Description:      description,
			Versions:         []string{sdkVersion},
			Deprecated:       deprecated,
			DeprecatedReason: deprecatedReason,
			SourceURL:        sourceURL,
		}
		events = append(events, ev)
	}

	return events, nil
}

// ParseFunctionPage parses a single API function reference page from
// docs.flightsimulator.com. The page structure is narrative — it contains:
//   - An <h1> with the function name.
//   - A <pre><code> block with the C signature.
//   - A parameters table with columns: "Parameter", "Type", "Description".
//   - A return value paragraph.
//   - An optional remarks paragraph.
func ParseFunctionPage(htmlBytes []byte, sdkVersion, sourceURL string) ([]corpus.Function, error) {
	doc, err := html.Parse(strings.NewReader(string(htmlBytes)))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	name := extractH1(doc)
	if name == "" {
		return nil, fmt.Errorf("no <h1> found in function page from %s", sourceURL)
	}

	description := extractParagraphAfterH1(doc)
	signature := extractPreCode(doc)
	returnType := extractReturnType(doc)
	remarks := extractRemarks(doc)
	params := extractFunctionParams(doc)

	fn := corpus.Function{
		Name:        name,
		Description: description,
		Signature:   signature,
		Parameters:  params,
		ReturnType:  returnType,
		Remarks:     remarks,
		SourceURL:   sourceURL,
	}

	return []corpus.Function{fn}, nil
}

// ParseStructurePage parses a single C structure definition page from
// docs.flightsimulator.com. The page structure is narrative — it contains:
//   - An <h1> with the structure name.
//   - A description paragraph.
//   - A members table with columns: "Member", "Type", "Description".
//   - An optional remarks paragraph.
//
// Some docs pages have a heading that references a different item (docs bug).
// In that case the URL filename stem is used as the authoritative name.
func ParseStructurePage(htmlBytes []byte, sdkVersion, sourceURL string) ([]corpus.Structure, error) {
	doc, err := html.Parse(strings.NewReader(string(htmlBytes)))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	name := extractH1(doc)
	urlName := urlFileStem(sourceURL)
	switch {
	case name == "" && urlName == "":
		return nil, fmt.Errorf("no <h1> found in structure page from %s", sourceURL)
	case name == "":
		name = urlName
	case urlName != "" && !strings.EqualFold(name, urlName) &&
		strings.HasPrefix(strings.ToUpper(urlName), "SIMCONNECT_"):
		// URL stem looks like a SimConnect API name but heading disagrees —
		// docs website bug (e.g. SIMCONNECT_STATE.htm whose h2 says
		// SIMCONNECT_SIMOBJECT_TYPE). Use the URL name as ground truth.
		log.Printf("structure page %s: heading %q differs from URL name %q — using URL name", sourceURL, name, urlName)
		name = urlName
	}

	description := extractParagraphAfterH1(doc)
	remarks := extractRemarks(doc)
	fields := extractStructFields(doc)

	st := corpus.Structure{
		Name:        name,
		Description: description,
		Fields:      fields,
		Remarks:     remarks,
		SourceURL:   sourceURL,
	}

	return []corpus.Structure{st}, nil
}

// ParseErrorCodePage parses the SIMCONNECT_EXCEPTION enumeration page from
// docs.flightsimulator.com. The page is expected to contain an HTML table with
// at least the columns: "Name", "Value", "Description".
func ParseErrorCodePage(htmlBytes []byte, sdkVersion, sourceURL string) ([]corpus.ErrorCode, error) {
	doc, err := html.Parse(strings.NewReader(string(htmlBytes)))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	rows := extractTableRows(doc)

	var codes []corpus.ErrorCode

	valueCounter := 0
	for i, row := range rows {
		if i == 0 {
			continue
		}
		cells := extractCells(row)
		if len(cells) < 2 {
			log.Printf("errorcode row %d: expected >=2 cells, got %d — skipping", i, len(cells))
			continue
		}

		name := cleanText(cellText(cells[0]))
		if name == "" {
			log.Printf("errorcode row %d: empty name — skipping", i)
			continue
		}

		var value int
		var description string

		if len(cells) >= 3 {
			// Three-column layout: Name | Value | Description
			valueStr := cleanText(cellText(cells[1]))
			parsed, err := strconv.Atoi(valueStr)
			if err != nil {
				log.Printf("errorcode row %d %q: non-integer value %q — using counter", i, name, valueStr)
				value = valueCounter
			} else {
				value = parsed
			}
			description = cleanText(cellText(cells[2]))
		} else {
			// Two-column layout: Name | Description — derive value from row order.
			value = valueCounter
			description = cleanText(cellText(cells[1]))
		}
		valueCounter++

		ec := corpus.ErrorCode{
			Name:        name,
			Value:       value,
			Description: description,
			SourceURL:   sourceURL,
		}
		codes = append(codes, ec)
	}

	return codes, nil
}

// ── HTML extraction helpers ───────────────────────────────────────────────────

// extractH1 returns the trimmed text content of the first heading element
// found via depth-first traversal. It tries h1 first, then falls back to h2
// since some SDK pages use h2 as the primary page title.
// Returns "" if neither heading tag is found.
func extractH1(doc *html.Node) string {
	for _, tag := range []string{"h1", "h2"} {
		var text string
		var walk func(*html.Node)
		walk = func(n *html.Node) {
			if text != "" {
				return
			}
			if n.Type == html.ElementNode && n.Data == tag {
				text = cleanText(nodeText(n))
				return
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
		}
		walk(doc)
		if text != "" {
			return text
		}
	}
	return ""
}

// extractTableRows returns all <tr> nodes found in the first <table> in doc.
func extractTableRows(doc *html.Node) []*html.Node {
	table := findFirst(doc, "table")
	if table == nil {
		return nil
	}
	return rowsFromTable(table)
}

// findAllTables returns every top-level <table> node in doc without recursing
// into nested tables.
func findAllTables(doc *html.Node) []*html.Node {
	var tables []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			tables = append(tables, n)
			return // do not recurse into nested tables
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return tables
}

// rowsFromTable extracts all <tr> nodes from a single <table> node, without
// recursing into nested tables.
func rowsFromTable(table *html.Node) []*html.Node {
	var rows []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			rows = append(rows, n)
			return // do not recurse into nested tables
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(table)
	return rows
}

// extractCells returns all <td> and <th> direct children of row.
func extractCells(row *html.Node) []*html.Node {
	var cells []*html.Node
	for c := row.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
			cells = append(cells, c)
		}
	}
	return cells
}

// cellText returns the full text content of a cell node, including text from
// nested elements such as <span> and <a>.
func cellText(n *html.Node) string {
	return nodeText(n)
}

// nodeText returns the concatenated text content of n and all its descendants.
func nodeText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

// extractDeprecated inspects a description cell for a <span class="deprecated">
// element. If found, it strips the deprecation notice from description and
// returns deprecated=true with the reason extracted from the span text.
//
// The deprecated span text is expected to follow the pattern:
//
//	"Deprecated: <reason>"
func extractDeprecated(cell *html.Node, fullText string) (deprecated bool, reason string) {
	span := findFirstWithClass(cell, "span", "deprecated")
	if span == nil {
		return false, ""
	}

	spanText := cleanText(nodeText(span))
	// Strip the leading "Deprecated:" label if present.
	reason = strings.TrimPrefix(spanText, "Deprecated:")
	reason = strings.TrimPrefix(reason, " ")
	return true, reason
}

// findFirstWithClass performs a depth-first search for the first element of
// tagName that has the given class name in its class attribute.
func findFirstWithClass(root *html.Node, tagName, className string) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == tagName {
			for _, attr := range n.Attr {
				if attr.Key == "class" {
					for _, cls := range strings.Fields(attr.Val) {
						if cls == className {
							found = n
							return
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return found
}

// findFirst returns the first element with the given tag name via depth-first
// traversal. Returns nil if none is found.
func findFirst(root *html.Node, tagName string) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == tagName {
			found = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return found
}

// extractParagraphAfterH1 returns the text of the first <p> element that
// appears after the <h1> in document order, trimmed. Returns "" if not found.
func extractParagraphAfterH1(doc *html.Node) string {
	// Collect all h1 and p nodes in document order.
	var nodes []struct {
		tag  string
		node *html.Node
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "h1" || n.Data == "p") {
			nodes = append(nodes, struct {
				tag  string
				node *html.Node
			}{n.Data, n})
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	pastH1 := false
	for _, item := range nodes {
		if item.tag == "h1" {
			pastH1 = true
			continue
		}
		if pastH1 && item.tag == "p" {
			return cleanText(nodeText(item.node))
		}
	}
	return ""
}

// extractPreCode returns the trimmed text content of the first <pre><code>
// or plain <pre> block found in doc. Used to capture C function signatures.
func extractPreCode(doc *html.Node) string {
	pre := findFirst(doc, "pre")
	if pre == nil {
		return ""
	}
	return cleanText(nodeText(pre))
}

// extractReturnType finds a section heading containing "return" and extracts
// the text of the following paragraph. Returns "" if not found.
func extractReturnType(doc *html.Node) string {
	var nodes []struct {
		tag  string
		node *html.Node
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h2", "h3", "p":
				nodes = append(nodes, struct {
					tag  string
					node *html.Node
				}{n.Data, n})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	pastReturn := false
	for _, item := range nodes {
		t := cleanText(nodeText(item.node))
		if (item.tag == "h2" || item.tag == "h3") && strings.Contains(strings.ToLower(t), "return") {
			pastReturn = true
			continue
		}
		if pastReturn && item.tag == "p" {
			return t
		}
	}
	return ""
}

// extractRemarks finds a section heading containing "remark" or "note" and
// extracts the text of the following paragraph. Returns "" if not found.
func extractRemarks(doc *html.Node) string {
	var nodes []struct {
		tag  string
		node *html.Node
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h2", "h3", "p":
				nodes = append(nodes, struct {
					tag  string
					node *html.Node
				}{n.Data, n})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	pastRemarks := false
	for _, item := range nodes {
		t := cleanText(nodeText(item.node))
		lower := strings.ToLower(t)
		if (item.tag == "h2" || item.tag == "h3") && (strings.Contains(lower, "remark") || strings.Contains(lower, "note")) {
			pastRemarks = true
			continue
		}
		if pastRemarks && item.tag == "p" {
			return t
		}
	}
	return ""
}

// extractFunctionParams extracts function parameters from the parameters table.
// Expects columns: Parameter/Name, Type, Description (direction is inferred).
func extractFunctionParams(doc *html.Node) []corpus.FunctionParam {
	rows := extractTableRows(doc)
	var params []corpus.FunctionParam

	for i, row := range rows {
		if i == 0 {
			continue
		}
		cells := extractCells(row)
		if len(cells) < 3 {
			log.Printf("function param row %d: expected >=3 cells, got %d — skipping", i, len(cells))
			continue
		}

		name := cleanText(cellText(cells[0]))
		typ := cleanText(cellText(cells[1]))
		description := cleanText(cellText(cells[2]))

		if name == "" {
			continue
		}

		// Direction is not always a separate column in function pages;
		// default to "in" and let description override if needed.
		direction := "in"
		if strings.Contains(strings.ToLower(description), "receives") ||
			strings.Contains(strings.ToLower(description), "address of") ||
			strings.Contains(strings.ToLower(name), "*") ||
			strings.Contains(typ, "*") {
			direction = "out"
		}

		params = append(params, corpus.FunctionParam{
			Name:        name,
			Type:        typ,
			Direction:   direction,
			Description: description,
		})
	}

	return params
}

// extractStructFields extracts structure member fields from the members table.
// Handles two layouts:
//   - Three-column: Member | Type | Description
//   - Two-column:   Member | Description  (Type stored as "")
func extractStructFields(doc *html.Node) []corpus.StructField {
	rows := extractTableRows(doc)
	var fields []corpus.StructField

	for i, row := range rows {
		if i == 0 {
			continue
		}
		cells := extractCells(row)
		if len(cells) < 2 {
			log.Printf("struct field row %d: expected >=2 cells, got %d — skipping", i, len(cells))
			continue
		}

		name := cleanText(cellText(cells[0]))
		if name == "" {
			continue
		}

		var typ, description string
		if len(cells) >= 3 {
			typ = cleanText(cellText(cells[1]))
			description = cleanText(cellText(cells[2]))
		} else {
			description = cleanText(cellText(cells[1]))
		}

		fields = append(fields, corpus.StructField{
			Name:        name,
			Type:        typ,
			Description: description,
		})
	}

	return fields
}

// ── Text utilities ────────────────────────────────────────────────────────────

// cleanText collapses internal whitespace, trims leading/trailing whitespace,
// and normalises line endings.
func cleanText(s string) string {
	// Normalise line endings.
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")

	// Collapse runs of spaces.
	parts := strings.Fields(s)
	return strings.Join(parts, " ")
}

// parseUnitList splits a comma- or semicolon-separated unit string into a
// slice, trimming each entry. Returns nil for blank input.
func parseUnitList(s string) []string {
	if s == "" {
		return nil
	}
	// Support both comma and semicolon separators.
	s = strings.ReplaceAll(s, ";", ",")
	parts := strings.Split(s, ",")
	units := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			units = append(units, p)
		}
	}
	if len(units) == 0 {
		return nil
	}
	return units
}

// urlFileStem extracts the filename stem (last path segment without extension)
// from a URL string. Returns "" if the URL has no recognisable last segment.
// Example: ".../SIMCONNECT_STATE.htm" → "SIMCONNECT_STATE"
func urlFileStem(u string) string {
	// Work with the path part only (ignore query/fragment).
	if idx := strings.Index(u, "?"); idx >= 0 {
		u = u[:idx]
	}
	parts := strings.Split(u, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		seg := parts[i]
		if seg == "" {
			continue
		}
		if idx := strings.LastIndex(seg, "."); idx >= 0 {
			return seg[:idx]
		}
		return seg
	}
	return ""
}
