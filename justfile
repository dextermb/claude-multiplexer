binary := "bin/multiplexer"
main := "./cmd/multiplexer"

# List the recipes.
default:
    @just --list

# Build the binary into bin/.
build:
    go build -o {{binary}} {{main}}

# Install the binary into GOBIN.
install:
    go install {{main}}

# Build the binary under NAME into ~/.local/bin, and add that directory to PATH.
install-as NAME="cmux":
    scripts/install-as.sh {{NAME}} {{main}}

# Start the terminal user interface.
tui *ARGS:
    go run {{main}} {{ARGS}}

# Start the interface with one session already open in DIR.
tui-in DIR=".":
    go run {{main}} --dir {{DIR}}

# Start the interface against the fake binary, with no network and no cost.
tui-fake:
    go build -o bin/fakeclaude ./internal/testutil/fakeclaude
    go run {{main}} --claude bin/fakeclaude --dir .

# Run any command. Example: just run run --dir . "hello"
run *ARGS:
    go run {{main}} {{ARGS}}

# Send one prompt to a session in DIR. Example: just ask "hello" /tmp
ask PROMPT DIR=".":
    go run {{main}} run --dir {{DIR}} {{quote(PROMPT)}}

# Send one prompt, and show the state changes and the full tool results.
ask-verbose PROMPT DIR=".":
    go run {{main}} run -v --dir {{DIR}} {{quote(PROMPT)}}

# Run the tests. Example: just test ./internal/session/
test PKG="./...":
    go test {{PKG}}

# Run the tests with the race detector.
test-race PKG="./...":
    go test -race {{PKG}}

# Run the tests three times, to catch a flaky one.
test-repeat PKG="./...":
    go test -race -count=3 {{PKG}}

# Run one test by name. Example: just test-one TestSessionRunsOneTurn
test-one NAME PKG="./...":
    go test -race -run {{quote(NAME)}} -v {{PKG}}

# Write a coverage profile, and open the report.
cover PKG="./...":
    go test -coverprofile=coverage.out {{PKG}}
    go tool cover -func=coverage.out | tail -1
    go tool cover -html=coverage.out

# Format the code.
fmt:
    gofmt -w .

# Fail if any file needs formatting.
fmt-check:
    @test -z "$(gofmt -l .)" || { echo "these files need gofmt:"; gofmt -l .; exit 1; }

# Report suspicious constructs.
vet:
    go vet ./...

# Tidy the module requirements.
tidy:
    go mod tidy

# Run every check before a commit.
check: fmt-check vet test-repeat

# Start a worktree on a new branch off master, for one effort. Example: just worktree add-auth
worktree NAME:
    scripts/worktree.sh {{NAME}}

# List the worktrees and their branches.
worktrees:
    git worktree list

# Merge a green, committed worktree back into master, then remove it. Run from the main worktree on master.
collapse NAME:
    scripts/collapse.sh {{NAME}}

# Remove the build output and the coverage profile.
clean:
    rm -rf bin coverage.out

# Remove the session state in ~/.multiplexier.
clean-state:
    rm -rf ~/.multiplexier/sessions

# Warning: this drives one real session through the interface, and it costs money.
test-real:
    MULTIPLEXIER_REAL=1 go test ./internal/tui/ -run TestRealSessionThroughTheInterface -v -timeout 300s

# Warning: this drives one real session that calls the mux tools, and it costs money.
probe-mcp:
    MULTIPLEXIER_REAL=1 go test ./internal/manager/ -run TestRealSessionCallsTheTools -v -timeout 300s

# Warning: this calls the real Claude API and it costs money.
smoke:
    go run {{main}} run --dir /tmp --model claude-haiku-4-5-20251001 "Reply with exactly one word: pong"

# Warning: this drives one real background job, to answer which id links BashOutput to a job. It costs money.
probe-jobs:
    MULTIPLEXIER_REAL=1 go test ./internal/manager/ -run TestRealSessionRunsABackgroundJob -v -timeout 300s

# Start the interface against the fake binary, with one background job already running.
tui-jobs:
    go build -o bin/fakeclaude ./internal/testutil/fakeclaude
    FAKECLAUDE_MODE=jobs go run {{main}} --claude bin/fakeclaude --dir .
