# Фильтрация и сортировка объявлений для фронтенда

Эта инструкция покрывает все POST-маршруты фильтрации/сортировки и их варианты с учетом лайков пользователя:

- `/service/filtered` и `/service/filtered/:user_id`
- `/work/filtered` и `/work/filtered/:user_id`
- `/rent/filtered` и `/rent/filtered/:user_id`
- `/ad/filtered` и `/ad/filtered/:user_id`
- `/work_ad/filtered` и `/work_ad/filtered/:user_id`
- `/rent_ad/filtered` и `/rent_ad/filtered/:user_id`

Маршруты без `:user_id` доступны без авторизации. Варианты с `:user_id` оборачиваются `authMiddleware`, поэтому требуют заголовок `Authorization: Bearer <jwt>` и корректный path-параметр `user_id`. Все хендлеры ожидают JSON в теле запроса и возвращают список сущностей в объекте-обертке (`{"services": [...]}`, `{"works": [...]}` и т.п.).【F:internal/handlers/service_handler.go†L271-L305】【F:cmd/routes.go†L223-L241】

## Общие правила
- **Метод:** `POST`
- **Тело:** JSON с полями, описанными ниже для каждой сущности
- **Авторизация:**
  - Без `:user_id` — не требуется, но если передан Bearer-токен, хендлер подставит `city_id` из токена, если он не указан во входе (актуально для всех сущностей).【F:internal/handlers/service_handler.go†L278-L291】
  - С `:user_id` — требуется валидный JWT и числовой `user_id` в пути; тело аналогично базовому варианту. Используется для возврата лайкнутых/отмеченных объектов в поле `liked`.【F:internal/handlers/service_handler.go†L889-L929】
- **Сортировка:** поле `sorting` принимает коды: `1` — по отзывам (рейтингу), `2` — по цене по убыванию, `3` — по цене по возрастанию. Если не задано, используется порядок по умолчанию из репозитория. Это правило едино для всех фильтров ниже.【F:internal/models/service.go†L77-L87】【F:internal/models/work.go†L78-L99】【F:internal/models/rent.go†L71-L87】【F:internal/models/ad.go†L70-L85】【F:internal/models/work_ad.go†L83-L105】【F:internal/models/rent_ad.go†L71-L87】
- **Геофильтр:** поля `latitude`, `longitude`, `radius_km` опциональны. При передаче координат репозитории проставляют `distance` в ответах (при наличии координат у объектов).【F:internal/models/service.go†L90-L118】【F:internal/models/work.go†L97-L129】【F:internal/models/rent.go†L85-L112】【F:internal/models/ad.go†L82-L110】【F:internal/models/work_ad.go†L101-L130】【F:internal/models/rent_ad.go†L84-L112】

## Сервисы `/service/filtered`
### Тело запроса (`FilterServicesRequest`)
| Поле | Тип | Обяз. | Описание |
| --- | --- | --- | --- |
| `category_id` | `[]int` | нет | ID категорий. |
| `subcategory_id` | `[]int` | нет | ID подкатегорий. |
| `price_from` / `price_to` | `float64` | нет | Диапазон цены. |
| `avg_rating` | `[]int` | нет | Допустимые значения рейтинга. |
| `negotiable` | `bool` | нет | Договорная цена. |
| `on_site` | `bool` | нет | Выезд на объект. |
| `open_now` | `bool` | нет | Только открытые сейчас. |
| `twenty_four_seven` | `bool` | нет | Круглосуточно. |
| `sorting` | `int` | нет | Код сортировки (см. выше). |
| `city_id` | `int` | нет | При отсутствии берётся из токена. |
| `latitude`, `longitude`, `radius_km` | `float64` | нет | Геофильтр, расстояние в км. |

### Ответ (`services`)
Массив объектов `FilteredService` с ценой, адресом, расписанием, медиа и флагами `liked`/`is_responded` для персонализации.【F:internal/models/service.go†L95-L123】

