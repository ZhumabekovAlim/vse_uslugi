# Курьерский API Barlyq Qyzmet

Этот документ описывает REST и WebSocket интерфейсы модуля курьерской доставки платформы Barlyq Qyzmet. Все примеры и маршруты подготовлены для публичного домена `https://api.barlyqqyzmet.kz` и ориентированы на разработчиков мобильных приложений курьеров, отправителей и операторов.

## Общие сведения

* Все ответы сервиса возвращаются в формате `application/json` и используют кодировку UTF-8.
* При ошибках сервис возвращает тело `{ "error": "сообщение" }` и HTTP-код в зависимости от ситуации (400 — ошибка валидации, 401 — нет авторизации, 403 — нет прав, 404 — объект не найден, 409 — конфликт статусов, 500 — внутренняя ошибка).【F:internal/courier/http/helpers.go†L49-L64】
* Модуль использует систему ролей. Аутентификация происходит через JWT, который проверяется внешними middleware, после чего в запрос добавляются идентификаторы:
  * `X-Sender-ID` — идентификатор отправителя (клиента) доставки; требуется для маршрутов клиента.【F:cmd/routes.go†L255-L289】【F:internal/courier/http/orders.go†L30-L36】
  * `X-Courier-ID` — идентификатор курьера; требуется для маршрутов исполнителя.【F:cmd/routes.go†L264-L309】【F:internal/courier/http/offers.go†L20-L27】
  * `Authorization: Bearer <token>` — маркер администратора или сервисного пользователя для маршрутов `/api/v1/admin/...` и `/api/v1/couriers`.
* Параметры пагинации: `limit` (по умолчанию 50) и `offset` (по умолчанию 0). Значения должны быть положительными числами.【F:internal/courier/http/helpers.go†L22-L42】
* Временные ограничения запросов составляют 5 секунд (таймаут на стороне сервера).【F:internal/courier/http/orders.go†L83-L99】

## Статусы и жизненный цикл заказов

Последовательность состояний заказа отражена в пакете `lifecycle` и должна соблюдаться обеими сторонами.【F:internal/courier/lifecycle/lifecycle.go†L5-L64】

| Этап | Код статуса | Кто устанавливает | Переходы |
| ---- | ----------- | ----------------- | -------- |
| Новый заказ создан | `new` | система при создании | `offered`, `canceled_by_sender` |
| Предложение отправлено курьерам | `offered` | курьер после оценки стоимости | `assigned`, `canceled_by_sender`, `canceled_by_courier` |
| Курьер назначен | `assigned` | отправитель подтверждает предложение | `courier_arrived`, `canceled_by_*`, `canceled_no_show` |
| Курьер прибыл | `courier_arrived` | курьер | `pickup_started`, `canceled_by_*`, `canceled_no_show` |
| Начата передача посылки | `pickup_started` | курьер | `pickup_done`, `canceled_by_courier` |
| Посылка забрана | `pickup_done` | курьер | `delivery_started`, `canceled_by_courier` |
| Доставка в пути | `delivery_started` | курьер | `delivered`, `canceled_by_courier` |
| Доставка завершена | `delivered` | курьер | `closed` |
| Заказ закрыт (оплата наличными подтверждена) | `closed` | курьер | — |
| Отменён отправителем | `canceled_by_sender` | отправитель | — |
| Отменён курьером | `canceled_by_courier` | курьер | — |
| Курьер не вышел на связь | `canceled_no_show` | курьер | — |

Дополнительные автоматические сокращённые переходы доступны через специальные маршруты (`start`, `waiting/advance`, `waypoints/next`), подробно описанные ниже.

## Расчёт рекомендуемой цены

Рекомендованная цена рассчитывается по формуле `(distance_m * price_per_km) / 1000` с округлением вниз и минимальной границей `min_price`. Значения по умолчанию: 120 ₸ за километр и минимальная цена 500 ₸, но они могут быть переопределены переменными окружения `COURIER_PRICE_PER_KM` и `COURIER_MIN_PRICE` соответственно.【F:internal/courier/pricing/pricing.go†L3-L15】【F:internal/courier/config.go†L9-L73】

## REST API

### Котировка маршрута

`POST /api/v1/courier/route/quote`

Возвращает рекомендуемую стоимость перевозки для заданной дистанции.

**Тело запроса**
```json
{
  "distance_m": 12500
}
```

**Ответ 200**
```json
{
  "recommended_price": 1500
}
```

### Управление заказами (отправитель)

#### Создание заказа

`POST /api/v1/courier/orders`

Заголовки: `X-Sender-ID: <id>`

