# Жизненный цикл заказа такси

## Что реализовано

Модуль `internal/taxi/lifecycle` описывает полный жизненный цикл поездки такси как надстройку над существующей системой состояний. Добавлены новые статусы заказов и переходы, позволяющие фиксировать прибытие водителя, интервалы ожидания, ход поездки, завершение и разные варианты отмены, не ломая прежние сценарии. 【F:internal/taxi/fsm/fsm.go†L10-L116】

Для работы с жизненным циклом введены:

* структура конфигурации с гео-порогами, тарифами ожидания, политиками кнопок и PIN-проверкой для посадки; сервис использует её при валидации и расчётах. 【F:internal/taxi/lifecycle/config.go†L5-L57】
* доменная модель заказа, которая хранит маршрут с точками, таймлайн статусов, сессии ожидания, журнал точек и контактов, а также детализированную разбивку чека. 【F:internal/taxi/lifecycle/order.go†L11-L150】
* сервис жизненного цикла, реализующий действия водителя и системы (прибытие, старт, точки, паузы, завершение, подтверждение наличной оплаты, отмены и no-show) с гео-валидацией и идемпотентностью. 【F:internal/taxi/lifecycle/service.go†L10-L347】

Пакет покрыт unit-тестами, демонстрирующими базовый сценарий поездки и крайние случаи отмены и no-show. 【F:internal/taxi/lifecycle/service_test.go†L10-L199】

## Конфигурация

`Config` задаёт параметры работы сервиса: радиусы геозон для прибытия, старта, промежуточных точек и финиша; порог скорости «почти стоим»; допустимую свежесть GPS; длительность бесплатного ожидания; ставки платного ожидания и пауз; признак необходимости PIN; TTL оффера; политики по кнопкам (лимиты повторных нажатий, задержки, окна учёта). 【F:internal/taxi/lifecycle/config.go†L5-L57】

Пример настройки:

```go
cfg := lifecycle.Config{
    ArrivalRadiusMeters:      50,
    StartRadiusMeters:        50,
    WaypointRadiusMeters:     30,
    FinishRadiusMeters:       50,
    StationarySpeedKPH:       5,
    CoordinateFreshness:      time.Minute,
    FreeWaitingWindow:        3 * time.Minute,
    PaidWaitingRatePerMinute: 100,
    PauseRatePerMinute:       200,
    RequireBoardingPIN:       true,
    OfferTTL:                 15 * time.Minute,
    ButtonPolicies: map[lifecycle.Action]lifecycle.ButtonPolicy{
        lifecycle.ActionArrive: {Cooldown: 10 * time.Second, MaxPresses: 3, TTL: time.Minute},
    },
}
svc := lifecycle.NewService(cfg)
```

## Жизненный цикл заказа

### Создание

`NewOrder` принимает идентификаторы, базовый тариф, валюту, момент создания и маршрут (минимум точка посадки и финиш). Заказ сразу получает статус `assigned`, таймлайн стартует событием «driver assigned», а срок действия оффера рассчитывается по TTL. 【F:internal/taxi/lifecycle/order.go†L170-L195】

### Прибытие к точке A

Метод `MarkDriverAtPickup` разблокирует состояние ожидания, если водитель находится в радиусе прибытия, скорость ниже порога, а телеметрия свежая. Действие идемпотентно: повторные нажатия не создают лишние сессии. При успехе фиксируется статус `driver_at_pickup`, запускается бесплатное ожидание `waiting_free`, и в журнал попадает отметка о точке A. 【F:internal/taxi/lifecycle/service.go†L61-L100】【F:internal/taxi/lifecycle/order.go†L213-L229】

### Управление ожиданием

`AdvanceWaiting` автоматически переводит бесплатное ожидание в платное, когда истекает окно `FreeWaitingWindow`. Завершённая бесплатная сессия закрывается и создаётся платная запись `waiting_paid`. 【F:internal/taxi/lifecycle/service.go†L102-L122】【F:internal/taxi/lifecycle/order.go†L238-L281】

Вызов `StartPause` / `EndPause` добавляет отдельную сессию ожидания «в пути» с собственной ставкой, что отражается в чековом разборе. 【F:internal/taxi/lifecycle/service.go†L186-L216】【F:internal/taxi/lifecycle/order.go†L238-L281】

### Старт поездки

`StartTrip` проверяет радиус старта, свежесть телеметрии и при включённом флаге — подтверждение PIN. После успешной проверки бесплатные/платные ожидания закрываются, фиксируется статус `in_progress`, стартовое время и базовая сумма чека. 【F:internal/taxi/lifecycle/service.go†L124-L157】

### Промежуточные точки

`ReachWaypoint` работает для точек типа `stop`: валидация по радиусу точки, логирование события и продвижение активного waypoint. Финишная точка пропускается этим методом, чтобы завершение происходило отдельным действием. 【F:internal/taxi/lifecycle/service.go†L159-L184】

### Паузы по просьбе пассажира

При паузе `StartPause` открывает сессию типа `in_trip_pause`, `EndPause` её закрывает, рассчитывая длительность и сумму по ставке `PauseRatePerMinute`. Данные сохраняются в `FareBreakdown`. 【F:internal/taxi/lifecycle/service.go†L186-L216】【F:internal/taxi/lifecycle/order.go†L238-L281】

### Завершение и расчёт

`FinishTrip` проверяет прибытие в финальную точку (или ближайшую доступную), закрывает текущие ожидания, отмечает достижение финала и переводит заказ в статус `at_last_point` с фиксацией времени. 【F:internal/taxi/lifecycle/service.go†L218-L257】

`ConfirmCashPayment` подтверждает получение наличных, переводя заказ в статус `completed` и закрывая активные ожидания. 【F:internal/taxi/lifecycle/service.go†L259-L274】

