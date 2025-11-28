# Бизнес-аккаунты, места и исполнители

Документ описывает серверную логику для роли **business** и связанных с ней
исполнителей **business_worker**: покупку мест (seats), создание и управление
сотрудниками, автосоздание чата с главным и правила входа по телефону или
логину. Примеры запросов и ответов приведены для текущих HTTP-роутов.

## Роли и ограничения

- `business` — владелец аккаунта, покупает места, создаёт/управляет
  исполнителями. Доступ к бизнес-ручкам защищён `JWTMiddlewareWithRole("business")`.
- `business_worker` — сотрудник. `JWTMiddleware` пропускает его по роли
  `business_worker` в ветке `requiredRole == "worker"` или напрямую через
  `requiredRole == "business_worker"`. Такой пользователь авторизуется по
  телефону или уникальному бизнес-логину.

## Данные и таблицы

Источники истины (используются напрямую в репозиториях):

- `business_accounts` — агрегированные поля `seats_total`, `seats_used`,
  `status` (`active`/`suspended`).
- `business_seat_purchases` — история покупок мест.
- `business_workers` — исполнители, связаны с `users` и бизнесом;
  содержит `login`, `chat_id`, `status`.

## Покупка мест (seats)

Ручка: `POST /business/purchase` (требуется роль `business`).

Логика (`BusinessService.PurchaseSeats`):
1. Валидирует, что `seats > 0`.
2. Получает или создаёт `business_accounts` для пользователя.
3. Сохраняет покупку в `business_seat_purchases` с суммой `seats*1000`, если
   `amount` не передан явно.
4. Увеличивает `seats_total` и активирует аккаунт.
5. Присваивает пользователю роль `business` (в таблице `users`).

Пример запроса:
```http
POST /business/purchase
Authorization: Bearer <token>
Content-Type: application/json

{
  "seats": 3,
  "provider": "test",
  "provider_txn_id": "abc-123"
}
```

Пример ответа 200 OK:
```json
{
  "account": {
    "id": 10,
    "business_user_id": 42,
    "seats_total": 5,
    "seats_used": 1,
    "status": "active",
    "created_at": "2024-05-10T09:00:00Z",
    "updated_at": null
  }
}
```

## Просмотр аккаунта

Ручка: `GET /business/account` (роль `business`).

Возвращает объект `account` и вычисленное поле `seats_left = seats_total - seats_used`.

Пример ответа 200 OK:
```json
{
  "account": {
    "id": 10,
    "business_user_id": 42,
    "seats_total": 5,
    "seats_used": 2,
    "status": "active",
    "created_at": "2024-05-10T09:00:00Z",
    "updated_at": null
  },
  "seats_left": 3
}
```

## Создание исполнителя

Ручка: `POST /business/workers` (роль `business`).

Валидации (`CreateWorker`):
- Аккаунт не должен быть в статусе `suspended`.
- `seats_used < seats_total`, иначе ошибка `no free seats available`.
- `login` уникален по таблице `business_workers`.

Шаги создания:
1. Хеширует пароль исполнителя (`bcrypt`).
2. Создаёт пользователя с ролью `business_worker` в `users`.
3. Создаёт чат между бизнесом и исполнителем (`ChatRepository.CreateChat`).
4. Записывает исполнителя в `business_workers` со статусом `active` и `chat_id`.
5. Инкрементирует `seats_used` у аккаунта бизнеса.

Тело запроса:
```json
{
  "login": "driver001",
  "name": "Иван",
  "surname": "Петров",
  "phone": "+79990001122",
  "password": "secret123"
}
```

Ответ 201 Created:
```json
{
  "worker": {
    "id": 7,
    "business_user_id": 42,
    "worker_user_id": 115,
    "login": "driver001",
    "chat_id": 501,
    "status": "active",
    "created_at": "2024-05-10T09:05:00Z",
    "updated_at": null,
    "user": {
      "id": 115,
      "name": "Иван",
      "surname": "Петров",
      "phone": "+79990001122"
    }
  }
}
```

## Список исполнителей бизнеса

