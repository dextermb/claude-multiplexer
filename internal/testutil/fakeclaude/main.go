package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type inputLine struct {
	Type    string `json:"type"`
	Message struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
	RequestID string `json:"request_id"`
	Request   struct {
		Subtype string `json:"subtype"`
	} `json:"request"`
}

func main() {
	if path := os.Getenv("FAKECLAUDE_ARGS_FILE"); path != "" {
		cwd, _ := os.Getwd()
		record := append([]string{"cwd=" + cwd}, os.Args[1:]...)
		_ = os.WriteFile(path, []byte(strings.Join(record, "\n")), 0o644)
	}

	sessionID := os.Getenv("FAKECLAUDE_SESSION_ID")
	if sessionID == "" {
		sessionID = "11111111-2222-3333-4444-555555555555"
	}

	switch os.Getenv("FAKECLAUDE_MODE") {
	case "crash":
		fmt.Fprintln(os.Stderr, "fakeclaude: exploded on purpose")
		os.Exit(1)
	case "noinit":
		os.Exit(0)
	case "garbage":
		fmt.Println("this line is not JSON")
	case "bigline":
		emitInit(sessionID)
		fmt.Printf("{\"type\":\"assistant\",\"padding\":\"%s\"}\n", strings.Repeat("x", 4096))
		os.Exit(0)
	case "exit-after-init":
		emitInit(sessionID)
		os.Exit(0)
	case "interruptible":
		emitInit(sessionID)
		runInterruptible(sessionID)
		os.Exit(0)
	case "question":
		emitInit(sessionID)
		runQuestion(sessionID)
		os.Exit(0)
	case "mcp":
		emitInit(sessionID)
		runMCP(sessionID)
		os.Exit(0)
	}

	replay := false
	partial := false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--replay-user-messages":
			replay = true
		case "--include-partial-messages":
			partial = true
		}
	}

	lazy := os.Getenv("FAKECLAUDE_MODE") == "lazyinit"
	if !lazy {
		emitInit(sessionID)
	}

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
		if lazy {
			emitInit(sessionID)
			lazy = false
		}
		if text == "quit" {
			os.Exit(0)
		}
		if replay {
			emit(map[string]any{
				"type":       "user",
				"session_id": sessionID,
				"isReplay":   true,
				"message": map[string]any{
					"role":    "user",
					"content": []map[string]any{{"type": "text", "text": text}},
				},
			})
		}
		if partial {
			for _, chunk := range split("echo: "+text, 3) {
				emit(map[string]any{
					"type":       "stream_event",
					"session_id": sessionID,
					"event": map[string]any{
						"type":  "content_block_delta",
						"index": 0,
						"delta": map[string]any{"type": "text_delta", "text": chunk},
					},
				})
			}
		}
		emit(map[string]any{
			"type":       "assistant",
			"session_id": sessionID,
			"message": map[string]any{
				"role":    "assistant",
				"model":   "fake-model",
				"content": []map[string]any{{"type": "text", "text": "echo: " + text}},
			},
		})
		emit(map[string]any{
			"type":           "result",
			"subtype":        "success",
			"is_error":       false,
			"duration_ms":    7,
			"num_turns":      1,
			"result":         "echo: " + text,
			"total_cost_usd": 0.25,
			"session_id":     sessionID,
		})
	}
}

func runInterruptible(sessionID string) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 16<<20)
	busy := false
	pending := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var in inputLine
		if err := json.Unmarshal([]byte(line), &in); err != nil {
			continue
		}
		switch in.Type {
		case "user":
			var text string
			for _, block := range in.Message.Content {
				if block.Type == "text" {
					text = block.Text
				}
			}
			pending = text
			busy = true
			emit(map[string]any{
				"type":       "assistant",
				"session_id": sessionID,
				"message": map[string]any{
					"role":    "assistant",
					"model":   "fake-model",
					"content": []map[string]any{{"type": "text", "text": "start: " + text}},
				},
			})
		case "control_request":
			emit(map[string]any{
				"type":     "control_response",
				"response": map[string]any{"subtype": "success", "request_id": in.RequestID},
			})
			if in.Request.Subtype != "interrupt" || !busy {
				continue
			}
			busy = false
			emit(map[string]any{
				"type":           "result",
				"subtype":        "error_during_execution",
				"is_error":       true,
				"duration_ms":    3,
				"num_turns":      1,
				"result":         "interrupted: " + pending,
				"total_cost_usd": 0.1,
				"session_id":     sessionID,
			})
		}
	}
}

