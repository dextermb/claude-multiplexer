package tui

import (
	"image/color"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
)

func TestTokensConvertToTheXtermColoursInTheDocs(t *testing.T) {
	cases := []struct {
		name  string
		token color.Color
		xterm ansi.IndexedColor
	}{
		{"colGround", colGround, 16},
		{"colSurface", colSurface, 232},
		{"colCode", colCode, 235},
		{"colFg", colFg, 231},
		{"colHeading", colHeading, 255},
		{"colSecondary", colSecondary, 251},
		{"colMuted", colMuted, 245},
		{"colDimmed", colDimmed, 242},
		{"colFaint", colFaint, 237},
		{"colBorder", colBorder, 237},
		{"colSubtle", colSubtle, 234},
		{"colScrim", colScrim, 235},
		{"colAccent", colAccent, 231},
		{"colAccentFg", colAccentFg, 16},
		{"colSubdued", colSubdued, 236},
		{"colPositive", colPositive, 78},
		{"colWarning", colWarning, 214},
		{"colDanger", colDanger, 203},
		{"colInfo", colInfo, 75},
	}
	for _, c := range cases {
		got := colorprofile.ANSI256.Convert(c.token)
		if got != c.xterm {
			t.Errorf("%s converts to %v, the docs say %d", c.name, got, c.xterm)
		}
	}
}