Ручка: `GET /business/workers` (роль `business`). Возвращает массив `workers`
с данными пользователя и `chat_id` для каждого исполнителя.

Ответ 200 OK (укороченный пример):
```json
{
  "workers": [
    {
      "id": 7,
      "business_user_id": 42,
      "worker_user_id": 115,
      "login": "driver001",
      "chat_id": 501,
      "status": "active",
      "user": {
        "id": 115,
        "name": "Иван",
        "surname": "Петров",
        "phone": "+79990001122"
      }
    }
  ]
}
```

## Обновление исполнителя

Ручка: `PUT /business/workers/:id` (роль `business`).

Правила:
- Аккаунт не должен быть `suspended`.
- `login` остаётся уникальным (валидация на другой ID).
- Допустимы изменения `login`, `name`, `surname`, `phone`, `password`, `status`.
- Если `status` пустой в запросе, сохраняется `active`.

Пример запроса:
```http
PUT /business/workers/7
Authorization: Bearer <token>
Content-Type: application/json

{
  "login": "driver-main",
  "phone": "+79990002233",
  "status": "disabled"
}
```

Пример ответа 200 OK (обрезан):
```json
{
  "worker": {
    "id": 7,
    "login": "driver-main",
    "status": "disabled",
    "chat_id": 501,
    "user": {
      "phone": "+79990002233"
    }
  }
}
```

## Отключение исполнителя

Ручка: `DELETE /business/workers/:id` (роль `business`).
Устанавливает `status = 'disabled'` в `business_workers`, чат и история сообщений
не удаляются. Ответ: HTTP 204 No Content.

## Авторизация business_worker

Ручка: `POST /user/sign_in` (общая для всех ролей).

- Пытается найти пользователя по `phone`, затем по бизнес-логину (`login` в теле
  запроса соответствует полю `name` сервисного метода).
- Пароль сравнивается с хешем в `users.password`.
- При успешном входе создаётся сессия и выдаются токены.

Пример запроса для исполнителя:
```json
{
  "login": "driver001",
  "password": "secret123"
}
```

Пример ответа 200 OK (структура `Tokens`):
```json
{
  "access_token": "<jwt>",
  "refresh_token": "<uuid>",
  "expires_in": 72000
}
```

## Сводка роутов

| Метод | Путь | Роль | Назначение |
|-------|------|------|------------|
| POST | `/business/purchase` | business | Покупка мест, активация бизнес-аккаунта |
| GET | `/business/account` | business | Просмотр лимитов `seats_total`, `seats_used`, `seats_left` |
| POST | `/business/workers` | business | Создать исполнителя, чат и занять seat |
| GET | `/business/workers` | business | Список исполнителей с `chat_id` |
| PUT | `/business/workers/:id` | business | Обновить данные/статус исполнителя |
| DELETE | `/business/workers/:id` | business | Логически отключить исполнителя |
| POST | `/user/sign_in` | public | Вход по телефону или бизнес-логину для `business_worker` |

## Привязка объявлений к исполнителям

| Метод | Путь | Роль | Назначение |
|-------|------|------|------------|
| POST | `/business/workers/:id/attach` | business | Привязать объявление к исполнителю |
| DELETE | `/business/workers/:id/detach` | business | Отвязать объявление от исполнителя |
| GET | `/business/workers/listings` | business | Получить карту `worker_user_id -> [listing]` |

### Логика (`BusinessService.AttachListing` / `DetachListing` / `ListWorkerListings`)

1. Проверяется, что `listing_type` (service/work/rent/ad/work_ad/rent_ad) и `listing_id` заданы.
2. Хендлер валидирует, что `:id` принадлежит текущему бизнесу (`GetWorkerByID`). При несоответствии → `404`.
3. На привязке вызывается `UpsertWorkerListing`, на отвязке — `DeleteWorkerListing`.
4. Список привязок возвращается картой, где ключ — `worker_user_id` исполнителя.

### Тела запросов

```json
{
  "listing_type": "service",
  "listing_id": 123
}
```

### Ответы

