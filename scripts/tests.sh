#!/bin/bash
set -euo pipefail

# Run all Go tests located in ./tests with verbose output.
# Script can be executed from any working directory.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${ROOT_DIR}"

echo "Running Go tests in ./tests (no cache, verbose)..."

# The project logger writes DEBUG lines to stdout. To keep test output readable,
# we filter DEBUG lines while preserving the original exit code (pipefail).
#
# Set SHOW_DEBUG=1 to disable filtering.
if [ "${SHOW_DEBUG:-0}" = "1" ]; then
  go test -count=1 -v ./tests
else
  go test -count=1 -v ./tests 2>&1 | sed '/DEBUG/d'
fi

