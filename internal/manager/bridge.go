package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

// bridge adapts the manager to the MCP tools, and publishes a notice for each
// change a tool makes, so the interface can show who did what. The methods are
// grouped by tool domain across the bridge_*.go files.
type bridge struct{ m *Manager }

func (b *bridge) SetTitle(name, title string) error {
	if err := b.m.SetTitle(name, title); err != nil {
		return err
	}
	notice := name + " cleared its title"
	if title != "" {
		notice = name + " renamed itself to " + title
	}
	b.m.notify(name, notice, true)
	return nil
}

func (b *bridge) SendFrom(target, from, text string) (int, error) {
	return b.m.SendFrom(target, from, text)
}

func (b *bridge) Stop(ctx context.Context, name, by string) error {
	if err := b.m.Stop(ctx, name); err != nil {
		return err
	}
	b.m.notify(name, by+" stopped "+name, false)
	return nil
}

func (b *bridge) Archive(name string, archived bool, by string) error {
	if err := b.m.Archive(name, archived); err != nil {
		return err
	}
	verb := " archived "
	if !archived {
		verb = " restored "
	}
	b.m.notify(name, by+verb+name, true)
	return nil
}

func (b *bridge) Create(dir, name, by string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrNotDirectory, dir)
	}
	created, err := b.m.Spawn(context.Background(), Spec{Dir: abs, Name: name, Parent: by})
	if err != nil {
		return "", err
	}
	b.m.notify(created, by+" created "+created, true)
	return created, nil
}

func (b *bridge) List() []mcp.Session { return b.m.List() }

func (b *bridge) Messages(name string, limit int) ([]mcp.Message, error) {
	return b.m.Messages(name, limit)
}

func (b *bridge) Jobs(name string) ([]mcp.Job, error) { return b.m.Jobs(name) }

func (b *bridge) StopJob(target, jobID, by string) (int, error) {
	return b.m.StopJobFrom(target, by, jobID)
}

func (b *bridge) ConfigPath() mcp.ConfigPath { return b.m.ConfigPath() }

func (b *bridge) TemplatePath(name string) (mcp.TemplatePath, error) { return b.m.TemplatePath(name) }
