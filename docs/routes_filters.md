# Сводка сортировок, фильтров и данных по роутам

## GET `/search/global`
- **Тип запроса:** query-параметры.
- **Фильтры:** `types` (service, ad, work, work_ad, rent, rent_ad), `categories`, `subcategories`, `price_from` / `priceFrom`, `price_to` / `priceTo`, `ratings`, `on_site`, `negotiable`, `latitude`/`longitude`, `radius`, `page`, `limit`. Параметры приводятся к числам, допускается CSV-формат для списков; координаты обязательны в паре; флаги `on_site` и `negotiable` валидируются на yes/no/1/0.【F:docs/global_search_api.md†L9-L37】【F:docs/global_search_api.md†L42-L83】
- **Сортировка:** сначала активный «топ», затем время его активации, затем дата создания объявления; внутри выборки учитывается переданный `sort_option` для конкретных типов через сервисы репозиториев.【F:docs/global_search_api.md†L1-L7】【F:docs/global_search_api.md†L107-L142】
- **Возвращаемые данные:** пагинированный список `results` с элементами разных типов (`service`, `ad`, `work`, `work_ad`, `rent`, `rent_ad`), каждый содержит объект сущности и поле `type`, дополнительные поля (`distance`, лайки и т.п.) приходят из соответствующих репозиториев; также возвращаются `total`, `page`, `limit`.【F:docs/global_search_api.md†L85-L104】

## POST `/service/filtered/:user_id`
- **Тип запроса:** тело JSON + path-параметр `user_id`.
- **Фильтры:** `category_id`, `subcategory_id`, `price_from`, `price_to`, `avg_rating`, `sorting` (1 — по отзывам, 2 — по цене по убыванию, 3 — по возрастанию), опционально `city_id`, `latitude`, `longitude`, `radius_km`.【F:internal/models/service.go†L16-L34】
- **Сортировка:** код в поле `sorting` определяет порядок (отзывы/цена ↓/цена ↑); дополнительно сервис сортирует топ-объявления перед выдачей.【F:internal/models/service.go†L24-L27】【F:internal/repositories/top_sort.go†L1-L45】
- **Возвращаемые данные:** массив `services`, каждый элемент — расширенный сервис с данными пользователя, ценой (от/до), адресом, координатами и статусами `liked`/`is_responded`, а также вложенными изображениями/видео и расстоянием при наличии координат.【F:internal/models/service.go†L36-L77】

## POST `/work/filtered/:user_id`
- **Тип запроса:** тело JSON + path-параметр `user_id`.
- **Фильтры:** `category_id`, `subcategory_id`, `price_from`, `price_to`, `avg_rating`, `sorting` (1 — отзывы, 2 — цена ↓, 3 — цена ↑), опционально `city_id`, `latitude`, `longitude`, `radius_km`.【F:internal/models/work.go†L16-L34】
- **Сортировка:** по коду `sorting`, после чего применяется приоритет «топ»-размещений с сохранением хронологии активации/создания.【F:internal/models/work.go†L26-L29】【F:internal/repositories/top_sort.go†L34-L41】
- **Возвращаемые данные:** массив работ с пользовательскими данными, ценой (от/до), временем работы, языками/образованием, координатами, `liked` и `is_responded`, медиа и расстоянием при наличии координат.【F:internal/models/work.go†L36-L78】

## POST `/rent/filtered/:user_id`
- **Тип запроса:** тело JSON + path-параметр `user_id`.
- **Фильтры:** `category_id`, `subcategory_id`, `price_from`, `price_to`, `avg_rating`, `sorting` (1 — отзывы, 2 — цена ↓, 3 — цена ↑), опционально `city_id`, `latitude`, `longitude`, `radius_km`.【F:internal/models/rent.go†L12-L29】
- **Сортировка:** по коду `sorting` с последующим приоритетом записей с активным «топом».【F:internal/models/rent.go†L24-L27】【F:internal/repositories/top_sort.go†L1-L45】
- **Возвращаемые данные:** список объявлений об аренде с данными владельца, ценой (от/до), временем работы, координатами, признаками `negotiable`/`hide_phone`, состоянием `liked`/`is_responded`, медиа и расстоянием при наличии координат.【F:internal/models/rent.go†L31-L73】

## POST `/ad/filtered/:user_id`
- **Тип запроса:** тело JSON + path-параметр `user_id`.
- **Фильтры:** `category_id`, `subcategory_id`, `price_from`, `price_to`, `avg_rating`, `sorting` (1 — отзывы, 2 — цена ↓, 3 — цена ↑), опционально `city_id`, `latitude`, `longitude`, `radius_km`.【F:internal/models/ad.go†L10-L27】
- **Сортировка:** по коду `sorting`, затем приоритет топовых объявлений по времени активации/создания.【F:internal/models/ad.go†L21-L24】【F:internal/repositories/top_sort.go†L1-L45】
- **Возвращаемые данные:** массив объявлений с данными пользователя, ценой (включая `ad_price_to`), адресом, признаками `on_site`, `negotiable`, `hide_phone`, флагами `liked`/`is_responded`, медиа и расстоянием при наличии координат.【F:internal/models/ad.go†L29-L68】

## POST `/work_ad/filtered/:user_id`
- **Тип запроса:** тело JSON + path-параметр `user_id`.
- **Фильтры:** `category_id`, `subcategory_id`, `price_from`, `price_to`, `avg_rating`, `sorting` (1 — отзывы, 2 — цена ↓, 3 — цена ↑), опционально `city_id`, `latitude`, `longitude`, `radius_km`.【F:internal/models/work_ad.go†L25-L43】
- **Сортировка:** по коду `sorting`, далее приоритет активного «топа» с учётом времени активации/создания объявления.【F:internal/models/work_ad.go†L35-L38】【F:internal/repositories/top_sort.go†L42-L45】
- **Возвращаемые данные:** список work_ad с контактной и личной информацией, ценой (от/до), графиком, языками/образованием, координатами, признаками `negotiable`/`hide_phone`, флагами `liked`/`is_responded`, медиа и расстоянием при наличии координат.【F:internal/models/work_ad.go†L45-L81】

## POST `/rent_ad/filtered/:user_id`
- **Тип запроса:** тело JSON + path-параметр `user_id`.
- **Фильтры:** `category_id`, `subcategory_id`, `price_from`, `price_to`, `avg_rating`, `sorting` (1 — отзывы, 2 — цена ↓, 3 — цена ↑), опционально `city_id`, `latitude`, `longitude`, `radius_km`.【F:internal/models/rent_ad.go†L12-L29】
- **Сортировка:** по коду `sorting` с приоритетом активных «топ»-размещений при выдаче.【F:internal/models/rent_ad.go†L24-L27】【F:internal/repositories/top_sort.go†L1-L45】
- **Возвращаемые данные:** массив rent_ad с данными пользователя, ценой (от/до), графиком, координатами, признаками `negotiable`/`hide_phone`, состояниями `liked`/`is_responded`, медиа и расстоянием при наличии координат.【F:internal/models/rent_ad.go†L31-L70】
