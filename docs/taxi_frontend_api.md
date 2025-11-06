# Такси API: руководство для фронтенда

Документ описывает HTTP и WebSocket интерфейсы такси-сервиса, необходимые для реализации пользовательских приложений пассажира и водителя. Все примеры приведены на русском языке.

## Общие сведения

- **Базовый URL**: `https://<домен>` (в Postman коллекции используется переменная `{{base_url}}`).
- **Версия API**: `v1`, все маршруты начинаются с `/api/v1/...` или `/ws/...` согласно регистрации роутов в сервере. 【F:internal/taxi/http/server.go†L249-L258】
- **Формат данных**: JSON UTF-8. Ответы и ошибки также возвращаются в JSON с ключом `error`, если применимо.
- **Аутентификация**: выполняется через служебные заголовки:
  - `X-Passenger-ID` — обязательный для запросов, выполняемых от имени пассажира. 【F:internal/taxi/http/server.go†L575-L578】【F:internal/taxi/http/server.go†L630-L633】【F:internal/taxi/http/server.go†L815-L818】
  - `X-Driver-ID` — обязателен, когда водитель подтверждает оффер. 【F:internal/taxi/http/server.go†L889-L892】
  - Для вебхука платежей требуется `X-AirbaPay-Signature`. 【F:internal/taxi/http/server.go†L1033-L1049】

## Сущности

### Водитель (Driver)

Полный список полей, принимаемых в телах POST/PUT, определён в `driverPayload` и проходит валидацию. Основные требования: статус (`offline|free|busy`), номер автомобиля, техпаспорт, фотографии машины и документов, фото водителя, телефон, ИИН. 【F:internal/taxi/http/server.go†L58-L127】

Ответ `driverResponse` дополняется идентификатором, рейтингом и временной меткой обновления. 【F:internal/taxi/http/server.go†L129-L170】

### Заказ (Order)

Структура заказа хранит координаты, цены, статус, тип оплаты, а также список точек маршрута (`OrderAddress`). 【F:internal/taxi/repo/orders.go†L11-L40】

Минимум два адреса обязательны для успешного создания. 【F:internal/taxi/repo/orders.go†L64-L95】

При формировании HTTP-ответа используется `orderResponse`: помимо базовых полей заказа он включает массив адресов и вложенный объект `driver` с полной карточкой водителя, если он назначен. 【F:internal/taxi/http/server.go†L181-L246】

### Статусы заказа

Допустимые переходы между статусами определены в FSM: `created → searching → accepted → arrived → picked_up → completed → paid → closed`, а также ветки отмены/не найден. 【F:internal/taxi/fsm/fsm.go†L9-L33】

## REST API

### `/api/v1/drivers`

| Метод | Описание | Особенности |
|-------|----------|-------------|
| `GET` | Получение списка водителей с пагинацией `limit/offset`. | Значения валидируются, по умолчанию `limit=100`. 【F:internal/taxi/http/server.go†L296-L329】 |
| `POST` | Создание водителя. | JSON из всех обязательных полей; валидация выполняется сервером. 【F:internal/taxi/http/server.go†L331-L374】 |

### `/api/v1/drivers/{id}`

| Метод | Описание | Особенности |
|-------|----------|-------------|
| `GET` | Получить карточку водителя. | Возвращает 404, если водитель не найден. 【F:internal/taxi/http/server.go†L377-L391】 |
| `PUT` | Полное обновление данных. | Требуются все поля; выполняется повторная валидация. 【F:internal/taxi/http/server.go†L393-L442】 |
| `DELETE` | Удаление водителя. | Ответ без тела, код 204. 【F:internal/taxi/http/server.go†L444-L457】 |

### `/api/v1/route/quote`

