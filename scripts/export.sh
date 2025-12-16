#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

CONFIGURATION="${1:-Debug}"
DEST_DIR="${2:-$ROOT_DIR/dist}"

"$ROOT_DIR/scripts/build.sh" "$CONFIGURATION"

APP_SRC="$ROOT_DIR/.derivedData/Build/Products/$CONFIGURATION/PKdictation.app"
APP_DEST="$DEST_DIR/PKdictation.app"

mkdir -p "$DEST_DIR"
rm -rf "$DEST_DIR/PKDication.app" >/dev/null 2>&1 || true
rm -rf "$APP_DEST"
ditto "$APP_SRC" "$APP_DEST"

echo "Exported app: $APP_DEST"