- `POST /attach` и `DELETE /detach` → `204 No Content`.
- `GET /business/workers/listings` →

```json
{
  "listings": {
    "115": [
      {
        "listing_type": "service",
        "listing_id": 123,
        "business_user_id": 42,
        "worker_user_id": 115
      }
    ]
  }
}
```

## Чаты бизнеса

| Метод | Путь | Роль | Назначение |
|-------|------|------|------------|
| GET | `/business/workers/chats` | business | Базовые чаты "бизнес ↔ исполнитель" с `chat_id` |
| GET | `/business/workers/listing_chats` | business | Все чаты по откликам/подтверждениям исполнителей |

### Важные правила

- При создании исполнителя чат создаётся автоматически (`ChatRepository.CreateChat`).
- Бизнес-исполнители не могут переписываться с клиентами (`MessageHandler` запрещает, если роль получателя `client`).
- При запросе истории сообщений бизнес-исполнитель проверяется на участие в чате и блокируется, если второй участник — клиент.

### Формат ответов

`GET /business/workers/chats`:

```json
{
  "workers": [
    {
      "id": 7,
      "business_user_id": 42,
      "worker_user_id": 115,
      "login": "driver001",
      "chat_id": 501,
      "status": "active"
    }
  ]
}
```

`GET /business/workers/listing_chats` возвращает агрегированный список чатов по объявлениям исполнителей.

## Карта и геометки бизнеса

| Метод | Путь | Роль | Назначение |
|-------|------|------|------------|
| GET | `/business/map/workers` | business | Координаты и активные объявления исполнителей бизнеса |
| GET | `/business/map/marker` | business | Аггрегированный маркер только текущего бизнеса |
| GET | `/map/business_markers` | public | Маркеры всех бизнесов с активными исполнителями |

### Логика (`LocationService`)

- `GetBusinessWorkers` передаёт `business_user_id` в фильтр и возвращает `workers: [ExecutorLocationGroup]`.
- `GetBusinessMarker` собирает средние координаты и количество онлайн-исполнителей для текущего бизнеса.
- `GetBusinessMarkers` отдаёт такие же маркеры для всех бизнесов.

Пример ответа `GET /business/map/marker`:

```json
{
  "marker": {
    "business_user_id": 42,
    "latitude": 51.12345,
    "longitude": 71.56789,
    "worker_count": 3
  }
}
```

Пример ответа `GET /business/map/workers` (усечённый):

```json
{
  "workers": [
    {
      "user_id": 115,
      "name": "Иван",
      "latitude": 51.12,
      "longitude": 71.56,
      "services": [
        {"id": 123, "name": "Ремонт"}
      ]
    }
  ]
}
```

## Пошаговые сценарии

### 1) Запуск бизнеса

1. Авторизоваться и вызвать `POST /business/purchase` с количеством мест.
2. Проверить лимиты через `GET /business/account`.
3. Создавать исполнителей (`POST /business/workers`) до исчерпания `seats_left`.

### 2) Управление исполнителем

1. Создать или найти исполнителя в списке `GET /business/workers`.
2. Обновить данные или статус через `PUT /business/workers/:id`.
3. Деактивировать, если нужно, `DELETE /business/workers/:id` (оставляет чат и историю).

### 3) Назначение объявлений

1. Выбрать исполнителя и объявление.
2. Привязать через `POST /business/workers/:id/attach`.
3. Проверить карту привязок `GET /business/workers/listings`.
4. При необходимости отвязать `DELETE /business/workers/:id/detach`.

### 4) Работа с чатами

1. Базовые чаты исполнителей доступны в `GET /business/workers/chats`.
2. Чаты по откликам/подтверждениям — `GET /business/workers/listing_chats`.
3. Отправлять сообщения через общий `POST /api/messages` (исполнители не могут писать клиентам).

### 5) Контроль на карте

1. `GET /business/map/marker` — быстрый статус онлайна.
2. `GET /business/map/workers` — кто именно онлайн и какие активные объявления привязаны.
3. Публичная карта всех бизнесов — `GET /map/business_markers`.
