#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIGURATION="${1:-Debug}"

"$ROOT_DIR/scripts/build.sh" "$CONFIGURATION"
echo "Build OK ($CONFIGURATION)."

