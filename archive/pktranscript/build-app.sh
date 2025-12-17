#!/usr/bin/env bash
set -euo pipefail

APP_NAME="PKTranscript"
BUNDLE_ID="com.example.pktranscript"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST_DIR="$ROOT_DIR/dist"
APP_DIR="$DIST_DIR/${APP_NAME}.app"

BIN_DIR="$APP_DIR/Contents/MacOS"
RES_DIR="$APP_DIR/Contents/Resources"

mkdir -p "$BIN_DIR" "$RES_DIR"

echo "Building binary..."
ARCH="${GOARCH:-}"
if [[ -z "$ARCH" ]]; then
  case "$(uname -m)" in
    arm64) ARCH="arm64" ;;
    x86_64) ARCH="amd64" ;;
    *) ARCH="arm64" ;;
  esac
fi
GOOS=darwin GOARCH="$ARCH" go build -o "$BIN_DIR/pktranscript" .

cat > "$APP_DIR/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>${APP_NAME}</string>
  <key>CFBundleDisplayName</key><string>${APP_NAME}</string>
  <key>CFBundleIdentifier</key><string>${BUNDLE_ID}</string>
  <key>CFBundleVersion</key><string>1</string>
  <key>CFBundleShortVersionString</key><string>0.1.0</string>
  <key>CFBundleExecutable</key><string>pktranscript</string>
  <key>LSUIElement</key><true/>
  <key>NSMicrophoneUsageDescription</key><string>PKTranscript a besoin du micro pour transcrire votre voix.</string>
  <key>NSSpeechRecognitionUsageDescription</key><string>PKTranscript a besoin de la reconnaissance vocale pour convertir votre voix en texte.</string>
</dict>
</plist>
EOF

echo -n "APPL????" > "$APP_DIR/Contents/PkgInfo"

echo "Ad-hoc signing (recommended for permissions prompts)..."
codesign --force --deep --sign - "$APP_DIR" >/dev/null 2>&1 || true

echo "Built: $APP_DIR"
