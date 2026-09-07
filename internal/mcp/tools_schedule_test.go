package mcp_test

import (
	"context"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

func controlClient(t *testing.T, sessions mcp.Sessions) *sdk.ClientSession {
	t.Helper()
	server := startServer(t, sessions)
	token, err := server.Register("boss", true)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return connect(t, server, token)
}

func TestCreateScheduleToolCreatesTheSchedule(t *testing.T) {
	sessions := newFakeSessions()
	client := controlClient(t, sessions)

	result := call(t, client, mcp.ToolCreateSchedule, map[string]any{
		"cron":   "*/5 * * * *",
		"dir":    "/tmp",
		"prompt": "poll the site",
		"name":   "poll",
	})
	if result.IsError {
		t.Fatalf("create_schedule failed: %s", resultText(result))
	}
	if _, ok := sessions.schedules["poll"]; !ok {
		t.Fatalf("schedule was not created: %+v", sessions.schedules)
	}
}

func TestCreateScheduleToolNeedsCronDirPrompt(t *testing.T) {
	sessions := newFakeSessions()
	client := controlClient(t, sessions)

	cases := []map[string]any{
		{"dir": "/tmp", "prompt": "x"},
		{"cron": "* * * * *", "prompt": "x"},
		{"cron": "* * * * *", "dir": "/tmp"},
	}
	for _, args := range cases {
		result := call(t, client, mcp.ToolCreateSchedule, args)
		if !result.IsError {
			t.Fatalf("create_schedule with %v did not fail", args)
		}
	}
	if len(sessions.schedules) != 0 {
		t.Fatalf("a bad create wrote a schedule: %+v", sessions.schedules)
	}
}

func TestListSchedulesToolReturnsTheSchedules(t *testing.T) {
	sessions := newFakeSessions()
	sessions.schedules["poll"] = mcp.Schedule{Name: "poll", Cron: "* * * * *", Enabled: true}
	client := controlClient(t, sessions)

	result := call(t, client, mcp.ToolListSchedules, map[string]any{})
	if result.IsError {
		t.Fatalf("list_schedules failed: %s", resultText(result))
	}
	if text := resultText(result); text == "" {
		t.Fatal("list_schedules returned no text")
	}
}

func TestRunScheduleToolRunsTheSchedule(t *testing.T) {
	sessions := newFakeSessions()
	sessions.schedules["poll"] = mcp.Schedule{Name: "poll", Enabled: true}
	client := controlClient(t, sessions)

	result := call(t, client, mcp.ToolRunSchedule, map[string]any{"name": "poll"})
	if result.IsError {
		t.Fatalf("run_schedule failed: %s", resultText(result))
	}
	if len(sessions.scheduleRuns) != 1 {
		t.Fatalf("runs = %d, want 1", len(sessions.scheduleRuns))
	}
}

func TestDeleteScheduleToolRemovesTheSchedule(t *testing.T) {
	sessions := newFakeSessions()
	sessions.schedules["poll"] = mcp.Schedule{Name: "poll"}
	client := controlClient(t, sessions)

	result := call(t, client, mcp.ToolDeleteSchedule, map[string]any{"name": "poll"})
	if result.IsError {
		t.Fatalf("delete_schedule failed: %s", resultText(result))
	}
	if _, ok := sessions.schedules["poll"]; ok {
		t.Fatal("schedule was not removed")
	}
}

func TestScheduleToolsNeedControl(t *testing.T) {
	server := startServer(t, newFakeSessions())
	token, err := server.Register("plain", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := client.CallTool(ctx, &sdk.CallToolParams{
		Name:      mcp.ToolCreateSchedule,
		Arguments: map[string]any{"cron": "* * * * *", "dir": "/tmp", "prompt": "x"},
	}); err == nil {
		t.Fatal("a session without control called create_schedule")
	}
}
