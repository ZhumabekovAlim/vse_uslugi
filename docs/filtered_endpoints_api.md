# Фильтрованные выборки объявлений (`/filtered` и `/filtered/:user_id`)

Документ описывает все эндпоинты вида `/filtered` и `/filtered/:user_id` для сущностей `service`, `ad`, `work`, `work_ad`, `rent` и `rent_ad`. Каждый маршрут доступен в двух вариантах:

* `POST /<entity>/filtered` — открытая выборка. Токен не требуется, но если в заголовке передан `Authorization: Bearer <jwt>`, то из него подставляется `city_id`, если он не указан в теле запроса.【F:internal/handlers/service_handler.go†L271-L305】【F:internal/handlers/ad_handler.go†L232-L260】
* `POST /<entity>/filtered/:user_id` — персонализированная выборка с учётом лайков/откликов. Требуется авторизация, путь включает идентификатор пользователя, а токен дополнительно может подставить `city_id`, если поле пустое.【F:cmd/routes.go†L233-L241】【F:internal/handlers/service_handler.go†L889-L924】

Все маршруты принимают тело JSON и возвращают объект с единственным ключом (`services`, `ads`, `works`, `works_ad`, `rents`, `rents_ad`), внутри которого находится массив записей соответствующего типа.

## Общие правила для всех `/filtered`

* **Диапазон цены** применяется только если переданы оба значения `price_from` и `price_to` больше нуля; в противном случае условие не добавляется.【F:internal/repositories/service_repository.go†L544-L547】【F:internal/repositories/ad_repository.go†L545-L548】【F:internal/repositories/work_repository.go†L543-L547】
* **Категории и подкатегории** фильтруются по CSV-массивам идентификаторов (`category_id`, `subcategory_id`). Пустые массивы не добавляют условий к запросу.【F:internal/repositories/service_repository.go†L549-L566】【F:internal/repositories/work_ad_repository.go†L541-L558】
* **Рейтинги** интерпретируются как минимальный порог: берётся наименьшее значение из массива `avg_rating` и добавляется условие `avg_rating >= <min>`.【F:internal/repositories/service_repository.go†L569-L573】【F:internal/repositories/work_repository.go†L569-L574】
* **География**: если переданы `latitude` и `longitude`, репозитории рассчитывают `distance` по формуле гаверсина. При наличии `radius_km` записи без координат или выходящие за пределы радиуса исключаются. Для `work` и `work_ad` флаг `nearby=true` без `radius_km` подставляет радиус по умолчанию 5 км при наличии координат пользователя.【F:internal/repositories/distance.go†L1-L34】【F:internal/repositories/service_repository.go†L622-L624】【F:internal/repositories/work_repository.go†L637-L640】【F:internal/repositories/work_ad_repository.go†L632-L636】
* **Сортировка** (поле `sorting`) общая для всех сущностей: `1` — по дате создания по возрастанию, `2` — по цене по убыванию, `3` — по цене по возрастанию, значение по умолчанию — дата создания по убыванию.【F:internal/repositories/service_repository.go†L576-L586】【F:internal/repositories/ad_repository.go†L642-L650】【F:internal/repositories/rent_repository.go†L637-L645】【F:internal/repositories/work_repository.go†L625-L634】【F:internal/repositories/work_ad_repository.go†L620-L629】【F:internal/repositories/rent_ad_repository.go†L631-L639】
* **«Топ»-поднятие**: после выборки массивы проходят через `liftListingsTopOnly`, который поднимает активные `top`-размещения выше остальных без изменения исходного порядка внутри групп. Это касается обеих версий эндпоинтов.【F:internal/repositories/top_sort.go†L87-L106】【F:internal/repositories/service_repository.go†L638-L640】【F:internal/repositories/work_ad_repository.go†L699-L701】

### Отличия `/filtered/:user_id`

* Эндпоинты с `:user_id` добавляют левое соединение с таблицами избранного/откликов и возвращают достоверные флаги `liked` и `is_responded` (в публичных `/filtered` они всегда ложные из-за отсутствия пользователя).【F:internal/repositories/service_repository.go†L705-L744】【F:internal/repositories/ad_repository.go†L841-L882】【F:internal/repositories/work_repository.go†L914-L958】【F:internal/repositories/work_ad_repository.go†L908-L953】【F:internal/repositories/rent_repository.go†L886-L932】【F:internal/repositories/rent_ad_repository.go†L867-L914】
* Эти маршруты требуют JWT-токен (см. `authMiddleware` в `cmd/routes.go`). Тело запроса совпадает с публичной версией.

## Поля запроса и ответа по сущностям

### Service (`/service/filtered`)

**Запрос.** Использует модель `FilterServicesRequest` с полями категорий/подкатегорий, ценового коридора, минимального рейтинга, булевых фильтров `negotiable`/`on_site`, флагов `open_now` и `twenty_four_seven`, сортировки и геопараметров (`city_id`, `latitude`, `longitude`, `radius_km`).【F:internal/models/service.go†L77-L93】 Значение `city_id` можно опустить — оно подставится из токена, если передан заголовок `Authorization`.【F:internal/handlers/service_handler.go†L278-L290】

