#!/bin/bash
# Экспорт графиков в doc/images (стиль как в HTML: сетка, легенда справа).
#
#   ./export_trajectory_images.sh 4blades
#   ./export_trajectory_images.sh 4blades --export-only
#   ./export_trajectory_images.sh 4blades --prez --export-only   # только trajectories, δ, Δφ → doc/images/prez
#
# Без --prez: полный набор plot_points.py в doc/images.
# С --prez: trajectories.png, decrement_deltas.png, interblade_phase.png в doc/images/prez.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

export PARALLEL="${PARALLEL:-false}"
export CONFIG="${CONFIG:-1blade}"
export THESIS_EXPORT=true

EXPORT_ONLY=false
PREZ=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --export-only)
            EXPORT_ONLY=true
            shift
            ;;
        --prez)
            PREZ=true
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

if $PREZ; then
    OUT_DIR="$ROOT_DIR/doc/images/prez"
    export PREZ_EXPORT_ONLY=true
    # Подписи на decrement_deltas.png — поменяйте значения под расчёт.
    : "${DECREMENT_ANNOT_DELTA:=-0.313}"
    : "${DECREMENT_ANNOT_DELTA0:=-0.04994}"
    export DECREMENT_ANNOT_DELTA DECREMENT_ANNOT_DELTA0
    : "${INTERBLADE_PHASE_ANNOT_VALUE:=90}"
    export INTERBLADE_PHASE_ANNOT_VALUE
else
    OUT_DIR="$ROOT_DIR/doc/images"
    unset PREZ_EXPORT_ONLY
    unset DECREMENT_ANNOT_DELTA
    unset DECREMENT_ANNOT_DELTA0
    unset INTERBLADE_PHASE_ANNOT_VALUE
fi
export PLOTS_OUTPUT_DIR="$OUT_DIR"

if ! $EXPORT_ONLY; then
    echo "Расчёт CONFIG=$CONFIG …"
    (cd "$SCRIPT_DIR" && ./run.sh)
fi

mkdir -p "$OUT_DIR"
echo "Построение графиков (HTML-стиль) → $OUT_DIR (CONFIG=$CONFIG)"
python3 "$ROOT_DIR/plots/plot_points.py"

if $PREZ; then
    echo "Готово: trajectories.png, decrement_deltas.png, interblade_phase.png → $OUT_DIR"
else
    echo "Готово: $OUT_DIR/trajectories.png и связанные графики"
fi
