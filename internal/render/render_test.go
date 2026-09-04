package render

import (
	"errors"
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

var errTest = errors.New("boom")

func decode(t *testing.T, line string) session.Event {
	t.Helper()
	ev, err := protocol.Decode([]byte(line))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return session.Event{Kind: session.KindProtocol, Protocol: ev}
}

func TestRenderInit(t *testing.T) {
	ev := decode(t, `{"type":"system","subtype":"init","session_id":"abc","model":"claude-opus-5","tools":["Read","Bash"],"permissionMode":"auto"}`)
	got := Renderer{}.Lines(ev)
	if len(got) != 1 || !strings.Contains(got[0].Text, "abc") || !strings.Contains(got[0].Text, "2 tools") {
		t.Fatalf("lines = %v", got)
	}
	if got[0].Class != ClassMeta {
		t.Fatalf("the init line must be meta, got %v", got[0].Class)
	}
}

func TestRenderTaskStarted(t *testing.T) {
	ev := decode(t, `{"type":"system","subtype":"task_started","task_id":"bao4ntmse","description":"Sleep 20 seconds then echo done","task_type":"local_bash"}`)
	got := Renderer{}.Lines(ev)
	if len(got) != 1 || got[0].Class != ClassMeta {
		t.Fatalf("lines = %v", got)
	}
	if got[0].Text != "⚙ started · Sleep 20 seconds then echo done" {
		t.Fatalf("text = %q", got[0].Text)
	}
}

func TestRenderTaskUpdatedShowsStatusWord(t *testing.T) {
	ev := decode(t, `{"type":"system","subtype":"task_updated","task_id":"b0zll5o88","patch":{"status":"completed","end_time":1}}`)
	got := Renderer{}.Lines(ev)
	if len(got) != 1 || got[0].Text != "⚙ done · b0zll5o88" {
		t.Fatalf("lines = %v", got)
	}
}

func TestRenderTaskNotificationIsSilent(t *testing.T) {
	ev := decode(t, `{"type":"system","subtype":"task_notification","task_id":"bao4ntmse","status":"stopped"}`)
	if got := (Renderer{}).Lines(ev); got != nil {
		t.Fatalf("a notification adds no pane line, got %v", got)
	}
}

func TestEveryLineCarriesItsClass(t *testing.T) {
	cases := []struct {
		name  string
		event session.Event
		want  []Class
	}{
		{
			name:  "assistant text and a tool call",
			event: decode(t, `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"},{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}`),
			want:  []Class{ClassText, ClassToolUse},
		},
		{
			name:  "a tool result",
			event: decode(t, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"out"}]}}`),
			want:  []Class{ClassToolResult},
		},
		{
			name:  "the turn result",
			event: decode(t, `{"type":"result","subtype":"success","num_turns":1}`),
			want:  []Class{ClassMeta},
		},
		{
			name:  "a stderr line",
			event: session.Event{Kind: session.KindStderr, Line: "warning: something"},
			want:  []Class{ClassStderr},
		},
		{
			name:  "a failure",
			event: session.Event{Kind: session.KindError, Err: errTest},
			want:  []Class{ClassError},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Renderer{}.Lines(tc.event)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d lines, want %d: %v", len(got), len(tc.want), got)
			}
			for i := range tc.want {
				if got[i].Class != tc.want[i] {
					t.Errorf("line %d has class %v, want %v", i, got[i].Class, tc.want[i])
				}
			}
		})
	}
}

func TestStateLinesAreMetaAndOnlyShownWhenVerbose(t *testing.T) {
	ev := session.Event{Kind: session.KindState, Prev: session.StateIdle, State: session.StateBusy}
	if got := (Renderer{}).Lines(ev); got != nil {
		t.Fatalf("lines = %v, want none", got)
	}
	got := Renderer{Verbose: true}.Lines(ev)
	if len(got) != 1 || got[0].Class != ClassMeta {
		t.Fatalf("lines = %v", got)
	}
}

func TestTextReturnsThePlainStrings(t *testing.T) {
	lines := []Line{{Class: ClassText, Text: "one"}, {Class: ClassMeta, Text: "two"}}
	got := Text(lines)
	if len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("Text = %q", got)
	}
}

func TestToolResultCarriesTheWholeBodyAndASummary(t *testing.T) {
	ev := decode(t, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"one\ntwo\nthree"}]}}`)
	got := Renderer{}.Lines(ev)
	if len(got) != 1 {
		t.Fatalf("lines = %v", got)
	}
	if got[0].Text != "← one\ntwo\nthree" {
		t.Fatalf("text = %q", got[0].Text)
	}
	if got[0].Summary != "← 3 lines" {
		t.Fatalf("summary = %q", got[0].Summary)
	}
}

