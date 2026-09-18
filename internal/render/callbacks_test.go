package render

import "testing"

func userEvent(t *testing.T, content string) []Line {
	t.Helper()
	return Renderer{}.Lines(decode(t, `{"type":"user","message":{"role":"user","content":[{"type":"text","text":`+content+`}]}}`))
}

func TestCallbackTaskNotification(t *testing.T) {
	got := userEvent(t, `"<task-notification>\n<task-id>b67xcvk3g</task-id>\n<status>stopped</status>\n<summary>No completion record was found.</summary>\n</task-notification>"`)
	if len(got) != 1 || got[0].Class != ClassMeta {
		t.Fatalf("lines = %v", got)
	}
	if got[0].Text != "⚙ killed · No completion record was found." {
		t.Fatalf("text = %q", got[0].Text)
	}
}

func TestCallbackTaskNotificationFallsBackToID(t *testing.T) {
	got := userEvent(t, `"<task-notification>\n<task-id>b1</task-id>\n<status>completed</status>\n</task-notification>"`)
	if len(got) != 1 || got[0].Text != "⚙ done · b1" {
		t.Fatalf("lines = %v", got)
	}
}

func TestCallbackLocalCommandStdout(t *testing.T) {
	got := userEvent(t, `"<local-command-stdout>Authentication successful.</local-command-stdout>"`)
	if len(got) != 1 || got[0].Class != ClassMeta || got[0].Text != "← Authentication successful." {
		t.Fatalf("lines = %v", got)
	}
}

func TestCallbackLocalCommandCaveatIsSilent(t *testing.T) {
	got := userEvent(t, `"<local-command-caveat>Caveat: local commands.</local-command-caveat>"`)
	if len(got) != 0 {
		t.Fatalf("a caveat adds no line, got %v", got)
	}
}

func TestCallbackSlashCommand(t *testing.T) {
	got := userEvent(t, `"<command-message>reload-skills</command-message>\n<command-name>/reload-skills</command-name>\n<command-args></command-args>"`)
	if len(got) != 1 || got[0].Class != ClassMeta || got[0].Text != "» /reload-skills" {
		t.Fatalf("lines = %v", got)
	}
}

func TestCallbackSlashCommandWithArgs(t *testing.T) {
	got := userEvent(t, `"<command-name>/loop</command-name>\n<command-args>5m /foo</command-args>"`)
	if len(got) != 1 || got[0].Text != "» /loop 5m /foo" {
		t.Fatalf("lines = %v", got)
	}
}

func TestCallbackStandaloneReminder(t *testing.T) {
	got := userEvent(t, `"<system-reminder>Do the thing.</system-reminder>"`)
	if len(got) != 1 || got[0].Class != ClassMeta || got[0].Text != "· system reminder" {
		t.Fatalf("lines = %v", got)
	}
}

func TestReplayPromptPeelsTrailingReminder(t *testing.T) {
	line := `{"type":"user","isReplay":true,"message":{"role":"user","content":[{"type":"text","text":"real prompt\n<system-reminder>a reminder</system-reminder>"}]}}`
	got := Renderer{}.Lines(decode(t, line))
	if len(got) != 2 {
		t.Fatalf("lines = %v, want 2", Text(got))
	}
	if got[0].Class != ClassPrompt || got[0].Text != "› real prompt" {
		t.Fatalf("prompt line = %v", got[0])
	}
	if got[1].Class != ClassMeta || got[1].Text != "· system reminder" {
		t.Fatalf("reminder line = %v", got[1])
	}
}

func TestPlainPromptIsNotACallback(t *testing.T) {
	got := Renderer{}.Lines(decode(t, `{"type":"user","isReplay":true,"message":{"role":"user","content":[{"type":"text","text":"just a prompt"}]}}`))
	if len(got) != 1 || got[0].Class != ClassPrompt || got[0].Text != "› just a prompt" {
		t.Fatalf("lines = %v", got)
	}
}

func TestIsPromptEcho(t *testing.T) {
	cases := []struct {
		name string
		line string
		want bool
	}{
		{"a prompt replay", `{"type":"user","isReplay":true,"message":{"role":"user","content":[{"type":"text","text":"do the thing"}]}}`, true},
		{"a slash command replay", `{"type":"user","isReplay":true,"message":{"role":"user","content":[{"type":"text","text":"<command-name>/review</command-name>"}]}}`, true},
		{"a task notification replay", `{"type":"user","isReplay":true,"message":{"role":"user","content":[{"type":"text","text":"<task-notification>\n<status>completed</status>\n</task-notification>"}]}}`, false},
		{"a reminder replay", `{"type":"user","isReplay":true,"message":{"role":"user","content":[{"type":"text","text":"<system-reminder>note</system-reminder>"}]}}`, false},
		{"a live prompt, not a replay", `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"do the thing"}]}}`, false},
		{"an assistant message", `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"an answer"}]}}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsPromptEcho(decode(t, tc.line).Protocol); got != tc.want {
				t.Fatalf("IsPromptEcho = %v, want %v", got, tc.want)
			}
		})
	}
}
