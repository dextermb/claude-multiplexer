// Package update checks whether a newer release of the multiplexer exists on
// GitHub, so the interface can warn the user. See docs/version-updates.md.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

// Repo is the GitHub repository the release check reads.
const Repo = "dextermb/claude-multiplexer"

const (
	latestURL    = "https://api.github.com/repos/" + Repo + "/releases/latest"
	userAgent    = "cmux-update-check"
	fetchTimeout = 10 * time.Second
	bodyLimit    = 1 << 20
)

// Release is the part of a GitHub release the check reads.
type Release struct {
	TagName     string    `json:"tag_name"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
}

// Current reads the running binary's version and build commit time from the Go
// build info. A binary installed from a tag carries a semver version; a build
// from source carries "(devel)" and a commit time. See docs/version-updates.md.
func Current() (version string, buildTime time.Time) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", time.Time{}
	}
	version = info.Main.Version
	for _, s := range info.Settings {
		if s.Key == "vcs.time" {
			if t, err := time.Parse(time.RFC3339, s.Value); err == nil {
				buildTime = t
			}
		}
	}
	return version, buildTime
}

// Latest fetches the most recent release from GitHub. It returns ok=false with
// no error when the repository has no release yet (a 404).
func Latest(ctx context.Context, client *http.Client) (Release, bool, error) {
	return latestFrom(ctx, client, latestURL)
}

func latestFrom(ctx context.Context, client *http.Client, url string) (rel Release, ok bool, err error) {
	if client == nil {
		client = &http.Client{Timeout: fetchTimeout}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Release{}, false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	res, err := client.Do(req)
	if err != nil {
		return Release{}, false, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return Release{}, false, nil
	}
	if res.StatusCode != http.StatusOK {
		return Release{}, false, fmt.Errorf("update: %s: status %d", url, res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, bodyLimit))
	if err != nil {
		return Release{}, false, err
	}
	if err := json.Unmarshal(body, &rel); err != nil {
		return Release{}, false, err
	}
	return rel, rel.TagName != "", nil
}

// Outdated reports whether rel is newer than the running binary. When the
// version parses as a semver, it compares the two tags. Otherwise (a build from
// source) it compares the build commit time to the release date. See
// docs/version-updates.md.
func Outdated(version string, buildTime time.Time, rel Release) bool {
	if rel.TagName == "" {
		return false
	}
	if cur, ok := parseSemver(version); ok {
		latest, ok := parseSemver(rel.TagName)
		return ok && compareSemver(latest, cur) > 0
	}
	if buildTime.IsZero() || rel.PublishedAt.IsZero() {
		return false
	}
	return rel.PublishedAt.After(buildTime)
}

// IsSemver reports whether s is a plain semver version, such as a released
// binary carries. A build from source is not.
func IsSemver(s string) bool {
	_, ok := parseSemver(s)
	return ok
}

type semver struct{ major, minor, patch int }

func parseSemver(s string) (semver, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return semver{}, false
	}
	var out semver
	dst := []*int{&out.major, &out.minor, &out.patch}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return semver{}, false
		}
		*dst[i] = n
	}
	return out, true
}

func compareSemver(a, b semver) int {
	switch {
	case a.major != b.major:
		return a.major - b.major
	case a.minor != b.minor:
		return a.minor - b.minor
	default:
		return a.patch - b.patch
	}
}
