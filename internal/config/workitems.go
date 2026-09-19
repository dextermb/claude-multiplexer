package config

import "strings"

// The provider names a work item belongs to. See docs/work-items.md.
const (
	WorkItemJira   = "jira"
	WorkItemLinear = "linear"
)

// The default MCP endpoint of each provider. A provider entry with no url takes
// its default. See docs/work-items.md.
const (
	DefaultJiraMCPURL   = "https://mcp.atlassian.com/v2/mcp"
	DefaultLinearMCPURL = "https://mcp.linear.app/mcp"
)

// WorkItems configures the work-item providers, keyed by provider. Both may be
// set at once. A nil provider, or a provider with no token, is off. See
// docs/work-items.md.
type WorkItems struct {
	Jira   *WorkItemProvider `json:"jira,omitempty"`
	Linear *WorkItemProvider `json:"linear,omitempty"`
}

// WorkItemProvider holds one provider's MCP endpoint and token. Email selects
// Jira Basic auth (email:token); with no email the client sends a bearer token.
// See docs/work-items.md.
type WorkItemProvider struct {
	Token string `json:"token,omitempty"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// Provider reads one provider's settings by name, or nil when it is not set.
func (w *WorkItems) Provider(name string) *WorkItemProvider {
	if w == nil {
		return nil
	}
	switch name {
	case WorkItemJira:
		return w.Jira
	case WorkItemLinear:
		return w.Linear
	}
	return nil
}

// Providers names the providers that hold a token, in a stable order. It is
// empty when the feature is off. The first name is the default a tool takes
// when the caller names none.
func (w *WorkItems) Providers() []string {
	if w == nil {
		return nil
	}
	var out []string
	if w.Jira.enabled() {
		out = append(out, WorkItemJira)
	}
	if w.Linear.enabled() {
		out = append(out, WorkItemLinear)
	}
	return out
}

// Enabled reports whether at least one provider holds a token.
func (w *WorkItems) Enabled() bool {
	return len(w.Providers()) > 0
}

func (p *WorkItemProvider) enabled() bool {
	return p != nil && strings.TrimSpace(p.Token) != ""
}

// SetProvider writes one provider entry, and makes the block when there is
// none. It returns the changed WorkItems, so a caller writes the settings file.
func (w *WorkItems) SetProvider(name string, entry *WorkItemProvider) *WorkItems {
	if w == nil {
		w = &WorkItems{}
	}
	switch name {
	case WorkItemJira:
		w.Jira = entry
	case WorkItemLinear:
		w.Linear = entry
	}
	return w
}

// ValidWorkItemProvider reports whether a name is a provider the program knows.
func ValidWorkItemProvider(name string) bool {
	return name == WorkItemJira || name == WorkItemLinear
}

// Endpoint gives the MCP url of a provider: its own url, or the provider
// default when it names none.
func (w *WorkItems) Endpoint(name string) string {
	p := w.Provider(name)
	if p != nil && strings.TrimSpace(p.URL) != "" {
		return strings.TrimSpace(p.URL)
	}
	switch name {
	case WorkItemJira:
		return DefaultJiraMCPURL
	case WorkItemLinear:
		return DefaultLinearMCPURL
	}
	return ""
}
