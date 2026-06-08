#!/bin/bash
# Экспорт амплитуд, энергии, декремента и Δφ в doc/images (стиль HTML).
#
#   ./export_metrics_images.sh 4blades
#   ./export_metrics_images.sh 4blades --export-only

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
OUT_DIR="$ROOT_DIR/doc/images"

export PARALLEL="${PARALLEL:-false}"
export CONFIG="${CONFIG:-4blades}"
export PLOTS_OUTPUT_DIR="$OUT_DIR"

EXPORT_ONLY=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --export-only)
            EXPORT_ONLY=true
            shift
            ;;
        --only)
            shift 2
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
echo "Построение метрик (HTML-стиль) → $OUT_DIR (CONFIG=$CONFIG)"
python3 "$ROOT_DIR/plots/plot_points.py"

echo "Готово: $OUT_DIR/{amplitudes,sum_energy,decrement_deltas,interblade_phase}.png"
