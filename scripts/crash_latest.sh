#!/usr/bin/env bash
set -euo pipefail

APP_NAME="PKdictation"
REPORT_DIR="$HOME/Library/Logs/DiagnosticReports"

if [[ ! -d "$REPORT_DIR" ]]; then
	echo "No crash report directory at: $REPORT_DIR"
	exit 0
fi

latest="$(ls -t "$REPORT_DIR"/"$APP_NAME"-*.ips 2>/dev/null | head -n 1 || true)"
if [[ -z "$latest" ]]; then
	echo "No $APP_NAME crash reports found in: $REPORT_DIR"
	exit 0
fi

echo "Latest crash report:"
echo "  $latest"
echo

head -n 1 "$latest" || true
echo

if command -v jq >/dev/null 2>&1; then
	echo "Summary:"
	tail -n +2 "$latest" | jq '{
		captureTime,
		osVersion: .osVersion.train + " (" + .osVersion.build + ")",
		exception: (.exception.type + " " + .exception.signal),
		termination: .termination.indicator,
		faultingThread,
		topFrames: (
			.threads[]
			| select(.triggered == true)
			| .frames[0:15]
			| map(.symbol // ("imageIndex=" + (.imageIndex|tostring)))
		)
	}' || true
else
	echo "jq not found; showing the first 80 lines:"
	sed -n '1,80p' "$latest"
fi
