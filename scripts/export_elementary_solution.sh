#!/bin/bash
# Элементарное решение: numSol vs anSol (D=±0.1) + график err(τ) → doc/images/prez/
#
#   ./export_elementary_solution.sh
#   ./export_elementary_solution.sh --no-run

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
OUT_DIR="$ROOT_DIR/doc/images/prez"

PY_ARGS=(--output-dir "$OUT_DIR")
for arg in "$@"; do
    PY_ARGS+=("$arg")
done

python3 "$ROOT_DIR/plots/export_damping_comparison.py" "${PY_ARGS[@]}"
python3 "$ROOT_DIR/plots/export_approximation_error.py" --output "$OUT_DIR/approximation_error.png"

echo "Готово: $OUT_DIR/damping_*.png, $OUT_DIR/approximation_error.png"
