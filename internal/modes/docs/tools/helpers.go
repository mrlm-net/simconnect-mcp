package tools

// intArg extracts an integer argument from an MCP args map.
// JSON numbers arrive as float64 from encoding/json; this coerces them.
// Falls back to defaultVal if the key is absent or the wrong type.
func intArg(args map[string]any, key string, defaultVal int) int {
	v, ok := args[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return defaultVal
}
