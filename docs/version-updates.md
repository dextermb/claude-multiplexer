# Update check

The multiplexer warns the user when a newer release exists. It reads the latest
release from GitHub once an hour, compares it to the running binary, and shows a
banner below the status bar when the binary is out of date. A key dismisses the
banner for a day. The check is in `internal/update`, and the banner is in
`internal/tui/updatecheck.go`.

## The comparison

The running binary reads its own version from the Go build info, the same source
as `cmux version`. Two cases decide "out of date":

1. The version is a real tag (`go install ...@v1.2.3` or `...@latest`). Compare
   the tag to the latest release tag with a semver compare. The binary is out of
   date when the release tag is greater.
2. The version is `(devel)` or empty (a build from source). A commit hash can
   not compare to a tag, so compare the build commit time (`vcs.time`) to the
   release `published_at`. The binary is out of date when the release is newer.
   When the build carries no commit time, no banner shows.

`go install ...@latest` resolves to the latest tag, so it takes case 1. A module
install carries no `vcs.*` stamps, but case 1 does not need them.

Both sides drop the `v` prefix before the semver compare, because a module
version needs the prefix but a release tag may drop it.

## The GitHub call

The check reads `https://api.github.com/repos/dextermb/claude-multiplexer/releases/latest`.
The call needs no token, and sends a `User-Agent` header, which GitHub requires.
One call per hour stays inside the unauthenticated limit of 60 per hour.

The endpoint returns Releases, not bare tags. So the release workflow creates a
Release, which also creates the tag. See [Release automation](#release-automation)
below.

A repository with no release yet returns a 404. The check treats a 404 as "no
release", not an error, so no banner shows until the first release exists.

## The state file

The check stores its result in `<root>/update-check.json`, inside the state
directory (`~/.claude-multiplexer` by default). The file holds:

- `lastCheck` — when the check last reached GitHub, so a check runs at most once
  an hour.
- `tag`, `url`, `publishedAt` — the latest release the check last saw.
- `dismissedUntil` — when a dismissal runs out.

The file is a cache, not a setting, so it stays out of `config.json`.

## The banner and the key

The banner shows when the binary is out of date, the check is on, and the
dismissal has run out. The dismiss key sets `dismissedUntil` to a day ahead and
saves the state file, so the banner stays hidden for a day.

The dismiss key is the global action `global.dismissUpdate`, bound to `ctrl+g` by
default. A global control key reaches the banner from the prompt. The key is
rebindable, like every other. See [config/keybindings.md](config/keybindings.md).

## The opt-out

The `checkUpdates` field in `config.json` turns the check off. A nil value, or
true, keeps the check on; false turns it off. When the check is off, no call
goes to GitHub and no banner shows. See [config.md](config.md).

## Data flow

```
Init / hourly tick
  -> read update-check.json
  -> if the last check is older than 1 hour, fetch the latest release and save
  -> compute out-of-date; set the banner state
Dismiss key
  -> set dismissedUntil = now + 24h; save; hide the banner
```

## Release automation

A push to `master` creates a Release with a patch bump. A manual run of the
`Release` workflow takes a `patch`, `minor`, or `major` choice. The workflow
finds the latest `v*` tag, bumps the part, and creates the Release with the
GitHub CLI, which also creates the tag. The first release is `v0.0.1`, because
the bump starts from `v0.0.0`.

The workflow is `.github/workflows/release.yml`. It targets `master`, the
repository default branch. Creating a Release with the built-in token does not
trigger another workflow run, so the push trigger does not loop.
