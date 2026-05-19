# Структура проекта

## Основная структура директорий

```
go-voting-app/
├── cmd/
│   ├── gateway/
│   │   └── main.go              # Gateway Service точка входа
│   ├── poll-manager/
│   │   └── main.go              # Poll Manager Service точка входа
│   └── anti-fraud/
│       └── main.go              # Anti-Fraud Service точка входа
│
├── antifraud/
│   ├── geoip.go                 # GeoIP Checker компонент
│   ├── limiter.go               # Rate Limiter и Deduplicator
│   └── processor.go             # Vote Processor главный компонент
│
├── cache/
│   └── redis.go                 # Redis клиент и операции
│
├── config/
│   └── config.go                # Конфигурация приложения
│
├── db/
│   └── db.go                    # PostgreSQL подключение и миграции
│
├── handler/
│   ├── gateway.go               # HTTP обработчики для Gateway
│   └── utils.go                 # Утилиты для работы с HTTP
│
├── models/
│   └── models.go                # Структуры данных
│
├── repository/
│   └── poll_repository.go       # Слой доступа к данным
│
├── service/
│   └── poll_service.go          # Бизнес-логика опросов
│
├── docker-compose.yml           # Docker Compose конфигурация
├── Dockerfile.gateway           # Dockerfile для Gateway Service
├── Dockerfile.poll-manager      # Dockerfile для Poll Manager Service
├── Dockerfile.anti-fraud        # Dockerfile для Anti-Fraud Service
├── go.mod                       # Go модуль
├── go.sum                       # Зависимости
├── Makefile                     # Команды для разработки
├── README.md                    # Основная документация
├── API_EXAMPLES.md              # Примеры использования API
├── .env.example                 # Пример конфигурации
├── .gitignore                   # Git ignore файл
└── ARCHITECTURE.md              # Документация архитектуры (этот файл)
```

## Описание компонентов

### cmd/ - Точки входа приложений

- **gateway/main.go** - REST API сервис для клиентов
  - Создание опросов
  - Голосование
  - Получение результатов
  - Публикация событий в NATS

- **poll-manager/main.go** - Управление данными опросов
  - CRUD операции над опросами
  - Работа с БД и кэшем
  - Вычисление результатов

- **anti-fraud/main.go** - Обработка голосов с защитой от накруток
  - Подписка на события NATS
  - Проверка IP и дедупликация
  - Rate Limiting

### antifraud/ - Компоненты Anti-Fraud сервиса

- **geoip.go**
  - `GeoIPChecker` - проверка типа IP
  - Классификация IP (residential/datacenter/vpn)
  - Кэширование результатов

- **limiter.go**
  - `RateLimiter` - ограничение частоты голосов
  - `Deduplicator` - предотвращение дублирующихся голосов

- **processor.go**
  - `VoteProcessor` - главная логика обработки голосов
  - Цепочка проверок
  - Запись результатов

### cache/ - Работа с Redis

- **redis.go**
  - `RedisCache` - Redis клиент
  - Операции: Get, Set, Incr, SAdd, SIsMember и т.д.

### config/ - Конфигурация

- **config.go**
  - `Config` структура
  - Загрузка из переменных окружения
  - Значения по умолчанию

### db/ - Работа с БД

- **db.go**
  - PostgreSQL подключение
  - Инициализация схемы БД
  - Определение таблиц и индексов

### handler/ - HTTP обработчики

- **gateway.go**
  - `GatewayHandler` структура
  - Методы обработки HTTP запросов

- **utils.go**
  - `GetClientIP()` - извлечение IP клиента

### models/ - Структуры данных

- **models.go**
  - `Poll` - опрос
  - `Option` - вариант ответа
  - `Vote` - голос
  - `VoteRequest`, `VoteResponse` - API контракты
  - `VoteEventMessage` - сообщение для NATS

### repository/ - Слой доступа к данным

- **poll_repository.go**
  - `PollRepository` структура
  - CRUD операции с опросами
  - Работа с вариантами и голосами

### service/ - Бизнес-логика

- **poll_service.go**
  - `PollService` структура
  - Создание опросов с вариантами
  - Получение результатов
  - Взаимодействие с кэшем и репозиторием

