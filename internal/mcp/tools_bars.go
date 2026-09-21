package mcp

import (
	"context"
	"encoding/json"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// addBarDefaultsTool gives every session the default composition of the two
// status bars, so an agent reads the default, then writes a copy it edits with
// set_config. See docs/config/bars.md.
func (s *Server) addBarDefaultsTool(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: ToolBarDefaults,
		Description: "The default composition of the two status bars: the session bar and the status bar. " +
			"Each is an ordered list of element ids, with left and right sides. " +
			"Read this to see the built-in elements and their order, then write a copy with " +
			ToolSetConfig + " on 'bars.session' or 'bars.status' to reorder them, remove one, or add a custom script element.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, barDefaultsOut, error) {
		out, err := barDefaults()
		return nil, out, err
	})
}

type barDefaultsOut struct {
	Session barSpecOut `json:"session"`
	Status  barSpecOut `json:"status"`
	Note    string     `json:"note"`
}

// barSpecOut is the shape the tool returns for one bar: an ordered list of
// built-in element ids per side. The defaults hold only built-in ids, so a
// string list matches the value and the generated output schema agree.
type barSpecOut struct {
	Left  []string `json:"left"`
	Right []string `json:"right"`
}

func barDefaults() (barDefaultsOut, error) {
	session, err := barSpecOutFor(config.BarSession)
	if err != nil {
		return barDefaultsOut{}, err
	}
	status, err := barSpecOutFor(config.BarStatus)
	if err != nil {
		return barDefaultsOut{}, err
	}
	return barDefaultsOut{
		Session: session,
		Status:  status,
		Note: "Write a copy with " + ToolSetConfig + " on 'bars.session' or 'bars.status'. " +
			"A bare string is a built-in id. An object {\"script\":\"path.sh\",\"label\":\"name\"} runs a Go, Python, or Bash script and shows its first line of output.",
	}, nil
}

func barSpecOutFor(bar string) (barSpecOut, error) {
	data, err := config.DefaultBarJSON(bar)
	if err != nil {
		return barSpecOut{}, err
	}
	var out barSpecOut
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		return barSpecOut{}, err
	}
	return out, nil
}
