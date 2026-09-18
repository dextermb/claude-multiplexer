package manager

import (
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

func streamDelta(text, parent string) session.Event {
	return session.Event{
		Kind: session.KindProtocol,
		Protocol: protocol.Event{
			Type:            protocol.TypeStreamEvent,
			ParentToolUseID: parent,
			Delta:           &protocol.Delta{Text: text},
		},
	}
}

// A local agent streams its partial deltas on the parent output. The partial
// line of the pane is the session's own, so an agent delta must not enter it.
func TestPartialIgnoresAgentDeltas(t *testing.T) {
	item := &entry{}

	if got := trackPartial(item, streamDelta("secret", "toolu_agent")); got != "" {
		t.Fatalf("an agent delta entered the partial line: %q", got)
	}
	if got := trackPartial(item, streamDelta("echo: ", "")); got != "echo: " {
		t.Fatalf("the session partial line = %q, want %q", got, "echo: ")
	}
	if got := trackPartial(item, streamDelta("more secret", "toolu_agent")); got != "echo: " {
		t.Fatalf("an agent delta changed the partial line: %q", got)
	}

	reset := session.Event{Kind: session.KindProtocol, Protocol: protocol.Event{Type: protocol.TypeAssistant}}
	if got := trackPartial(item, reset); got != "" {
		t.Fatalf("the session turn did not reset the partial line: %q", got)
	}
}
