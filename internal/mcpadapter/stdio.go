package mcpadapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
)

// ServeStdio runs the MCP server over stdin/stdout using newline-delimited
// JSON-RPC messages. Each line read from stdin is decoded as a JSON-RPC
// request; responses are written as single-line JSON to stdout.
//
// This is the standard stdio transport used by Claude Code and other local
// MCP clients that launch the server as a subprocess.
//
// ServeStdio blocks until stdin is closed, ctx is cancelled, or a scanner
// error occurs.
func (s *Server) ServeStdio(ctx context.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	enc := json.NewEncoder(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !scanner.Scan() {
			return scanner.Err()
		}

		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			_ = enc.Encode(errResp(nil, -32700, "parse error"))
			continue
		}

		if req.isNotification() {
			s.handleNotif(req.Method)
			continue
		}

		resp := s.dispatch(ctx, &req)
		_ = enc.Encode(resp)
	}
}
