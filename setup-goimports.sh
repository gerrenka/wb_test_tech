#!/bin/bash

echo "🚀 Проверка установки goimports..."

# Проверка, установлен ли goimports
if ! command -v goimports &> /dev/null
then
    echo "🔧 goimports не найден, устанавливаю..."
    go install golang.org/x/tools/cmd/goimports@latest

    # Добавим GOPATH/bin в PATH, если нужно
    GOPATH_BIN=$(go env GOPATH)/bin
    if [[ ":$PATH:" != *":$GOPATH_BIN:"* ]]; then
        echo "📦 Добавляю $GOPATH_BIN в PATH (в ~/.zshrc)..."
        echo "export PATH=\"\$PATH:$GOPATH_BIN\"" >> ~/.zshrc
        source ~/.zshrc
    fi
else
    echo "✅ goimports уже установлен"
fi

echo "✨ Привожу импорты в порядок во всем проекте..."
goimports -w .

echo "✅ Готово!"
echo "✨ Привожу импорты в порядок во всем проекте..."
goimports -w . 2>&1 | tee -a goimports.log
echo "✅ Готово!"