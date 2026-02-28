// Package mcpadapter implements a minimal Model Context Protocol (MCP) server
// over HTTP/SSE using only the standard library and Gin. It supports both the
// Streamable HTTP transport (POST /mcp) and the legacy SSE transport (GET /sse,
// POST /message) as defined in the MCP 2024-11-05 specification.
package mcpadapter