**Ответ.** Ключ `services` содержит массив `FilteredService` с данными пользователя, ценой (`service_price`, `service_price_to`), признаками `on_site`, `negotiable`, `hide_phone`, временем работы, медиа, координатами и флагами `liked`/`is_responded`; при передаче координат добавляется `distance`.【F:internal/models/service.go†L95-L123】

**Пример запроса:**

```http
POST /service/filtered/:user_id
Authorization: Bearer <token>
Content-Type: application/json

{
  "category_id": [10, 11],
  "price_from": 500,
  "price_to": 50000,
  "avg_rating": [4],
  "negotiable": false,
  "on_site": true,
  "sorting": 2,
  "latitude": 55.75,
  "longitude": 37.62,
  "radius_km": 10
}
```

### Ad (`/ad/filtered`)

**Запрос.** Поля аналогичны `service`, дополнительно включают работу по времени (`open_now`, `twenty_four_seven`) и `on_site`. Используется модель `FilterAdRequest`.【F:internal/models/ad.go†L72-L88】

**Ответ.** Ключ `ads` содержит массив `FilteredAd` с пользовательскими данными, адресом, `on_site`, `negotiable`, `hide_phone`, опциональными `order_date`/`order_time`, геодистанцией и флагами `liked`/`is_responded`.【F:internal/models/ad.go†L90-L119】

### Work (`/work/filtered`)

**Запрос.** Помимо базовых полей цен и рейтингов, модель `FilterWorkRequest` добавляет фильтры `work_experience`, `schedule`, `payment_period`, `distance_work` (онлайн/офлайн), массивы `languages` и `education`, а также флаги `negotiable_only`, `nearby`, `open_now`, `twenty_four_seven`. При `nearby=true` и наличии координат радиус по умолчанию 5 км, если `radius_km` не задан.【F:internal/models/work.go†L78-L100】【F:internal/repositories/work_repository.go†L637-L640】

**Ответ.** Ключ `works` содержит `FilteredWork` с ценой (`work_price`, `work_price_to`), описанием, опытом, графиком, форматом работы, языками/образованием, временем работы, координатами, `distance` и флагами `liked`/`is_responded`. Телефон скрывается, если `hide_phone=true`.【F:internal/models/work.go†L102-L135】

### WorkAd (`/work_ad/filtered`)

**Запрос.** Идентичен `work`, но относится к откликам работодателей. Используется `FilterWorkAdRequest` с теми же наборами фильтров и геопараметров, включая `nearby`.【F:internal/models/work_ad.go†L83-L105】【F:internal/repositories/work_ad_repository.go†L632-L636】

**Ответ.** Ключ `works_ad` содержит `FilteredWorkAd`: данные пользователя, контакты кандидата (`first_name`, `last_name`, `birth_date`, `contact_number`), параметры работы, медиа, координаты, `distance` и флаги `liked`/`is_responded`.【F:internal/models/work_ad.go†L107-L144】

### Rent (`/rent/filtered`)

**Запрос.** Модель `FilterRentRequest` добавляет фильтры `rent_type` и `deposit` к ценам, рейтингам и `negotiable`, а также флаги `open_now` и `twenty_four_seven`. Геополя и `city_id` работают так же, как в других сущностях.【F:internal/models/rent.go†L71-L88】

**Ответ.** Ключ `rents` содержит `FilteredRent` с ценой (`rent_price`, `rent_price_to`), признаками `negotiable`/`hide_phone`, временем работы, медиа, координатами, `distance` и флагами `liked`/`is_responded`.【F:internal/models/rent.go†L90-L116】

### RentAd (`/rent_ad/filtered`)

**Запрос.** Аналогичен `rent`, использует `FilterRentAdRequest` с теми же полями фильтрации и геопараметрами. 【F:internal/models/rent_ad.go†L73-L90】

**Ответ.** Ключ `rents_ad` содержит `FilteredRentAd` с параметрами аренды, условиями `deposit`, опциональными `order_date`/`order_time`, координатами, `distance` и персональными флагами `liked`/`is_responded`.【F:internal/models/rent_ad.go†L92-L120】

## Что получает фронт в результате

* Пагинация отсутствует в теле ответа — эндпоинты возвращают полный набор, отсортированный по `sorting` и поднятию «топов».
* Поля `liked` и `is_responded` информативны только в вариантах `/:user_id`; для публичных маршрутов они остаются `false` по умолчанию.
* При переданных координатах каждая запись получает `distance` (км) для отображения ближайших предложений; записи без координат не фильтруются, если `radius_km` не задан.
* Время работы (`work_time_from`/`work_time_to`) используется фильтрами `open_now` и `twenty_four_seven`, поэтому фронт может отображать статус доступности на основании тех же полей.
