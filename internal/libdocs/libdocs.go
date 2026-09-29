// Package libdocs serves the guides of the github.com/mrlm-net/simconnect Go
// library: the client and manager, facilities, pkg/airport (layouts, taxi
// routing, procedures), pkg/nav (airways, weather, flight plans) and
// pkg/traffic (AI traffic, sequencing, separation).
//
// The guides are embedded at the library version go.mod requires; refresh
// them with go generate after an upgrade.
package libdocs

//go:generate go run ../../tools/libdocs -out assets

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
)

//go:embed assets/*.md assets/VERSION
var assets embed.FS

// Guide is one library guide.
type Guide struct {
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Section     string    `json:"section,omitempty"`
	Order       int       `json:"-"`
	Sections    []Section `json:"-"`
	Body        string    `json:"-"`
}

// Section is a "##" chapter of a guide; the text before the first one is the
// introduction, with an empty heading.
type Section struct {
	Heading string
	Text    string
}

// Hit is a search result: a chapter of a guide.
type Hit struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Heading string `json:"heading,omitempty"`
	Excerpt string `json:"excerpt"`
	Score   int    `json:"-"`
}

// Library is the loaded set of guides.
type Library struct {
	Version string
	guides  []*Guide
	bySlug  map[string]*Guide
}

// Load reads the embedded guides.
func Load() (*Library, error) {
	return LoadFS(assets, "assets")
}

// LoadFS reads the guides from dir in fsys.
func LoadFS(fsys fs.FS, dir string) (*Library, error) {
	lib := &Library{bySlug: map[string]*Guide{}}
	if v, err := fs.ReadFile(fsys, path.Join(dir, "VERSION")); err == nil {
		lib.Version = strings.TrimSpace(string(v))
	}
	files, err := fs.Glob(fsys, path.Join(dir, "*.md"))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		data, err := fs.ReadFile(fsys, f)
		if err != nil {
			return nil, err
		}
		g := parse(strings.TrimSuffix(path.Base(f), ".md"), strings.ReplaceAll(string(data), "\r\n", "\n"))
		lib.guides = append(lib.guides, g)
		lib.bySlug[g.Slug] = g
	}
	if len(lib.guides) == 0 {
		return nil, fmt.Errorf("libdocs: no guides in %s", dir)
	}
	sort.SliceStable(lib.guides, func(i, j int) bool {
		a, b := lib.guides[i], lib.guides[j]
		if a.Section != b.Section {
			return a.Section < b.Section
		}
		if a.Order != b.Order {
			return a.Order < b.Order
		}
		return a.Slug < b.Slug
	})
	return lib, nil
}

// Guides returns the guides, by section and order; section filters when set.
func (l *Library) Guides(section string) []*Guide {
	var out []*Guide
	for _, g := range l.guides {
		if section == "" || strings.EqualFold(g.Section, section) {
			out = append(out, g)
		}
	}
	return out
}

// Sections returns the section names in use.
func (l *Library) Sections() []string {
	seen := map[string]bool{}
	var out []string
	for _, g := range l.guides {
		if g.Section != "" && !seen[g.Section] {
			seen[g.Section] = true
			out = append(out, g.Section)
		}
	}
	return out
}

// Guide returns the guide by slug ("airport-layout"; ".md" is accepted).
func (l *Library) Guide(slug string) (*Guide, bool) {
	g, ok := l.bySlug[strings.TrimSuffix(strings.ToLower(strings.TrimSpace(slug)), ".md")]
	return g, ok
}

// Chapter returns the chapter of g whose heading matches heading, ignoring
// case; a heading that is a prefix of exactly one chapter matches too.
func (g *Guide) Chapter(heading string) (Section, bool) {
	h := strings.ToLower(strings.TrimSpace(heading))
	var prefix []Section
	for _, s := range g.Sections {
		sh := strings.ToLower(s.Heading)
		if sh == h {
			return s, true
		}
		if h != "" && strings.HasPrefix(sh, h) {
			prefix = append(prefix, s)
		}
	}
	if len(prefix) == 1 {
		return prefix[0], true
	}
	return Section{}, false
}

// Headings returns the chapter headings of g.
func (g *Guide) Headings() []string {
	var out []string
	for _, s := range g.Sections {
		if s.Heading != "" {
			out = append(out, s.Heading)
		}
	}
	return out
}

// Search finds the chapters containing every word of query, ignoring case,
// best first: words in a title or heading count more than in the text.
func (l *Library) Search(query string, limit int) []Hit {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return nil
	}
	var hits []Hit
	for _, g := range l.guides {
		title := strings.ToLower(g.Title + " " + g.Description)
		for _, s := range g.Sections {
			head := strings.ToLower(s.Heading)
			text := strings.ToLower(s.Text)
			score := 0
			for _, w := range words {
				n := strings.Count(text, w)
				if n == 0 && !strings.Contains(head, w) && !strings.Contains(title, w) {
					score = -1
					break
				}
				score += min(n, 10)
				if strings.Contains(head, w) {
					score += 20
				}
				if strings.Contains(title, w) {
					score += 5
				}
			}
			if score < 0 {
				continue
			}
			hits = append(hits, Hit{Slug: g.Slug, Title: g.Title, Heading: s.Heading, Excerpt: excerpt(s.Text, words[0]), Score: score})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

// excerpt returns about 240 characters of text around the first occurrence of word.
func excerpt(text, word string) string {
	const width = 240
	flat := strings.Join(strings.Fields(text), " ")
	i := strings.Index(strings.ToLower(flat), word)
	start := 0
	if i > width/3 {
		start = i - width/3
	}
	end := min(len(flat), start+width)
	// Keep to rune boundaries.
	for start > 0 && start < len(flat) && flat[start]&0xC0 == 0x80 {
		start--
	}
	for end < len(flat) && flat[end]&0xC0 == 0x80 {
		end++
	}
	out := flat[start:end]
	if start > 0 {
		out = "…" + out
	}
	if end < len(flat) {
		out += "…"
	}
	return out
}

// parse reads the front matter and splits the body into "##" chapters.
// Headings inside fenced code blocks are not chapters.
func parse(slug, doc string) *Guide {
	g := &Guide{Slug: slug}
	body := doc
	if strings.HasPrefix(doc, "---\n") {
		if end := strings.Index(doc[4:], "\n---"); end >= 0 {
			for _, line := range strings.Split(doc[4:4+end], "\n") {
				k, v, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				v = strings.Trim(strings.TrimSpace(v), `"'`)
				switch strings.TrimSpace(k) {
				case "title":
					g.Title = v
				case "description":
					g.Description = v
				case "section":
					g.Section = v
				case "order":
					g.Order, _ = strconv.Atoi(v)
				}
			}
			body = strings.TrimLeft(doc[4+end+4:], "\n")
		}
	}
	g.Body = body

	cur := Section{}
	var text strings.Builder
	fenced := false
	flush := func() {
		cur.Text = strings.TrimSpace(text.String())
		if cur.Heading != "" || cur.Text != "" {
			g.Sections = append(g.Sections, cur)
		}
		text.Reset()
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
		}
		if !fenced && strings.HasPrefix(line, "## ") {
			flush()
			cur = Section{Heading: strings.TrimSpace(line[3:])}
			continue
		}
		if !fenced && g.Title == "" && strings.HasPrefix(line, "# ") {
			g.Title = strings.TrimSpace(line[2:])
		}
		text.WriteString(line)
		text.WriteByte('\n')
	}
	flush()
	if g.Title == "" {
		g.Title = slug
	}
	return g
}