`FareBreakdown.Total()` суммирует базовую стоимость, платное ожидание, паузы, доп.км и скидки. 【F:internal/taxi/lifecycle/order.go†L103-L118】

### Отмены и no-show

`CancelByPassenger` и `CancelByDriver` доступны до завершения. Они закрывают активные ожидания, фиксируют статус отмены и причину в заметке. Повторные вызовы безопасны. 【F:internal/taxi/lifecycle/service.go†L276-L320】

`MarkNoShow` разрешён из состояний ожидания, требует подтверждения нахождения у точки A и закрывает ожидания, после чего устанавливает статус `no_show`. 【F:internal/taxi/lifecycle/service.go†L322-L347】

Unit-тесты демонстрируют: полный happy-path (прибытие → платное ожидание → старт → waypoint → пауза → финиш → подтверждение оплаты), а также сценарии no-show и отмены пассажиром. 【F:internal/taxi/lifecycle/service_test.go†L10-L199】

## Примерный REST-интерфейс

Ниже — предлагаемые HTTP-роуты для драйверского приложения. Каждый маршрут должен вызывать соответствующий метод `Service`, применять бизнес-валидации и обеспечивать идемпотентность.

### 1. «Я на месте»

`POST /api/taxi/orders/{orderId}/arrive`

```json
{
  "timestamp": "2023-10-05T10:10:00Z",
  "position": {"lat": 43.25, "lon": 76.9},
  "speed_kph": 2.0
}
```

Backend вызывает `Service.MarkDriverAtPickup`, который проверяет скорость, радиус и свежесть координат, и при успехе переводит заказ в `waiting_free`. 【F:internal/taxi/lifecycle/service.go†L61-L100】

### 2. Автоперевод ожидания

`POST /api/taxi/orders/{orderId}/waiting/advance`

Без тела: достаточно серверного времени. Роут вызывает `Service.AdvanceWaiting` и при возврате `true` оповещает клиента о переходе к платному ожиданию. 【F:internal/taxi/lifecycle/service.go†L102-L122】

### 3. Старт поездки

`POST /api/taxi/orders/{orderId}/start`

```json
{
  "timestamp": "2023-10-05T10:15:00Z",
  "position": {"lat": 43.25, "lon": 76.9},
  "speed_kph": 0.5,
  "pin_confirmed": true
}
```

Маршрут вызывает `Service.StartTrip`, который проверяет радиус старта, PIN и закрывает ожидания. 【F:internal/taxi/lifecycle/service.go†L124-L157】

### 4. Промежуточная точка

`POST /api/taxi/orders/{orderId}/waypoints/next`

```json
{
  "timestamp": "2023-10-05T10:25:00Z",
  "position": {"lat": 43.255, "lon": 76.905},
  "speed_kph": 10.0
}
```

Роут вызывает `Service.ReachWaypoint`, что подтверждает достижение текущей промежуточной точки и записывает событие в журнал. 【F:internal/taxi/lifecycle/service.go†L159-L184】【F:internal/taxi/lifecycle/order.go†L213-L229】

### 5. Пауза и возобновление

* `POST /api/taxi/orders/{orderId}/pause`
* `POST /api/taxi/orders/{orderId}/resume`

Тела могут содержать только отметку времени. Эти действия открывают и закрывают сессию ожидания в пути, начисляя стоимость по ставке `PauseRatePerMinute`. 【F:internal/taxi/lifecycle/service.go†L186-L216】【F:internal/taxi/lifecycle/order.go†L238-L281】

### 6. Завершение поездки

`POST /api/taxi/orders/{orderId}/finish`

```json
{
  "timestamp": "2023-10-05T10:35:00Z",
  "position": {"lat": 43.26, "lon": 76.91},
  "speed_kph": 1.0
}
```

Роут вызывает `Service.FinishTrip`, который проверяет геозону финиша и фиксирует статус `at_last_point`. 【F:internal/taxi/lifecycle/service.go†L218-L257】

### 7. Подтверждение наличной оплаты

`POST /api/taxi/orders/{orderId}/confirm-cash`

```json
{
  "timestamp": "2023-10-05T10:36:00Z"
}
```

Маршрут вызывает `Service.ConfirmCashPayment`, переводя заказ в `completed` и закрывая все ожидания. 【F:internal/taxi/lifecycle/service.go†L259-L274】

### 8. Отмены и no-show

* `POST /api/taxi/orders/{orderId}/cancel`
  * Тело: `{ "by": "passenger", "reason": "changed plans" }` → `CancelByPassenger`
  * Тело: `{ "by": "driver", "reason": "flat tire" }` → `CancelByDriver`
* `POST /api/taxi/orders/{orderId}/no-show`
  * Тело содержит телеметрию прибытия водителя для валидации радиуса и времени.

Эти маршруты фиксируют статус отмены или `no_show`, закрывая ожидания и добавляя причину в таймлайн. 【F:internal/taxi/lifecycle/service.go†L276-L347】

## Аудит и аналитика

`Order` хранит полный таймлайн статусов, журнал точек и историю сессий ожидания с длительностью и начислениями. Эти данные позволяют строить аналитику по времени подачи, платному ожиданию, паузам, no-show и структуре итогового чека. 【F:internal/taxi/lifecycle/order.go†L103-L150】【F:internal/taxi/lifecycle/order.go†L213-L281】

Unit-тест `TestLifecycleHappyPath` показывает итоговый чек: базовый тариф 1500 ₸ + 2 минуты платного ожидания (200 ₸) + 2 минуты паузы (400 ₸). 【F:internal/taxi/lifecycle/service_test.go†L10-L133】
