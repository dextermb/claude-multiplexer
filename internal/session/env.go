package session

import "strings"

// buildEnv merges the environment for the child process. It starts from base,
// drops every key in scrub, drops every key that add overrides, then appends
// add. So add wins over base for a shared key, and a scrubbed key is gone unless
// add sets it again. A hoisted session uses this to inject exactly one Claude
// credential; see docs/peers/hoisted.md.
func buildEnv(base, add, scrub []string) []string {
	drop := make(map[string]bool, len(scrub)+len(add))
	for _, key := range scrub {
		drop[key] = true
	}
	for _, entry := range add {
		drop[envKey(entry)] = true
	}
	out := make([]string, 0, len(base)+len(add))
	for _, entry := range base {
		if drop[envKey(entry)] {
			continue
		}
		out = append(out, entry)
	}
	return append(out, add...)
}

// envKey reads the name of an environment entry, the part before the first "=".
func envKey(entry string) string {
	if i := strings.IndexByte(entry, '='); i >= 0 {
		return entry[:i]
	}
	return entry
}
