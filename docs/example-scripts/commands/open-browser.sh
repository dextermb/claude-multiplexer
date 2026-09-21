#!/usr/bin/env bash
# A key command that opens the default browser at its home page.
#
# A command reads the selected session as JSON on its stdin, the same payload a
# custom bar element reads. This example does not use it, so it drains stdin and
# ignores it. The interface reads the first line of stdout as a status notice, so
# the script prints one short line. See docs/config/commands.md.
#
# In the settings file, bind a trigger to this script:
#
#   "commands": [
#     { "keys": "b o", "label": "browser", "script": "commands/open-browser.sh" }
#   ]
cat >/dev/null

url="https://example.com"

case "$(uname)" in
Darwin) open "$url" ;;
*) xdg-open "$url" >/dev/null 2>&1 ;;
esac

printf 'opened %s\n' "$url"