## Работа `/work/filtered`
### Тело запроса (`FilterWorkRequest`)
| Поле | Тип | Обяз. | Описание |
| --- | --- | --- | --- |
| `category_id`, `subcategory_id` | `[]int` | нет | Категории. |
| `price_from` / `price_to` | `float64` | нет | Диапазон ставки. |
| `avg_rating` | `[]int` | нет | Рейтинги. |
| `negotiable_only` | `bool` | нет | Только договорная оплата. |
| `twenty_four_seven`, `nearby`, `open_now` | `bool` | нет | Режим работы и близость. |
| `work_experience`, `schedule`, `payment_period`, `distance_work`, `languages`, `education` | `[]string` | нет | Атрибуты вакансии/резюме. |
| `sorting` | `int` | нет | Код сортировки. |
| `city_id`, `latitude`, `longitude`, `radius_km` | `int/float64` | нет | Геофильтр и город. |

### Ответ (`works`)
Массив `FilteredWork` с полями опыта, расписания, способа работы, языков, образования, координат и флагов `liked`/`is_responded`.【F:internal/models/work.go†L102-L134】

## Аренда `/rent/filtered`
### Тело запроса (`FilterRentRequest`)
| Поле | Тип | Обяз. | Описание |
| --- | --- | --- | --- |
| `category_id`, `subcategory_id` | `[]int` | нет | Категории. |
| `price_from` / `price_to` | `float64` | нет | Диапазон аренды. |
| `avg_rating` | `[]int` | нет | Рейтинги. |
| `negotiable` | `bool` | нет | Договорная цена. |
| `rent_type`, `deposit` | `[]string` | нет | Тип аренды и депозит. |
| `open_now`, `twenty_four_seven` | `bool` | нет | Режим работы. |
| `sorting` | `int` | нет | Код сортировки. |
| `city_id`, `latitude`, `longitude`, `radius_km` | `int/float64` | нет | Геофильтр и город. |

### Ответ (`rents`)
Массив `FilteredRent` с ценой, адресом, медиа, координатами и флагами `liked`/`is_responded`.【F:internal/models/rent.go†L90-L117】

## Объявления `/ad/filtered`
### Тело запроса (`FilterAdRequest`)
Поля идентичны фильтрам сервиса, включая `negotiable`, `on_site`, временные флаги и геофильтр.【F:internal/models/ad.go†L70-L110】

### Ответ (`ads`)
Массив `FilteredAd` с данными пользователя, ценой, адресом, расписанием, медиа и полями персонализации `liked`/`is_responded`.【F:internal/models/ad.go†L88-L115】

## Поиск сотрудников `/work_ad/filtered`
### Тело запроса (`FilterWorkAdRequest`)
Повторяет поля `FilterWorkRequest`, включая опыт, расписание, период оплаты, дистанционный формат, языки/образование и геофильтр.【F:internal/models/work_ad.go†L83-L104】

### Ответ (`works_ad`)
Массив `FilteredWorkAd` с деталями объявлений, персональной информацией, медиа, координатами и флагами `liked`/`is_responded`.【F:internal/models/work_ad.go†L107-L143】

## Запросы на аренду `/rent_ad/filtered`
### Тело запроса (`FilterRentAdRequest`)
Поля совпадают с `FilterRentRequest`: категории, ценовой диапазон, `negotiable`, тип аренды, депозит, временные флаги и геофильтр.【F:internal/models/rent_ad.go†L71-L87】

### Ответ (`rents_ad`)
Массив `FilteredRentAd` с ценой, адресом, медиа, координатами и персонализацией лайков/откликов.【F:internal/models/rent_ad.go†L90-L117】

## Использование вариантов `/:user_id`
- Path-параметр `:user_id` обязателен и должен быть числом. Ошибка парсинга приводит к `400 Bad Request` еще до обращения к сервису.【F:internal/handlers/service_handler.go†L889-L895】
- Авторизация через `authMiddleware` обязательна. В ответах поле `liked` будет корректно заполнено с учётом пользователя, а гео-информация (`city_id`) также может подтягиваться из токена при его наличии.【F:cmd/routes.go†L233-L241】【F:internal/handlers/service_handler.go†L902-L916】
- Тело запроса полностью совпадает с базовым вариантом без `:user_id`, поэтому фронту можно переиспользовать один и тот же DTO и просто добавлять path-параметр при сценариях, завязанных на лайки/отклики.
