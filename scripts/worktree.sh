#!/usr/bin/env bash
set -euo pipefail

if [ $# -ne 1 ]; then
    echo "usage: scripts/worktree.sh NAME" >&2
    exit 2
fi

name="$1"
root="$(dirname "$(git rev-parse --path-format=absolute --git-common-dir)")"
dir="$root/.worktrees/$name"

git worktree add -b "$name" "$dir" master
echo "ready: $dir on branch $name — do the work there, then 'just collapse $name'"
