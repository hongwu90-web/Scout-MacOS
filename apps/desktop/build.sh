#!/bin/bash
set -euo pipefail

# Delegate to scripts/build-macos-app.sh to maintain full compatibility
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

exec "$ROOT_DIR/scripts/build-macos-app.sh" "$@"
