# Task Manager

## Запуск

1. Создать `.env` из `.env.example`
2. Заполнить в `.env` значение `JWT_SECRET`
3. Запустить контейнеры

```powershell
docker compose up --build
```

## Проверка

Можно проверить, например, с windows
```powershell
Invoke-RestMethod http://localhost:8080/health
```