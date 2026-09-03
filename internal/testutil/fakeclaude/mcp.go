package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// runMCP answers each prompt by calling one MCP tool. The prompt is the tool
// name, a space, and the JSON arguments, for example:
//
//	rename_session {"title":"Billing rewrite"}
func runMCP(sessionID string) {
	client, err := dialMCP()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fakeclaude: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 16<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var in inputLine
		if err := json.Unmarshal([]byte(line), &in); err != nil {
			continue
		}
		if in.Type != "user" {
			continue
		}
		var text string
		for _, block := range in.Message.Content {
			if block.Type == "text" {
				text = block.Text
			}
		}
		if text == "quit" {
			os.Exit(0)
		}
		answer := callTool(client, text)
		emit(map[string]any{
			"type":       "assistant",
			"session_id": sessionID,
			"message": map[string]any{
				"role":    "assistant",
				"model":   "fake-model",
				"content": []map[string]any{{"type": "text", "text": answer}},
			},
		})
		emit(map[string]any{
			"type":           "result",
			"subtype":        "success",
			"is_error":       false,
			"duration_ms":    7,
			"num_turns":      1,
			"result":         answer,
			"total_cost_usd": 0.25,
			"session_id":     sessionID,
		})
	}
}

func callTool(client *sdk.ClientSession, text string) string {
	name, rest, _ := strings.Cut(text, " ")
	args := map[string]any{}
	if rest = strings.TrimSpace(rest); rest != "" {
		if err := json.Unmarshal([]byte(rest), &args); err != nil {
			return "bad arguments: " + err.Error()
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := client.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return "call failed: " + err.Error()
	}
	var out strings.Builder
	if result.IsError {
		out.WriteString("tool error: ")
	}
	for _, item := range result.Content {
		if content, ok := item.(*sdk.TextContent); ok {
			out.WriteString(content.Text)
		}
	}
	return out.String()
}

type mcpConfig struct {
	MCPServers map[string]struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	} `json:"mcpServers"`
}

func dialMCP() (*sdk.ClientSession, error) {
	path := ""
	for i, arg := range os.Args {
		if arg == "--mcp-config" && i+1 < len(os.Args) {
			path = os.Args[i+1]
		}
	}
	if path == "" {
		return nil, fmt.Errorf("no --mcp-config argument")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg mcpConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	entry, ok := cfg.MCPServers["mux"]
	if !ok {
		return nil, fmt.Errorf("no mux server in %s", path)
	}

	client := sdk.NewClient(&sdk.Implementation{Name: "fakeclaude", Version: "0"}, nil)
	transport := &sdk.StreamableClientTransport{
		Endpoint:   entry.URL,
		HTTPClient: &http.Client{Transport: headerTransport{headers: entry.Headers}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return client.Connect(ctx, transport, nil)
}

type headerTransport struct{ headers map[string]string }

func (h headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	for key, value := range h.headers {
		clone.Header.Set(key, value)
	}
	return http.DefaultTransport.RoundTrip(clone)
}
