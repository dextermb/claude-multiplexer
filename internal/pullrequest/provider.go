package pullrequest

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// provider builds one batched GraphQL query for many branches, and reads the
// response back into one Result per branch. See docs/pull-requests.md.
type provider interface {
	fetch(ctx context.Context, t transport, queries []Query) []Result
}

func providerFor(name string) provider {
	if name == config.PullRequestGitLab {
		return gitlab{}
	}
	return github{}
}

// runFetch is the shared flow: build the query, send it, then parse. A
// transport error marks every query in the chunk with that error.
func runFetch(
	ctx context.Context,
	t transport,
	queries []Query,
	build func([]Query) (string, map[string]string),
	parse func(json.RawMessage, []Query) []Result,
) []Result {
	query, vars := build(queries)
	data, err := t.graphql(ctx, query, vars)
	if err != nil {
		out := make([]Result, len(queries))
		for i := range out {
			out[i] = Result{Err: err}
		}
		return out
	}
	return parse(data, queries)
}

// aliasOf names the GraphQL alias of the i-th query in a batch.
func aliasOf(i int) string {
	return "a" + strconv.Itoa(i)
}
