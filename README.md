# Task Manager

## Запуск

1. Создать `.env` из `.env.example`
2. Указать `JWT_SECRET` длиной не менее 32 символов
3. Запустить приложение, MySQL и Redis

```powershell
docker compose up --build
```

Повторный запуск не удаляет данные MySQL

## Проверка

```powershell
Invoke-RestMethod http://localhost:8080/health
```

## Конфигурация

| Переменная | Назначение | Значение по умолчанию в Docker Compose |
| --- | --- | --- |
| `HTTP_ADDR` | Адрес API внутри контейнера | `:8080` |
| `JWT_SECRET` | Ключ подписи JWT | Обязательная переменная |
| `JWT_TTL` | Срок действия JWT | `24h` |
| `APP_PORT` | Порт API на хосте | `8080` |
| `MYSQL_DSN` | Строка подключения API к MySQL | `task_manager:task_manager@tcp(mysql:3306)/task_manager?parseTime=true&loc=UTC&multiStatements=true` |
| `MYSQL_DATABASE` | Имя базы данных | `task_manager` |
| `MYSQL_USER` | Пользователь MySQL | `task_manager` |
| `MYSQL_PASSWORD` | Пароль пользователя MySQL | `task_manager` |
| `MYSQL_ROOT_PASSWORD` | Пароль root MySQL | `root` |
| `MYSQL_PORT` | Порт MySQL на хосте | `3306` |
| `REDIS_ADDR` | Адрес Redis внутри контейнера | `redis:6379` |
| `REDIS_PORT` | Порт Redis на хосте | `6379` |

При изменении параметров MySQL в `.env` нужно согласованно изменить `MYSQL_DSN`

## Логи

API пишет JSON-логи в stdout

Для HTTP-запроса логируются метод, путь, статус, длительность, размер ответа и `X-Request-ID` при наличии

## Миграции

Миграции лежат в `internal/migration/*.up.sql`

При старте API применяет неприменённые миграции и фиксирует их в таблице `schema_migrations`

## Примеры запросов

Регистрация пользователя

```powershell
$user = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/register `
  -ContentType 'application/json' `
  -Body '{"name":"Иван","email":"ivan@example.com","password":"password123"}'
```

Вход и создание команды

```powershell
$login = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/login `
  -ContentType 'application/json' `
  -Body '{"email":"ivan@example.com","password":"password123"}'

$headers = @{ Authorization = "Bearer $($login.token)" }

Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/teams `
  -Headers $headers `
  -ContentType 'application/json' `
  -Body '{"name":"Команда разработки"}'
```

Полная спецификация API находится в [`docs/openapi.yaml`](docs/openapi.yaml)