- **Метод**: `POST`
- **Вход**: адреса `from_address`/`to_address` либо структуры `from/to` с координатами. Один из вариантов обязателен для каждой точки. 【F:internal/taxi/http/server.go†L477-L529】
- **Ответ**: координаты маршрута, расстояние в метрах, ETA в секундах, рекомендованная и минимальная цена. Цена округляется вниз до шага 50 и не опускается ниже минимума из конфигурации. 【F:internal/taxi/http/server.go†L535-L561】
- **Ошибки**: 400 при некорректном JSON/отсутствии точек, 502 при сбое геосервиса. 【F:internal/taxi/http/server.go†L496-L499】【F:internal/taxi/http/server.go†L697-L699】

### `/api/v1/orders`

| Метод | Описание | Особенности |
|-------|----------|-------------|
| `GET` | История поездок пассажира. | Требуется `X-Passenger-ID`; поддерживает `limit`/`offset`, возвращает вложенные данные водителя для назначенных заказов. 【F:internal/taxi/http/server.go†L574-L626】【F:internal/taxi/http/server.go†L181-L246】【F:internal/taxi/repo/orders.go†L113-L146】 |
| `POST` | Создание заказа. | Требуется `X-Passenger-ID`; валидирует цену, маршрут и запускает диспетчеризацию. 【F:internal/taxi/http/server.go†L629-L761】【F:internal/taxi/repo/orders.go†L64-L95】 |

**POST /api/v1/orders**

- **Тело**: маршрут `from`, `to`, промежуточные `stops`, ожидаемое расстояние `distance_m`, ETA `eta_s`, выбранная цена `client_price`, метод оплаты `online|cash`, комментарий `notes`. 【F:internal/taxi/http/server.go†L635-L655】
- **Валидация**: цена ≥ минимальной и метод оплаты из допустимого списка; каждая точка маршрута должна содержать координаты и формирует минимум две точки. 【F:internal/taxi/http/server.go†L661-L687】
- **Проверка маршрута**: сервер сверяет дистанцию и ETA с матрицей дорог и отклонения более 10% отклоняются. 【F:internal/taxi/http/server.go†L689-L773】
- **Ответ**: `order_id` и пересчитанная рекомендованная цена. 【F:internal/taxi/http/server.go†L723-L762】
- **Сайд-эффекты**: сохраняются адреса и создаётся запись диспетчеризации со стартовым радиусом; после создания запускается немедленный тик поиска. 【F:internal/taxi/repo/orders.go†L64-L95】【F:internal/taxi/http/server.go†L739-L759】

**GET /api/v1/orders**

- **Параметры**: `limit` (по умолчанию 50) и `offset` ≥ 0. 【F:internal/taxi/http/server.go†L580-L597】
- **Ответ**: объект с массивом `orders`, включающим адреса и профиль водителя, если он назначен. 【F:internal/taxi/http/server.go†L608-L626】【F:internal/taxi/http/server.go†L181-L246】

**GET /api/v1/driver/orders**

- **Заголовок**: `X-Driver-ID`.
- **Параметры**: `limit` (по умолчанию 50) и `offset` ≥ 0. 【F:internal/taxi/http/server.go†L716-L735】
- **Ответ**: объект с массивом `orders`; каждый элемент включает адреса, а при успешной загрузке профиля текущего водителя — вложенную карточку `driver`. 【F:internal/taxi/http/server.go†L739-L759】【F:internal/taxi/http/server.go†L181-L246】

### `/api/v1/orders/{id}/reprice`

- **Метод**: `POST`
- **Вход**: `client_price` ≥ минимальной цене. 【F:internal/taxi/http/server.go†L849-L858】
- **Действие**: обновляет цену и перезапускает диспетчеризацию. 【F:internal/taxi/http/server.go†L862-L880】

### `/api/v1/orders/{id}/status`

- **Метод**: `POST`
- **Вход**: `status` — целевой статус.
- **Правила**: переход разрешён только если соответствует FSM, иначе 409. 【F:internal/taxi/http/server.go†L948-L983】【F:internal/taxi/fsm/fsm.go†L9-L33】
- **Контроль завершения**: для статуса `completed` требуется координаты водителя; завершение отклоняется, если расстояние до конечной точки >300 м. 【F:internal/taxi/http/server.go†L962-L973】
- **Побочный эффект**: пассажиру отправляется WebSocket-уведомление. 【F:internal/taxi/http/server.go†L986-L987】
- **Оплата**: при `completed` и методе `online` создаётся платёж в AirbaPay. 【F:internal/taxi/http/server.go†L988-L990】

