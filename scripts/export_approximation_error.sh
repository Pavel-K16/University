#!/bin/bash
# График погрешности err(τ) по таблице 1 (КПА, с. 10) → doc/images/

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

python3 "$ROOT_DIR/plots/export_approximation_error.py" "$@"
echo "Готово: $ROOT_DIR/doc/images/approximation_error.png"
