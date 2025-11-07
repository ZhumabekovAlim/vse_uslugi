# Межгород: REST и WebSocket интерфейсы

Документ описывает работу межгородского сервиса. Он объединяет информацию об HTTP-маршрутах и формате WebSocket-уведомлений, которые теперь рассылаются одновременно пассажирским и водительским клиентам.

## Базовая информация

- **Базовый URL**: `https://<домен>` (в Postman-коллекциях используйте переменную `{{base_url}}`).
- **Версия API**: `v1`. Все эндпоинты находятся под префиксом `/api/v1/intercity/...` и регистрируются в HTTP-сервере. 【F:internal/taxi/http/server.go†L450-L468】
- **Формат данных**: JSON UTF-8. Ошибки возвращаются в виде `{ "error": "описание" }`.
- **Статусы объявлений**: `open` (активно) и `closed` (закрыто). 【F:internal/taxi/http/server.go†L260-L262】
- **Роль создателя** (`creator_role`): фиксирует, кто создал объявление — пассажир или водитель. 【F:internal/taxi/http/server.go†L335-L336】

## Структура объявления

Базовая запись хранится в таблице `intercity_orders` и преобразуется в ответ `intercityOrderResponse`. Поля ответа формируются сервером и включают профиль водителя (если он есть) и контактный телефон, вычисляемый по приоритету пассажир/водитель. 【F:internal/taxi/repo/intercity_orders.go†L11-L127】【F:internal/taxi/http/server.go†L319-L407】

Пример ответа:

```json
{
  "id": 452,
  "passenger_id": 123,
  "driver_id": 0,
  "from": "Алматы",
  "to": "Нур-Султан",
  "trip_type": "companions",
  "comment": "Берём двух попутчиков без багажа",
  "price": 18000,
  "contact_phone": "+77071234567",
  "departure_date": "2024-06-15",
  "departure_time": "08:30",
  "status": "open",
  "creator_role": "passenger",
  "created_at": "2024-05-20T10:12:00Z",
  "updated_at": "2024-05-20T10:12:00Z"
}
```

## REST API

### Создание объявления — `POST /api/v1/intercity/orders`

- Требуется указать **ровно один** идентификатор автора (`passenger_id` или `driver_id`). 【F:internal/taxi/http/server.go†L290-L295】
- Обязательные поля: `from`, `to`, `trip_type`, `departure_date`, `price`. 【F:internal/taxi/http/server.go†L296-L307】
- `trip_type` принимает только значения `companions`, `parcel` или `solo`. 【F:internal/taxi/http/server.go†L254-L303】
- `price` не может быть отрицательным. 【F:internal/taxi/http/server.go†L313-L315】
- `departure_time`, если задан, проверяется на формат `HH:MM`. 【F:internal/taxi/http/server.go†L308-L312】
- `departure_date` парсится как `YYYY-MM-DD`; некорректный формат приводит к ошибке 400. 【F:internal/taxi/http/server.go†L1469-L1472】
- После сохранения сервер возвращает полную карточку объявления и инициирует WebSocket-рассылку события `created`. 【F:internal/taxi/http/server.go†L1489-L1521】

Пример тела запроса:

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

### Получение объявления — `GET /api/v1/intercity/orders/{id}`

Возвращает объект `intercityOrderResponse`. Если объявление создал водитель, внутри присутствует блок `driver` с машиной, рейтингом и контактами. 【F:internal/taxi/http/server.go†L348-L407】

### Поиск объявлений — `POST /api/v1/intercity/orders/list`

Поддерживает фильтры по направлениям, дате, времени, статусу и автору. Лимит и смещение проверяются на отрицательные значения. 【F:internal/taxi/http/server.go†L420-L457】【F:internal/taxi/repo/intercity_orders.go†L156-L170】

### Закрытие объявления — `POST /api/v1/intercity/orders/{id}/close`

- Требуется `passenger_id` владельца объявления. 【F:internal/taxi/http/server.go†L1533-L1548】
- После изменения статуса на `closed` сервер возвращает актуальную карточку и транслирует событие `closed` в оба WebSocket-хаба. 【F:internal/taxi/http/server.go†L1549-L1577】

## WebSocket-уведомления

Изменения в межгородских объявлениях доставляются одновременно пассажирским и водительским клиентам по существующим WS-подключениям.

### Подключение

- Пассажиры: `wss://<домен>/ws/passenger?passenger_id=<id>` 【F:internal/taxi/ws/passenger.go†L35-L53】
- Водители: `wss://<домен>/ws/driver?driver_id=<id>&city=<slug>` (параметр `city` опционален). 【F:internal/taxi/ws/driver.go†L55-L86】

Оба хаба принимают по одному соединению на пользователя и следят за таймаутами: пассажирский канал использует 60-секундный heartbeat, водительский — до 1000 секунд между пингами. 【F:internal/taxi/ws/passenger.go†L55-L88】【F:internal/taxi/ws/driver.go†L88-L147】

### Формат событий

Все межгородские уведомления используют общий payload:

```json
{
  "type": "intercity_order",
  "action": "created", // или "closed"
  "order": { ... intercityOrderResponse ... }
}
```

Структура описана типом `IntercityEvent`. 【F:internal/taxi/ws/intercity.go†L3-L8】

### Триггеры

- **Создание объявления** — событие `action: "created"`. Отправляется всем активным пассажирам и водителям сразу после записи объявления в БД. 【F:internal/taxi/http/server.go†L1489-L1521】
- **Закрытие объявления** — событие `action: "closed"`. Транслируется после успешного обновления статуса. 【F:internal/taxi/http/server.go†L1549-L1577】

Полученное поле `order` всегда содержит актуальное состояние карточки, поэтому фронтенду не нужно выполнять дополнительный HTTP-запрос для синхронизации списка.

### Пример клиента

```js
const ws = new WebSocket('wss://example.com/ws/passenger?passenger_id=123');

ws.onmessage = (event) => {
  const payload = JSON.parse(event.data);
  if (payload.type === 'intercity_order') {
    if (payload.action === 'created') {
      addOrderToList(payload.order);
    }
    if (payload.action === 'closed') {
      markOrderClosed(payload.order.id);
    }
  }
};
```

### Поведение при отключении

При разрыве соединения хабы очищают сохранённые подключения и статусы, чтобы новые события не отправлялись в устаревшие сессии. 【F:internal/taxi/ws/passenger.go†L60-L88】【F:internal/taxi/ws/driver.go†L99-L121】

## Инструменты для тестирования

В репозитории есть Postman-коллекции:

- `docs/taxi_api_postman_collection.json` — общие такси маршруты
- `docs/performer_confirmation_flow.postman_collection.json` — сценарии подтверждения исполнителя

Для теста WebSocket удобно использовать `wscat` или девтулзы браузера, подписавшись одновременно как пассажир и водитель и создавая объявления через REST API.