**Тело запроса**
```json
{
  "distance_m": 12500,
  "eta_s": 2400,
  "client_price": 2000,
  "payment_method": "cash",
  "comment": "Курьер позвонит за 10 минут",
  "route_points": [
    {
      "address": "Алматы, Абая 10",
      "lat": 43.238949,
      "lon": 76.889709,
      "entrance": "1",
      "intercom": "12#34",
      "phone": "+77001234567"
    },
    {
      "address": "Алматы, Достык 50",
      "lat": 43.240101,
      "lon": 76.913456,
      "floor": "3",
      "apt": "15"
    }
  ]
}
```

**Ответ 201**
```json
{
  "order_id": 4102,
  "recommended_price": 1500,
  "status": "new"
}
```

При нарушении правил (меньше двух точек, цена ниже `min_price`, некорректный метод оплаты) возвращается 400 с текстом ошибки.【F:internal/courier/http/orders.go†L30-L78】

#### Список заказов отправителя

`GET /api/v1/courier/orders`

Заголовки: `X-Sender-ID`

Параметры: `limit`, `offset`.

**Ответ 200**
```json
{
  "orders": [
    {
      "id": 4102,
      "sender_id": 912,
      "courier_id": null,
      "distance_m": 12500,
      "eta_s": 2400,
      "recommended_price": 1500,
      "client_price": 2000,
      "payment_method": "cash",
      "status": "new",
      "comment": "Курьер позвонит за 10 минут",
      "created_at": "2023-07-30T10:12:45Z",
      "updated_at": "2023-07-30T10:12:45Z",
      "route_points": [
        {
          "address": "Алматы, Абая 10",
          "lat": 43.238949,
          "lon": 76.889709,
          "entrance": "1",
          "intercom": "12#34",
          "phone": "+77001234567"
        },
        {
          "address": "Алматы, Достык 50",
          "lat": 43.240101,
          "lon": 76.913456,
          "floor": "3",
          "apt": "15"
        }
      ]
    }
  ]
}
```

#### Активный заказ отправителя

`GET /api/v1/courier/orders/active`

Возвращает один активный заказ либо 404, если активных нет.【F:internal/courier/http/orders.go†L109-L148】

#### Получение заказа по ID

`GET /api/v1/courier/orders/{id}`

Доступно как отправителю, так и назначенному курьеру. Возвращает полную карточку заказа или 404, если заказ не найден.【F:internal/courier/http/orders.go†L262-L307】【F:internal/courier/http/orders.go†L471-L495】

#### Отмена заказа отправителем

`POST /api/v1/courier/orders/{id}/cancel`

Заголовки: `X-Sender-ID`

**Тело запроса (опционально)**
```json
{
  "reason": "Получатель отложил доставку"
}
```

**Ответ 200**
```json
{
  "status": "canceled_by_sender"
}
```

#### Изменение цены заказа

`POST /api/v1/courier/orders/{id}/reprice`

Заголовки: `X-Sender-ID`

**Тело запроса**
```json
{
  "client_price": 2200
}
```

Ответ содержит обновлённый заказ или новую цену.【F:internal/courier/http/orders.go†L328-L369】

#### Принятие решения по статусу вручную

`POST /api/v1/courier/orders/{id}/status`

Заголовки: отправитель — для `canceled_by_sender`, курьер — для остальных.

**Тело запроса**
```json
{
  "status": "canceled_by_sender",
  "note": "Покупатель не вышел на связь"
}
```

### Управление заказами (курьер)

Все маршруты требуют заголовок `X-Courier-ID` и отработают только если заказ назначен этому курьеру.【F:internal/courier/http/orders.go†L372-L459】

#### Список заказов курьера

`GET /api/v1/courier/orders`

Возвращает заказы, назначенные курьеру.【F:internal/courier/http/orders.go†L185-L220】

#### Активный заказ курьера

`GET /api/v1/courier/orders/active`

Возвращает текущий активный заказ или 404.【F:internal/courier/http/orders.go†L222-L249】

#### Подтверждение прибытия

`POST /api/v1/courier/orders/{id}/arrive`

Ответ: `{ "status": "courier_arrived" }` при успешном переходе.【F:internal/courier/http/orders.go†L303-L346】

#### Быстрый запуск маршрута

`POST /api/v1/courier/orders/{id}/start`

За один вызов переводит заказ через статусы `pickup_started → pickup_done → delivery_started`. Используется, когда курьер уже забрал посылку и отправляется к получателю.【F:internal/courier/http/orders.go†L430-L459】

#### Завершение доставки

