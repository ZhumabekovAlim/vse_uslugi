# Глобальный поиск `/search/global`

Полностью переописанная документация для фронтенда по маршруту `POST /search/global`: тело запроса, параметры, сортировки, геофильтры, типоспецифические фильтры и примеры. Эндпоинт зарегистрирован как `mux.Post("/search/global", standardMiddleware.ThenFunc(app.globalSearchHandler.Search))`.【F:cmd/routes.go†L214-L214】

## Авторизация и заголовки

* JWT не обязателен. Если передать `Authorization: Bearer <jwt>`, из токена извлекается `user_id`; он прокидывается во все репозитории и влияет на персональные поля (лайк/избранное, отклики). При ошибке парсинга или отсутствии заголовка `user_id=0`.【F:internal/handlers/global_search_handler.go†L28-L131】【F:internal/handlers/global_search_handler.go†L129-L165】
* Ответ всегда JSON (`application/json; charset=utf-8`).【F:internal/handlers/global_search_handler.go†L157-L165】

## Базовые поля тела (JSON)

| Параметр | Тип/допустимые значения | Обязателен | Значение по умолчанию | Применение |
| --- | --- | --- | --- | --- |
| `types` | Массив строк из `service`, `ad`, `work`, `work_ad`, `rent`, `rent_ad`; дубликаты удаляются, максимум 6 | да | — | Определяет, какие домены участвуют в поиске. Неверное значение → 400, отсутствие → 400.【F:internal/handlers/global_search_handler.go†L39-L132】 |
| `categories` | Массив целых | нет | — | Общий фильтр категорий для всех выбранных типов.【F:internal/handlers/global_search_handler.go†L86-L88】 |
| `subcategories` | Массив целых | нет | — | Общий фильтр подкатегорий; дальше преобразуется в строки для репозиториев.【F:internal/handlers/global_search_handler.go†L86-L88】【F:internal/services/global_search_service.go†L53-L62】 |
| `limit` | `>0` | нет | `20` | Размер страницы результатов.【F:internal/handlers/global_search_handler.go†L90-L92】【F:internal/services/global_search_service.go†L36-L57】 |
| `page` | `>0` | нет | `1` | Номер страницы (применяется после объединения и сортировки).【F:internal/handlers/global_search_handler.go†L90-L92】【F:internal/services/global_search_service.go†L36-L57】 |
| `price_from` / `priceFrom` | число | нет | `0` | Нижняя граница цены. Оба написания равнозначны.【F:internal/handlers/global_search_handler.go†L93-L95】 |
| `price_to` / `priceTo` | число | нет | `0` | Верхняя граница цены.【F:internal/handlers/global_search_handler.go†L93-L95】 |
| `ratings` | Массив чисел | нет | — | Набор допустимых рейтингов (используется как множество значений).【F:internal/handlers/global_search_handler.go†L95-L97】【F:internal/services/global_search_service.go†L63-L70】 |
| `sort_option` / `sortOption` | целое ≥0 | нет | `0` | Код сортировки (см. раздел «Сортировка»). При пустом/некорректном значении → `0`.【F:internal/handlers/global_search_handler.go†L95-L100】【F:internal/services/global_search_service.go†L193-L275】 |
| `latitude`, `longitude` | числа | нет | — | Геоточка пользователя. Должны идти парой, иначе 400. При наличии включают расчёт `distance`.【F:internal/handlers/global_search_handler.go†L114-L122】【F:internal/services/global_search_service.go†L46-L120】【F:internal/services/global_search_service.go†L358-L374】 |
| `radius` | число > 0 | нет | — | Км-радиус вокруг точки пользователя. Работает только вместе с координатами: записи без координат или за пределами радиуса исключаются; дополнительно сортировка переключается на дистанцию.【F:internal/handlers/global_search_handler.go†L124-L130】【F:internal/services/global_search_service.go†L46-L194】 |
| `on_site` | `true/false` | нет | — | Фильтр «выезд на объект», применяется только к `service` и `ad`. Ошибочное значение → 400.【F:internal/handlers/global_search_handler.go†L102-L109】【F:internal/services/global_search_service.go†L70-L106】 |
| `negotiable` | `true/false` | нет | — | Фильтр «договорная цена» для всех типов. Ошибочное значение → 400.【F:internal/handlers/global_search_handler.go†L102-L109】【F:internal/services/global_search_service.go†L70-L183】 |
| `remote` | `true/false` | нет | — | Применяется к `work`/`work_ad`: `true` — только удалёнка, `false` — только не-удалённые записи; пусто — без фильтра.【F:internal/handlers/global_search_handler.go†L101-L103】【F:internal/services/global_search_service.go†L109-L145】【F:internal/repositories/work_repository.go†L310-L336】 |

