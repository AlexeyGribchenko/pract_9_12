# Быстрый старт

## Требования

- Go 1.21+
- Docker и Docker Compose
- Git

## Запуск с Docker Compose (Рекомендуется)

### 1. Клонируйте репозиторий

```bash
cd go-voting-app
```

### 2. Запустите приложение

```bash
docker-compose up --build
```

Это автоматически создаст и запустит все контейнеры:
- PostgreSQL (порт 5432)
- Redis (порт 6379)
- NATS (порт 4222)
- Gateway Service (порт 8001)
- Poll Manager Service (порт 8002)
- Anti-Fraud Service (порт 8003)

### 3. Проверьте здоровье сервисов

```bash
# Gateway Health
curl http://localhost:8001/health

# Poll Manager Health
curl http://localhost:8002/health

# Anti-Fraud Service
curl http://localhost:8003/health
```

## Тестирование API

### 1. Создайте опрос

```bash
curl -X POST http://localhost:8001/polls \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Лучший язык программирования?",
    "options": ["Go", "Python", "JavaScript", "Rust"]
  }' | jq
```

Сохраните `id` и `admin_key` из ответа.

### 2. Проголосуйте

Откройте несколько терминалов и отправьте голоса с разных IP:

```bash
# Замените на реальные ID из предыдущего шага
POLL_ID="your-poll-id"
OPTION_ID="your-option-id"

curl -X POST http://localhost:8001/polls/$POLL_ID/vote \
  -H "Content-Type: application/json" \
  -H "X-Forwarded-For: 192.168.1.100" \
  -d '{"option_id": "'$OPTION_ID'"}' | jq
```

Попробуйте голосовать с тем же IP дважды - должна быть ошибка.

### 3. Получите результаты

```bash
curl http://localhost:8001/polls/$POLL_ID/results | jq
```

### 4. Закройте опрос

```bash
ADMIN_KEY="your-admin-key"

curl -X POST http://localhost:8001/polls/$POLL_ID/close \
  -H "Content-Type: application/json" \
  -d '{"admin_key": "'$ADMIN_KEY'"}' | jq
```

После закрытия голосование невозможно.

## Локальная разработка

### 1. Установите зависимости

```bash
go mod download
go mod tidy
```

### 2. Запустите сервисы отдельно

Откройте три терминала и запустите каждый сервис:

**Терминал 1 - Gateway Service:**
```bash
go run ./cmd/gateway/main.go
```

**Терминал 2 - Poll Manager Service:**
```bash
go run ./cmd/poll-manager/main.go
```

**Терминал 3 - Anti-Fraud Service:**
```bash
go run ./cmd/anti-fraud/main.go
```

Убедитесь, что PostgreSQL, Redis и NATS запущены (используйте Docker Compose для этого или установите локально).

## Полезные команды

```bash
# Используйте Makefile для удобства
make docker-up      # Запустить Docker Compose
make docker-down    # Остановить Docker Compose
make docker-rebuild # Пересобрать и перезапустить
make logs           # Показать логи всех сервисов
make logs-gateway   # Показать логи Gateway
make logs-pm        # Показать логи Poll Manager
make logs-af        # Показать логи Anti-Fraud
```

## Отладка

### Просмотр логов

```bash
# Все сервисы
docker-compose logs -f

# Конкретный сервис
docker-compose logs -f gateway
docker-compose logs -f poll-manager
docker-compose logs -f anti-fraud
```

### Подключение к БД

```bash
docker exec -it voting-postgres psql -U postgres -d voting

# Полезные SQL запросы
SELECT * FROM polls;
SELECT * FROM options;
SELECT * FROM votes;
SELECT COUNT(*) FROM votes WHERE poll_id='<poll_id>';
```

### Проверка Redis

```bash
docker exec -it voting-redis redis-cli

# Полезные команды
KEYS *
GET poll:status:*
SMEMBERS votes:ips:*
GET vote:count:*
```

### Проверка NATS

```bash
# Сообщения в очереди
curl http://localhost:8222/subsz
curl http://localhost:8222/varz
```

## Решение проблем

### Проблема: Контейнеры не стартуют

**Решение:**
```bash
# Очистите все контейнеры и volumes
docker-compose down -v

# Пересоберите с нуля
docker-compose up --build
```

### Проблема: Ошибка подключения к БД

**Решение:**
```bash
# Проверьте, что PostgreSQL готов
docker-compose logs postgres

# Пересоздайте контейнер
docker-compose up --force-recreate postgres
```

### Проблема: Голосование не работает

**Решение:**
```bash
# Проверьте логи Anti-Fraud Service
docker-compose logs -f anti-fraud

# Убедитесь, что опрос активен
curl http://localhost:8002/polls/<poll_id>
```

## Дополнительные ресурсы

- [API Примеры](API_EXAMPLES.md) - Подробные примеры всех API
- [Архитектура](ARCHITECTURE.md) - Описание архитектуры
- [README](README.md) - Основная документация

## Контакты и поддержка

Если у вас есть вопросы, создайте issue или обратитесь в документацию.
