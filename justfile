binary := "bin/multiplexier"
main := "./cmd/multiplexier"

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
    #!/usr/bin/env bash
    set -euo pipefail
    dir="$HOME/.local/bin"
    mkdir -p "$dir"
    go build -o "$dir/{{NAME}}" {{main}}
    echo "built: $dir/{{NAME}}"
    case ":$PATH:" in
        *":$dir:"*) echo "$dir is already on PATH" ;;
        *) echo "export PATH=\"$dir:\$PATH\"" >> "$HOME/.zshrc"
           echo "added $dir to PATH in ~/.zshrc — open a new shell or run: export PATH=\"$dir:\$PATH\"" ;;
    esac

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
    #!/usr/bin/env bash
    set -euo pipefail
    root="$(dirname "$(git rev-parse --path-format=absolute --git-common-dir)")"
    dir="$root/.worktrees/{{NAME}}"
    git worktree add -b "{{NAME}}" "$dir" master
    echo "ready: $dir on branch {{NAME}} — do the work there, then 'just collapse {{NAME}}'"

# List the worktrees and their branches.
worktrees:
    git worktree list

# Merge a green, committed worktree back into master, then remove it. Run from the main worktree on master.
collapse NAME:
    #!/usr/bin/env bash
    set -euo pipefail
    root="$(dirname "$(git rev-parse --path-format=absolute --git-common-dir)")"
    dir="$root/.worktrees/{{NAME}}"
    if [ -n "$(git -C "$dir" status --porcelain)" ]; then
        echo "commit the work in $dir before you collapse it"; exit 1
    fi
    git merge "{{NAME}}"
    go test ./...
    git worktree remove "$dir"
    git branch -d "{{NAME}}"
    echo "collapsed {{NAME}} into master"

# Remove the build output and the coverage profile.
clean:
    rm -rf bin coverage.out

# Remove the session state in ~/.multiplexier.
clean-state:
    rm -rf ~/.multiplexier/sessions

# Warning: this drives one real session through the interface, and it costs money.
test-real:
    MULTIPLEXIER_REAL=1 go test ./internal/tui/ -run TestRealSessionThroughTheInterface -v -timeout 300s

# Warning: this calls the real Claude API and it costs money.
smoke:
    go run {{main}} run --dir /tmp --model claude-haiku-4-5-20251001 "Reply with exactly one word: pong"
