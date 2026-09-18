package git

import "strings"

// Hunk is one @@ change block of a unified file diff. NewStart and NewCount are
// the new-side range from the header, so a reader names the change as
// path:NewStart-NewEnd without the body. See docs/tui/review.md.
type Hunk struct {
	Header   string
	Body     []string
	NewStart int
	NewCount int
}

// Hunks splits a unified file diff into its hunks. It drops the file header
// lines that come before the first @@. A diff with no @@ (a rename with no
// content change, or a binary change) has no hunks.
func Hunks(diff string) []Hunk {
	var out []Hunk
	idx := -1
	for _, line := range strings.Split(strings.TrimRight(diff, "\n"), "\n") {
		if strings.HasPrefix(line, "@@") {
			start, count := hunkNewRange(line)
			out = append(out, Hunk{Header: line, NewStart: start, NewCount: count})
			idx = len(out) - 1
			continue
		}
		if idx < 0 {
			continue
		}
		out[idx].Body = append(out[idx].Body, line)
	}
	return out
}

// NewEnd is the last line of the hunk on the new side. A pure-deletion hunk has
// a count of zero, so its end is its start: the line the removal sits above.
func (h Hunk) NewEnd() int {
	if h.NewCount <= 0 {
		return h.NewStart
	}
	return h.NewStart + h.NewCount - 1
}

// hunkNewRange reads the new-side start and count from a hunk header, from the
// "+c,d" field, for example 12 and 6 in "@@ -3,4 +12,6 @@". A field with no
// comma, such as "+12", counts as one line.
func hunkNewRange(header string) (start, count int) {
	plus := strings.IndexByte(header, '+')
	if plus < 0 {
		return 0, 0
	}
	field := header[plus+1:]
	if space := strings.IndexByte(field, ' '); space >= 0 {
		field = field[:space]
	}
	if comma := strings.IndexByte(field, ','); comma >= 0 {
		return atoi(field[:comma]), atoi(field[comma+1:])
	}
	return atoi(field), 1
}
