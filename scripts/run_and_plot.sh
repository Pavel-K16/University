#!/bin/bash

start_ts=$(date +%s)
# Если передан аргумент — имя конфига (без .json),
# прокидываем его в переменную окружения CONFIG,
# чтобы и Go-программа, и Python-скрипт строили всё по одному и тому же конфигу.
if [ -n "$1" ]; then
    export CONFIG="$1"
fi

# Второй аргумент — PARALLEL=true (nsolv sweep): в Python строятся карты decrementStore и freqStore.
# При PARALLEL=false эти карты не строятся. Если не задан — false.
export PARALLEL=false
if [ -n "$2" ]; then
    lower="$(printf '%s' "$2" | tr '[:upper:]' '[:lower:]')"
    case "$lower" in
        1|true|yes|on)
            export PARALLEL=true
            ;;
        *)
            export PARALLEL=false
            ;;
    esac
fi

# 1) Запускаем основной расчёт (run.sh читает CONFIG и PARALLEL из окружения)
./run.sh

# 2) Графики и HTML в plots/ (автомасштаб, сырые δ/Δφ)
python3 ../plots/plot_points.py

# 3) Экспорт в doc/images/ (фиксированные оси δ/Δφ для диплома)
DOC_IMAGES_DIR="../doc/images"
mkdir -p "$DOC_IMAGES_DIR"
PLOTS_OUTPUT_DIR="$DOC_IMAGES_DIR" THESIS_EXPORT=true python3 ../plots/plot_points.py
echo "Экспорт в $DOC_IMAGES_DIR: trajectories, amplitudes, sum_energy, decrement_deltas, interblade_phase"

# xdg-open ../plots/index.html
end_ts=$(date +%s)

echo "Overall time taken: $((end_ts - start_ts)) seconds"