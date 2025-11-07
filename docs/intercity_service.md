# Межгород: руководство по API для пассажиров и водителей

Документ описывает REST-интерфейс межгородского сервиса. В нём собраны все маршруты, схемы запросов и ответы, необходимые как пассажирам, так и водителям для работы с объявлениями. Все примеры приведены на русском языке.

## Общие сведения

- **Базовый URL**: `https://<домен>` (в Postman-коллекциях используйте переменную `{{base_url}}`).
- **Версия API**: `v1`. Все описанные маршруты начинаются с `/api/v1/intercity/...` согласно регистрации обработчиков. 【F:internal/taxi/http/server.go†L461-L475】
- **Формат данных**: JSON UTF-8. Ошибки возвращаются в виде `{"error": "описание"}`.
- **Часовые и календарные форматы**:
  - Дата отправления — строка в формате `YYYY-MM-DD` (ISO 8601). 【F:internal/taxi/http/server.go†L357-L358】【F:internal/taxi/http/server.go†L1398-L1409】
  - Время отправления — строка в формате `HH:MM` (24 часа). 【F:internal/taxi/http/server.go†L330-L331】【F:internal/taxi/http/server.go†L1420-L1425】
- **Типы поездок**: `companions` (попутчики), `parcel` (доставка посылок), `solo` (индивидуальная поездка). Любое другое значение вызовет ошибку. 【F:internal/taxi/http/server.go†L254-L303】
- **Статусы объявления**: `open` (активно) и `closed` (закрыто). 【F:internal/taxi/http/server.go†L260-L262】
- **Роль создателя**: `creator_role` фиксирует, кто разместил объявление — пассажир или водитель. 【F:internal/taxi/http/server.go†L335-L336】【F:internal/taxi/repo/intercity_orders.go†L27-L33】【F:db/migrations/000075_intercity_orders_driver_ads.up.sql†L1-L11】

## Модель данных объявления

Сервер хранит объявления в таблице `intercity_orders`. Основные поля: откуда, куда, тип поездки, цена, дата/время выезда, статус, идентификаторы пассажира и водителя. Для отображения карточки объявления дополнительно подтягиваются телефон для связи, модель машины, ФИО и рейтинг водителя, а также время обновления его профиля. 【F:internal/taxi/repo/intercity_orders.go†L11-L127】

Ответ API `intercityOrderResponse` содержит:

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | `int64` | Идентификатор объявления. |
| `passenger_id` | `int64` | ID пассажира (0, если объявление создал водитель). |
| `driver_id` | `int64?` | ID водителя, если объявление создано водителем. |
| `from` / `to` | `string` | Города/локации отправления и назначения. |
| `trip_type` | `string` | Тип поездки (`companions` / `parcel` / `solo`). |
| `comment` | `string?` | Дополнительная информация. |
| `price` | `int` | Стоимость в тенге. |
| `contact_phone` | `string` | Телефон для связи: сначала телефон пассажира, если он есть, иначе — телефон водителя. 【F:internal/taxi/repo/intercity_orders.go†L87-L118】 |
| `departure_date` | `string` | Дата выезда (`YYYY-MM-DD`). |
| `departure_time` | `string?` | Время выезда (`HH:MM`). |
| `status` | `string` | Текущий статус (`open` или `closed`). |
| `created_at`, `updated_at` | `string` (ISO timestamp) | Временные метки создания и последнего изменения. |
| `closed_at` | `string?` | Время закрытия, если объявление закрыто. |
| `creator_role` | `string` | `passenger` или `driver`. |
| `driver` | `object?` | Карточка водителя (ID, модель машины, ФИО, рейтинг, фото и дата обновления профиля). 【F:internal/taxi/http/server.go†L319-L407】 |

## Обязательные проверки при создании

При размещении объявления сервер выполняет следующие проверки:

1. Передан **ровно один** идентификатор: либо `passenger_id`, либо `driver_id`. Если указаны оба или ни один, запрос отклоняется. 【F:internal/taxi/http/server.go†L290-L295】
2. Поля `from`, `to`, `trip_type`, `departure_date` обязательны. 【F:internal/taxi/http/server.go†L296-L307】
3. `departure_time`, если указан, должен быть в формате `HH:MM`. 【F:internal/taxi/http/server.go†L308-L312】
4. `price` не может быть отрицательной. 【F:internal/taxi/http/server.go†L313-L315】
5. Дата выезда парсится как `YYYY-MM-DD`; при ошибке сервер вернёт 400. 【F:internal/taxi/http/server.go†L1398-L1407】