## Типоспецифические фильтры

| Типы | Параметры | Описание |
| --- | --- | --- |
| `service`, `ad` | `on_site`, `negotiable` | Фильтры по выезду и договорной цене передаются в соответствующие репозитории услуг/объявлений.【F:internal/services/global_search_service.go†L70-L107】 |
| `ad`, `rent_ad` | `order_date`, `order_time` | Строковые фильтры по слоту записи/показа. При передаче добавляют условия равенства к полям `order_date`/`order_time`; пустые значения игнорируются.【F:internal/handlers/global_search_handler.go†L82-L117】【F:internal/services/global_search_service.go†L70-L187】【F:internal/repositories/ad_repository.go†L295-L377】【F:internal/repositories/rent_ad_repository.go†L275-L364】 |
| `rent`, `rent_ad` | `rent_types`, `deposits` | Массив строк. Ограничивают тип аренды и условия залога.【F:internal/handlers/global_search_handler.go†L82-L84】【F:internal/services/global_search_service.go†L146-L183】 |
| `work`, `work_ad` | `work_experience`, `work_schedule`/`work_schedules`, `payment_period`/`payment_periods`, `remote`, `languages`, `education`/`educations` | Массивы строк опыта, графиков, периодов оплаты, форматов (дистанционная работа), языков и образований. Все передаются в репозитории вакансий/кандидатов.【F:internal/handlers/global_search_handler.go†L84-L98】【F:internal/services/global_search_service.go†L108-L145】 |

## Сортировка

Сначала применяется приоритет активного «топ»-продвижения; затем учитывается `sortOption`; затем — дата/время активации топа и создания записи.

| `sortOption` | Правило (после учёта «топа») |
| --- | --- |
| `0` или пусто | Новые раньше старых (дата создания по убыванию).【F:internal/services/global_search_service.go†L252-L275】 |
| `1` | Старые раньше новых (дата создания по возрастанию).【F:internal/services/global_search_service.go†L252-L258】 |
| `2` | Цена по возрастанию (дешевле → дороже). Записи без цены идут после тех, у кого цена есть.【F:internal/services/global_search_service.go†L258-L275】 |
| `3` | Цена по убыванию (дороже → дешевле).【F:internal/services/global_search_service.go†L265-L275】 |

Если заданы `latitude`, `longitude` и `radius`, поверх таблицы выше сначала сортируется по возрастанию `distance`; внутри одинаковой дистанции действует приоритет «топа», затем правило `sortOption`. Записи без расстояния (нет координат у объявления) всегда ниже записей с расстоянием при активном geo-sort.【F:internal/services/global_search_service.go†L193-L249】

## Геологика и пагинация

* Координаты обязательны парой; одиночное значение приводит к 400 (`both latitude and longitude must be provided`).【F:internal/handlers/global_search_handler.go†L117-L126】
* `radius` отфильтровывает объявления без координат и все записи дальше указанного радиуса; расстояние считается по гаверсину в километрах.【F:internal/handlers/global_search_handler.go†L123-L125】【F:internal/services/global_search_service.go†L146-L183】【F:internal/services/global_search_service.go†L358-L388】
* Для равномерной выдачи из каждого типа берётся `perTypeLimit = limit * page`, затем все записи объединяются, сортируются и только потом режутся пагинацией по `page/limit`. Это гарантирует полноценную выборку на любой странице.【F:internal/services/global_search_service.go†L36-L212】

## Формат ответа

```json
{
  "results": [
    { "type": "service", "distance": 1.2, "service": { /* поля Service */ } },
    { "type": "ad", "ad": { /* поля Ad */ } }
  ],
  "total": 42,
  "page": 1,
  "limit": 20
}
```

* `results` — смешанный список `GlobalSearchItem`; заполнен ровно один объект сущности (`service`/`ad`/`work`/`work_ad`/`rent`/`rent_ad`) плюс `type`. `distance` присутствует только при передаче координат и наличии координат у записи.【F:internal/models/global_search.go†L17-L40】【F:internal/services/global_search_service.go†L82-L183】
* `total` — число записей после объединения, сортировки и применения радиуса; `page`/`limit` возвращаются как принятые сервисом.【F:internal/services/global_search_service.go†L189-L213】
* При пустом результате `results=[]`, `total=0`, `page` и `limit` сохраняются.【F:internal/services/global_search_service.go†L189-L213】

