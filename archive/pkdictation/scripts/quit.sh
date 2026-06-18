#!/usr/bin/env bash
set -euo pipefail

APP_NAME="PKdictation"

if ! command -v osascript >/dev/null 2>&1; then
	echo "osascript not found; cannot quit app politely." >&2
	exit 2
fi

osascript -e "tell application \"$APP_NAME\" to quit" >/dev/null 2>&1 || true
echo "Requested quit for $APP_NAME."
