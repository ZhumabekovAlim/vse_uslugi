# Такси API: руководство для фронтенда

Документ описывает HTTP и WebSocket интерфейсы такси-сервиса, необходимые для реализации пользовательских приложений пассажира и водителя. Все примеры приведены на русском языке.

## Общие сведения

- **Базовый URL**: `https://<домен>` (в Postman коллекции используется переменная `{{base_url}}`).
- **Версия API**: `v1`, все маршруты начинаются с `/api/v1/...` или `/ws/...` согласно регистрации роутов в сервере. 【F:internal/taxi/http/server.go†L172-L181】
- **Формат данных**: JSON UTF-8. Ответы и ошибки также возвращаются в JSON с ключом `error`, если применимо.
- **Аутентификация**: выполняется через служебные заголовки:
  - `X-Passenger-ID` — обязательный для запросов, выполняемых от имени пассажира. 【F:internal/taxi/http/server.go†L495-L500】
  - `X-Driver-ID` — обязателен, когда водитель подтверждает оффер. 【F:internal/taxi/http/server.go†L708-L716】
  - Для вебхука платежей требуется `X-AirbaPay-Signature`. 【F:internal/taxi/http/server.go†L821-L829】

## Сущности

### Водитель (Driver)

Полный список полей, принимаемых в телах POST/PUT, определён в `driverPayload` и проходит валидацию. Основные требования: статус (`offline|free|busy`), номер автомобиля, техпаспорт, фотографии машины и документов, фото водителя, телефон, ИИН. 【F:internal/taxi/http/server.go†L57-L126】

Ответ `driverResponse` дополняется идентификатором, рейтингом и временной меткой обновления. 【F:internal/taxi/http/server.go†L128-L170】

### Заказ (Order)

Структура заказа хранит координаты, цены, статус, тип оплаты, а также список точек маршрута (`OrderAddress`). 【F:internal/taxi/repo/orders.go†L11-L40】

Минимум два адреса обязательны для успешного создания. 【F:internal/taxi/repo/orders.go†L64-L90】

### Статусы заказа

Допустимые переходы между статусами определены в FSM: `created → searching → accepted → arrived → picked_up → completed → paid → closed`, а также ветки отмены/не найден. 【F:internal/taxi/fsm/fsm.go†L9-L33】

## REST API

### `/api/v1/drivers`

| Метод | Описание | Особенности |
|-------|----------|-------------|
| `GET` | Получение списка водителей с пагинацией `limit/offset`. | Значения валидируются, по умолчанию `limit=100`. 【F:internal/taxi/http/server.go†L219-L252】 |
| `POST` | Создание водителя. | JSON из всех обязательных полей; валидация выполняется сервером. 【F:internal/taxi/http/server.go†L254-L297】 |

### `/api/v1/drivers/{id}`

| Метод | Описание | Особенности |
|-------|----------|-------------|
| `GET` | Получить карточку водителя. | Возвращает 404, если водитель не найден. 【F:internal/taxi/http/server.go†L300-L314】 |
| `PUT` | Полное обновление данных. | Требуются все поля; выполняется повторная валидация. 【F:internal/taxi/http/server.go†L316-L342】 |
| `DELETE` | Удаление водителя. | Ответ без тела, код 204. 【F:internal/taxi/http/server.go†L343-L369】 |

### `/api/v1/route/quote`

- **Метод**: `POST`
- **Вход**: адреса `from_address`/`to_address` либо структуры `from/to` с координатами. Один из вариантов обязателен для каждой точки. 【F:internal/taxi/http/server.go†L406-L453】
- **Ответ**: координаты маршрута, расстояние в метрах, ETA в секундах, рекомендованная и минимальная цена. Цена округляется вниз до шага 50 и не опускается ниже минимума из конфигурации. 【F:internal/taxi/http/server.go†L458-L484】
- **Ошибки**: 400 при некорректном JSON/отсутствии точек, 502 при сбое геосервиса.

### `/api/v1/orders`

- **Метод**: `POST`
- **Заголовок**: `X-Passenger-ID`.
- **Вход**: маршрут `from`, `to`, массив промежуточных `stops`, расстояние `distance_m`, ETA `eta_s`, выбранная цена `client_price`, метод оплаты `online|cash`, комментарий `notes`. 【F:internal/taxi/http/server.go†L501-L523】
- **Проверки**:
  - Цена не может быть ниже минимальной. 【F:internal/taxi/http/server.go†L527-L533】
  - Каждая точка должна содержать координаты; расстояние и ETA проверяются по матрице маршрутов. 【F:internal/taxi/http/server.go†L536-L577】
- **Ответ**: `order_id` и рекомендованная цена от сервера. 【F:internal/taxi/http/server.go†L615-L627】
- **Побочные эффекты**: заказ создаётся со статусом `searching`, запускается запись в таблице диспетчеризации и триггерится первый тик поиска водителей. 【F:internal/taxi/http/server.go†L589-L625】【F:internal/taxi/repo/orders.go†L64-L95】

### `/api/v1/orders/{id}/reprice`

- **Метод**: `POST`
- **Вход**: `client_price` ≥ минимальной цене. 【F:internal/taxi/http/server.go†L658-L688】
- **Действие**: обновляет цену и перезапускает диспетчеризацию. 【F:internal/taxi/http/server.go†L689-L699】

