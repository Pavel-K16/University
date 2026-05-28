#!/bin/bash

# Путь к файлу main.go
MAIN_FILE="../cmd/nsolv.go"

# Если передан аргумент — имя конфига (без .json), прокидываем в переменную окружения CONFIG
if [ -n "$1" ]; then
    export CONFIG="$1"
fi

# Компилируем и запускаем (запуск из корня проекта: scripts/run.sh [conf])
go build -o myapp "$MAIN_FILE"

if [ $? -eq 0 ]; then
    ./myapp
    rm -f myapp
else
    echo "Ошибка компиляции."
fi