func TestToolResultLeavesTheSummaryEmptyWhenItIsOneShortLine(t *testing.T) {
	ev := decode(t, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"short"}]}}`)
	got := Renderer{}.Lines(ev)
	if len(got) != 1 || got[0].Summary != "" {
		t.Fatalf("summary should be empty, got %q", got[0].Summary)
	}
	if got[0].Text != "← short" {
		t.Fatalf("text = %q", got[0].Text)
	}
}

func TestToolResultSummariesALongSingleLine(t *testing.T) {
	long := strings.Repeat("x", 300)
	ev := decode(t, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"`+long+`"}]}}`)
	got := Renderer{ToolWidth: 20}.Lines(ev)
	if len(got) != 1 {
		t.Fatalf("lines = %v", got)
	}
	if !strings.HasSuffix(got[0].Summary, "…") {
		t.Fatalf("a clipped summary must end in an ellipsis, got %q", got[0].Summary)
	}
	if got[0].Text != "← "+long {
		t.Fatalf("text = %q, want the whole line", got[0].Text)
	}
}

func TestAPromptMarksItsContinuationLines(t *testing.T) {
	got := PromptLines("one\ntwo\nthree")
	if len(got) != 3 {
		t.Fatalf("lines = %v", got)
	}
	if got[0].Cont {
		t.Error("the first line of a prompt starts a block")
	}
	if !got[1].Cont || !got[2].Cont {
		t.Error("every later line of a prompt continues that block")
	}
}

func TestBashOutputIsABlockUnderItsCommand(t *testing.T) {
	got := BashLines("ls", "one\ntwo", nil)
	if len(got) != 3 {
		t.Fatalf("lines = %v", got)
	}
	if got[0].Cont || got[1].Cont {
		t.Error("the command and the first output line each start a block")
	}
	if !got[2].Cont {
		t.Error("a later output line continues the output block")
	}
}

func TestRenderToolUseUsesTheCommand(t *testing.T) {
	ev := decode(t, `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"checking"},{"type":"tool_use","name":"Bash","input":{"command":"ls -la","description":"list"}}]}}`)
	got := Text(Renderer{}.Lines(ev))
	want := []string{"checking", "→ Bash ls -la"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("lines = %q, want %q", got, want)
	}
}

func TestPrintFoldsAMultiLineToolResult(t *testing.T) {
	ev := decode(t, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"a\nb\nc"}]}}`)
	if got := Print(Renderer{}.Lines(ev)); len(got) != 1 || got[0] != "← 3 lines" {
		t.Fatalf("lines = %q", got)
	}
	if got := Print(Renderer{Verbose: true}.Lines(ev)); len(got) != 1 || got[0] != "← a\nb\nc" {
		t.Fatalf("verbose lines = %q", got)
	}
}

func TestRenderAReplayedPrompt(t *testing.T) {
	ev := decode(t, `{"type":"user","isReplay":true,"message":{"role":"user","content":[{"type":"text","text":"first line\nsecond line"}]}}`)
	got := Renderer{}.Lines(ev)
	if len(got) != 2 {
		t.Fatalf("lines = %v, want 2", got)
	}
	if got[0].Text != "› first line" || got[1].Text != "  second line" {
		t.Fatalf("lines = %v", Text(got))
	}
	for i, line := range got {
		if line.Class != ClassPrompt {
			t.Errorf("line %d has class %v, want ClassPrompt", i, line.Class)
		}
	}
}

func TestRenderResult(t *testing.T) {
	ev := decode(t, `{"type":"result","subtype":"success","duration_ms":1234,"num_turns":2,"total_cost_usd":0.1234,"usage":{"input_tokens":10,"output_tokens":20}}`)
	got := Text(Renderer{}.Lines(ev))
	if len(got) != 1 {
		t.Fatalf("lines = %q", got)
	}
	for _, want := range []string{"✓ success", "1.2s", "$0.1234", "2 turns", "10 in / 20 out"} {
		if !strings.Contains(got[0], want) {
			t.Errorf("%q does not contain %q", got[0], want)
		}
	}
}

func TestRenderMarksAnErrorResult(t *testing.T) {
	ev := decode(t, `{"type":"result","subtype":"success","is_error":true,"duration_ms":338,"num_turns":1}`)
	got := Text(Renderer{}.Lines(ev))
	if len(got) != 1 || !strings.HasPrefix(got[0], "✗ error") {
		t.Fatalf("lines = %q", got)
	}
}

func TestRenderClipsLongToolInput(t *testing.T) {
	long := strings.Repeat("x", 300)
	ev := decode(t, `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"`+long+`"}}]}}`)
	got := Text(Renderer{ToolWidth: 20}.Lines(ev))
	if len(got) != 1 || len([]rune(got[0])) != len([]rune("→ Bash "))+21 {
		t.Fatalf("lines = %q", got)
	}
}
