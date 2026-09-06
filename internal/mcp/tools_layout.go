package mcp

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) addLayoutTools(server *sdk.Server, caller string) {
	sdk.AddTool(server, &sdk.Tool{
		Name: ToolListLayouts,
		Description: "The named interface layouts, the global active layout, and the layout of one session. " +
			"A layout sets the prompt bar height, the session list width, the task panel width, and the diff panel width. " +
			"Give a session name, or leave it empty for this session.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listLayoutsIn) (*sdk.CallToolResult, LayoutList, error) {
		target, err := targetOrSelf(in.Session, caller)
		if err != nil {
			return nil, LayoutList{}, err
		}
		out, err := s.sessions.Layouts(target)
		if err != nil {
			return nil, LayoutList{}, err
		}
		return nil, out, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolSaveLayout,
		Description: "Create or replace a named layout. " +
			"It captures the current dimensions of this session, so a call with only a name saves what the interface shows now. " +
			"Give a dimension to override the captured one, so the same call edits one field of an existing layout. " +
			"It writes the settings file of the multiplexer.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in saveLayoutIn) (*sdk.CallToolResult, saveLayoutOut, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return nil, saveLayoutOut{}, ErrNoLayout
		}
		dims := LayoutDims{
			PromptMin:    in.PromptMin,
			PromptMax:    in.PromptMax,
			SidebarWidth: in.SidebarWidth,
			TaskWidth:    in.TaskWidth,
			DiffWidth:    in.DiffWidth,
		}
		if err := checkDims(dims); err != nil {
			return nil, saveLayoutOut{}, err
		}
		path, err := s.sessions.SaveLayout(name, dims, caller)
		if err != nil {
			return nil, saveLayoutOut{}, err
		}
		return nil, saveLayoutOut{OK: true, Path: path, Name: name,
			Message: "the layout " + name + " is saved, in " + path}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolDeleteLayout,
		Description: "Remove a named layout from the settings file of the multiplexer. " +
			"A session or the global default that named it takes the built-in defaults again.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in deleteLayoutIn) (*sdk.CallToolResult, deleteLayoutOut, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return nil, deleteLayoutOut{}, ErrNoLayout
		}
		path, changed, err := s.sessions.DeleteLayout(name, caller)
		if err != nil {
			return nil, deleteLayoutOut{}, err
		}
		message := "the layout " + name + " was not there, in " + path
		if changed {
			message = "the layout " + name + " is deleted, in " + path
		}
		return nil, deleteLayoutOut{OK: true, Path: path, Changed: changed, Message: message}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolSetLayout,
		Description: "Activate a layout. " +
			"The scope 'session' sets this session, so it overrides the global default. " +
			"The scope 'all' sets the global default for every session. " +
			"The layout must exist.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in setLayoutIn) (*sdk.CallToolResult, setLayoutOut, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return nil, setLayoutOut{}, ErrNoLayout
		}
		scope, err := scopeOrSession(in.Scope)
		if err != nil {
			return nil, setLayoutOut{}, err
		}
		path, err := s.sessions.SetLayout(name, scope, caller)
		if err != nil {
			return nil, setLayoutOut{}, err
		}
		where := caller
		if scope == ScopeAll {
			where = "every session"
		}
		return nil, setLayoutOut{OK: true, Path: path, Scope: scope,
			Message: "the layout " + name + " is active for " + where}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolUnsetLayout,
		Description: "Take a layout off. " +
			"The scope 'session' clears this session, so it takes the global default again. " +
			"The scope 'all' clears the global default, so every session takes the built-in defaults.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in unsetLayoutIn) (*sdk.CallToolResult, unsetLayoutOut, error) {
		scope, err := scopeOrSession(in.Scope)
		if err != nil {
			return nil, unsetLayoutOut{}, err
		}
		path, changed, err := s.sessions.UnsetLayout(scope, caller)
		if err != nil {
			return nil, unsetLayoutOut{}, err
		}
		where := caller
		if scope == ScopeAll {
			where = "the global default"
		}
		message := where + " had no layout"
		if changed {
			message = where + " takes no layout now"
		}
		return nil, unsetLayoutOut{OK: true, Path: path, Scope: scope, Changed: changed, Message: message}, nil
	})
}

func scopeOrSession(scope string) (string, error) {
	switch strings.TrimSpace(scope) {
	case "", ScopeSession:
		return ScopeSession, nil
	case ScopeAll:
		return ScopeAll, nil
	default:
		return "", ErrBadScope
	}
}

func checkDims(dims LayoutDims) error {
	for _, v := range []*int{dims.PromptMin, dims.PromptMax, dims.SidebarWidth, dims.TaskWidth, dims.DiffWidth} {
		if v != nil && *v < 1 {
			return ErrBadDim
		}
	}
	return nil
}
