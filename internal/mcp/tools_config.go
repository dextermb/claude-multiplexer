package mcp

import (
	"context"
	"encoding/json"
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
		Name: ToolConfigKeys,
		Description: "The settings keys " + ToolSetConfig + " and " + ToolUnsetConfig + " accept, as dot paths, each with its JSON type. " +
			"A '<key>' segment stands for a name you choose, such as a block bucket in 'blockCaps.<key>' or a layout in 'layouts.<key>.sidebarSize'. " +
			"Read this to name a real key before you set one.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, ConfigKeys, error) {
		return nil, ConfigKeys{Keys: config.Keys()}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolTemplatePath,
		Description: "The directories one session reads a preset prompt from, in the order they are read. " +
			"The last directory wins when two hold the same name. Give a session name, or leave it empty for this session.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in templatePathIn) (*sdk.CallToolResult, TemplatePath, error) {
		out, err := templatePath(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolSetConfig,
		Description: "Set one value in the settings file of the multiplexer by a dot path, such as 'editor', 'blockCap', 'blockCaps.tool', or 'layouts.wide.sidebarSize'. " +
			"Give 'value' as the JSON value to write. It writes the settings file, and makes that file when there is none. " +
			"It rejects a key or a type the settings do not allow.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in setConfigIn) (*sdk.CallToolResult, setConfigOut, error) {
		out, err := setConfig(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolUnsetConfig,
		Description: "Remove one value from the settings file of the multiplexer by a dot path, such as 'blockCap' or 'layouts.wide', so that key takes its default again.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in unsetConfigIn) (*sdk.CallToolResult, unsetConfigOut, error) {
		out, err := unsetConfig(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolGetKeybindings,
		Description: "The resolved keybindings of the interface: every action, the keys it answers to now, and whether the settings changed it from the default. " +
			"Give 'action' to filter to one action, such as 'session.rename', or to one context, such as 'session'. " +
			"Read this to see a binding before you change it with " + ToolSetKeybinding + ".",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in getKeybindingsIn) (*sdk.CallToolResult, getKeybindingsOut, error) {
		return nil, getKeybindings(s.sessions, in), nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolSetKeybinding,
		Description: "Bind one or more keys to an action of the interface, such as 'session.rename' or 'global.quit'. " +
			"Give 'action' as '<context>.<action>' and 'keys' as the keys, such as [\"N\"] or [\"n\",\"ctrl+n\"]. " +
			"It refuses a reserved key (?, esc, ctrl+c), an unknown action, or a clash with another binding. " +
			"It answers with 'warning' when the new binding takes a key a default action used, so you can rebind that action too. " +
			"Read " + ToolConfigKeys + " for the 'keybindings.<context>.<action>' paths.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in setKeybindingIn) (*sdk.CallToolResult, setKeybindingOut, error) {
		out, err := setKeybinding(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolResetKeybinding,
		Description: "Clear one keybinding, so the action takes its built-in keys again. Give 'action' as '<context>.<action>', such as 'session.rename'.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in resetKeybindingIn) (*sdk.CallToolResult, resetKeybindingOut, error) {
		out, err := resetKeybinding(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolGetCommands,
		Description: "The key commands of the interface: each trigger, its label, and its script. " +
			"A command binds a key trigger to a script, so a press runs the script. " +
			"Read this to see a command before you change it with " + ToolAddCommand + " or " + ToolRemoveCommand + ".",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, getCommandsOut, error) {
		return nil, getCommands(s.sessions), nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolAddCommand,
		Description: "Bind a key trigger to a script, so a press runs the script. " +
			"Give 'keys' as the trigger (a standalone key such as 'ctrl+g', or a leader and a second key such as 'b o'), 'label' as a name, and 'script' as the file. " +
			"The script runs off the main loop, reads the selected session as JSON on stdin, and its first stdout line shows as a notice. " +
			"A command with the same label is replaced. It validates the whole set before it writes, so it refuses a reserved key (?, esc, ctrl+c), a bad trigger, or a clash with a key of the interface.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in addCommandIn) (*sdk.CallToolResult, addCommandOut, error) {
		out, err := addCommand(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRemoveCommand,
		Description: "Remove a key command by its label, so its trigger runs nothing again. Give 'label' as the name of the command to remove.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in removeCommandIn) (*sdk.CallToolResult, removeCommandOut, error) {
		out, err := removeCommand(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolSetEditor,
		Description: "Set the editor the human opens a session directory with, and whether it draws in the terminal. " +
			"It writes the settings file of the multiplexer, and makes that file when there is none. " +
			"Give the editor, the terminal flag, or both.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in setEditorIn) (*sdk.CallToolResult, setEditorOut, error) {
		out, err := setEditor(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolUnsetEditor,
		Description: "Take the editor out of the settings file of the multiplexer, so the human falls back to $EDITOR and to the settings of Claude Code. " +
			"Clear the editor, the terminal flag, or both.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in unsetEditorIn) (*sdk.CallToolResult, unsetEditorOut, error) {
		out, err := unsetEditor(s.sessions, caller, in)
		return nil, out, err
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
		out, err := setBlockCap(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolUnsetBlockCap,
		Description: "Take a block cap out of the settings file of the multiplexer. " +
			"Give 'type' to clear one kind of block, so it takes the default again, or no type to clear the default of " +
			strconv.Itoa(config.DefaultBlockCap) + " rows.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in unsetBlockCapIn) (*sdk.CallToolResult, unsetBlockCapOut, error) {
		out, err := unsetBlockCap(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolSetAutoArchive,
		Description: "Archive a stopped session on its own, after it is idle for a number of days. " +
			"Give 'days' as one or more. A stopped session with no turn for that many days is archived, so the sidebar stays short. " +
			"It writes the settings file of the multiplexer, and the setting holds for every session.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in setAutoArchiveIn) (*sdk.CallToolResult, setAutoArchiveOut, error) {
		out, err := setAutoArchive(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolUnsetAutoArchive,
		Description: "Turn auto-archive off, so a stopped session stays until the human archives it. " +
			"It takes the setting out of the settings file of the multiplexer.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, unsetAutoArchiveOut, error) {
		out, err := unsetAutoArchive(s.sessions, caller)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolStopWhenIdle,
		Description: "Arm this session to stop itself the next time it is idle, after it finishes the current turn and every queued prompt. " +
			"Unlike " + ToolStop + ", a session can arm this on itself, because the stop is deferred. " +
			"With archive, the session archives itself after the stop. Call with stop false and archive false to disarm. " +
			"A scheduled run uses this to clean itself up when its work is done.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in stopWhenIdleIn) (*sdk.CallToolResult, okOut, error) {
		out, err := stopWhenIdle(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolSetWorkingDir,
		Description: "Say which directory this session works in now, so the human opens that one instead of the directory the session started in. " +
			"Call it after you move into a worktree. The directory must exist.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in setWorkingDirIn) (*sdk.CallToolResult, workingDirOut, error) {
		out, err := setWorkingDir(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolUnsetWorkingDir,
		Description: "Take the working directory off this session, so the human opens the directory the session started in again. " +
			"Call it after you collapse a worktree.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, workingDirOut, error) {
		out, err := unsetWorkingDir(s.sessions, caller)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolListProject,
		Description: "The directories of a session's project, in order. " +
			"A session works in one directory by default, and a project lets it work in several, so the diff panel groups the changes by directory. " +
			"Give a session name, or leave it empty for this session.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listProjectIn) (*sdk.CallToolResult, projectOut, error) {
		out, err := listProject(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolAddProjectDir,
		Description: "Add one directory to this session's project, so the diff panel shows its changes in a section of its own. " +
			"Call it for each code base a single change spans. Add the root of a code base, not a subdirectory of it. The directory must exist.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in projectDirIn) (*sdk.CallToolResult, projectOut, error) {
		out, err := addProjectDir(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolRemoveProject,
		Description: "Take one directory out of this session's project, so the diff panel no longer shows its changes. " +
			"Call it when a directory is no longer part of the change.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in projectDirIn) (*sdk.CallToolResult, projectOut, error) {
		out, err := removeProjectDir(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolSetProject,
		Description: "Replace the whole ordered set of directories of this session's project. " +
			"Give every directory the change spans. Every directory must exist. An empty list clears the project.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in setProjectIn) (*sdk.CallToolResult, projectOut, error) {
		out, err := setProject(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolClearProject,
		Description: "Empty this session's project, so the session works in one directory again. " +
			"The diff panel shows one section again.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, projectOut, error) {
		out, err := clearProject(s.sessions, caller)
		return nil, out, err
	})
}

func templatePath(cfg ConfigPort, caller string, in templatePathIn) (TemplatePath, error) {
	target, err := targetOrSelf(in.Session, caller)
	if err != nil {
		return TemplatePath{}, err
	}
	return cfg.TemplatePath(target)
}

func setConfig(cfg ConfigPort, caller string, in setConfigIn) (setConfigOut, error) {
	path := strings.TrimSpace(in.Path)
	if path == "" {
		return setConfigOut{}, ErrNoConfigPath
	}
	value, err := configValue(in.Value)
	if err != nil {
		return setConfigOut{}, err
	}
	file, err := cfg.SetConfig(path, value, caller)
	if err != nil {
		return setConfigOut{}, err
	}
	return setConfigOut{OK: true, Path: file, Message: path + " is set in " + file}, nil
}

// configValue turns the tool input into the JSON value to write. A client that
// cannot send a JSON array or object sends it as a string of that JSON instead,
// so a string that holds an array or an object is unwrapped to the value it
// holds. A scalar, and a plain string, pass through as themselves.
func configValue(v any) (json.RawMessage, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	s, ok := v.(string)
	if !ok {
		return raw, nil
	}
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || (trimmed[0] != '[' && trimmed[0] != '{') {
		return raw, nil
	}
	if !json.Valid([]byte(trimmed)) {
		return raw, nil
	}
	return json.RawMessage(trimmed), nil
}

func unsetConfig(cfg ConfigPort, caller string, in unsetConfigIn) (unsetConfigOut, error) {
	path := strings.TrimSpace(in.Path)
	if path == "" {
		return unsetConfigOut{}, ErrNoConfigPath
	}
	file, changed, err := cfg.UnsetConfig(path, caller)
	if err != nil {
		return unsetConfigOut{}, err
	}
	message := path + " was not set in " + file
	if changed {
		message = path + " is no longer set in " + file
	}
	return unsetConfigOut{OK: true, Path: file, Changed: changed, Message: message}, nil
}

func getKeybindings(cfg ConfigPort, in getKeybindingsIn) getKeybindingsOut {
	filter := strings.TrimSpace(in.Action)
	var out getKeybindingsOut
	for _, e := range cfg.Keybindings() {
		if filter != "" && string(e.Action) != filter && string(e.Context) != filter {
			continue
		}
		out.Keybindings = append(out.Keybindings, keybindingEntry{
			Action:  string(e.Action),
			Context: string(e.Context),
			Keys:    e.Keys,
			Custom:  e.Custom,
		})
	}
	return out
}

func setKeybinding(cfg ConfigPort, caller string, in setKeybindingIn) (setKeybindingOut, error) {
	action := strings.TrimSpace(in.Action)
	if action == "" {
		return setKeybindingOut{}, ErrNoKeyAction
	}
	if len(in.Keys) == 0 {
		return setKeybindingOut{}, ErrNoKeys
	}
	file, warning, err := cfg.SetKeybinding(action, in.Keys, caller)
	if err != nil {
		return setKeybindingOut{}, err
	}
	message := action + " is bound to " + strings.Join(in.Keys, " ") + " in " + file
	return setKeybindingOut{OK: true, Path: file, Message: message, Warning: warning}, nil
}

func resetKeybinding(cfg ConfigPort, caller string, in resetKeybindingIn) (resetKeybindingOut, error) {
	action := strings.TrimSpace(in.Action)
	if action == "" {
		return resetKeybindingOut{}, ErrNoKeyAction
	}
	file, changed, err := cfg.ResetKeybinding(action, caller)
	if err != nil {
		return resetKeybindingOut{}, err
	}
	message := action + " was not bound in " + file
	if changed {
		message = action + " takes its default keys again, in " + file
	}
	return resetKeybindingOut{OK: true, Path: file, Changed: changed, Message: message}, nil
}

func getCommands(cfg ConfigPort) getCommandsOut {
	var out getCommandsOut
	for _, c := range cfg.Commands() {
		out.Commands = append(out.Commands, commandEntry{Keys: c.Keys, Label: c.Label, Script: c.Script})
	}
	return out
}

func addCommand(cfg ConfigPort, caller string, in addCommandIn) (addCommandOut, error) {
	keyList := strings.TrimSpace(in.Keys)
	label := strings.TrimSpace(in.Label)
	script := strings.TrimSpace(in.Script)
	if keyList == "" {
		return addCommandOut{}, ErrNoKeys
	}
	if label == "" {
		return addCommandOut{}, ErrNoCommandLabel
	}
	if script == "" {
		return addCommandOut{}, ErrNoCommandScript
	}
	file, err := cfg.AddCommand(keyList, label, script, caller)
	if err != nil {
		return addCommandOut{}, err
	}
	message := label + " runs " + script + " on " + keyList + " in " + file
	return addCommandOut{OK: true, Path: file, Message: message}, nil
}

func removeCommand(cfg ConfigPort, caller string, in removeCommandIn) (removeCommandOut, error) {
	label := strings.TrimSpace(in.Label)
	if label == "" {
		return removeCommandOut{}, ErrNoCommandLabel
	}
	file, changed, err := cfg.RemoveCommand(label, caller)
	if err != nil {
		return removeCommandOut{}, err
	}
	message := label + " was not a command in " + file
	if changed {
		message = label + " is no longer a command in " + file
	}
	return removeCommandOut{OK: true, Path: file, Changed: changed, Message: message}, nil
}

func setEditor(cfg ConfigPort, caller string, in setEditorIn) (setEditorOut, error) {
	editor := strings.TrimSpace(in.Editor)
	if editor == "" && in.Terminal == nil {
		return setEditorOut{}, ErrNoEditor
	}
	path, err := cfg.SetEditor(editor, in.Terminal, caller)
	if err != nil {
		return setEditorOut{}, err
	}
	return setEditorOut{OK: true, Path: path, Message: editorMessage(editor, in.Terminal, path)}, nil
}

func unsetEditor(cfg ConfigPort, caller string, in unsetEditorIn) (unsetEditorOut, error) {
	path, changed, err := cfg.UnsetEditor(strings.TrimSpace(in.Field), caller)
	if err != nil {
		return unsetEditorOut{}, err
	}
	return unsetEditorOut{OK: true, Path: path, Changed: changed, Message: clearedMessage(in.Field, changed, path)}, nil
}

func setBlockCap(cfg ConfigPort, caller string, in setBlockCapIn) (setBlockCapOut, error) {
	bucket := strings.TrimSpace(in.Type)
	if bucket != "" && !config.ValidBucket(bucket) {
		return setBlockCapOut{}, ErrBadType
	}
	if in.Unlimited && in.Rows != nil {
		return setBlockCapOut{}, ErrCapBoth
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
			return setBlockCapOut{}, ErrBadCap
		}
		value := *in.Rows
		rows = &value
	default:
		return setBlockCapOut{}, ErrBadCap
	}
	path, err := cfg.SetBlockCap(bucket, rows, caller)
	if err != nil {
		return setBlockCapOut{}, err
	}
	return setBlockCapOut{OK: true, Path: path, Type: bucket, Rows: rows, Message: blockCapMessage(bucket, rows, path)}, nil
}

func unsetBlockCap(cfg ConfigPort, caller string, in unsetBlockCapIn) (unsetBlockCapOut, error) {
	bucket := strings.TrimSpace(in.Type)
	if bucket != "" && !config.ValidBucket(bucket) {
		return unsetBlockCapOut{}, ErrBadType
	}
	path, changed, err := cfg.UnsetBlockCap(bucket, caller)
	if err != nil {
		return unsetBlockCapOut{}, err
	}
	return unsetBlockCapOut{OK: true, Path: path, Changed: changed, Message: blockCapClearedMessage(bucket, changed, path)}, nil
}

func setAutoArchive(cfg ConfigPort, caller string, in setAutoArchiveIn) (setAutoArchiveOut, error) {
	if in.Days < 1 {
		return setAutoArchiveOut{}, ErrBadDays
	}
	path, err := cfg.SetAutoArchive(in.Days, caller)
	if err != nil {
		return setAutoArchiveOut{}, err
	}
	return setAutoArchiveOut{OK: true, Path: path, Days: in.Days,
		Message: "a stopped session is now archived after " + strconv.Itoa(in.Days) + " days idle, in " + path}, nil
}

func unsetAutoArchive(cfg ConfigPort, caller string) (unsetAutoArchiveOut, error) {
	path, changed, err := cfg.UnsetAutoArchive(caller)
	if err != nil {
		return unsetAutoArchiveOut{}, err
	}
	message := "auto-archive was not set in " + path
	if changed {
		message = "auto-archive is off, in " + path
	}
	return unsetAutoArchiveOut{OK: true, Path: path, Changed: changed, Message: message}, nil
}

func stopWhenIdle(ctrl ControlPort, caller string, in stopWhenIdleIn) (okOut, error) {
	stop := true
	if in.Stop != nil {
		stop = *in.Stop
	}
	if err := ctrl.StopWhenIdle(caller, stop, in.Archive); err != nil {
		return okOut{}, err
	}
	switch {
	case in.Archive:
		return okOut{OK: true, Message: caller + " will archive itself when idle"}, nil
	case stop:
		return okOut{OK: true, Message: caller + " will stop itself when idle"}, nil
	default:
		return okOut{OK: true, Message: caller + " will not stop itself when idle"}, nil
	}
}

func setWorkingDir(cfg ConfigPort, caller string, in setWorkingDirIn) (workingDirOut, error) {
	path := strings.TrimSpace(in.Path)
	if path == "" {
		return workingDirOut{}, ErrNoDir
	}
	full, err := cfg.SetWorkingDir(path, caller)
	if err != nil {
		return workingDirOut{}, err
	}
	return workingDirOut{OK: true, Path: full, Changed: true,
		Message: caller + " works in " + full + " now"}, nil
}

func unsetWorkingDir(cfg ConfigPort, caller string) (workingDirOut, error) {
	changed, err := cfg.UnsetWorkingDir(caller)
	if err != nil {
		return workingDirOut{}, err
	}
	message := caller + " had no working directory"
	if changed {
		message = caller + " works in the directory it started in again"
	}
	return workingDirOut{OK: true, Changed: changed, Message: message}, nil
}

func listProject(cfg ConfigPort, caller string, in listProjectIn) (projectOut, error) {
	target, err := targetOrSelf(in.Session, caller)
	if err != nil {
		return projectOut{}, err
	}
	dirs, err := cfg.Project(target)
	if err != nil {
		return projectOut{}, err
	}
	return projectOut{OK: true, Dirs: dirs, Message: projectListMessage(target, dirs)}, nil
}

func addProjectDir(cfg ConfigPort, caller string, in projectDirIn) (projectOut, error) {
	path := strings.TrimSpace(in.Path)
	if path == "" {
		return projectOut{}, ErrNoDir
	}
	dirs, err := cfg.AddProjectDir(path, caller)
	if err != nil {
		return projectOut{}, err
	}
	return projectOut{OK: true, Dirs: dirs, Changed: true,
		Message: caller + " has " + strconv.Itoa(len(dirs)) + " directories in its project now"}, nil
}

func removeProjectDir(cfg ConfigPort, caller string, in projectDirIn) (projectOut, error) {
	path := strings.TrimSpace(in.Path)
	if path == "" {
		return projectOut{}, ErrNoDir
	}
	dirs, err := cfg.RemoveProjectDir(path, caller)
	if err != nil {
		return projectOut{}, err
	}
	return projectOut{OK: true, Dirs: dirs, Changed: true,
		Message: caller + " has " + strconv.Itoa(len(dirs)) + " directories in its project now"}, nil
}

func setProject(cfg ConfigPort, caller string, in setProjectIn) (projectOut, error) {
	dirs, err := cfg.SetProject(in.Paths, caller)
	if err != nil {
		return projectOut{}, err
	}
	return projectOut{OK: true, Dirs: dirs, Changed: true,
		Message: caller + " has " + strconv.Itoa(len(dirs)) + " directories in its project now"}, nil
}

func clearProject(cfg ConfigPort, caller string) (projectOut, error) {
	changed, err := cfg.ClearProject(caller)
	if err != nil {
		return projectOut{}, err
	}
	message := caller + " had no project"
	if changed {
		message = caller + " works in one directory again"
	}
	return projectOut{OK: true, Changed: changed, Message: message}, nil
}

func projectListMessage(session string, dirs []string) string {
	if len(dirs) == 0 {
		return session + " has no project, so it works in one directory"
	}
	return session + " has " + strconv.Itoa(len(dirs)) + " directories in its project"
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