func runQuestion(sessionID string) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 16<<20)
	busy := false
	asked := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var in inputLine
		if err := json.Unmarshal([]byte(line), &in); err != nil {
			continue
		}
		switch in.Type {
		case "user":
			var text string
			for _, block := range in.Message.Content {
				if block.Type == "text" {
					text = block.Text
				}
			}
			if !asked {
				asked = true
				busy = true
				emit(map[string]any{
					"type":       "assistant",
					"session_id": sessionID,
					"message": map[string]any{
						"role":  "assistant",
						"model": "fake-model",
						"content": []map[string]any{{
							"type": "tool_use",
							"id":   "toolu_1",
							"name": "AskUserQuestion",
							"input": map[string]any{"questions": []map[string]any{{
								"question": "Which colour do you prefer?",
								"header":   "Colour",
								"options": []map[string]any{
									{"label": "Red", "description": "You prefer red."},
									{"label": "Blue", "description": "You prefer blue."},
								},
								"multiSelect": false,
							}}},
						}},
					},
				})
				emit(map[string]any{
					"type":       "user",
					"session_id": sessionID,
					"message": map[string]any{
						"role": "user",
						"content": []map[string]any{{
							"type":        "tool_result",
							"tool_use_id": "toolu_1",
							"is_error":    true,
							"content":     "Answer questions?",
						}},
					},
				})
				continue
			}
			busy = true
			emit(map[string]any{
				"type":       "assistant",
				"session_id": sessionID,
				"message": map[string]any{
					"role":    "assistant",
					"model":   "fake-model",
					"content": []map[string]any{{"type": "text", "text": "answered: " + text}},
				},
			})
			emit(map[string]any{
				"type":           "result",
				"subtype":        "success",
				"is_error":       false,
				"duration_ms":    5,
				"num_turns":      1,
				"result":         "answered: " + text,
				"total_cost_usd": 0.2,
				"session_id":     sessionID,
			})
			busy = false
		case "control_request":
			emit(map[string]any{
				"type":     "control_response",
				"response": map[string]any{"subtype": "success", "request_id": in.RequestID},
			})
			if in.Request.Subtype != "interrupt" || !busy {
				continue
			}
			busy = false
			emit(map[string]any{
				"type":           "result",
				"subtype":        "error_during_execution",
				"is_error":       true,
				"duration_ms":    3,
				"num_turns":      1,
				"result":         "interrupted",
				"total_cost_usd": 0.1,
				"session_id":     sessionID,
			})
		}
	}
}

func split(text string, parts int) []string {
	runes := []rune(text)
	if parts < 1 || len(runes) < parts {
		return []string{text}
	}
	size := len(runes) / parts
	out := make([]string, 0, parts)
	for i := 0; i < parts; i++ {
		start := i * size
		end := start + size
		if i == parts-1 {
			end = len(runes)
		}
		out = append(out, string(runes[start:end]))
	}
	return out
}

func emitInit(sessionID string) {
	cwd, _ := os.Getwd()
	emit(map[string]any{
		"type":           "system",
		"subtype":        "init",
		"session_id":     sessionID,
		"model":          "fake-model",
		"cwd":            cwd,
		"tools":          []string{"Read", "Bash"},
		"permissionMode": "auto",
	})
}

func emit(v any) {
	line, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Println(string(line))
}
