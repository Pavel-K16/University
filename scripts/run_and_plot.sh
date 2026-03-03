#!/bin/bash

# 1) Запускаем основной расчёт
./run.sh "$@"

# 2) После завершения строим графики и HTML
python3 ../plots/plot_points.py

# xdg-open ../plots/index.html