`POST /api/v1/courier/orders/{id}/finish`

Переводит заказ в статус `delivered` и рассылает событие обновления.【F:internal/courier/http/orders.go†L303-L346】

#### Подтверждение наличной оплаты

`POST /api/v1/courier/orders/{id}/confirm-cash`

Переводит заказ из `delivered` в `closed` после подтверждения получения наличных.【F:internal/courier/http/orders.go†L303-L346】

#### Пошаговые статусы ожидания

`POST /api/v1/courier/orders/{id}/waiting/advance`

Выполняет следующий переход в цепочке `assigned → courier_arrived → pickup_started → pickup_done → delivery_started`. Возвращает новый статус или 409, если заказ не в ожидающем состоянии.【F:internal/courier/http/orders.go†L346-L397】

#### Переход к следующей точке маршрута

`POST /api/v1/courier/orders/{id}/waypoints/next`

Доступно на этапе `delivery_started`. Переводит заказ в `delivered`. Используется для многоадресных доставок.【F:internal/courier/http/orders.go†L399-L430】

#### Пауза и возобновление

`POST /api/v1/courier/orders/{id}/pause`

`POST /api/v1/courier/orders/{id}/resume`

Оба маршрута проверяют, что заказ принадлежит текущему курьеру, и возвращают текущий статус заказа без изменения. Предназначены для будущих сценариев паузы и могут использоваться для аналитики приложения.【F:internal/courier/http/orders.go†L499-L547】

#### Отмена курьером

`POST /api/v1/courier/orders/{id}/cancel`

**Ответ** `{ "status": "canceled_by_courier" }` при успехе.【F:internal/courier/http/orders.go†L459-L494】

#### Отметка «Клиент не вышел»

`POST /api/v1/courier/orders/{id}/no-show`

Переводит заказ в `canceled_no_show` и фиксирует событие.【F:internal/courier/http/orders.go†L303-L346】

### Предложения по цене

#### Курьер предлагает цену

`POST /api/v1/courier/offers/price`

Заголовки: `X-Courier-ID`

**Тело запроса**
```json
{
  "order_id": 4102,
  "price": 2100
}
```

Ответ: `{ "status": "offered" }`. Заказ автоматически переводится в статус `offered` (если ещё не назначен).【F:internal/courier/http/offers.go†L16-L69】

#### Отправитель принимает предложение

`POST /api/v1/courier/offers/accept`

Заголовки: `X-Sender-ID`

**Тело запроса**
```json
{
  "order_id": 4102,
  "courier_id": 305,
  "price": 2100
}
```

Ответ: `{ "status": "assigned" }`. Курьер закрепляется за заказом.【F:internal/courier/http/offers.go†L71-L120】

#### Отправитель отклоняет предложение

`POST /api/v1/courier/offers/decline`

Ответ: `{ "status": "declined" }`.【F:internal/courier/http/offers.go†L122-L163】

#### Унифицированный ответ на предложение

`POST /api/v1/courier/offers/respond`

Тело содержит `decision: "accept"` или `"decline"`. Возвращает `status: "assigned"` или `status: "declined"`. Используется для интеграции с интерфейсами, где решение принимается единым действием.【F:internal/courier/http/offers.go†L165-L228】

### Профиль курьера и статистика

Маршруты требуют сервисной авторизации (заголовок `Authorization`).

* `POST /api/v1/couriers` — создать или обновить профиль курьера. Поля `user_id`, `first_name`, `last_name`, `courier_photo`, `iin`, `date_of_birth` обязательны. Дата рождения — `YYYY-MM-DD`. Возвращает профиль и статистику заказов курьера.【F:internal/courier/http/couriers.go†L17-L117】
* `GET /api/v1/courier/{id}/profile` — получить профиль и статистику курьера.【F:internal/courier/http/couriers.go†L119-L163】
* `GET /api/v1/courier/{id}/reviews` — на данный момент возвращает пустой список отзывов.【F:internal/courier/http/couriers.go†L165-L175】
* `GET /api/v1/courier/{id}/stats` — агрегированные показатели (`total_orders`, `active_orders`, `completed_orders`, `canceled_orders`).【F:internal/courier/http/couriers.go†L177-L188】【F:internal/courier/repo/orders.go†L520-L581】

### Баланс курьера

* `POST /api/v1/courier/balance/deposit`
* `POST /api/v1/courier/balance/withdraw`

Обе операции пока не реализованы и возвращают 501 с текстом `courier balance operations are not supported yet`.【F:internal/courier/http/balance.go†L3-L10】

### Административные маршруты

