package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOutdatedSemver(t *testing.T) {
	rel := Release{TagName: "v1.2.0"}
	cases := []struct {
		version string
		want    bool
	}{
		{"v1.1.0", true},
		{"v1.2.0", false},
		{"v1.3.0", false},
		{"1.1.0", true},
	}
	for _, c := range cases {
		if got := Outdated(c.version, time.Time{}, rel); got != c.want {
			t.Errorf("Outdated(%q, v1.2.0) = %v, want %v", c.version, got, c.want)
		}
	}
}

func TestOutdatedDevBuild(t *testing.T) {
	build := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := Release{TagName: "v0.0.9", PublishedAt: build.Add(time.Hour)}
	older := Release{TagName: "v0.0.9", PublishedAt: build.Add(-time.Hour)}
	if !Outdated("(devel)", build, newer) {
		t.Error("a dev build older than the release must be out of date")
	}
	if Outdated("(devel)", build, older) {
		t.Error("a dev build newer than the release must not be out of date")
	}
	if Outdated("(devel)", time.Time{}, newer) {
		t.Error("a dev build with no commit time must not be out of date")
	}
}

func TestOutdatedNoRelease(t *testing.T) {
	if Outdated("v0.0.1", time.Now(), Release{}) {
		t.Error("an empty release must never be out of date")
	}
}

func TestLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("Latest must send a User-Agent")
		}
		w.Write([]byte(`{"tag_name":"v2.0.0","html_url":"https://example/x","published_at":"2026-02-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	rel, ok, err := latestFrom(context.Background(), srv.Client(), srv.URL)
	if err != nil || !ok {
		t.Fatalf("Latest ok=%v err=%v", ok, err)
	}
	if rel.TagName != "v2.0.0" {
		t.Errorf("tag = %q, want v2.0.0", rel.TagName)
	}
}

func TestLatestNoRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, ok, err := latestFrom(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("a 404 must not be an error: %v", err)
	}
	if ok {
		t.Error("a 404 must report no release")
	}
}

func TestParseSemver(t *testing.T) {
	if _, ok := parseSemver("(devel)"); ok {
		t.Error("(devel) must not parse as a semver")
	}
	if _, ok := parseSemver(""); ok {
		t.Error("an empty string must not parse as a semver")
	}
	if _, ok := parseSemver("v1.2"); ok {
		t.Error("a two-part version must not parse as a semver")
	}
	if v, ok := parseSemver("v1.2.3"); !ok || v.major != 1 || v.minor != 2 || v.patch != 3 {
		t.Errorf("parseSemver(v1.2.3) = %+v ok=%v", v, ok)
	}
}
