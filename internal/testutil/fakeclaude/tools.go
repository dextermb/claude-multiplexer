package main

import (
	"fmt"
	"strings"
	"time"
)

func emitToolCalls(sessionID string) {
	var body []string
	for i := 1; i <= 20; i++ {
		body = append(body, fmt.Sprintf("line %d of the test output", i))
	}
	calls := []struct {
		id, name string
		input    map[string]any
		result   string
		delay    time.Duration
	}{
		{"toolu_r1", "Read", map[string]any{"file_path": "internal/auth/middleware.go"}, strings.Repeat("code\n", 412), 0},
		{"toolu_e1", "Edit", map[string]any{"file_path": "internal/auth/refresh.go", "old_string": "a", "new_string": "a\nb\nc"}, "The file has been updated.", 0},
		{"toolu_g1", "Grep", map[string]any{"pattern": "owner:", "path": "docs/"}, "Found 9 files\ndocs/a.md", 0},
		{"toolu_b1", "Bash", map[string]any{"command": "go test ./internal/auth/..."}, strings.Join(body, "\n"), 3 * time.Second},
	}
	for _, c := range calls {
		emit(map[string]any{
			"type":       "assistant",
			"session_id": sessionID,
			"message": map[string]any{
				"role":    "assistant",
				"model":   "fake-model",
				"content": []map[string]any{{"type": "tool_use", "id": c.id, "name": c.name, "input": c.input}},
			},
		})
		time.Sleep(c.delay)
		emit(map[string]any{
			"type":       "user",
			"session_id": sessionID,
			"message": map[string]any{
				"role":    "user",
				"content": []map[string]any{{"type": "tool_result", "tool_use_id": c.id, "content": c.result}},
			},
		})
	}
}
