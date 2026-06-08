#!/bin/bash
# Экспорт графиков в doc/images (стиль как в HTML: сетка, легенда справа).
#
#   ./export_trajectory_images.sh 4blades
#   ./export_trajectory_images.sh 4blades --export-only
#
# Строит trajectories.png и остальные графики plot_points.py в doc/images.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
OUT_DIR="$ROOT_DIR/doc/images"

export PARALLEL="${PARALLEL:-false}"
export CONFIG="${CONFIG:-1blade}"
export PLOTS_OUTPUT_DIR="$OUT_DIR"

EXPORT_ONLY=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --export-only)
            EXPORT_ONLY=true
            shift
            ;;
        --all|--combined|--node|--fixed-y|--auto-y|--t-max)
            shift
            [[ "$1" == --node || "$1" == --t-max ]] && shift
            ;;
        --*)
            shift
            ;;
        *)
            export CONFIG="$1"
            shift
            ;;
    esac
done

if ! $EXPORT_ONLY; then
    echo "Расчёт CONFIG=$CONFIG …"
    (cd "$SCRIPT_DIR" && ./run.sh)
fi

mkdir -p "$OUT_DIR"
echo "Построение графиков (HTML-стиль) → $OUT_DIR (CONFIG=$CONFIG)"
python3 "$ROOT_DIR/plots/plot_points.py"

echo "Готово: $OUT_DIR/trajectories.png и связанные графики"
