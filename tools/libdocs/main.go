// Command libdocs copies the guides of the github.com/mrlm-net/simconnect Go
// library, at the version go.mod requires, into internal/libdocs/assets.
//
// Run it after upgrading the library:
//
//	go generate ./internal/libdocs/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const module = "github.com/mrlm-net/simconnect"

func main() {
	out := flag.String("out", "assets", "output directory")
	flag.Parse()
	if err := run(*out); err != nil {
		fmt.Fprintf(os.Stderr, "libdocs: %v\n", err)
		os.Exit(1)
	}
}

func run(out string) error {
	cmd := exec.Command("go", "list", "-m", "-json", module)
	b, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("go list -m %s: %w", module, err)
	}
	var mod struct{ Version, Dir string }
	if err := json.Unmarshal(b, &mod); err != nil {
		return err
	}
	if mod.Dir == "" {
		return fmt.Errorf("%s %s is not downloaded; run go mod download", module, mod.Version)
	}

	src := filepath.Join(mod.Dir, "docs")
	files, err := filepath.Glob(filepath.Join(src, "*.md"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no guides in %s", src)
	}

	// Replace the previous version's guides entirely: renamed or removed
	// guides must not linger.
	old, _ := filepath.Glob(filepath.Join(out, "*.md"))
	for _, f := range old {
		if err := os.Remove(f); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		// Normalise line endings so the embedded corpus is identical on every OS.
		data = []byte(strings.ReplaceAll(string(data), "\r\n", "\n"))
		if err := os.WriteFile(filepath.Join(out, filepath.Base(f)), data, 0o644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(out, "VERSION"), []byte(mod.Version+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Printf("libdocs: %d guides of %s %s\n", len(files), module, mod.Version)
	return nil
}
