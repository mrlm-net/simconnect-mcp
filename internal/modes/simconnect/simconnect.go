//go:build windows

// Package simconnect implements the live-data mode (Milestone 2).
// It connects to a running MSFS/P3D/FSX instance via the SimConnect SDK
// and exposes real-time simulation variables through MCP tool endpoints.
// This package is Windows-only and requires the SimConnect DLL.
package simconnect
