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
