#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

CONFIGURATION="${1:-Debug}"
PROJECT="$ROOT_DIR/PKdictation.xcodeproj"
SCHEME="PKdictation"
DERIVED_DATA_PATH="$ROOT_DIR/.derivedData"
ARCH="$(uname -m)"
DESTINATION="platform=macOS,arch=$ARCH"

xcodebuild \
	-project "$PROJECT" \
	-scheme "$SCHEME" \
	-configuration "$CONFIGURATION" \
	-derivedDataPath "$DERIVED_DATA_PATH" \
	-destination "$DESTINATION" \
	build

APP_PATH="$DERIVED_DATA_PATH/Build/Products/$CONFIGURATION/PKdictation.app"
echo "Built app: $APP_PATH"
