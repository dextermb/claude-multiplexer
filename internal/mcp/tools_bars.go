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
	Session json.RawMessage `json:"session"`
	Status  json.RawMessage `json:"status"`
	Note    string          `json:"note"`
}

func barDefaults() (barDefaultsOut, error) {
	session, err := config.DefaultBarJSON(config.BarSession)
	if err != nil {
		return barDefaultsOut{}, err
	}
	status, err := config.DefaultBarJSON(config.BarStatus)
	if err != nil {
		return barDefaultsOut{}, err
	}
	return barDefaultsOut{
		Session: json.RawMessage(session),
		Status:  json.RawMessage(status),
		Note: "Write a copy with " + ToolSetConfig + " on 'bars.session' or 'bars.status'. " +
			"A bare string is a built-in id. An object {\"script\":\"path.sh\",\"label\":\"name\"} runs a Go, Python, or Bash script and shows its first line of output.",
	}, nil
}