## Маршруты для пассажиров

### Создать объявление

- **Метод**: `POST`
- **URL**: `/api/v1/intercity/orders`
- **Тело**:

```json
{
  "passenger_id": 123,
  "from": "Алматы",
  "to": "Нур-Султан",
  "trip_type": "companions",
  "comment": "Берём двух попутчиков без багажа",
  "price": 18000,
  "departure_date": "2024-06-15",
  "departure_time": "08:30"
}
```

- **Ответ 201** (пример):

```json
{
  "id": 452,
  "passenger_id": 123,
  "from": "Алматы",
  "to": "Нур-Султан",
  "trip_type": "companions",
  "comment": "Берём двух попутчиков без багажа",
  "price": 18000,
  "contact_phone": "+77071234567",
  "departure_date": "2024-06-15",
  "departure_time": "08:30",
  "status": "open",
  "created_at": "2024-05-20T10:12:00Z",
  "updated_at": "2024-05-20T10:12:00Z",
  "creator_role": "passenger"
}
```

После успешного создания сервер сам вычислит `contact_phone` и заполнит его телефоном пассажира. 【F:internal/taxi/repo/intercity_orders.go†L87-L118】

### Получить объявление

- **Метод**: `GET`
- **URL**: `/api/v1/intercity/orders/{id}`
- **Ответ 200**: объект `intercityOrderResponse`. Если объявление принадлежит водителю, вложенный объект `driver` будет содержать модель машины, ФИО, рейтинг и фото. 【F:internal/taxi/http/server.go†L348-L407】【F:internal/taxi/http/server.go†L1447-L1461】

### Список объявлений (поиск)

- **Метод**: `POST`
- **URL**: `/api/v1/intercity/orders/list`
- **Тело (все поля опциональны)**:

```json
{
  "from": "Алматы",
  "to": "Астана",
  "date": "2024-06-15",
  "time": "08:30",
  "status": "open",
  "passenger_id": 123,
  "limit": 20,
  "offset": 0
}
```

- **Фильтры**:
  - `from` / `to` — подстрочный поиск по локациям (регистр не учитывается). 【F:internal/taxi/repo/intercity_orders.go†L156-L163】
  - `date` — точное совпадение даты выезда. 【F:internal/taxi/repo/intercity_orders.go†L164-L167】
  - `time` — точное совпадение времени (формат `HH:MM`). 【F:internal/taxi/repo/intercity_orders.go†L168-L170】
  - `status` — `open`, `closed` или `all` (возвращает оба статуса). 【F:internal/taxi/http/server.go†L453-L455】【F:internal/taxi/http/server.go†L1361-L1364】
  - `passenger_id` / `driver_id` — показывают объявления конкретного автора. 【F:internal/taxi/http/server.go†L1346-L1351】
  - `limit` и `offset` контролируют пагинацию. Значения <0 отклоняются. 【F:internal/taxi/http/server.go†L430-L437】【F:internal/taxi/http/server.go†L1330-L1337】

- **Ответ 200**:

```json
{
  "orders": [
    {
      "id": 452,
      "passenger_id": 123,
      "from": "Алматы",
      "to": "Нур-Султан",
      "trip_type": "companions",
      "price": 18000,
      "contact_phone": "+77071234567",
      "departure_date": "2024-06-15",
      "departure_time": "08:30",
      "status": "open",
      "creator_role": "passenger"
    }
  ],
  "limit": 20,
  "offset": 0
}
```

Если в выдаче присутствуют объявления водителей, для каждого добавляется вложенный блок `driver` с их профилем. 【F:internal/taxi/http/server.go†L380-L407】

### Закрыть объявление

- **Метод**: `POST`
- **URL**: `/api/v1/intercity/orders/{id}/close`
- **Тело**:

```json
{
  "passenger_id": 123
}
```

Сервер проверит, что объявление принадлежит указанному пассажиру и находится в статусе `open`. В случае успеха статус станет `closed`, а в ответ вернётся обновлённый объект. 【F:internal/taxi/http/server.go†L1463-L1491】【F:internal/taxi/repo/intercity_orders.go†L242-L258】

## Маршруты для водителей

Маршруты те же, что и для пассажиров, но с другими полями тела запроса и ожидаемым содержимым `creator_role`.

