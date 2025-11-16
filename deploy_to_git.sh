#!/bin/bash
# deploy_to_github.sh — один скрипт: создаёт репозиторий на GitHub и заливает проект
# Требования: gh CLI установлен и авторизован (gh auth login), git настроен
# Пользователь: wwsilin, email: wwsilin@ya.ru
# Выполнить: chmod +x deploy_to_github.sh && ./deploy_to_github.sh

set -e  # Остановка при ошибке

REPO_NAME="quiz-app"
USER="wwsilin"
EMAIL="wwsilin@ya.ru"

echo "Настройка Git..."
git config --global user.name "$USER"
git config --global user.email "$EMAIL"

echo "Инициализация Git-репозитория..."
if [ ! -d ".git" ]; then
  git init
fi

# Добавляем .gitignore
cat > .gitignore << 'EOF'
# Go
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out

# Binaries
quiz-app

# Logs
quiz.log

# IDE
.idea/
.vscode/

# OS
.DS_Store
Thumbs.db
EOF

echo "Добавление файлов в Git..."
git add .

echo "Первый коммит..."
git commit -m "Initial commit: full quiz application with CSV, timer, logging, adaptive UI" --no-edit || echo "Commit skipped (already exists)"

echo "Создание репозитория на GitHub через gh CLI..."
if ! gh repo view "$USER/$REPO_NAME" >/dev/null 2>&1; then
  gh repo create "$REPO_NAME" --public --source=. --remote=origin --push
  echo "Репозиторий $REPO_NAME создан и запушен!"
else
  echo "Репозиторий уже существует. Пушим изменения..."
  git remote add origin "https://github.com/$USER/$REPO_NAME.git" 2>/dev/null || true
  git branch -M main
  git push -u origin main
fi

echo "Готово! Проект залит на GitHub:"
echo "   https://github.com/$USER/$REPO_NAME"
echo ""
echo "Логи тестов: quiz.log (не заливается благодаря .gitignore)"
