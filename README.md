# Система голосований с защитой от накруток

[![Coverage Status](https://coveralls.io/repos/github/AlexeyGribchenko/pract_9_12/badge.svg?branch=main)](https://coveralls.io/github/AlexeyGribchenko/pract_9_12?branch=main)

## Описание проекта

Этот проект содержит архитектурное проектирование и полную реализацию высоконагруженной системы голосований на языке Go с встроенной защитой от автоматических накруток. Система предоставляет REST API для создания и проведения опросов, а также защиту от накруток через проверку IP-адресов, дедупликацию и rate limiting.

## Структура проекта

Проект разделен на две практики:

### 📐 [Practice 1](practice1/) - Архитектурное проектирование

Первая практика содержит архитектурные диаграммы системы на основе C4 Model:

#### Диаграммы C4 Model

1. **Context Diagram** ([context.puml](practice1/context.puml))
   - Контекстная диаграмма системы
   - Взаимодействие с внешними системами
   - Пользователи и их роли

2. **Container Diagram** ([container.puml](practice1/container.puml))
   - Основные контейнеры/микросервисы
   - Взаимодействие между сервисами
   - Используемые технологии и хранилища

3. **Component Diagram** ([component.puml](practice1/component.puml))
   - Компоненты Anti-Fraud сервиса
   - Внутренние взаимодействия
   - Паттерны обработки

#### Подробнее: [Practice 1 README](practice1/README.md)

---

### 🚀 [Practice 2](practice2/) - Реализация приложения

Вторая практика содержит полную реализацию системы голосований на Go.

#### Структура приложения

```
practice2/
└── go-voting-app/              # Основное приложение
    ├── cmd/                    # Точки входа сервисов
    │   ├── gateway/            # REST API Gateway
    │   ├── poll-manager/       # Менеджер опросов
    │   └── anti-fraud/         # Сервис защиты от накруток
    ├── antifraud/              # Компоненты Anti-Fraud
    ├── cache/                  # Redis интеграция
    ├── config/                 # Конфигурация
    ├── db/                     # PostgreSQL интеграция
    ├── handler/                # HTTP обработчики
    ├── models/                 # Структуры данных
    ├── repository/             # Data Access Layer
    └── service/                # Business Logic
```

#### Основные компоненты

**Три микросервиса:**

1. **Gateway Service** (порт 8001)
   - REST API входная точка
   - Управление опросами и голосованием
   - Публикация событий в NATS

2. **Poll Manager Service** (порт 8002)
   - CRUD операции над опросами
   - Кэширование в Redis
   - Персистентное хранилище в PostgreSQL

3. **Anti-Fraud Service** (порт 8003)
   - Асинхронная обработка голосов
   - Проверка типа IP-адреса
   - Дедупликация и Rate Limiting

#### Технологический стек

- **Язык**: Go 1.21+
- **API Router**: Chi
- **Message Broker**: NATS
- **Cache**: Redis
- **Database**: PostgreSQL
- **Containerization**: Docker & Docker Compose

#### Быстрый старт

```bash
cd practice2/go-voting-app

# С Docker Compose (рекомендуется)
docker-compose up --build

# Или локальная разработка
go run ./cmd/gateway/main.go       # Терминал 1
go run ./cmd/poll-manager/main.go  # Терминал 2
go run ./cmd/anti-fraud/main.go    # Терминал 3
```

**Сервисы доступны по адресам:**
- Gateway: http://localhost:8001
- Poll Manager: http://localhost:8002
- Anti-Fraud: http://localhost:8003
- PostgreSQL: localhost:5432
- Redis: localhost:6379
- NATS: localhost:4222

#### API Endpoints

- `POST /polls` - создание опроса
- `GET /polls/{pollID}` - информация об опросе
- `POST /polls/{pollID}/vote` - голосование
- `GET /polls/{pollID}/results` - результаты опроса

#### Подробнее

Полная документация находится в:
- [README](practice2/go-voting-app/README.md)
- [Архитектура](practice2/go-voting-app/ARCHITECTURE.md)
- [Примеры API](practice2/go-voting-app/API_EXAMPLES.md)
- [Быстрый старт](practice2/go-voting-app/QUICKSTART.md)

---

## Ключевые особенности

✅ **Микросервисная архитектура** - независимые масштабируемые сервисы

✅ **Защита от накруток:**
- GeoIP проверка (residential/datacenter/vpn)
- Rate Limiting - ограничение частоты голосов
- Дедупликация - исключение дублирующихся голосов

✅ **Асинхронная обработка** - NATS для очередей событий

✅ **Кэширование** - Redis для высокой производительности

✅ **Персистентность** - PostgreSQL для надежного хранения данных

✅ **Containerization** - Docker для простого развертывания

---

## Использованные подходы

### Practice 1 - Проектирование с ИИ

При разработке архитектуры использовалась языковая модель DeepSeek для:
- Определения структуры и логики работы сервиса
- Рассмотрения множества вариантов реализации
- Генерации диаграмм PlantUML

Сгенерированный код требовал ручной доработки для оптимизации структуры и связей между компонентами.

### Practice 2 - Реализация

Полная реализация системы включает:
- Трехуровневую архитектуру (handlers → services → repositories)
- Тестирование компонентов (unit тесты для каждого слоя)
- Интеграцию внешних сервисов (Redis, PostgreSQL, NATS)
- Docker-контейнеризацию всех сервисов

---

## Документация

- [Practice 1 - Архитектурные диаграммы](practice1/README.md)
- [Practice 2 - Реализация приложения](practice2/go-voting-app/README.md)
- [Архитектура системы](practice2/go-voting-app/ARCHITECTURE.md)
- [API Примеры](practice2/go-voting-app/API_EXAMPLES.md)

---

## Требования

- Docker и Docker Compose
- Go 1.21+ (для локальной разработки)

## Контакты

Проект создан в рамках курса "ИПОРиПИС" (Информационные процессы, организация, реализация и проектирование информационных систем)