### Создать объявление водителя

- **Метод**: `POST`
- **URL**: `/api/v1/intercity/orders`
- **Тело**:

```json
{
  "driver_id": 98,
  "from": "Алматы",
  "to": "Талдыкорган",
  "trip_type": "solo",
  "comment": "Toyota Camry 2021, 3 свободных места",
  "price": 24000,
  "departure_date": "2024-06-16",
  "departure_time": "07:00"
}
```

- **Ответ 201**: объект `intercityOrderResponse` с `creator_role = "driver"` и вложенным профилем водителя (`car_model`, `full_name`, `rating`, `photo`, `profile_updated_at`). 【F:internal/taxi/http/server.go†L348-L407】【F:internal/taxi/http/server.go†L1411-L1418】

Телефон для связи будет взят из карточки водителя, если у объявления нет пассажира. 【F:internal/taxi/repo/intercity_orders.go†L87-L118】

### Список объявлений водителя

Используется тот же маршрут `/api/v1/intercity/orders/list`, но с фильтрацией по `driver_id`:

```json
{
  "driver_id": 98,
  "status": "open"
}
```

Это вернёт все активные объявления, созданные конкретным водителем. Фильтрация и пагинация работают аналогично пассажирскому сценарию. 【F:internal/taxi/http/server.go†L1346-L1358】【F:internal/taxi/repo/intercity_orders.go†L176-L183】

### Просмотр объявления водителя клиентами

Клиентам доступен `GET /api/v1/intercity/orders/{id}`. Если объявление принадлежит водителю, блок `driver` будет содержать:

```json
"driver": {
  "id": 98,
  "car_model": "Toyota Camry",
  "full_name": "Иванов Иван Иванович",
  "rating": 4.9,
  "photo": "https://cdn.example.com/driver-photos/98.jpg",
  "profile_updated_at": "2024-05-10T09:30:00Z"
}
```

Эти данные берутся из таблицы `drivers` и связанного пользователя. 【F:internal/taxi/repo/intercity_orders.go†L29-L127】【F:internal/taxi/http/server.go†L367-L405】

## Обработка ошибок

- Некорректный JSON, отрицательные значения, неверные форматы даты/времени — ответ `400 Bad Request` с текстом ошибки. 【F:internal/taxi/http/server.go†L1317-L1326】【F:internal/taxi/http/server.go†L1398-L1424】
- Попытка создать объявление без идентификаторов или с неверным `trip_type` также приводит к `400`. 【F:internal/taxi/http/server.go†L290-L315】
- Запрос к несуществующему объявлению — `404 Not Found`. 【F:internal/taxi/http/server.go†L1451-L1460】
- Попытка закрыть чужое либо уже закрытое объявление — `404` (обновлена 0 строк). 【F:internal/taxi/http/server.go†L1477-L1483】【F:internal/taxi/repo/intercity_orders.go†L242-L258】
- Прочие сбои (ошибки базы данных) возвращают `500 Internal Server Error`. 【F:internal/taxi/http/server.go†L1373-L1387】【F:internal/taxi/http/server.go†L1434-L1444】

## Типичные сценарии

### Пассажир размещает запрос и ищет попутчиков

1. Отправляет `POST /api/v1/intercity/orders` с `passenger_id` и параметрами поездки.
2. При необходимости закрывает объявление после подбора спутников через `POST /api/v1/intercity/orders/{id}/close`.
3. Отслеживает активные и закрытые объявления через `/api/v1/intercity/orders/list` с фильтром по `passenger_id` и статусу.

### Водитель публикует собственный рейс

1. Отправляет `POST /api/v1/intercity/orders` с `driver_id`.
2. Использует `/api/v1/intercity/orders/list` с `driver_id` для контроля активных объявлений.
3. Пассажиры получают профиль водителя и контактный телефон при просмотре объявления через `GET /api/v1/intercity/orders/{id}`.

### Клиент ищет подходящее объявление водителя

1. Выполняет `/api/v1/intercity/orders/list` с фильтрами `from`, `to`, `date`, `status`.
2. Открывает конкретную карточку через `GET /api/v1/intercity/orders/{id}` и видит профиль водителя.
3. Созванивается по номеру из `contact_phone` и договаривается о поездке.

Документ охватывает полный цикл работы межгородского сервиса как со стороны пассажиров, так и со стороны водителей. Используйте описанные примеры при интеграции и тестировании.
