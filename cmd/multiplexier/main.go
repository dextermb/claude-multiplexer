package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
	"github.com/dextermb/claude-multiplexer/internal/template"
	"github.com/dextermb/claude-multiplexer/internal/tui"
)

const usage = `multiplexier — a Claude Code multiplexer

Usage:
  multiplexier [flags]              Start the terminal user interface.
  multiplexier run [flags] PROMPT   Start one session, send one prompt, print the stream.
  multiplexier templates [flags]    List the preset prompts, and the fields each one takes.

Run "multiplexier -h" or "multiplexier run -h" for the flags.
`

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "run":
			os.Exit(runCommand(args[1:]))
		case "tui":
			os.Exit(tuiCommand(args[1:]))
		case "templates":
			os.Exit(templatesCommand(args[1:]))
		case "help":
			fmt.Print(usage)
			os.Exit(0)
		}
	}
	os.Exit(tuiCommand(args))
}

func templatesCommand(argv []string) int {
	fs := flag.NewFlagSet("templates", flag.ExitOnError)
	root := fs.String("root", "", "state directory (default ~/.multiplexier)")
	dir := fs.String("dir", ".", "directory whose project templates to include")
	if err := fs.Parse(argv); err != nil {
		return 2
	}

	stateRoot, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "multiplexier: %v\n", err)
		return 1
	}
	sessionDir, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "multiplexier: %v\n", err)
		return 1
	}

	dirs := template.Dirs(stateRoot, sessionDir)
	all := template.Load(dirs...)
	if len(all) == 0 {
		fmt.Println("No templates yet. Put a markdown file in one of these:")
		for _, dir := range dirs {
			fmt.Printf("  %s\n", dir)
		}
		return 0
	}

	width := 0
	for _, tpl := range all {
		if n := len(tpl.Name) + 1; n > width {
			width = n
		}
	}
	for _, tpl := range all {
		fields := strings.Join(fieldSummary(tpl), " ")
		fmt.Printf("%-*s  %-28s  %s\n", width, "/"+tpl.Name, fields, tpl.Description)
	}
	return 0
}

func fieldSummary(tpl template.Template) []string {
	out := make([]string, 0, len(tpl.Fields))
	for _, field := range tpl.Fields {
		if field.Default != "" {
			out = append(out, field.Name+"="+field.Default)
			continue
		}
		out = append(out, field.Name)
	}
	return out
}

func resolveRoot(root string) (string, error) {
	if root != "" {
		return filepath.Abs(root)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, manager.DefaultRootName), nil
}

func tuiCommand(argv []string) int {
	fs := flag.NewFlagSet("tui", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, usage, "\nFlags:\n")
		fs.PrintDefaults()
	}
	root := fs.String("root", "", "state directory (default ~/.multiplexier)")
	model := fs.String("model", "", "default model for a new session")
	mode := fs.String("permission-mode", session.DefaultPermissionMode,
		"default permission mode: acceptEdits, auto, bypassPermissions, default, dontAsk, or plan")
	claudePath := fs.String("claude", session.DefaultClaudePath, "path to the claude binary")
	dir := fs.String("dir", "", "start one session in this directory at once")
	control := fs.Bool("control", false, "let the session started by --dir drive the other sessions")
	maxLines := fs.Int("max-lines", manager.DefaultMaxLines, "output lines kept in memory for each session")
	verbose := fs.Bool("v", false, "show state changes, thinking, and full tool results")
	if err := fs.Parse(argv); err != nil {
		return 2
	}

	mgr, err := manager.New(manager.Options{
		Root:                  *root,
		Renderer:              render.Renderer{Verbose: *verbose},
		MaxLines:              *maxLines,
		ClaudePath:            *claudePath,
		DefaultModel:          *model,
		DefaultPermissionMode: *mode,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "multiplexier: %v\n", err)
		return 1
	}

	if err := mgr.StartMCP(); err != nil {
		fmt.Fprintf(os.Stderr, "multiplexier: %v\n", err)
		return 1
	}

	initialDir := *dir
	if initialDir != "" {
		abs, err := filepath.Abs(initialDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "multiplexier: %v\n", err)
			return 2
		}
		initialDir = abs
	}

	if err := tui.Run(tui.Options{
		Manager:               mgr,
		DefaultModel:          *model,
		DefaultPermissionMode: *mode,
		InitialDir:            initialDir,
		InitialControl:        *control,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "multiplexier: %v\n", err)
		return 1
	}
	return 0
}

func runCommand(argv []string) int {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	dir := fs.String("dir", ".", "working directory for the session")
	model := fs.String("model", "", "model for the session")
	mode := fs.String("permission-mode", session.DefaultPermissionMode,
		"permission mode: acceptEdits, auto, bypassPermissions, default, dontAsk, or plan")
	claudePath := fs.String("claude", session.DefaultClaudePath, "path to the claude binary")
	transcript := fs.String("transcript", "", "path for the JSON Lines transcript")
	resume := fs.String("resume", "", "resume a stored Claude session by id")
	allowed := fs.String("allowed-tools", "", "comma separated list of allowed tools")
	verbose := fs.Bool("v", false, "show state changes, thinking, and full tool results")
	if err := fs.Parse(argv); err != nil {
		return 2
	}

	prompt, err := readPrompt(fs.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "multiplexier: %v\n", err)
		return 2
	}

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "multiplexier: %v\n", err)
		return 2
	}

	cfg := session.Config{
		Name:           filepath.Base(absDir),
		Dir:            absDir,
		Model:          *model,
		PermissionMode: *mode,
		ResumeID:       *resume,
		ClaudePath:     *claudePath,
		TranscriptPath: *transcript,
	}
	if *allowed != "" {
		cfg.AllowedTools = strings.Split(*allowed, ",")
	}

	s, err := session.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "multiplexier: %v\n", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := s.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "multiplexier: %v\n", err)
		return 1
	}
	if err := s.Send(prompt); err != nil {
		fmt.Fprintf(os.Stderr, "multiplexier: %v\n", err)
		return 1
	}

	code := stream(ctx, s, render.Renderer{Verbose: *verbose})

	shutdown, cancel := context.WithTimeout(context.Background(), session.DefaultStopGrace)
	defer cancel()
	if err := s.Stop(shutdown); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(os.Stderr, "multiplexier: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	return code
}

func stream(ctx context.Context, s *session.Session, r render.Renderer) int {
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	code := 0
	for {
		select {
		case <-ctx.Done():
			return 130
		case ev, ok := <-s.Events():
			if !ok {
				if s.State() == session.StateFailed {
					return 1
				}
				return code
			}
			for _, line := range r.Lines(ev) {
				fmt.Fprintln(out, line.Text)
			}
			out.Flush()
			if ev.Kind == session.KindProtocol && ev.Protocol.Type == protocol.TypeResult {
				if ev.Protocol.Result != nil && ev.Protocol.Result.IsError {
					code = 1
				}
				return code
			}
		}
	}
}

func readPrompt(args []string) (string, error) {
	joined := strings.TrimSpace(strings.Join(args, " "))
	if joined != "" && joined != "-" {
		return joined, nil
	}
	info, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return "", errors.New("no prompt given")
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	prompt := strings.TrimSpace(string(data))
	if prompt == "" {
		return "", errors.New("no prompt given")
	}
	return prompt, nil
}
