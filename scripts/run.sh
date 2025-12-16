#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

CONFIGURATION="${1:-Debug}"
RELAUNCH="${2:-}"

"$ROOT_DIR/scripts/build.sh" "$CONFIGURATION"

APP_PATH="$ROOT_DIR/.derivedData/Build/Products/$CONFIGURATION/PKdictation.app"

if [[ "$RELAUNCH" == "--relaunch" ]]; then
	osascript -e 'tell application "PKdictation" to quit' >/dev/null 2>&1 || true
	sleep 0.3
	pkill -x PKdictation >/dev/null 2>&1 || true
	for _ in {1..20}; do
		if pgrep -x PKdictation >/dev/null 2>&1; then
			sleep 0.1
		else
			break
		fi
	done
fi

if [[ -d "$APP_PATH" ]]; then
	echo "Launching app (no Dock icon): $APP_PATH"
else
	echo "App not found at: $APP_PATH" >&2
	exit 1
fi

set +e
open_output="$(open "$APP_PATH" 2>&1)"
open_exit=$?
set -e

if [[ $open_exit -ne 0 ]]; then
	echo "$open_output" >&2
	echo >&2
	echo "Launch failed. If you're running this inside a sandboxed terminal, 'open' may be blocked." >&2
	echo "Try launching from Finder (double-click) or from Terminal.app:" >&2
	echo "  open \"$APP_PATH\"" >&2
	exit "$open_exit"
fi

sleep 0.3
if "$ROOT_DIR/scripts/status.sh" >/dev/null 2>&1; then
	"$ROOT_DIR/scripts/status.sh" || true
else
	echo "Tip: look for the 'waveform' icon in the macOS menu bar."
fi
