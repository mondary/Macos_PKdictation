#!/usr/bin/env bash
set -euo pipefail

APP_NAME="PKvoice"
BUNDLE_ID="com.example.pkvoice"
APP_VERSION="${APP_VERSION:-2.1}"

SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SRC_DIR/.." && pwd)"
DIST_DIR="$ROOT_DIR/release"
APP_DIR="$DIST_DIR/${APP_NAME}.app"
APP_SRC_DIR="$SRC_DIR/app"

BIN_DIR="$APP_DIR/Contents/MacOS"
RES_DIR="$APP_DIR/Contents/Resources"

mkdir -p "$BIN_DIR" "$RES_DIR"

GO_CACHE_DIR="$SRC_DIR/.cache/go"
export GOCACHE="${GOCACHE:-$GO_CACHE_DIR/build}"
export GOMODCACHE="${GOMODCACHE:-$GO_CACHE_DIR/mod}"
mkdir -p "$GOCACHE" "$GOMODCACHE"

echo "Building binary... (${APP_VERSION})"
ARCH="${GOARCH:-}"
if [[ -z "$ARCH" ]]; then
  case "$(uname -m)" in
    arm64) ARCH="arm64" ;;
    x86_64) ARCH="amd64" ;;
    *) ARCH="arm64" ;;
  esac
fi
(
  cd "$APP_SRC_DIR"
  GOOS=darwin GOARCH="$ARCH" go build -o "$BIN_DIR/pkvoice" .
)

ICON_SRC="$SRC_DIR/assets/PKvoice.icns"
ICON_DST="$RES_DIR/PKvoice.icns"
if [[ -f "$ICON_SRC" ]]; then
  cp "$ICON_SRC" "$ICON_DST"
fi

cat > "$APP_DIR/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>${APP_NAME}</string>
  <key>CFBundleDisplayName</key><string>${APP_NAME}</string>
  <key>CFBundleIdentifier</key><string>${BUNDLE_ID}</string>
  <key>CFBundleVersion</key><string>${APP_VERSION}</string>
  <key>CFBundleShortVersionString</key><string>${APP_VERSION}</string>
  <key>CFBundleExecutable</key><string>pkvoice</string>
  <key>CFBundleIconFile</key><string>PKvoice</string>
  <key>CFBundleIconName</key><string>PKvoice</string>
  <key>LSUIElement</key><true/>
  <key>NSMicrophoneUsageDescription</key><string>PKvoice a besoin du micro pour transcrire votre voix.</string>
  <key>NSSpeechRecognitionUsageDescription</key><string>PKvoice a besoin de la reconnaissance vocale pour convertir votre voix en texte.</string>
</dict>
</plist>
EOF

echo -n "APPL????" > "$APP_DIR/Contents/PkgInfo"

echo "Ad-hoc signing (recommended for permissions prompts)..."
codesign --force --deep --sign - "$APP_DIR" >/dev/null 2>&1 || true

echo "Built: $APP_DIR"
echo "Version: ${APP_VERSION}"
