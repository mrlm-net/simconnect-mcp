package mcpadapter

import (
	"context"
	"encoding/json"
)

// ToolHandlerFunc is the signature for a tool implementation.
// args contains the parsed input arguments; the function returns a result or error.
type ToolHandlerFunc func(ctx context.Context, args map[string]any) (*CallToolResult, error)

// Tool is the MCP tool definition exposed to clients.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema is the JSON Schema object describing a tool's input parameters.
type InputSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]SchemaProperty `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

// SchemaProperty describes a single input parameter.
type SchemaProperty struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// Content is a single content item in a tool result.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// CallToolResult is returned by a tool handler.
type CallToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// ToolBuilder constructs a Tool definition with a fluent API.
type ToolBuilder struct {
	tool Tool
}

// NewTool starts building a tool with the given name.
func NewTool(name string) *ToolBuilder {
	return &ToolBuilder{
		tool: Tool{
			Name:        name,
			InputSchema: InputSchema{Type: "object", Properties: make(map[string]SchemaProperty)},
		},
	}
}

// Description sets the tool's human-readable description.
func (b *ToolBuilder) Description(desc string) *ToolBuilder {
	b.tool.Description = desc
	return b
}

// StringParam adds a string input parameter.
func (b *ToolBuilder) StringParam(name, description string) *ToolBuilder {
	return b.addParam(name, "string", description)
}

// NumberParam adds a number input parameter.
func (b *ToolBuilder) NumberParam(name, description string) *ToolBuilder {
	return b.addParam(name, "number", description)
}

// BoolParam adds a boolean input parameter.
func (b *ToolBuilder) BoolParam(name, description string) *ToolBuilder {
	return b.addParam(name, "boolean", description)
}

// Required marks the named parameters as required.
func (b *ToolBuilder) Required(names ...string) *ToolBuilder {
	b.tool.InputSchema.Required = append(b.tool.InputSchema.Required, names...)
	return b
}

// Build returns the completed Tool definition.
func (b *ToolBuilder) Build() Tool {
	return b.tool
}

func (b *ToolBuilder) addParam(name, typ, description string) *ToolBuilder {
	b.tool.InputSchema.Properties[name] = SchemaProperty{Type: typ, Description: description}
	return b
}

// TextResult returns a successful tool result containing plain text.
func TextResult(text string) *CallToolResult {
	return &CallToolResult{Content: []Content{{Type: "text", Text: text}}}
}

// ErrorResult returns a tool result that signals an error to the LLM.
// Prefer this over returning a Go error — the LLM can self-correct on tool errors.
func ErrorResult(msg string) *CallToolResult {
	return &CallToolResult{Content: []Content{{Type: "text", Text: msg}}, IsError: true}
}

// JSONResult returns a successful tool result with JSON-encoded content.
func JSONResult[T any](v T) (*CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &CallToolResult{Content: []Content{{Type: "text", Text: string(b)}}}, nil
}
