#!/usr/bin/env bash
set -euo pipefail

if [ $# -ne 2 ]; then
    echo "usage: scripts/install-as.sh NAME MAIN_PACKAGE" >&2
    exit 2
fi

name="$1"
package="$2"

cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

dir="$HOME/.local/bin"
target="$dir/$name"
mkdir -p "$dir"
rm -f "$target"
go build -o "$target" "$package"

head="$(git rev-parse HEAD)"
built="$(go version -m "$target" | sed -n 's/^.*vcs\.revision=//p')"
if [ "$built" != "$head" ]; then
    echo "the binary reports ${built:-no revision}, but this tree is at $head — the build did not take"
    exit 1
fi
echo "built: $target at ${head:0:7}"

case ":$PATH:" in
    *":$dir:"*) echo "$dir is already on PATH" ;;
    *) echo "export PATH=\"$dir:\$PATH\"" >> "$HOME/.zshrc"
       echo "added $dir to PATH in ~/.zshrc — open a new shell or run: export PATH=\"$dir:\$PATH\"" ;;
esac

found="$(command -v "$name" || true)"
if [ -n "$found" ] && [ "$found" != "$target" ]; then
    echo "warning: $name on PATH is $found, so it hides the binary you just built"
fi
