package manager

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

// errNoLentCredential is returned when a hoisted session names a peer that holds
// no lent credential. See docs/peers/hoisted.md.
var errNoLentCredential = errors.New("manager: the peer has no lent credential to hoist")

// hoist prepares a local session to run with a peer's Claude credential. It sets
// the credential environment variable, scrubs every other Claude credential from
// the inherited environment, and points CLAUDE_CONFIG_DIR at a per-lender
// directory seeded from the borrower's own setup. See docs/peers/hoisted.md.
func (m *Manager) hoist(cfg *session.Config, lender string) error {
	host, ok := m.peerByName(lender)
	if !ok {
		return fmt.Errorf("%w: %s", errUnknownPeer, lender)
	}
	if host.Credential == nil {
		return fmt.Errorf("%w: %s", errNoLentCredential, lender)
	}
	envVar := config.CredentialEnvVar(host.Credential.Type)
	if envVar == "" {
		return errBadCredentialType
	}
	dir := filepath.Join(m.opts.Root, "hoisted", safeName(lender))
	if err := seedHoistDir(dir, config.ClaudeDir()); err != nil {
		return err
	}
	cfg.Env = []string{
		envVar + "=" + host.Credential.Value,
		"CLAUDE_CONFIG_DIR=" + dir,
	}
	cfg.EnvScrub = config.AuthEnvVars
	return nil
}

// peerByName reads one configured peer host by name.
func (m *Manager) peerByName(name string) (config.PeerHost, bool) {
	for _, host := range m.peerHosts() {
		if host.Name == name {
			return host, true
		}
	}
	return config.PeerHost{}, false
}

// safeName turns a peer name into a directory-safe name, so a label with a slash
// or a space cannot escape the hoisted directory.
func safeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "peer"
	}
	return b.String()
}

// seedHoistDir makes the per-lender config directory and seeds it from the
// borrower's Claude Code setup: it copies the commands, skills, and rules, and
// symlinks claude.json, so the borrower's own setup applies and the lender's
// projects persist per lender. A missing source is skipped, so a hoisted session
// still starts. See docs/peers/hoisted.md.
func seedHoistDir(dir, src string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if src == "" {
		return nil
	}
	for _, name := range []string{"commands", "skills", "rules"} {
		_ = copyTree(filepath.Join(src, name), filepath.Join(dir, name))
	}
	_ = copyFile(filepath.Join(src, "CLAUDE.md"), filepath.Join(dir, "CLAUDE.md"))
	if from := borrowerClaudeJSON(src); from != "" {
		_ = linkFile(from, filepath.Join(dir, ".claude.json"))
	}
	return nil
}

// borrowerClaudeJSON finds the borrower's claude.json: beside the config
// directory, or in the home directory. It returns "" when neither is there.
func borrowerClaudeJSON(src string) string {
	beside := filepath.Join(src, ".claude.json")
	if _, err := os.Stat(beside); err == nil {
		return beside
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	inHome := filepath.Join(home, ".claude.json")
	if _, err := os.Stat(inHome); err == nil {
		return inHome
	}
	return ""
}

// linkFile points dst at from with a symlink, replacing any dst that is there,
// so a re-seed keeps the link current.
func linkFile(from, dst string) error {
	abs, err := filepath.Abs(from)
	if err != nil {
		return err
	}
	_ = os.Remove(dst)
	return os.Symlink(abs, dst)
}

// copyTree copies a directory tree from src to dst. It is a no-op when src is
// not there.
func copyTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return copyFile(src, dst)
	}
	if err := os.MkdirAll(dst, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := copyTree(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

// copyFile copies one file from src to dst. It is a no-op when src is not there.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
