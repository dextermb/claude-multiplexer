// Package workitem is the multiplexer's MCP client to the Jira and Linear work
// item servers. It links a session to one work item, reads its status, and
// changes it. See docs/work-items.md.
package workitem

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// Item is one work item: the provider it lives on, its key, and the fields the
// interface shows. See docs/work-items.md.
type Item struct {
	Provider string `json:"provider"`
	Key      string `json:"key"`
	Title    string `json:"title,omitempty"`
	URL      string `json:"url,omitempty"`
	Status   string `json:"status,omitempty"`
	StatusID string `json:"status_id,omitempty"`
}

// Status is one status a work item may take: the display name the platform
// offers, and the raw id the provider transitions to.
type Status struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

// Provider reads and changes the status of a work item on one platform. Each
// method opens a fresh MCP connection, so the provider holds no live state. See
// docs/work-items.md.
type Provider interface {
	// Resolve reads the item by key: its title, url, and current status.
	Resolve(ctx context.Context, key string) (Item, error)
	// Statuses reads the statuses the item may move to from its current state.
	Statuses(ctx context.Context, key string) ([]Status, error)
	// SetStatus moves the item to the named status, and returns the updated item.
	SetStatus(ctx context.Context, key, target string) (Item, error)
}

var (
	// ErrDisabled is the failure when no provider is configured.
	ErrDisabled = errors.New("workitem: no provider is configured")
	// ErrUnknownProvider is the failure when a name is not a known provider.
	ErrUnknownProvider = errors.New("workitem: the provider must be jira or linear")
	// ErrNoProvider is the failure when the caller names no provider and more than
	// one is configured.
	ErrNoProvider = errors.New("workitem: name the provider, because more than one is configured")
	// ErrNoKey is the failure when a work-item key is empty.
	ErrNoKey = errors.New("workitem: this needs a work-item key")
	// ErrNoStatus is the failure when a target status is empty.
	ErrNoStatus = errors.New("workitem: this needs a target status")
)

// Set holds the configured providers. The manager keeps one, built from the
// settings file. A nil or empty config keeps the feature off. See
// docs/work-items.md.
type Set struct {
	cfg *config.WorkItems
}

// NewSet builds the provider set from the work-item settings.
func NewSet(cfg *config.WorkItems) *Set {
	return &Set{cfg: cfg}
}

// Enabled reports whether at least one provider is configured.
func (s *Set) Enabled() bool {
	return s != nil && s.cfg.Enabled()
}

// Providers names the configured providers, in a stable order.
func (s *Set) Providers() []string {
	if s == nil {
		return nil
	}
	return s.cfg.Providers()
}

// Resolve links to an item on the named provider. An empty name takes the sole
// configured provider.
func (s *Set) Resolve(ctx context.Context, provider, key string) (Item, error) {
	p, name, err := s.provider(provider)
	if err != nil {
		return Item{}, err
	}
	key, err = cleanKey(key)
	if err != nil {
		return Item{}, err
	}
	item, err := p.Resolve(ctx, key)
	if err != nil {
		return Item{}, err
	}
	item.Provider = name
	item.Key = key
	return item, nil
}

// Statuses reads the target statuses of an item on the named provider.
func (s *Set) Statuses(ctx context.Context, provider, key string) ([]Status, error) {
	p, _, err := s.provider(provider)
	if err != nil {
		return nil, err
	}
	key, err = cleanKey(key)
	if err != nil {
		return nil, err
	}
	return p.Statuses(ctx, key)
}

// SetStatus moves an item on the named provider to the target status.
func (s *Set) SetStatus(ctx context.Context, provider, key, target string) (Item, error) {
	p, name, err := s.provider(provider)
	if err != nil {
		return Item{}, err
	}
	key, err = cleanKey(key)
	if err != nil {
		return Item{}, err
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return Item{}, ErrNoStatus
	}
	item, err := p.SetStatus(ctx, key, target)
	if err != nil {
		return Item{}, err
	}
	item.Provider = name
	item.Key = key
	return item, nil
}

// provider resolves the named provider, or the sole configured one when the
// name is empty. It returns the provider and its resolved name.
func (s *Set) provider(name string) (Provider, string, error) {
	if !s.Enabled() {
		return nil, "", ErrDisabled
	}
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		names := s.cfg.Providers()
		if len(names) != 1 {
			return nil, "", ErrNoProvider
		}
		name = names[0]
	}
	if name != config.WorkItemJira && name != config.WorkItemLinear {
		return nil, "", ErrUnknownProvider
	}
	settings := s.cfg.Provider(name)
	if settings == nil || strings.TrimSpace(settings.Token) == "" {
		return nil, "", fmt.Errorf("%w: %s", ErrDisabled, name)
	}
	endpoint := s.cfg.Endpoint(name)
	auth := authorization(settings)
	switch name {
	case config.WorkItemJira:
		return &jira{conn: connector{endpoint: endpoint, authorization: auth}}, name, nil
	default:
		return &linear{conn: connector{endpoint: endpoint, authorization: auth}}, name, nil
	}
}

func cleanKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", ErrNoKey
	}
	return key, nil
}

// matchStatus finds the status whose name equals target, without case. It
// returns the match and whether one was found.
func matchStatus(list []Status, target string) (Status, bool) {
	for _, s := range list {
		if strings.EqualFold(strings.TrimSpace(s.Name), strings.TrimSpace(target)) {
			return s, true
		}
	}
	return Status{}, false
}

func statusNames(list []Status) string {
	names := make([]string, 0, len(list))
	for _, s := range list {
		names = append(names, s.Name)
	}
	return strings.Join(names, ", ")
}