## Поток данных

### Создание опроса

```
Client
  │
  ├─→ POST /polls (title, options)
  │
  ├─→ Gateway Handler
  │
  ├─→ Poll Service
  │
  ├─→ Poll Repository
  │
  ├─→ PostgreSQL (INSERT poll, options)
  │
  ├─→ Redis (cache poll status)
  │
  └─← Ответ: {id, admin_key, title}
```

### Голосование

```
Client
  │
  ├─→ POST /polls/{pollID}/vote (option_id)
  │
  ├─→ Gateway Handler
  │
  ├─→ Validation & IP extraction
  │
  ├─→ NATS Publish (votes.new)
  │
  └─← Ответ: {success, message}
  
  ↓
  
Anti-Fraud Service
  │
  ├─ NATS Subscribe (votes.new)
  │
  ├─ Vote Event Processing
  │  ├─ Status Checker (poll active?)
  │  ├─ GeoIP Checker (residential IP?)
  │  ├─ Deduplicator (not voted before?)
  │  ├─ Rate Limiter (limit exceeded?)
  │  └─ Vote Recorder (record vote)
  │
  ├─ PostgreSQL (INSERT vote)
  │
  └─ Redis (increment counters)
```

### Получение результатов

```
Client
  │
  ├─→ GET /polls/{pollID}/results
  │
  ├─→ Gateway Handler
  │
  ├─→ Poll Service
  │
  ├─→ Poll Repository (get options)
  │
  ├─→ Redis (get vote counts)
  │
  └─← Ответ: {poll_id, title, status, results{...}}
```

## Взаимодействие сервисов

### Синхронное взаимодействие

```
Gateway Service ←→ Poll Manager Service
  │
  ├─ Получение конфигурации опроса
  ├─ Проверка существования опроса
  └─ Получение результатов
```

### Асинхронное взаимодействие

```
Gateway Service
  │
  └─→ NATS (vote event) ←─ Anti-Fraud Service
```

## Технологические решения

### Почему Go?

- Высокая производительность
- Встроенная поддержка конкурентности (goroutines)
- Простой синтаксис
- Быстрая компиляция

### Почему Chi Router?

- Легкий и быстрый маршрутизатор
- Встроенная поддержка middleware
- Удобный API

### Почему NATS?

- Высокая производительность (миллионы сообщений в секунду)
- Простая архитектура
- Consumer Group для масштабирования
- Встроенная отказоустойчивость

### Почему Redis?

- Очень быстрый доступ к данным (< 1ms)
- Встроенная поддержка TTL
- Множества (sets) для дедупликации
- Счетчики (incr) для голосов

### Почему PostgreSQL?

- Надежное хранилище данных
- Встроенная поддержка ACID
- Индексы для быстрого поиска
- Масштабируемость

## Масштабирование

### Горизонтальное масштабирование

```
Load Balancer
  │
  ├─ Gateway Service 1
  ├─ Gateway Service 2
  ├─ Gateway Service 3
  │
  ├─ Anti-Fraud Service 1 (Consumer Group)
  ├─ Anti-Fraud Service 2 (Consumer Group)
  └─ Anti-Fraud Service 3 (Consumer Group)
  
  ↓
  
  Shared Resources
  │
  ├─ PostgreSQL (Master + Replicas)
  ├─ Redis (Cluster)
  └─ NATS (Cluster)
```

### Вертикальное масштабирование

- Увеличение ресурсов сервера
- Увеличение пула подключений БД
- Увеличение объема памяти Redis

## Мониторинг и логирование

### Метрики

- Количество опросов
- Количество голосов
- Время обработки голоса
- Процент отклоненных голосов

### Логирование

- Все операции логируются
- Ошибки логируются с stack trace
- Можно настроить разные уровни логирования

## Безопасность

### Защита от накруток

1. **GeoIP Проверка** - блокирует VPN и datacenter
2. **Дедупликация** - один IP = один голос за опрос
3. **Rate Limiting** - ограничение частоты запросов
4. **Асинхронная обработка** - защита от DDoS

### Другие меры

- HTTPS в продакшене
- API-ключи администраторов
- Валидация входных данных
- SQL injection защита (параметризованные запросы)
