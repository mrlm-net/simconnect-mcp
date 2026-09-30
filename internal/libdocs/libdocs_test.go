package libdocs

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

// TestEmbeddedMatchesDependency fails when go.mod moves to another library
// version without refreshing the guides (go generate ./internal/libdocs/).
func TestEmbeddedMatchesDependency(t *testing.T) {
	lib, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	mod, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(mod), "\n") {
		if f := strings.Fields(line); len(f) >= 2 && f[0] == "github.com/mrlm-net/simconnect" {
			if f[1] != lib.Version {
				t.Fatalf("embedded guides are %s, go.mod requires %s: run go generate ./internal/libdocs/", lib.Version, f[1])
			}
			return
		}
	}
	t.Fatal("github.com/mrlm-net/simconnect not in go.mod")
}

func TestLoadEmbedded(t *testing.T) {
	lib, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if n := len(lib.Guides("")); n < 30 {
		t.Errorf("got %d guides, want at least 30", n)
	}
	g, ok := lib.Guide("airport-layout.md")
	if !ok {
		t.Fatal("airport-layout not found")
	}
	if g.Title != "Airport Layout & Taxi Routing" || g.Section != "airport" {
		t.Errorf("front matter: title %q section %q", g.Title, g.Section)
	}
	if _, ok := g.Chapter("Taxi graph"); !ok {
		t.Errorf("no Taxi graph chapter in %v", g.Headings())
	}
}

const doc = "---\ntitle: \"Holds\"\ndescription: Holding patterns\nsection: traffic\norder: 2\n---\n\n# Holds\n\nIntro text.\n\n" +
	"## Entry\n\nDirect, teardrop or parallel entry by heading.\n\n```go\n## not a heading\n```\n\n" +
	"## Stack\n\nLevels 1000 ft apart; the lowest leaves first.\n"

func testLib(t *testing.T) *Library {
	t.Helper()
	lib, err := LoadFS(fstest.MapFS{
		"d/holds.md": {Data: []byte(doc)},
		"d/VERSION":  {Data: []byte("v9.9.9\n")},
	}, "d")
	if err != nil {
		t.Fatal(err)
	}
	return lib
}

func TestParse(t *testing.T) {
	lib := testLib(t)
	g, _ := lib.Guide("holds")
	if lib.Version != "v9.9.9" || g.Title != "Holds" || g.Description != "Holding patterns" || g.Section != "traffic" || g.Order != 2 {
		t.Fatalf("front matter: %+v", g)
	}
	if h := g.Headings(); strings.Join(h, "|") != "Entry|Stack" {
		t.Fatalf("headings %v: a fenced \"##\" is not a chapter", h)
	}
	c, ok := g.Chapter("ent")
	if !ok || !strings.Contains(c.Text, "## not a heading") {
		t.Fatalf("chapter by prefix: %v %q", ok, c.Text)
	}
}

func TestSearch(t *testing.T) {
	lib := testLib(t)
	hits := lib.Search("lowest LEVELS", 5)
	if len(hits) != 1 || hits[0].Heading != "Stack" {
		t.Fatalf("hits %+v", hits)
	}
	// A heading word ranks its chapter first.
	hits = lib.Search("entry", 5)
	if len(hits) == 0 || hits[0].Heading != "Entry" {
		t.Fatalf("hits %+v", hits)
	}
	if hits := lib.Search("nothing-like-this", 5); len(hits) != 0 {
		t.Fatalf("hits %+v", hits)
	}
}
