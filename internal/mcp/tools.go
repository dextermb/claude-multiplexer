package mcp

import (
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) build(caller string, control bool) *sdk.Server {
	server := sdk.NewServer(
		&sdk.Implementation{Name: ServerName, Version: version},
		&sdk.ServerOptions{Instructions: instructions()},
	)

	s.addReadTools(server, caller)
	s.addConfigTools(server, caller)

	if control {
		s.addControlTools(server, caller)
	}

	return server
}

// targetOrSelf reads a session argument, and it returns the caller when the
// argument is empty, so a job tool defaults to the calling session.
func targetOrSelf(arg, caller string) (string, error) {
	if strings.TrimSpace(arg) == "" {
		return caller, nil
	}
	return cleanTarget(arg)
}
