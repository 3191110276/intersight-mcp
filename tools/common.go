package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/sandbox"
)

const (
	ToolSearch = "search"
	ToolQuery  = "query"
	ToolMutate = "mutate"

	searchTitle = "Intersight Spec Search"
	queryTitle  = "Intersight Query"
	mutateTitle = "Intersight Mutate"

	metricsAppResourceURI      = "ui://metrics/frame.html"
	metricsAppResourceMimeType = "text/html+skybridge"
)

var (
	inputSchemaJSON = json.RawMessage(`{
  "type": "object",
  "properties": {
    "code": {
      "type": "string",
      "minLength": 1,
      "maxLength": 102400,
      "description": "JavaScript source to execute as the body of an async function."
    }
  },
  "required": ["code"],
  "additionalProperties": false
}`)

	mutateInputSchemaJSON = json.RawMessage(`{
  "type": "object",
  "properties": {
    "changeSummary": {
      "type": "string",
      "minLength": 1,
      "maxLength": 1000,
      "description": "Human-readable summary of what the mutation will change."
    },
    "code": {
      "type": "string",
      "minLength": 1,
      "maxLength": 102400,
      "description": "JavaScript source to execute as the body of an async function."
    }
  },
  "required": ["changeSummary", "code"],
  "additionalProperties": false
}`)

	outputSchemaJSON = json.RawMessage(`{
  "oneOf": [
    {
      "type": "object",
      "properties": {
        "ok": { "const": true },
        "result": {},
        "logs": {
          "type": "array",
          "items": { "type": "string" }
        }
      },
      "required": ["ok", "result", "logs"],
      "additionalProperties": false
    },
    {
      "type": "object",
      "properties": {
        "ok": { "const": false },
        "error": {
          "type": "object",
          "properties": {
            "type": { "type": "string" },
            "message": { "type": "string" },
            "hint": { "type": "string" },
            "retryable": { "type": "boolean" },
            "status": { "type": "integer" },
            "details": {}
          },
          "required": ["type", "message", "hint", "retryable"],
          "additionalProperties": false
        },
        "logs": {
          "type": "array",
          "items": { "type": "string" }
        }
      },
      "required": ["ok", "error", "logs"],
      "additionalProperties": false
    }
  ]
}`)
)

type codeInput struct {
	Code string `json:"code"`
}

type mutateInput struct {
	ChangeSummary string `json:"changeSummary"`
	Code          string `json:"code"`
}

type Limiter struct {
	slots chan struct{}
}

func NewLimiter(maxConcurrent int) *Limiter {
	if maxConcurrent <= 0 {
		return nil
	}
	return &Limiter{slots: make(chan struct{}, maxConcurrent)}
}

func (l *Limiter) Acquire() bool {
	if l == nil {
		return true
	}
	select {
	case l.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (l *Limiter) Release() {
	if l == nil {
		return
	}
	select {
	case <-l.slots:
	default:
	}
}

func (l *Limiter) Limit() int {
	if l == nil {
		return 0
	}
	return cap(l.slots)
}

func InputSchema() json.RawMessage {
	return append(json.RawMessage(nil), inputSchemaJSON...)
}

func MutateInputSchema() json.RawMessage {
	return append(json.RawMessage(nil), mutateInputSchemaJSON...)
}

func OutputSchema() json.RawMessage {
	return append(json.RawMessage(nil), outputSchemaJSON...)
}

func ServerTools(searchExec, queryExec, mutateExec sandbox.Executor, limiter *Limiter, exposeMetricsApps bool) []mcpserver.ServerTool {
	return []mcpserver.ServerTool{
		NewSearchTool(searchExec),
		NewQueryTool(queryExec, limiter, exposeMetricsApps),
		NewMutateTool(mutateExec, limiter),
	}
}

func newServerTool(name, title, description string, mode sandbox.Mode, exec sandbox.Executor, limiter *Limiter, readOnly, destructive, _ bool) mcpserver.ServerTool {
	inputSchema := InputSchema()
	if mode == sandbox.ModeMutate {
		inputSchema = MutateInputSchema()
	}

	tool := mcp.NewTool(name,
		mcp.WithDescription(description),
		mcp.WithRawInputSchema(inputSchema),
		mcp.WithRawOutputSchema(OutputSchema()),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           title,
			ReadOnlyHint:    mcp.ToBoolPtr(readOnly),
			DestructiveHint: mcp.ToBoolPtr(destructive),
		}),
	)
	tool.InputSchema = mcp.ToolInputSchema{}
	tool.OutputSchema = mcp.ToolOutputSchema{}

	return mcpserver.ServerTool{
		Tool:    tool,
		Handler: NewToolHandler(mode, exec, limiter),
	}
}