### `/api/v1/orders/{id}/status`

- **Метод**: `POST`
- **Вход**: `status` — целевой статус.
- **Правила**: переход разрешён только если соответствует FSM, иначе 409. 【F:internal/taxi/http/server.go†L767-L799】【F:internal/taxi/fsm/fsm.go†L9-L33】
- **Побочный эффект**: пассажиру отправляется WebSocket-уведомление. 【F:internal/taxi/http/server.go†L791-L794】
- **Оплата**: при `completed` и методе `online` создаётся платёж в AirbaPay. 【F:internal/taxi/http/server.go†L793-L818】

### `/api/v1/offers/accept`

- **Метод**: `POST`
- **Заголовок**: `X-Driver-ID`.
- **Вход**: `order_id`.
- **Логика**: проверяет существование водителя, подтверждает оффер, назначает водителя заказу и уведомляет пассажира о назначении. 【F:internal/taxi/http/server.go†L708-L742】

### `/api/v1/payments/airbapay/webhook`

- **Метод**: `POST`
- **Заголовок**: `X-AirbaPay-Signature` с HMAC подписью.
- **Действие**: сохраняет payload, при статусе `paid` обновляет заказ до `paid`, переводит платеж в состояние `paid` и уведомляет пассажира. 【F:internal/taxi/http/server.go†L821-L870】

## WebSocket API

### `/ws/driver`

- **Подключение**: GET `wss://<домен>/ws/driver?driver_id=<id>&city=<slug>`. ID обязателен; город опционален (по умолчанию `default`). 【F:internal/taxi/ws/driver.go†L66-L92】
- **Исходящие сообщения клиента**: периодические JSON с координатами и статусом водителя. Статус по умолчанию `free`, если не передан. 【F:internal/taxi/ws/driver.go†L112-L133】
- **Входящие сообщения сервера**: события `order_offer` со структурой `DriverOfferPayload` (ID заказа, маршрут, цена, ETA, срок действия). 【F:internal/taxi/ws/driver.go†L29-L147】

### `/ws/passenger`

- **Подключение**: GET `wss://<домен>/ws/passenger?passenger_id=<id>`. 【F:internal/taxi/ws/passenger.go†L39-L57】
- **Сообщения сервера**: объекты `PassengerEvent` с типами `order_assigned`, `order_status` и т. д., содержащие статус, радиус поиска или текстовое сообщение. 【F:internal/taxi/ws/passenger.go†L12-L97】
- **Назначение**: доставка событий о назначении водителя, изменении статуса заказа, расширении радиуса поиска.

## Потоки взаимодействия

### Создание поездки пассажиром

1. Клиент рассчитывает цену через `/api/v1/route/quote` (опционально). 【F:internal/taxi/http/server.go†L400-L484】
2. Отправляет `/api/v1/orders` с выбранной ценой и маршрутом. 【F:internal/taxi/http/server.go†L495-L627】
3. Подписывается на `/ws/passenger` для получения статусов. 【F:internal/taxi/http/server.go†L877-L879】【F:internal/taxi/ws/passenger.go†L39-L97】
4. Получает push о назначении водителя и дальнейших изменениях.

### Работа водителя

1. Подключается к `/ws/driver`, передаёт координаты и статус. 【F:internal/taxi/ws/driver.go†L66-L134】
2. Получает `order_offer`, подтверждает его через `/api/v1/offers/accept`. 【F:internal/taxi/ws/driver.go†L29-L147】【F:internal/taxi/http/server.go†L708-L742】
3. По мере выполнения заказа отправляет обновления статуса через `/api/v1/orders/{id}/status` (например, `arrived`, `picked_up`, `completed`). 【F:internal/taxi/http/server.go†L767-L799】【F:internal/taxi/fsm/fsm.go†L9-L33】

### Онлайн-оплата

1. После установки статуса `completed` с методом оплаты `online` сервер автоматически создаёт платеж AirbaPay и сохраняет запись. 【F:internal/taxi/http/server.go†L793-L818】
2. По webhook от платёжного провайдера статус заказа переводится в `paid`, пассажиру приходит событие. 【F:internal/taxi/http/server.go†L821-L870】

## Работа с ошибками

- Неверный JSON, отсутствующие обязательные поля или параметры → `400 Bad Request` с сообщением из валидации. 【F:internal/taxi/http/server.go†L254-L263】【F:internal/taxi/http/server.go†L523-L533】
- Отсутствие требуемых заголовков → `401 Unauthorized`. 【F:internal/taxi/http/server.go†L495-L500】【F:internal/taxi/http/server.go†L708-L714】
- Нарушение бизнес-правил (например, неверный статус) → `409 Conflict`. 【F:internal/taxi/http/server.go†L767-L785】
- Прочие ошибки сервера и внешних сервисов → `500/502` с кратким описанием. 【F:internal/taxi/http/server.go†L244-L245】【F:internal/taxi/http/server.go†L563-L565】

## Инструменты для тестирования

Готовая Postman-коллекция `docs/taxi_api_postman_collection.json` содержит примеры всех описанных запросов, включая заголовки и тела на русском языке. Используйте переменную `{{base_url}}` для переключения между окружениями.

