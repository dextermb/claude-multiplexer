#!/usr/bin/env bash
# A custom bar element that rotates between two words on each refresh.
#
# A bar script starts fresh every refresh, so it cannot hold state in memory.
# This script keeps the last word in a state file, and toggles it each run, so
# the bar shows "hello", then "world!", then "hello", and so on.
#
# The bar reads the first line of stdout, so the script writes the state file on
# its own and prints only the word to show. See docs/config/bars.md.
#
# In the settings file, point a custom element at this script and refresh it once
# a second:
#
#   "bars": { "status": { "right": [
#     { "script": "bars/hello-world.sh", "label": "hello", "refresh": "1s" }
#   ] } }
state="${TMPDIR:-/tmp}/cmux-hello-world.state"

last=""
[ -f "$state" ] && last=$(cat "$state")

if [ "$last" = "hello" ]; then
	next="world!"
else
	next="hello"
fi

printf '%s\n' "$next" >"$state"
printf '%s\n' "$next"