func NewToolHandler(mode sandbox.Mode, exec sandbox.Executor, limiter *Limiter) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if exec == nil {
			return nil, errors.New("tool executor is not configured")
		}

		var (
			code          string
			changeSummary string
		)
		if mode == sandbox.ModeMutate {
			var input mutateInput
			if err := request.BindArguments(&input); err != nil {
				return nil, err
			}
			if strings.TrimSpace(input.ChangeSummary) == "" {
				return nil, errors.New(`required argument "changeSummary" not found`)
			}
			code = input.Code
			changeSummary = input.ChangeSummary
		} else {
			var input codeInput
			if err := request.BindArguments(&input); err != nil {
				return nil, err
			}
			code = input.Code
		}
		if strings.TrimSpace(code) == "" {
			return nil, errors.New(`required argument "code" not found`)
		}

		if limiter != nil {
			if !limiter.Acquire() {
				return toolErrorResult(contracts.LimitError{
					Message: fmt.Sprintf("Concurrent execution limit reached (%d)", limiter.Limit()),
				}, nil), nil
			}
			defer limiter.Release()
		}

		_ = changeSummary
		result, err := exec.Execute(ctx, code, mode)
		if err != nil {
			return toolErrorResult(err, result.Logs), nil
		}
		return toolSuccessResult(request.Params.Name, result), nil
	}
}

func toolSuccessResult(_ string, result sandbox.Result) *mcp.CallToolResult {
	envelope := contracts.Success(result.Value, result.Logs)
	content := []mcp.Content{mcp.NewTextContent(renderSuccessText(envelope))}
	return &mcp.CallToolResult{
		Result:            mcp.Result{},
		Content:           content,
		StructuredContent: envelope,
		IsError:           false,
	}
}

func toolErrorResult(err error, logs []string) *mcp.CallToolResult {
	envelope := contracts.NormalizeError(err, logs)
	return &mcp.CallToolResult{
		Content:           []mcp.Content{mcp.NewTextContent(renderErrorText(envelope))},
		StructuredContent: envelope,
		IsError:           true,
	}
}

func renderSuccessText(envelope contracts.SuccessEnvelope) string {
	var buf bytes.Buffer
	buf.WriteString(compactJSON(envelope.Result))
	if len(envelope.Logs) > 0 {
		buf.WriteString("\n\nLogs:\n")
		buf.WriteString(strings.Join(envelope.Logs, "\n"))
	}
	return buf.String()
}

func renderErrorText(envelope contracts.ErrorEnvelope) string {
	var buf bytes.Buffer
	buf.WriteString("error.type: ")
	buf.WriteString(envelope.Error.Type)
	buf.WriteString("\nerror.message: ")
	buf.WriteString(envelope.Error.Message)
	buf.WriteString("\nerror.hint: ")
	buf.WriteString(envelope.Error.Hint)
	if len(envelope.Logs) > 0 {
		buf.WriteString("\n\nLogs:\n")
		buf.WriteString(strings.Join(envelope.Logs, "\n"))
	}
	return buf.String()
}

func compactJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(data)
}
