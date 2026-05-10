#!/bin/bash

start_ts=$(date +%s)
# Если передан аргумент — имя конфига (без .json),
# прокидываем его в переменную окружения CONFIG,
# чтобы и Go-программа, и Python-скрипт строили всё по одному и тому же конфигу.
if [ -n "$1" ]; then
    export CONFIG="$1"
fi

# Второй аргумент — параллельный режим (sweep / только карты декремента в Python).
# Если не задан — считаем false.
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

# 2) После завершения строим графики и HTML (plot_points.py тоже читает CONFIG и PARALLEL)
python3 ../plots/plot_points.py

# xdg-open ../plots/index.html
end_ts=$(date +%s)

echo "Overall time taken: $((end_ts - start_ts)) seconds"