# Taxi Admin API

Ниже перечислены административные HTTP-эндпоинты для управления модулем такси. Все эндпоинты принимают и возвращают данные в формате JSON. Для пагинации везде используются параметры `limit` и `offset`.

## Список водителей
- **URL:** `GET /api/v1/admin/taxi/drivers`
- **Параметры запроса:**
  - `limit` — максимальное количество записей (по умолчанию 100).
  - `offset` — смещение (по умолчанию 0).
- **Ответ:**
  ```json
  {
    "drivers": [
      {
        "id": 1,
        "user_id": 42,
        "name": "Имя",
        "surname": "Фамилия",
        "middlename": "Отчество",
        "status": "offline",
        "approval_status": "approved",
        "is_banned": false,
        "car_number": "123ABC01",
        "rating": 5.0,
        ...
      }
    ],
    "limit": 100,
    "offset": 0
  }
  ```

## Управление баном водителя
- **URL:** `POST /api/v1/admin/taxi/drivers/{driver_id}/ban`
- **Тело запроса:**
  ```json
  { "banned": true }
  ```
  Значение `true` блокирует водителя (он не сможет выходить на линию и принимать заказы), `false` снимает блокировку.
- **Ответ:** объект водителя после изменения.

## Подтверждение регистрации водителя
- **URL:** `POST /api/v1/admin/taxi/drivers/{driver_id}/approval`
- **Тело запроса:**
  ```json
  { "status": "approved" }
  ```
  Допустимые значения: `approved` или `rejected`. При `rejected` статус водителя автоматически переводится в `offline`.
- **Ответ:** объект водителя после изменения.

## История всех заказов такси
- **URL:** `GET /api/v1/admin/taxi/orders`
- **Параметры запроса:** `limit`, `offset`.
- **Ответ:**
  ```json
  {
    "orders": [
      {
        "id": 10,
        "passenger_id": 5,
        "driver_id": 3,
        "status": "completed",
        "payment_method": "cash",
        "driver": {
          "name": "Имя",
          "surname": "Фамилия",
          "approval_status": "approved",
          ...
        },
        "passenger": {
          "name": "Имя",
          "surname": "Фамилия",
          ...
        }
      }
    ],
    "limit": 100,
    "offset": 0
  }
  ```

## История межгородских заказов
- **URL:** `GET /api/v1/admin/taxi/intercity/orders`
- **Параметры запроса:** `limit`, `offset`.
- **Ответ:** массив заказов с полной информацией о водителе и пассажире (ФИО берётся из таблицы `users`).

> **Примечание:** Во всех административных эндпоинтах при ошибках пагинации возвращается код `400 Bad Request` с соответствующим сообщением.