### Какие поля приходят внутри сущностей

Сущности возвращаются в тех же моделях, что и в профильных списках: с медиаданными, ценами, координатами, признаками `liked`/`is_responded`, топ-информацией и т.п., т.к. сервис просто оборачивает записи из репозиториев без модификации.【F:internal/services/global_search_service.go†L70-L183】【F:internal/models/global_search.go†L17-L34】

## Примеры запросов

### Базовый поиск по нескольким типам

```
POST /search/global HTTP/1.1
Authorization: Bearer <token>
Content-Type: application/json

{
  "types": ["service", "ad", "rent"],
  "categories": [10, 11],
  "limit": 10,
  "page": 1,
  "price_from": 0,
  "price_to": 50000,
  "ratings": [4, 5]
}
```

### Геопоиск с сортировкой по дистанции

```
POST /search/global HTTP/1.1
Content-Type: application/json

{
  "types": ["service", "ad"],
  "limit": 15,
  "page": 1,
  "latitude": 55.7522,
  "longitude": 37.6156,
  "radius": 10
}
```

* Показывает только объявления с координатами в 10 км от точки; ближайшие — первыми, потом «топ», потом правило `sortOption` (по умолчанию — новые выше).

### Пример с фильтрами работы/кадров

```
POST /search/global HTTP/1.1
Content-Type: application/json

{
  "types": ["work", "work_ad"],
  "categories": [30],
  "limit": 20,
  "work_experience": ["middle", "senior"],
  "work_schedules": ["full_time", "part_time"],
  "payment_periods": ["month", "week"],
  "remote": true,
  "languages": ["ru", "en"],
  "educations": ["bachelor", "master"],
  "sort_option": 2
}
```

* Фильтрует по опыту, графикам, периодам выплат, дистанционному формату, языкам и образованию; сортирует по цене возрастанию, внутри приоритета «топ».

## Ошибки

| Условие | Код | Сообщение |
| --- | --- | --- |
| Нет `types` | 400 | `types parameter is required`【F:internal/handlers/global_search_handler.go†L28-L31】 |
| Недопустимый тип в `types` | 400 | `unsupported listing type: <type>`【F:internal/handlers/global_search_handler.go†L34-L45】 |
| Все типы пустые/невалидные | 400 | `at least one valid type must be provided`【F:internal/handlers/global_search_handler.go†L54-L56】 |
| Типов больше 6 | 400 | `no more than 6 listing types allowed`【F:internal/handlers/global_search_handler.go†L57-L59】 |
| Неверный `remote`/`on_site`/`negotiable` | 400 | `invalid <param> value`【F:internal/handlers/global_search_handler.go†L99-L115】 |
| Одни координаты без пары | 400 | `both latitude and longitude must be provided`【F:internal/handlers/global_search_handler.go†L117-L126】 |
| Радиус ≤ 0 или некорректный | 400 | `radius must be greater than zero` / `invalid radius`【F:internal/handlers/global_search_handler.go†L123-L125】 |
| Хэндлер не сконфигурирован | 500 | `service unavailable` (отсутствует сервис в хэндлере)【F:internal/handlers/global_search_handler.go†L22-L25】 |
| Неподдерживаемый тип (отсутствует репозиторий) | 500 | `unsupported listing type: <type>` (ошибка сервиса)【F:internal/services/global_search_service.go†L70-L186】 |

## Поток обработки

1. Хэндлер валидирует `types`, числовые/булевые параметры, координаты и собирает `GlobalSearchRequest`.【F:internal/handlers/global_search_handler.go†L28-L155】
2. Сервис нормализует `limit/page`, считает `perTypeLimit = limit * page`, вызывает профильные репозитории по каждому типу, оборачивает результаты в `GlobalSearchItem` и добавляет `distance`, если заданы координаты. Репозитории используют `user_id` из токена для расчёта персональных флагов.【F:internal/services/global_search_service.go†L36-L183】
3. Все записи объединяются, сортируются с учётом дистанции/«топа»/`sortOption`, после чего режутся по `page/limit`; в ответ возвращаются `results`, `total`, `page`, `limit`.【F:internal/services/global_search_service.go†L193-L213】
