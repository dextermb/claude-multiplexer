#!/usr/bin/env bash
set -euo pipefail

if [ $# -ne 1 ]; then
    echo "usage: scripts/collapse.sh NAME" >&2
    exit 2
fi

name="$1"
root="$(dirname "$(git rev-parse --path-format=absolute --git-common-dir)")"
dir="$root/.worktrees/$name"

cd "$root"

if [ -n "$(git -C "$dir" status --porcelain)" ]; then
    echo "commit the work in $dir before you collapse it"
    exit 1
fi

if ! git merge --no-ff --no-commit "$name"; then
    git merge --abort
    echo "the merge of $name has conflicts — settle them by hand, then commit and collapse again"
    exit 1
fi

if ! go test ./...; then
    git merge --abort
    echo "the tests fail after the merge, so nothing landed on master — fix $name in $dir and collapse again"
    exit 1
fi

git commit --no-edit
git worktree remove "$dir"
git branch -d "$name"
echo "collapsed $name into master"
