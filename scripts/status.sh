#!/usr/bin/env bash
set -euo pipefail

APP_NAME="PKdictation"

if ! command -v osascript >/dev/null 2>&1; then
	echo "osascript not found; cannot check app status." >&2
	exit 2
fi

# Avoid relying on `pgrep`/`ps`, which may be restricted in some environments.
if osascript -e "application \"$APP_NAME\" is running" 2>/dev/null | grep -qx "true"; then
	echo "$APP_NAME is running."
	exit 0
fi

echo "$APP_NAME is not running."
exit 1
