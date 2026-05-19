# Структура созданного проекта

```
go-voting-app/
├── README.md                              # Основная документация
├── QUICKSTART.md                          # Быстрый старт
├── ARCHITECTURE.md                        # Описание архитектуры
├── API_EXAMPLES.md                        # Примеры использования API
├── Makefile                               # Команды разработки
├── .gitignore                             # Git ignore
├── .env.example                           # Пример конфигурации
│
├── go.mod                                 # Go модуль
├── go.sum                                 # Зависимости (будет создан при первой сборке)
│
├── docker-compose.yml                     # Docker Compose конфигурация
├── Dockerfile.gateway                     # Docker для Gateway Service
├── Dockerfile.poll-manager                # Docker для Poll Manager Service
├── Dockerfile.anti-fraud                  # Docker для Anti-Fraud Service
│
├── cmd/                                   # Точки входа приложений
│   ├── gateway/
│   │   └── main.go                        # Gateway Service
│   ├── poll-manager/
│   │   └── main.go                        # Poll Manager Service
│   └── anti-fraud/
│       └── main.go                        # Anti-Fraud Service
│
├── antifraud/                             # Anti-Fraud компоненты
│   ├── geoip.go                           # GeoIP Checker
│   ├── limiter.go                         # Rate Limiter & Deduplicator
│   └── processor.go                       # Vote Processor
│
├── cache/                                 # Redis клиент
│   └── redis.go                           # Redis операции
│
├── config/                                # Конфигурация
│   └── config.go                          # Config loader
│
├── db/                                    # Database
│   └── db.go                              # PostgreSQL подключение
│
├── handler/                               # HTTP обработчики
│   ├── gateway.go                         # Gateway обработчики
│   └── utils.go                           # Утилиты
│
├── models/                                # Структуры данных
│   └── models.go                          # Models
│
├── repository/                            # Data Access Layer
│   └── poll_repository.go                 # Poll Repository
│
└── service/                               # Business Logic
    └── poll_service.go                    # Poll Service
```

## Быстрые команды для запуска

### С Docker Compose (рекомендуется)
```bash
cd go-voting-app
docker-compose up --build
```

### Локальная разработка
```bash
# Терминал 1
go run ./cmd/gateway/main.go

# Терминал 2
go run ./cmd/poll-manager/main.go

# Терминал 3
go run ./cmd/anti-fraud/main.go
```

## Что было реализовано

### ✅ Три микросервиса
1. **Gateway Service** - REST API точка входа
2. **Poll Manager Service** - Управление опросами
3. **Anti-Fraud Service** - Защита от накруток

### ✅ Архитектура
- Асинхронная обработка через NATS
- Кэширование в Redis
- Персистентное хранилище в PostgreSQL
- Цепочка проверок для безопасности

### ✅ Компоненты Anti-Fraud
- GeoIP Checker - проверка типа IP
- Rate Limiter - ограничение частоты запросов
- Deduplicator - предотвращение дублирующихся голосов
- Vote Processor - главная логика обработки

### ✅ API Endpoints
- POST /polls - создание опроса
- GET /polls/{pollID} - информация об опросе
- POST /polls/{pollID}/vote - голосование
- GET /polls/{pollID}/results - результаты
- POST /polls/{pollID}/close - закрытие опроса
- GET /health - проверка здоровья

### ✅ Защита от накруток
- Блокировка VPN и дата-центров
- Дедупликация по IP
- Rate limiting (5/мин, 50/час, 200/день)
- Асинхронная обработка

### ✅ Документация
- README.md - основная документация
- QUICKSTART.md - быстрый старт
- ARCHITECTURE.md - архитектура
- API_EXAMPLES.md - примеры API
- Inline комментарии в коде

### ✅ DevOps
- Docker Compose для локального развертывания
- Dockerfile для каждого сервиса
- Health checks
- Environment configuration
- Makefile для удобства

## Как использовать

1. **Прочитайте QUICKSTART.md** для быстрого старта
2. **Запустите с Docker Compose**: `docker-compose up --build`
3. **Тестируйте API** используя примеры из API_EXAMPLES.md
4. **Изучайте архитектуру** в ARCHITECTURE.md

## Технологии

- Go 1.21
- PostgreSQL
- Redis
- NATS
- Chi Router
- Docker & Docker Compose

## Автор

Создано на основе диаграмм C4 Model из practice1
