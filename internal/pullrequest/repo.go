package pullrequest

import (
	"net/url"
	"strings"
)

// ParseRepo reads a git remote url into a Repo: the host, the owner path, and
// the repository name. It reads the SSH form (git@host:owner/name.git), the
// ssh:// form, and the HTTPS form. A GitLab subgroup keeps its full path in the
// owner. It returns ok false for a url it cannot read. See docs/pull-requests.md.
func ParseRepo(remoteURL string) (Repo, bool) {
	raw := strings.TrimSpace(remoteURL)
	if raw == "" {
		return Repo{}, false
	}

	var host, path string
	switch {
	case strings.Contains(raw, "://"):
		u, err := url.Parse(raw)
		if err != nil {
			return Repo{}, false
		}
		host = u.Hostname()
		path = u.Path
	case strings.HasPrefix(raw, "git@") || scpLike(raw):
		rest := raw
		if at := strings.Index(rest, "@"); at >= 0 {
			rest = rest[at+1:]
		}
		h, p, ok := strings.Cut(rest, ":")
		if !ok {
			return Repo{}, false
		}
		host, path = h, p
	default:
		return Repo{}, false
	}

	host = strings.TrimSpace(host)
	path = strings.Trim(path, "/")
	path = strings.TrimSuffix(path, ".git")
	if host == "" || path == "" {
		return Repo{}, false
	}

	segments := strings.Split(path, "/")
	if len(segments) < 2 {
		return Repo{}, false
	}
	name := segments[len(segments)-1]
	owner := strings.Join(segments[:len(segments)-1], "/")
	if owner == "" || name == "" {
		return Repo{}, false
	}
	return Repo{Host: host, Owner: owner, Name: name}, true
}

// scpLike reports whether raw is the scp-like SSH form host:owner/name, without
// a scheme. It requires a colon before the first slash, so a Windows path or a
// url with a port does not match.
func scpLike(raw string) bool {
	colon := strings.Index(raw, ":")
	if colon < 0 {
		return false
	}
	slash := strings.Index(raw, "/")
	return slash < 0 || colon < slash
}
