# Proxy Service (API Gateway) для Кинобездны

Маршрутизирует трафик `/api/movies` между монолитом и `movies-service`
с использованием фиче-флага и процента миграции (`Strangler Fig` pattern).

## Переменные окружения
- `PORT` — порт запуска (по умолчанию 8000)
- `MONOLITH_URL` — адрес монолита (пример: `http://monolith:8080`)
- `MOVIES_SERVICE_URL` — адрес сервиса movies (пример: `http://movies-service:8081`)
- `GRADUAL_MIGRATION` — `true`/`false`, включает процентную маршрутизацию
- `MOVIES_MIGRATION_PERCENT` — число 0-100, доля трафика в новый сервис

## Запуск
```bash
docker compose up -d --build proxy-service

