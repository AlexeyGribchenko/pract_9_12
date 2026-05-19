# Система голосований с защитой от накруток - Реализация на Go

## Описание

Это реализация высоконагруженной системы голосований на языке Go с встроенной защитой от автоматических накруток. Архитектура соответствует диаграммам C4 Model и состоит из трёх основных сервисов:

### Сервисы

1. **Gateway Service** (порт 8001)
   - REST API входная точка
   - Принимает запросы на создание опросов, голосование и получение результатов
   - Проверяет API-ключи администраторов
   - Публикует события голосования в NATS очередь

2. **Poll Manager Service** (порт 8002)
   - Управление опросами (CRUD операции)
   - Кэширование конфигураций опросов в Redis
   - Хранение метаданных в PostgreSQL
   - Вычисление результатов в реальном времени

3. **Anti-Fraud Service** (порт 8003)
   - Асинхронная обработка голосов через NATS
   - Проверка типа IP-адреса (residential/datacenter/vpn)
   - Дедупликация голосов по IP
   - Rate limiting (ограничение частоты запросов)
   - Защита от VPN и дата-центров

## Технологический стек

- **Language**: Go 1.21
- **API Gateway**: Chi Router
- **Message Broker**: NATS
- **Cache**: Redis
- **Database**: PostgreSQL
- **Docker**: Docker Compose для локального развертывания

## Установка и запуск

### Требования

- Docker и Docker Compose
- Go 1.21+ (для локальной разработки)

### Запуск через Docker Compose

```bash
cd go-voting-app
docker-compose up --build
```

Сервисы будут доступны по адресам:
- Gateway: http://localhost:8001
- Poll Manager: http://localhost:8002
- Anti-Fraud: http://localhost:8003
- PostgreSQL: localhost:5432
- Redis: localhost:6379
- NATS: localhost:4222

## API Endpoints

### Gateway Service (REST API)

#### 1. Создание опроса

```bash
POST /polls
Content-Type: application/json

{
  "title": "Ваш любимый язык программирования?",
  "options": ["Go", "Python", "JavaScript", "Rust"]
}

# Ответ:
{
  "id": "uuid-poll-id",
  "admin_key": "uuid-admin-key",
  "title": "Ваш любимый язык программирования?"
}
```

#### 2. Получение информации об опросе

```bash
GET /polls/{pollID}

# Ответ:
{
  "id": "uuid-poll-id",
  "title": "Ваш любимый язык программирования?",
  "status": "active",
  "created_at": "2024-01-15T10:30:00Z",
  "closed_at": null
}
```

#### 3. Голосование

```bash
POST /polls/{pollID}/vote
Content-Type: application/json

{
  "option_id": "uuid-option-id"
}

# Ответ:
{
  "success": true,
  "message": "Vote submitted for processing"
}
```

#### 4. Получение результатов

```bash
GET /polls/{pollID}/results

# Ответ:
{
  "poll_id": "uuid-poll-id",
  "title": "Ваш любимый язык программирования?",
  "status": "active",
  "results": {
    "option-1": {
      "option": "option-1",
      "text": "Go",
      "count": 42
    },
    "option-2": {
      "option": "option-2",
      "text": "Python",
      "count": 38
    },
    ...
  }
}
```

#### 5. Закрытие опроса

```bash
POST /polls/{pollID}/close
Content-Type: application/json

{
  "admin_key": "uuid-admin-key"
}

# Ответ:
{
  "success": true,
  "message": "Poll closed"
}
```

#### 6. Проверка здоровья сервиса

```bash
GET /health

# Ответ:
{
  "status": "ok",
  "service": "gateway"
}
```

## Архитектура

### Компоненты Anti-Fraud Service

1. **NATS Consumer** - Получает события голосования из очереди
2. **Status Checker** - Проверяет, активен ли опрос
3. **GeoIP Checker** - Проверяет тип IP (residential/datacenter/VPN)
4. **Deduplicator** - Проверяет, не голосовал ли IP уже
5. **Rate Limiter** - Ограничивает частоту голосов
6. **Vote Recorder** - Записывает голос в БД и обновляет счетчики

### Поток обработки голоса

```
1. Клиент отправляет POST /polls/{pollID}/vote
   ↓
2. Gateway проверяет опрос и публикует событие в NATS (votes.new)
   ↓
3. Anti-Fraud Service получает событие
   ↓
4. Цепочка проверок:
   - Статус опроса (должен быть active)
   - Тип IP (должен быть residential)
   - Дедупликация (не голосовал ли уже)
   - Rate Limiting (не превышен ли лимит)
   ↓
5. Если все проверки пройдены:
   - Записывается голос в PostgreSQL
   - Увеличивается счетчик в Redis
   - IP заносится в список проголосовавших
   ↓
6. Клиент может получить результаты через GET /polls/{pollID}/results
```

## Конфигурация

Переменные окружения:

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=voting

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# NATS
NATS_URL=nats://localhost:4222

# Ports
GATEWAY_PORT=8001
POLL_MANAGER_PORT=8002
ANTI_FRAUD_PORT=8003

# GeoIP API
GEOIP_API_URL=https://ipqualityscore.com/api/json/ip
GEOIP_API_KEY=your-api-key
```

## Тестирование

### Пример использования с curl

```bash
# 1. Создание опроса
curl -X POST http://localhost:8001/polls \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Лучший язык программирования?",
    "options": ["Go", "Python", "JavaScript"]
  }'

# Сохраняем poll_id и admin_key из ответа

# 2. Получение информации об опросе
curl http://localhost:8001/polls/{poll_id}

# 3. Голосование
curl -X POST http://localhost:8001/polls/{poll_id}/vote \
  -H "Content-Type: application/json" \
  -d '{"option_id": "{option_id}"}'

# 4. Получение результатов
curl http://localhost:8001/polls/{poll_id}/results

# 5. Закрытие опроса
curl -X POST http://localhost:8001/polls/{poll_id}/close \
  -H "Content-Type: application/json" \
  -d '{"admin_key": "{admin_key}"}'
```

## Защита от накруток

### Механизмы защиты

1. **GeoIP Проверка**
   - Блокирует IP из дата-центров
   - Блокирует VPN сервисы
   - Пропускает только residential IP
   - Результаты кэшируются на 24 часа

2. **Дедупликация**
   - Каждый IP может голосовать только один раз за опрос
   - Информация хранится в Redis множестве (set)

3. **Rate Limiting**
   - Максимум 5 голосов в минуту с одного IP
   - Максимум 50 голосов в час с одного IP
   - Максимум 200 голосов в день с одного IP

4. **Асинхронная обработка**
   - Голоса обрабатываются асинхронно через NATS
   - Предотвращает перегрузку системы

## Масштабируемость

- **Горизонтальное масштабирование**: Можно запустить несколько экземпляров каждого сервиса
- **Redis Cluster**: Для работы с большим объемом кэша
- **PostgreSQL Replica**: Для читаемых запросов
- **NATS Clustering**: Для высокой доступности очереди сообщений

## Лицензия

MIT

## Документация архитектуры

Диаграммы C4 Model находятся в соседних файлах:
- [context.puml](../context.puml) - Контекстная диаграмма
- [container.puml](../container.puml) - Диаграмма контейнеров
- [component.puml](../component.puml) - Диаграмма компонентов Anti-Fraud Service
