package mcp

import (
	"context"
	"strconv"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

func (s *Server) addConfigTools(server *sdk.Server, caller string) {
	sdk.AddTool(server, &sdk.Tool{
		Name: ToolConfigPath,
		Description: "The settings files of the multiplexer, in the order they are read. " +
			"'active' is the file that is read now, and it is absent when there is none. " +
			"'target' is the file that " + ToolSetEditor + " and " + ToolSetBlockCap + " write.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, ConfigPath, error) {
		return nil, s.sessions.ConfigPath(), nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolTemplatePath,
		Description: "The directories one session reads a preset prompt from, in the order they are read. " +
			"The last directory wins when two hold the same name. Give a session name, or leave it empty for this session.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in templatePathIn) (*sdk.CallToolResult, TemplatePath, error) {
		target, err := targetOrSelf(in.Session, caller)
		if err != nil {
			return nil, TemplatePath{}, err
		}
		out, err := s.sessions.TemplatePath(target)
		if err != nil {
			return nil, TemplatePath{}, err
		}
		return nil, out, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolSetEditor,
		Description: "Set the editor the human opens a session directory with, and whether it draws in the terminal. " +
			"It writes the settings file of the multiplexer, and makes that file when there is none. " +
			"Give the editor, the terminal flag, or both.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in setEditorIn) (*sdk.CallToolResult, setEditorOut, error) {
		editor := strings.TrimSpace(in.Editor)
		if editor == "" && in.Terminal == nil {
			return nil, setEditorOut{}, ErrNoEditor
		}
		path, err := s.sessions.SetEditor(editor, in.Terminal, caller)
		if err != nil {
			return nil, setEditorOut{}, err
		}
		return nil, setEditorOut{OK: true, Path: path, Message: editorMessage(editor, in.Terminal, path)}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolUnsetEditor,
		Description: "Take the editor out of the settings file of the multiplexer, so the human falls back to $EDITOR and to the settings of Claude Code. " +
			"Clear the editor, the terminal flag, or both.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in unsetEditorIn) (*sdk.CallToolResult, unsetEditorOut, error) {
		path, changed, err := s.sessions.UnsetEditor(strings.TrimSpace(in.Field), caller)
		if err != nil {
			return nil, unsetEditorOut{}, err
		}
		return nil, unsetEditorOut{OK: true, Path: path, Changed: changed, Message: clearedMessage(in.Field, changed, path)}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolSetBlockCap,
		Description: "Set how much of one block the session pane draws before it caps it. " +
			"A block is one piece of content: a prompt, one message, one tool result, or the output of a ! command. " +
			"Give 'type' to cap one kind of block (prompt, message, tool, meta, bash, or error), or no type to set the default for the rest. " +
			"The question_option and question_description types cap the question modal, not the pane, and default to 2 lines. " +
			"Give 'rows' to draw that many rows (0 draws only the marker), or 'unlimited' to never cap. " +
			"The human opens the rest of a capped block in the pane. It writes the settings file of the multiplexer.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in setBlockCapIn) (*sdk.CallToolResult, setBlockCapOut, error) {
		bucket := strings.TrimSpace(in.Type)
		if bucket != "" && !config.ValidBucket(bucket) {
			return nil, setBlockCapOut{}, ErrBadType
		}
		if in.Unlimited && in.Rows != nil {
			return nil, setBlockCapOut{}, ErrCapBoth
		}
		var rows *int
		switch {
		case in.Unlimited:
			if bucket == "" {
				zero := 0
				rows = &zero
			}
		case in.Rows != nil:
			if *in.Rows < 0 {
				return nil, setBlockCapOut{}, ErrBadCap
			}
			value := *in.Rows
			rows = &value
		default:
			return nil, setBlockCapOut{}, ErrBadCap
		}
		path, err := s.sessions.SetBlockCap(bucket, rows, caller)
		if err != nil {
			return nil, setBlockCapOut{}, err
		}
		return nil, setBlockCapOut{OK: true, Path: path, Type: bucket, Rows: rows, Message: blockCapMessage(bucket, rows, path)}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolUnsetBlockCap,
		Description: "Take a block cap out of the settings file of the multiplexer. " +
			"Give 'type' to clear one kind of block, so it takes the default again, or no type to clear the default of " +
			strconv.Itoa(config.DefaultBlockCap) + " rows.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in unsetBlockCapIn) (*sdk.CallToolResult, unsetBlockCapOut, error) {
		bucket := strings.TrimSpace(in.Type)
		if bucket != "" && !config.ValidBucket(bucket) {
			return nil, unsetBlockCapOut{}, ErrBadType
		}
		path, changed, err := s.sessions.UnsetBlockCap(bucket, caller)
		if err != nil {
			return nil, unsetBlockCapOut{}, err
		}
		return nil, unsetBlockCapOut{OK: true, Path: path, Changed: changed, Message: blockCapClearedMessage(bucket, changed, path)}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolSetWorkingDir,
		Description: "Say which directory this session works in now, so the human opens that one instead of the directory the session started in. " +
			"Call it after you move into a worktree. The directory must exist.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in setWorkingDirIn) (*sdk.CallToolResult, workingDirOut, error) {
		path := strings.TrimSpace(in.Path)
		if path == "" {
			return nil, workingDirOut{}, ErrNoDir
		}
		full, err := s.sessions.SetWorkingDir(path, caller)
		if err != nil {
			return nil, workingDirOut{}, err
		}
		return nil, workingDirOut{OK: true, Path: full, Changed: true,
			Message: caller + " works in " + full + " now"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolUnsetWorkingDir,
		Description: "Take the working directory off this session, so the human opens the directory the session started in again. " +
			"Call it after you collapse a worktree.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, workingDirOut, error) {
		changed, err := s.sessions.UnsetWorkingDir(caller)
		if err != nil {
			return nil, workingDirOut{}, err
		}
		message := caller + " had no working directory"
		if changed {
			message = caller + " works in the directory it started in again"
		}
		return nil, workingDirOut{OK: true, Changed: changed, Message: message}, nil
	})
}

func editorMessage(editor string, terminal *bool, path string) string {
	var parts []string
	if editor != "" {
		parts = append(parts, "the editor is now "+editor)
	}
	if terminal != nil {
		kind := "a window editor"
		if *terminal {
			kind = "a terminal editor"
		}
		parts = append(parts, "it is "+kind)
	}
	return strings.Join(parts, ", ") + ", in " + path
}

func blockCapMessage(bucket string, rows *int, path string) string {
	if bucket == "" {
		if rows == nil || *rows == 0 {
			return "the pane now caps no block, in " + path
		}
		return "one block now draws " + strconv.Itoa(*rows) + " rows before the pane caps it, in " + path
	}
	if rows == nil {
		return "the pane now caps no " + bucket + " block, in " + path
	}
	if *rows == 0 {
		return "a " + bucket + " block now draws only the marker, in " + path
	}
	return "a " + bucket + " block now draws " + strconv.Itoa(*rows) + " rows before the pane caps it, in " + path
}

func blockCapClearedMessage(bucket string, changed bool, path string) string {
	if bucket != "" {
		if !changed {
			return "the " + bucket + " block cap was not set in " + path
		}
		return "the " + bucket + " block cap is no longer set in " + path + ", so a " + bucket + " block takes the default"
	}
	if !changed {
		return "the block cap was not set in " + path
	}
	return "the block cap is no longer set in " + path + ", so one block draws " +
		strconv.Itoa(config.DefaultBlockCap) + " rows"
}

func clearedMessage(field string, changed bool, path string) string {
	what := "the editor settings"
	switch strings.TrimSpace(field) {
	case "editor":
		what = "the editor"
	case "terminal":
		what = "the terminal flag of the editor"
	}
	if !changed {
		return what + " was not set in " + path
	}
	return what + " is no longer set in " + path
}
