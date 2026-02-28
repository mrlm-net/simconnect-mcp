//go:build windows

// Package simconnect implements the live SimConnect data operating mode
// (MCP_MODE=simconnect). It connects to a running Microsoft Flight Simulator
// instance via the github.com/mrlm-net/simconnect library and exposes live
// simulator variables and events as MCP tools.
//
// This package is gated by //go:build windows and must only be compiled on Windows.
package simconnect