### `/api/v1/orders/{id}`

| Метод | Описание | Особенности |
|-------|----------|-------------|
| `GET` | Получить подробности заказа. | Требуется `X-Passenger-ID`; возвращает 403 для чужих заказов и включает карточку водителя. 【F:internal/taxi/http/server.go†L775-L847】 |

### `/api/v1/offers/accept`

- **Метод**: `POST`
- **Заголовок**: `X-Driver-ID`.
- **Вход**: `order_id`.
- **Логика**: проверяет существование водителя, подтверждает оффер, назначает водителя заказу и уведомляет пассажира о назначении. 【F:internal/taxi/http/server.go†L884-L920】

### `/api/v1/payments/airbapay/webhook`

- **Метод**: `POST`
- **Заголовок**: `X-AirbaPay-Signature` с HMAC подписью.
- **Действие**: сохраняет payload, при статусе `paid` обновляет заказ до `paid`, переводит платеж в состояние `paid` и уведомляет пассажира. 【F:internal/taxi/http/server.go†L1028-L1077】

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

1. Клиент рассчитывает цену через `/api/v1/route/quote` (опционально). 【F:internal/taxi/http/server.go†L477-L561】
2. Отправляет `/api/v1/orders` с выбранной ценой и маршрутом. 【F:internal/taxi/http/server.go†L629-L761】
3. Подписывается на `/ws/passenger` для получения статусов. 【F:internal/taxi/http/server.go†L249-L258】【F:internal/taxi/ws/passenger.go†L39-L97】
4. Получает push о назначении водителя и дальнейших изменениях.

### Работа водителя

1. Подключается к `/ws/driver`, передаёт координаты и статус. 【F:internal/taxi/ws/driver.go†L66-L134】
2. Получает `order_offer`, подтверждает его через `/api/v1/offers/accept`. 【F:internal/taxi/ws/driver.go†L29-L147】【F:internal/taxi/http/server.go†L884-L920】
3. По мере выполнения заказа отправляет обновления статуса через `/api/v1/orders/{id}/status` (например, `arrived`, `picked_up`, `completed`). 【F:internal/taxi/http/server.go†L948-L992】【F:internal/taxi/fsm/fsm.go†L9-L33】

### Онлайн-оплата

1. После установки статуса `completed` с методом оплаты `online` сервер автоматически создаёт платеж AirbaPay и сохраняет запись. 【F:internal/taxi/http/server.go†L988-L1025】
2. По webhook от платёжного провайдера статус заказа переводится в `paid`, пассажиру приходит событие. 【F:internal/taxi/http/server.go†L1028-L1077】

## Работа с ошибками

- Неверный JSON, отсутствующие обязательные поля или параметры → `400 Bad Request` с сообщением из валидации. 【F:internal/taxi/http/server.go†L331-L336】【F:internal/taxi/http/server.go†L657-L667】
- Отсутствие требуемых заголовков → `401 Unauthorized`. 【F:internal/taxi/http/server.go†L575-L578】【F:internal/taxi/http/server.go†L889-L892】
- Нарушение бизнес-правил (например, неверный статус) → `409 Conflict`. 【F:internal/taxi/http/server.go†L957-L973】
- Прочие ошибки сервера и внешних сервисов → `500/502` с кратким описанием. 【F:internal/taxi/http/server.go†L320-L323】【F:internal/taxi/http/server.go†L697-L699】

## Инструменты для тестирования

Готовая Postman-коллекция `docs/taxi_api_postman_collection.json` содержит примеры всех описанных запросов, включая заголовки и тела на русском языке. Используйте переменную `{{base_url}}` для переключения между окружениями.

