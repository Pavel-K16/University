#!/bin/bash
# Численное vs аналитическое для D=0.1 и D=-0.1 → doc/images/
#
#   ./export_damping_comparison.sh
#   ./export_damping_comparison.sh --no-run   # только перерисовка

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

PY_ARGS=()
for arg in "$@"; do
    PY_ARGS+=("$arg")
done

python3 "$ROOT_DIR/plots/export_damping_comparison.py" "${PY_ARGS[@]}"
echo "Готово: $ROOT_DIR/doc/images/damping_*.png"
