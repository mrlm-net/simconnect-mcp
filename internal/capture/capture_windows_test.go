//go:build windows

package capture

import (
	"errors"
	"os"
	"testing"
)

// The simulator window when it runs here (skipped otherwise); with
// CAPTURE_OUT set, the picture is written there.
func TestWindow(t *testing.T) {
	img, err := Window()
	if errors.Is(err, ErrNoWindow) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	small := Scale(img, 800)
	if small.Bounds().Dx() > 800 {
		t.Errorf("scaled to %d wide", small.Bounds().Dx())
	}
	data, mime, err := Encode(small, "jpeg", 80)
	if err != nil || mime != "image/jpeg" || len(data) == 0 {
		t.Fatalf("encode: %v %s %d", err, mime, len(data))
	}
	if out := os.Getenv("CAPTURE_OUT"); out != "" {
		os.WriteFile(out, data, 0o644)
	}
}
