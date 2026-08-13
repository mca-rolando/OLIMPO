#!/usr/bin/env bash
set -euo pipefail

E2E_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"$E2E_DIR/server-e2e.sh"
"$E2E_DIR/upgrade-26.08-01-to-26.08-02.sh"
