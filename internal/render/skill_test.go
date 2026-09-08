package render

import (
	"strings"
	"testing"
)

func track(t *testing.T, tr *SkillTracker, line string) []Line {
	t.Helper()
	ev := decode(t, line)
	return tr.Track(ev.Protocol, Renderer{}.Lines(ev))
}

const (
	skillCall   = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Skill","input":{"skill":"agent-browser"}}]}}`
	skillResult = `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"Launching skill: agent-browser"}]}}`
	skillDump   = `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"Base directory for this skill: /x\n\n# agent-browser\nline three"}]}}`
	bareDump    = `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"# Update Config Skill\n\nModify the config."}]}}`
	assistant   = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"On it."}]}}`
)

func TestSkillTrackerMarksDump(t *testing.T) {
	var tr SkillTracker
	track(t, &tr, skillCall)
	track(t, &tr, skillResult)
	got := track(t, &tr, skillDump)
	if len(got) != 1 {
		t.Fatalf("dump lines = %v", got)
	}
	if got[0].Class != ClassSkill {
		t.Fatalf("dump class = %v, want ClassSkill", got[0].Class)
	}
	if !strings.Contains(got[0].Summary, "skill") {
		t.Fatalf("dump summary = %q, want a skill summary", got[0].Summary)
	}
}

// A skill without bundled files dumps its raw markdown, with no base-directory
// line, so the tracker cannot key on the text. It keys on the Skill call before.
func TestSkillTrackerMarksBareDump(t *testing.T) {
	var tr SkillTracker
	track(t, &tr, skillCall)
	track(t, &tr, skillResult)
	got := track(t, &tr, bareDump)
	if len(got) != 1 || got[0].Class != ClassSkill {
		t.Fatalf("bare dump = %v, want one ClassSkill line", got)
	}
}

func TestSkillTrackerHoldsAcrossResult(t *testing.T) {
	var tr SkillTracker
	track(t, &tr, skillCall)
	got := track(t, &tr, skillResult)
	if len(got) != 1 || got[0].Class != ClassToolResult {
		t.Fatalf("result = %v, want the tool result unchanged", got)
	}
	if !tr.armed {
		t.Fatal("the tracker must stay armed across the tool result")
	}
}

func TestSkillTrackerDisarmsOnAssistant(t *testing.T) {
	var tr SkillTracker
	track(t, &tr, skillCall)
	track(t, &tr, assistant)
	got := track(t, &tr, skillDump)
	if got[0].Class != ClassText {
		t.Fatalf("class = %v, want ClassText because an assistant message disarmed", got[0].Class)
	}
}

func TestSkillTrackerIgnoresPromptReplay(t *testing.T) {
	var tr SkillTracker
	track(t, &tr, skillCall)
	track(t, &tr, skillResult)
	replay := `{"type":"user","isReplay":true,"message":{"role":"user","content":[{"type":"text","text":"a real prompt"}]}}`
	got := track(t, &tr, replay)
	if got[0].Class != ClassPrompt {
		t.Fatalf("class = %v, want ClassPrompt; a replay is not a skill dump", got[0].Class)
	}
}
