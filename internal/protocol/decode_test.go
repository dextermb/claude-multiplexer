package protocol

import (
	"errors"
	"strings"
	"testing"
)

func TestDecodeInit(t *testing.T) {
	line := `{"type":"system","subtype":"init","session_id":"abc","model":"claude-opus-5","cwd":"/tmp","tools":["Read","Bash"],"permissionMode":"auto"}`
	ev, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("Decode returned %v", err)
	}
	if !ev.IsInit() {
		t.Fatalf("expected an init event, got %q/%q", ev.Type, ev.Subtype)
	}
	if ev.Init.SessionID != "abc" || ev.Init.Model != "claude-opus-5" {
		t.Fatalf("unexpected init payload: %+v", ev.Init)
	}
	if len(ev.Init.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(ev.Init.Tools))
	}
}

func TestDecodeAssistantBlocks(t *testing.T) {
	line := `{"type":"assistant","session_id":"abc","message":{"role":"assistant","content":[{"type":"text","text":"hello"},{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"ls"}},{"type":"text","text":"world"}]}}`
	ev, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("Decode returned %v", err)
	}
	if got := ev.Text(); got != "hello\nworld" {
		t.Fatalf("Text() = %q", got)
	}
	if len(ev.Message.Content) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(ev.Message.Content))
	}
	tool := ev.Message.Content[1]
	if tool.Name != "Bash" || string(tool.Input) != `{"command":"ls"}` {
		t.Fatalf("unexpected tool block: %+v", tool)
	}
}

