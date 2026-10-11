//go:build windows

package tools

import (
	"context"
	"fmt"

	"github.com/mrlm-net/simconnect-mcp/internal/capture"
	"github.com/mrlm-net/simconnect-mcp/internal/mcpadapter"
)

// RegisterScreenshotTool registers screenshot: a picture of the simulator
// window, drawn even when other windows cover it (not when minimized).
// It needs no SimConnect, only the simulator on this machine.
func RegisterScreenshotTool(mcp *mcpadapter.Server) {
	tool := mcpadapter.NewTool("screenshot").
		Description("A picture of the simulator window as it is now (also when covered by other windows, not minimized).").
		NumberParam("max_width", "pixels, scaled down to it; default 1600, 0 full size").
		StringParam("format", "jpeg (default) or png").
		NumberParam("quality", "jpeg quality 1–100, default 85").
		Build()
	mcp.AddTool(tool, func(ctx context.Context, args map[string]any) (*mcpadapter.CallToolResult, error) {
		format := strArg(args, "format")
		if format != "" && format != "jpeg" && format != "png" {
			return mcpadapter.ErrorResult("INVALID_ARGUMENT: format must be jpeg or png"), nil
		}
		maxW := 1600
		if v, ok := args["max_width"].(float64); ok {
			maxW = int(v)
		}
		q := 85
		if v, ok := args["quality"].(float64); ok {
			q = int(v)
		}
		img, err := capture.Window()
		if err != nil {
			return mcpadapter.ErrorResult("NOT_APPLICABLE: " + err.Error()), nil
		}
		full := img.Bounds()
		img = capture.Scale(img, maxW)
		data, mime, err := capture.Encode(img, format, q)
		if err != nil {
			return mcpadapter.ErrorResult("NOT_APPLICABLE: encoding: " + err.Error()), nil
		}
		b := img.Bounds()
		return mcpadapter.ImageResult(data, mime, fmt.Sprintf("simulator window %dx%d (sent %dx%d, %d KB)", full.Dx(), full.Dy(), b.Dx(), b.Dy(), len(data)/1024)), nil
	})
}