Требуют заголовок `Authorization` администратора.

* `GET /api/v1/admin/courier/orders` — список всех заказов с пагинацией.【F:internal/courier/http/admin.go†L14-L45】
* `GET /api/v1/admin/courier/orders/stats` — агрегированная статистика заказов (`total_orders`, `active_orders`, `completed_orders`, `canceled_orders`).【F:internal/courier/http/admin.go†L47-L71】【F:internal/courier/repo/orders.go†L462-L519】
* `GET /api/v1/admin/courier/couriers` — список профилей курьеров.【F:internal/courier/http/admin.go†L73-L108】
* `GET /api/v1/admin/courier/couriers/stats` — количество курьеров по статусам (`total_couriers`, `pending_couriers`, `active_couriers`, `banned_couriers`).【F:internal/courier/http/admin.go†L110-L139】【F:internal/courier/repo/couriers.go†L49-L129】
* `POST /api/v1/admin/courier/couriers/{id}/ban` — установить или снять бан курьера. Тело `{ "ban": true }` или `{ "ban": false }`. Возвращает текущий статус профиля (`banned` или `active`).【F:internal/courier/http/admin.go†L141-L190】
* `POST /api/v1/admin/courier/couriers/{id}/approval` — обновить статус проверки курьера. Пустое тело присваивает статус `approved`, можно передать `{ "status": "pending" }` и т.д.【F:internal/courier/http/admin.go†L192-L222】

## WebSocket API

Для получения push-событий доступны два канала:

* `GET /ws/courier` — поток событий для курьеров. Требуется JWT с ролью `worker`; идентификатор курьера передаётся через параметр `courier_id` в query string или заголовок `X-Courier-ID`.【F:cmd/routes.go†L310-L316】【F:internal/courier/ws/hub.go†L1-L214】
* `GET /ws/sender` — поток событий для отправителей. Требуется JWT с ролью `client`; идентификатор задаётся через `sender_id` или заголовок `X-Sender-ID`.【F:cmd/routes.go†L315-L316】【F:internal/courier/ws/hub.go†L214-L256】

Поддерживаются авто-пинг и авто-переподключение. Сообщения имеют два типа:

### События заказов

```json
{
  "type": "order_updated",
  "order": {
    "id": 4102,
    "status": "delivery_started",
    "sender_id": 912,
    "courier_id": 305,
    "route_points": [ ... ]
  }
}
```

Типы: `order_created`, `order_updated`. События транслируются как курьеру, так и отправителю, когда заказ изменяется или создаётся.【F:internal/courier/http/events.go†L1-L58】

### События предложений

```json
{
  "type": "offer_updated",
  "order_id": 4102,
  "courier_id": 305,
  "status": "accepted",
  "price": 2100
}
```

Событие отправляется обеим сторонам при изменении статуса предложения (`proposed`, `accepted`, `declined`).【F:internal/courier/http/events.go†L60-L96】

## Примеры пошагового сценария

1. Отправитель рассчитывает рекомендуемую цену через `/api/v1/courier/route/quote`.
2. Создаёт заказ `/api/v1/courier/orders` со своей ценой (`client_price`). Заказ получает статус `new`.
3. Курьеры через WebSocket получают событие `order_created` и могут предложить стоимость через `/api/v1/courier/offers/price`. Заказ переходит в `offered`.
4. Отправитель выбирает предложение и подтверждает его `/api/v1/courier/offers/accept`. Заказ становится `assigned`, закрепляется за курьером.
5. Курьер прибывает, отмечает `/arrive` → `courier_arrived`.
6. Курьер переводит заказ в рабочий статус `/waiting/advance` (или сразу `/start`), выполняет доставку и завершает `/finish`.
7. При оплате наличными курьер подтверждает `/confirm-cash`, после чего заказ считается закрытым (`closed`). При онлайн-оплате этот шаг можно пропустить.

## Postman коллекция

Полная коллекция с описанными запросами и примерами находится в файле [`docs/courier_postman_collection.json`](./courier_postman_collection.json). Её можно импортировать в Postman или любой совместимый клиент.

## Контрольный список интеграции

* Реализуйте обработку всех кодов статусов из таблицы жизненного цикла.
* Сохраняйте и передавайте идентификаторы `sender_id` и `courier_id` в HTTP-заголовках и при подключении к WebSocket.
* При получении ответа 409 повторно запросите заказ, чтобы синхронизировать состояние.
* Минимальная цена контролируется сервером — обновляйте UI по сообщению об ошибке.
* Следите за таймаутами запросов: операции должны укладываться в 5 секунд.

