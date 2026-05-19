# Примеры использования API

## 1. Создание опроса

**Запрос:**
```bash
POST http://localhost:8001/polls
Content-Type: application/json

{
  "title": "Лучший язык программирования?",
  "options": ["Go", "Python", "JavaScript", "Rust"]
}
```

**Ответ (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "admin_key": "660e8400-e29b-41d4-a716-446655440001",
  "title": "Лучший язык программирования?"
}
```

---

## 2. Получение информации об опросе

**Запрос:**
```bash
GET http://localhost:8001/polls/550e8400-e29b-41d4-a716-446655440000
```

**Ответ (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Лучший язык программирования?",
  "admin_key": "660e8400-e29b-41d4-a716-446655440001",
  "status": "active",
  "created_at": "2024-01-15T10:30:00Z",
  "closed_at": null
}
```

---

## 3. Голосование

**Запрос:**
```bash
POST http://localhost:8001/polls/550e8400-e29b-41d4-a716-446655440000/vote
Content-Type: application/json
X-Forwarded-For: 192.168.1.100

{
  "option_id": "770e8400-e29b-41d4-a716-446655440002"
}
```

**Ответ (200 OK):**
```json
{
  "success": true,
  "message": "Vote submitted for processing"
}
```

**Ошибки:**
- 400 Bad Request - Опрос не активен или option_id не предоставлен
- 404 Not Found - Опрос не найден
- 429 Too Many Requests - Превышен лимит запросов
- 403 Forbidden - IP заблокирован (VPN/Datacenter)

---

## 4. Получение результатов

**Запрос:**
```bash
GET http://localhost:8001/polls/550e8400-e29b-41d4-a716-446655440000/results
```

**Ответ (200 OK):**
```json
{
  "poll_id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Лучший язык программирования?",
  "status": "active",
  "results": {
    "770e8400-e29b-41d4-a716-446655440002": {
      "option": "770e8400-e29b-41d4-a716-446655440002",
      "text": "Go",
      "count": 42
    },
    "880e8400-e29b-41d4-a716-446655440003": {
      "option": "880e8400-e29b-41d4-a716-446655440003",
      "text": "Python",
      "count": 38
    },
    "990e8400-e29b-41d4-a716-446655440004": {
      "option": "990e8400-e29b-41d4-a716-446655440004",
      "text": "JavaScript",
      "count": 25
    },
    "aa0e8400-e29b-41d4-a716-446655440005": {
      "option": "aa0e8400-e29b-41d4-a716-446655440005",
      "text": "Rust",
      "count": 15
    }
  }
}
```

---

## 5. Закрытие опроса

**Запрос:**
```bash
POST http://localhost:8001/polls/550e8400-e29b-41d4-a716-446655440000/close
Content-Type: application/json

{
  "admin_key": "660e8400-e29b-41d4-a716-446655440001"
}
```

**Ответ (200 OK):**
```json
{
  "success": true,
  "message": "Poll closed"
}
```

**Ошибки:**
- 401 Unauthorized - Неверный admin_key

---

## 6. Проверка статуса голоса

**Запрос:**
```bash
GET http://localhost:8001/polls/550e8400-e29b-41d4-a716-446655440000/vote-status
X-Forwarded-For: 192.168.1.100
```

**Ответ (200 OK):**
```json
{
  "poll_id": "550e8400-e29b-41d4-a716-446655440000",
  "ip": "192.168.1.100",
  "has_voted": true
}
```

---

## 7. Health Check

**Запрос:**
```bash
GET http://localhost:8001/health
```

**Ответ (200 OK):**
```json
{
  "status": "ok",
  "service": "gateway"
}
```

---

## Примеры с использованием Python

```python
import requests
import json

BASE_URL = "http://localhost:8001"

# 1. Создание опроса
response = requests.post(
    f"{BASE_URL}/polls",
    json={
        "title": "Лучший язык программирования?",
        "options": ["Go", "Python", "JavaScript", "Rust"]
    }
)
poll_data = response.json()
poll_id = poll_data["id"]
admin_key = poll_data["admin_key"]
print(f"Poll created: {poll_id}")

# 2. Голосование
response = requests.post(
    f"{BASE_URL}/polls/{poll_id}/vote",
    json={
        "option_id": "option-id-here"
    },
    headers={"X-Forwarded-For": "192.168.1.100"}
)
print(f"Vote response: {response.json()}")

# 3. Получение результатов
response = requests.get(f"{BASE_URL}/polls/{poll_id}/results")
results = response.json()
print(f"Results: {json.dumps(results, indent=2)}")

# 4. Закрытие опроса
response = requests.post(
    f"{BASE_URL}/polls/{poll_id}/close",
    json={"admin_key": admin_key}
)
print(f"Close response: {response.json()}")
```

---

## Примеры с использованием cURL

### Создание опроса
```bash
curl -X POST http://localhost:8001/polls \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Лучший язык программирования?",
    "options": ["Go", "Python", "JavaScript", "Rust"]
  }' | jq
```

### Сохранение ID для дальнейшего использования
```bash
POLL_ID="550e8400-e29b-41d4-a716-446655440000"
ADMIN_KEY="660e8400-e29b-41d4-a716-446655440001"
OPTION_ID="770e8400-e29b-41d4-a716-446655440002"
```

### Голосование
```bash
curl -X POST http://localhost:8001/polls/$POLL_ID/vote \
  -H "Content-Type: application/json" \
  -H "X-Forwarded-For: 192.168.1.100" \
  -d "{\"option_id\": \"$OPTION_ID\"}" | jq
```

### Получение результатов
```bash
curl -X GET http://localhost:8001/polls/$POLL_ID/results | jq
```

### Закрытие опроса
```bash
curl -X POST http://localhost:8001/polls/$POLL_ID/close \
  -H "Content-Type: application/json" \
  -d "{\"admin_key\": \"$ADMIN_KEY\"}" | jq
```

---

## Load Testing с использованием Apache Bench

```bash
# Создание опроса
ab -n 100 -c 10 -p create_poll.json -T application/json http://localhost:8001/polls

# Голосование (требует готового опроса)
ab -n 1000 -c 50 -p vote.json -T application/json http://localhost:8001/polls/$POLL_ID/vote
```
