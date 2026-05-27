# Subscription Service

REST-сервис на Go для агрегации данных об онлайн-подписках пользователей.

Проект выполнен как тестовое задание Junior Golang Developer.

## Стек

- Go
- Chi
- PostgreSQL
- pgx
- Squirrel
- Docker Compose
- Swagger/OpenAPI
- slog

## Возможности

- Создание подписки
- Получение списка подписок
- Фильтрация списка по `user_id` и `service_name`
- Получение подписки по ID
- Обновление подписки
- Удаление подписки
- Подсчет суммарной стоимости подписок за выбранный период
- Swagger-документация
- Запуск через Docker Compose

## Запуск

```bash
docker compose up --build
```

API base URL:

```text
http://localhost:28080
```

Проверка сервиса:

```text
http://localhost:28080/health
```

Swagger UI:

```text
http://localhost:18081
```

## Пример создания подписки

```bash
curl -X POST http://localhost:28080/api/v1/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "Yandex Plus",
    "price": 400,
    "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
    "start_date": "07-2025"
  }'
```

## Пример подсчета суммы

```bash
curl "http://localhost:28080/api/v1/subscriptions/summary?period_start=07-2025&period_end=12-2025"
```

Если подписка стоит `400` рублей и активна с `07-2025` по `12-2025`, ответ будет:

```json
{
  "period_end": "12-2025",
  "period_start": "07-2025",
  "total_price": 2400
}
```

## API

Основные ручки:

```text
GET    /health
GET    /api/v1/subscriptions
GET    /api/v1/subscriptions/summary
GET    /api/v1/subscriptions/{id}
POST   /api/v1/subscriptions
PUT    /api/v1/subscriptions/{id}
DELETE /api/v1/subscriptions/{id}
```

## Конфигурация

Конфигурация передается через переменные окружения:

```env
HTTP_ADDR=:8080
DATABASE_URL=postgres://admin:admin@postgres:5432/subscriptions?sslmode=disable
```

Для локального запуска вне Docker используется:

```env
HTTP_ADDR=:28080
DATABASE_URL=postgres://admin:admin@localhost:5433/subscriptions?sslmode=disable
```

## Миграции

SQL-миграция находится в папке:

```text
migrations/001_init.sql
```

При первом запуске PostgreSQL через Docker Compose миграция применяется автоматически.

## Логирование

Сервис использует структурированные JSON-логи через `slog`.

Логируются:

- запуск сервиса
- подключение к PostgreSQL
- ошибки работы с базой
- входящие HTTP-запросы через middleware
