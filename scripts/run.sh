#!/bin/bash

# Путь к файлу main.go 
MAIN_FILE="../cmd/nsolv.go"

# Компилируем файл main.go
go build -o myapp "$MAIN_FILE"

# Проверяем, была ли компиляция успешной
if [ $? -eq 0 ]; then
    ./myapp
    rm myapp 
else
    echo "Ошибка компиляции."
fi