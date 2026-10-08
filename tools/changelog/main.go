// Command changelog writes the website's copy of the changelog,
// docs/changelog.md: its own front matter followed by CHANGELOG.md without
// the top "# Changelog" title. CHANGELOG.md is the source; never edit
// docs/changelog.md by hand.
//
// Run it after editing CHANGELOG.md:
//
//	go generate ./tools/changelog/
package main

//go:generate go run . -in ../../CHANGELOG.md -out ../../docs/changelog.md

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	in := flag.String("in", "CHANGELOG.md", "source changelog")
	out := flag.String("out", "docs/changelog.md", "website copy (its front matter is kept)")
	flag.Parse()
	if err := run(*in, *out); err != nil {
		fmt.Fprintf(os.Stderr, "changelog: %v\n", err)
		os.Exit(1)
	}
}

func run(in, out string) error {
	src, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	dst, err := os.ReadFile(out)
	if err != nil {
		return err
	}
	nl := []byte("\n")
	if bytes.Contains(dst, []byte("\r\n")) {
		nl = []byte("\r\n")
	}
	fm, err := frontMatter(normalize(dst))
	if err != nil {
		return fmt.Errorf("%s: %w", out, err)
	}
	body := stripTitle(normalize(src))

	var b bytes.Buffer
	b.Write(fm)
	b.WriteString("\n")
	b.Write(body)
	res := bytes.ReplaceAll(b.Bytes(), []byte("\n"), nl)
	return os.WriteFile(out, res, 0o644)
}

func normalize(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}

// frontMatter returns the leading "---" block, closing line included.
func frontMatter(b []byte) ([]byte, error) {
	if !bytes.HasPrefix(b, []byte("---\n")) {
		return nil, errors.New("no front matter")
	}
	end := bytes.Index(b[4:], []byte("\n---\n"))
	if end < 0 {
		return nil, errors.New("front matter not closed")
	}
	return b[:4+end+5], nil
}

// stripTitle drops the "# " title line and the blank lines after it.
func stripTitle(b []byte) []byte {
	if bytes.HasPrefix(b, []byte("# ")) {
		if i := bytes.IndexByte(b, '\n'); i >= 0 {
			b = b[i+1:]
		} else {
			b = nil
		}
	}
	return bytes.TrimLeft(b, "\n")
}
