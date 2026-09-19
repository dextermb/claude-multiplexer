package pullrequest

import "testing"

func TestParseRepoReadsTheForms(t *testing.T) {
	cases := []struct {
		in    string
		host  string
		owner string
		name  string
		ok    bool
	}{
		{"git@github.com:owner/name.git", "github.com", "owner", "name", true},
		{"https://github.com/owner/name.git", "github.com", "owner", "name", true},
		{"https://github.com/owner/name", "github.com", "owner", "name", true},
		{"ssh://git@gitlab.com/group/sub/name.git", "gitlab.com", "group/sub", "name", true},
		{"git@gitlab.corp:group/sub/name.git", "gitlab.corp", "group/sub", "name", true},
		{"https://user@github.corp/owner/name.git", "github.corp", "owner", "name", true},
		{"", "", "", "", false},
		{"not a url", "", "", "", false},
		{"https://github.com/owner", "", "", "", false},
	}
	for _, c := range cases {
		repo, ok := ParseRepo(c.in)
		if ok != c.ok {
			t.Errorf("%q ok = %v, want %v", c.in, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if repo.Host != c.host || repo.Owner != c.owner || repo.Name != c.name {
			t.Errorf("%q = %+v, want %s %s %s", c.in, repo, c.host, c.owner, c.name)
		}
	}
}