func TestDecodeToolResultWithStringContent(t *testing.T) {
	line := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"file.go"}]}}`
	ev, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("Decode returned %v", err)
	}
	block := ev.Message.Content[0]
	if block.ToolUseID != "t1" {
		t.Fatalf("unexpected tool_use_id %q", block.ToolUseID)
	}
	if got := block.Content.Text(); got != "file.go" {
		t.Fatalf("nested content = %q", got)
	}
}

func TestDecodeMarksAReplayedPrompt(t *testing.T) {
	line := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"hello"}]},"isReplay":true}`
	ev, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("Decode returned %v", err)
	}
	if !ev.IsReplay {
		t.Fatal("the replay marker was lost")
	}
	if ev.Message.Content.Text() != "hello" {
		t.Fatalf("text = %q", ev.Message.Content.Text())
	}

	toolResult := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"out"}]}}`
	ev, err = Decode([]byte(toolResult))
	if err != nil {
		t.Fatalf("Decode returned %v", err)
	}
	if ev.IsReplay {
		t.Fatal("a tool result must not look like a replayed prompt")
	}
}

func TestDecodeResult(t *testing.T) {
	line := `{"type":"result","subtype":"success","is_error":false,"duration_ms":1200,"num_turns":3,"result":"done","total_cost_usd":0.42,"session_id":"abc","usage":{"input_tokens":10,"output_tokens":20}}`
	ev, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("Decode returned %v", err)
	}
	if ev.Result.NumTurns != 3 || ev.Result.TotalCostUSD != 0.42 {
		t.Fatalf("unexpected result: %+v", ev.Result)
	}
	if ev.Result.Usage.OutputTokens != 20 {
		t.Fatalf("unexpected usage: %+v", ev.Result.Usage)
	}
}

func TestDecodeStreamEvent(t *testing.T) {
	line := `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"par"}}}`
	ev, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("Decode returned %v", err)
	}
	if ev.Delta.Kind != "content_block_delta" || ev.Delta.Text != "par" {
		t.Fatalf("unexpected delta: %+v", ev.Delta)
	}
}

func TestDecodeTaskStarted(t *testing.T) {
	line := `{"type":"system","subtype":"task_started","task_id":"bao4ntmse","tool_use_id":"toolu_01","description":"Sleep 20 seconds then echo done","task_type":"local_bash","session_id":"abc"}`
	ev, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("Decode returned %v", err)
	}
	if ev.Task == nil {
		t.Fatal("expected a Task payload")
	}
	if ev.Task.TaskID != "bao4ntmse" || ev.Task.TaskType != "local_bash" {
		t.Fatalf("unexpected task: %+v", ev.Task)
	}
	if ev.Task.Description != "Sleep 20 seconds then echo done" {
		t.Fatalf("unexpected description: %q", ev.Task.Description)
	}
	if ev.Task.Patch != nil {
		t.Fatalf("a start event has no patch: %+v", ev.Task.Patch)
	}
}

func TestDecodeTaskUpdated(t *testing.T) {
	line := `{"type":"system","subtype":"task_updated","task_id":"b0zll5o88","patch":{"status":"completed","end_time":1788506380063},"session_id":"abc"}`
	ev, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("Decode returned %v", err)
	}
	if ev.Task == nil || ev.Task.Patch == nil {
		t.Fatalf("expected a Task with a patch, got %+v", ev.Task)
	}
	if ev.Task.Patch.Status != "completed" || ev.Task.Patch.EndTime != 1788506380063 {
		t.Fatalf("unexpected patch: %+v", ev.Task.Patch)
	}
}

func TestDecodeTaskNotification(t *testing.T) {
	line := `{"type":"system","subtype":"task_notification","task_id":"bao4ntmse","tool_use_id":"toolu_01","status":"stopped","output_file":"","summary":"Sleep 20 seconds then echo done","session_id":"abc"}`
	ev, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("Decode returned %v", err)
	}
	if ev.Task == nil {
		t.Fatal("expected a Task payload")
	}
	if ev.Task.Status != "stopped" || ev.Task.Summary != "Sleep 20 seconds then echo done" {
		t.Fatalf("unexpected notification: %+v", ev.Task)
	}
}

func TestDecodeUnknownTypeSurvives(t *testing.T) {
	line := `{"type":"something_new_in_2027","payload":{"a":1}}`
	ev, err := Decode([]byte(line))
	if err != nil {
		t.Fatalf("an unknown type must not fail, got %v", err)
	}
	if ev.Type != "something_new_in_2027" {
		t.Fatalf("unexpected type %q", ev.Type)
	}
	if string(ev.Raw) != line {
		t.Fatalf("the raw line must be kept")
	}
}

func TestDecodeMalformedLine(t *testing.T) {
	ev, err := Decode([]byte("not json at all"))
	if !errors.Is(err, ErrNotJSON) {
		t.Fatalf("expected ErrNotJSON, got %v", err)
	}
	if string(ev.Raw) != "not json at all" {
		t.Fatalf("the raw line must be kept")
	}
}

func TestReaderSkipsBlankLinesAndKeepsOrder(t *testing.T) {
	input := strings.Join([]string{
		`{"type":"system","subtype":"init","session_id":"abc"}`,
		"",
		"   ",
		`{"type":"result","subtype":"success","num_turns":1}`,
	}, "\n") + "\n"

	r := NewReader(strings.NewReader(input))
	first, err := r.Next()
	if err != nil || !first.IsInit() {
		t.Fatalf("first event: %+v, %v", first, err)
	}
	second, err := r.Next()
	if err != nil || second.Type != TypeResult {
		t.Fatalf("second event: %+v, %v", second, err)
	}
	if _, err := r.Next(); err == nil {
		t.Fatal("expected EOF after the last line")
	}
}

func TestReaderHandlesLineLongerThanBuffer(t *testing.T) {
	long := strings.Repeat("a", 200<<10)
	input := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"` + long + `"}]}}` + "\n"

	r := NewReader(strings.NewReader(input))
	ev, err := r.Next()
	if err != nil {
		t.Fatalf("Next returned %v", err)
	}
	if len(ev.Text()) != len(long) {
		t.Fatalf("text length = %d, want %d", len(ev.Text()), len(long))
	}
}

func TestReaderTruncatesOversizeLine(t *testing.T) {
	input := `{"type":"assistant","text":"` + strings.Repeat("b", 4096) + `"}` + "\n" +
		`{"type":"result","subtype":"success"}` + "\n"

	r := NewReader(strings.NewReader(input))
	r.SetMaxLine(512)

	first, err := r.Next()
	if !errors.Is(err, ErrNotJSON) {
		t.Fatalf("expected ErrNotJSON for a truncated line, got %v", err)
	}
	if !first.Truncated {
		t.Fatal("expected Truncated to be set")
	}
	if len(first.Raw) > 512 {
		t.Fatalf("raw length = %d, want at most 512", len(first.Raw))
	}
	second, err := r.Next()
	if err != nil || second.Type != TypeResult {
		t.Fatalf("the reader must recover on the next line: %+v, %v", second, err)
	}
}

func TestReaderHandlesFinalLineWithoutNewline(t *testing.T) {
	r := NewReader(strings.NewReader(`{"type":"result","subtype":"success"}`))
	ev, err := r.Next()
	if err != nil || ev.Type != TypeResult {
		t.Fatalf("event: %+v, err: %v", ev, err)
	}
}
