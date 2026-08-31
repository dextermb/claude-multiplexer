package tui

import "strings"

// contextWindow returns the context-window size for a model, or 0 when the
// model is not known. The bar shows a percentage only for a known window.
func contextWindow(model string) int {
	name := strings.ToLower(model)
	switch {
	case name == "":
		return 0
	case strings.Contains(name, "[1m]"), strings.Contains(name, "-1m"):
		return 1_000_000
	case strings.Contains(name, "claude"):
		return 200_000
	default:
		return 0
	}
}
