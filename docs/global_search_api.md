# Глобальный поиск объявлений `/search/global`

Маршрут `GET /search/global` агрегирует результаты из всех доменных сущностей платформы (услуги, объявления, работа, аренда и т.д.) и возвращает единый список, отсортированный по активности «топ»-продвижения и дате создания. Если в запросе одновременно переданы `latitude`, `longitude` и `radius`, результаты сортируются по возрастанию дистанции до пользователя (чем ближе — тем выше в списке) с последующим упорядочиванием по «топу» и дате при равном расстоянии. Маршрут зарегистрирован в `cmd/routes.go` и оборачивается стандартным middleware-стеком: `mux.Get("/search/global", standardMiddleware.ThenFunc(app.globalSearchHandler.Search))`.【F:cmd/routes.go†L197-L197】【F:internal/services/global_search_service.go†L141-L160】

## Требования к авторизации и заголовкам

* Заголовок `Authorization: Bearer <jwt>` необязателен, но если передан валидный токен, из него извлекается `user_id`, который попадёт в запрос к сервису. Это влияет на персонализированные выборки репозиториев (например, отметки избранного). При отсутствии токена или в случае ошибки разбора идентификатор равен 0.【F:internal/handlers/global_search_handler.go†L80-L101】【F:internal/handlers/global_search_handler.go†L177-L191】
* Дополнительные заголовки не используются. Ответ всегда возвращается в формате `application/json` и кодировке UTF-8.【F:internal/handlers/global_search_handler.go†L101-L103】

## Параметры запроса

| Параметр | Тип | Обязательность | Значение по умолчанию | Описание |
| --- | --- | --- | --- | --- |
| `types` | строка (CSV) | да | — | Список типов объявлений (через запятую). Допустимые значения: `service`, `ad`, `work`, `work_ad`, `rent`, `rent_ad`. Пробелы удаляются, дубликаты игнорируются. Минимум 1, максимум 6 типов.【F:internal/handlers/global_search_handler.go†L26-L59】【F:internal/models/top.go†L14-L44】【F:internal/models/top.go†L108-L127】|
| `categories` | строка (CSV целых чисел) | нет | `null` | Набор идентификаторов категорий. Неверные или пустые значения отбрасываются. Если после фильтрации список пуст, параметр не передаётся в сервис.【F:internal/handlers/global_search_handler.go†L61-L63】【F:internal/handlers/global_search_handler.go†L105-L124】|
| `subcategories` | строка (CSV целых чисел) | нет | `null` | Аналогично `categories`, но для подкатегорий. В сервисе преобразуются к строковым идентификаторам для репозиториев.【F:internal/handlers/global_search_handler.go†L61-L63】【F:internal/services/global_search_service.go†L50-L54】|
| `limit` | положительное целое | нет | `20` | Размер страницы. Отрицательные/нулевые значения заменяются на 20.【F:internal/handlers/global_search_handler.go†L64-L65】【F:internal/handlers/global_search_handler.go†L147-L155】【F:internal/services/global_search_service.go†L35-L38】|
| `page` | положительное целое | нет | `1` | Номер страницы. Минимум 1.【F:internal/handlers/global_search_handler.go†L64-L66】【F:internal/handlers/global_search_handler.go†L147-L155】【F:internal/services/global_search_service.go†L40-L44】|
| `priceFrom`, `price_from` | число с плавающей точкой | нет | `0` | Нижняя граница цены. Поддерживаются оба написания (camelCase и snake_case). Если значение не парсится, используется 0.【F:internal/handlers/global_search_handler.go†L66-L73】【F:internal/handlers/global_search_handler.go†L167-L175】|
| `priceTo`, `price_to` | число с плавающей точкой | нет | `0` | Верхняя граница цены. Аналогично `priceFrom`.|
| `ratings` | строка (CSV чисел) | нет | `null` | Список минимальных рейтингов. Непарсибельные элементы пропускаются, пустой результат трактуется как отсутствие фильтра.【F:internal/handlers/global_search_handler.go†L74-L75】【F:internal/handlers/global_search_handler.go†L126-L145】|
| `sortOption`, `sort_option` | неотрицательное целое | нет | `0` | Код сортировки, передаваемый напрямую в репозитории. Неверные значения приводят к 0.【F:internal/handlers/global_search_handler.go†L74-L78】【F:internal/handlers/global_search_handler.go†L157-L165】|
| `on_site` | `"yes"/"no"`, `"да"/"нет"`, `true/false/1/0` | нет | `null` | Фильтр только для типов `service` и `ad`. Принимает значения «да/yes» или «нет/no» в любом регистре; при пустом значении фильтр не применяется. Неверное значение приводит к 400.【F:internal/handlers/global_search_handler.go†L79-L94】【F:internal/models/global_search.go†L5-L14】【F:internal/services/global_search_service.go†L62-L90】|
| `negotiable` | `"yes"/"no"`, `"да"/"нет"`, `true/false/1/0` | нет | `null` | Фильтр по признаку «договорная цена». Применяется ко всем типам, но для `service` и `ad` идёт вместе с `on_site`. Неверные значения приводят к 400.【F:internal/handlers/global_search_handler.go†L79-L94】【F:internal/models/global_search.go†L5-L14】【F:internal/services/global_search_service.go†L62-L136】|
| `latitude` и `longitude` | числа с плавающей точкой | нет | — | Текущее местоположение пользователя. Если указаны оба значения, в ответе появится `distance`, вычисленное по формуле гаверсина. При передаче только одного из параметров вернётся 400. При совместной передаче с `radius` включает сортировку по дистанции по возрастанию.【F:internal/handlers/global_search_handler.go†L98-L136】【F:internal/services/global_search_service.go†L31-L56】【F:internal/services/global_search_service.go†L141-L160】【F:internal/services/global_search_service.go†L245-L268】|
| `radius` | положительное число | нет | — | Радиус поиска в километрах. Работает только вместе с `latitude`/`longitude`: объявления без координат или находящиеся дальше радиуса будут отброшены. Также включает сортировку результатов по расстоянию от пользователя (ближайшие первыми).【F:internal/handlers/global_search_handler.go†L118-L136】【F:internal/services/global_search_service.go†L37-L56】【F:internal/services/global_search_service.go†L66-L160】|

## Формат ответа

Сервис возвращает пагинированный объект `GlobalSearchResponse`:

```json
{
  "results": [
    {
      "type": "service",
      "service": { "id": 123, "title": "..." }
    },
    {
      "type": "ad",
      "ad": { "id": 987, "title": "..." }
    }
  ],
  "total": 42,
  "page": 1,
  "limit": 20
}
```

Каждый элемент `results` содержит поле `type` и ровно один заполненный объект сущности (`service`, `ad`, `work`, `work_ad`, `rent` или `rent_ad`). Пустые сущности опускаются благодаря `omitempty`. 【F:internal/models/global_search.go†L17-L34】
При указании координат в запросе (`latitude`, `longitude`) возвращается дополнительное поле `distance` (в километрах), отражающее удалённость объявления от пользователя. Если координаты или координаты объявления недоступны, поле опускается.【F:internal/models/global_search.go†L17-L34】【F:internal/services/global_search_service.go†L31-L56】【F:internal/services/global_search_service.go†L245-L282】

### Правила сортировки

* Без координат результаты отсортированы по активности «топ»-продвижения и дате создания: активный «топ» → время активации «топа» → дата публикации.【F:internal/services/global_search_service.go†L141-L160】
* При наличии `latitude`, `longitude` и `radius` сортировка переключается на расстояние до пользователя: объявления с меньшим `distance` идут первыми; при равной дистанции сохраняется приоритет по «топу» и дате. Таким образом, фронт получает ближайшие предложения в рамках указанного радиуса, но при одинаковом расстоянии первыми будут активные «топы».【F:internal/services/global_search_service.go†L37-L160】【F:internal/services/global_search_service.go†L245-L282】

## Ошибки и коды ответов

| Условие | HTTP-код | Сообщение |
| --- | --- | --- |
| Отсутствует параметр `types` | 400 | `types parameter is required`【F:internal/handlers/global_search_handler.go†L26-L30】|
| Передан недопустимый тип | 400 | `unsupported listing type: <type>`【F:internal/handlers/global_search_handler.go†L32-L44】|
| Ни один валидный тип не найден | 400 | `at least one valid type must be provided`【F:internal/handlers/global_search_handler.go†L52-L55】|
| Передано больше 6 типов | 400 | `no more than 6 listing types allowed`【F:internal/handlers/global_search_handler.go†L56-L58】|
| Неверное значение `on_site` или `negotiable` | 400 | `invalid on_site value` / `invalid negotiable value`【F:internal/handlers/global_search_handler.go†L79-L96】|
| Недоступен сервис глобального поиска | 500 | `service unavailable` (ошибка конфигурации обработчика)【F:internal/handlers/global_search_handler.go†L21-L24】|
| Ошибка бизнес-логики или репозиториев | 500 | Текст ошибки, проброшенный из `GlobalSearchService` или репозитория. Например, `unsupported listing type` при попытке запросить тип, для которого не инициализирован соответствующий репозиторий.【F:internal/handlers/global_search_handler.go†L95-L99】【F:internal/services/global_search_service.go†L29-L138】|

## Алгоритм обработки запроса

1. Middleware гарантирует базовую защиту и логирование, затем управление передаётся `globalSearchHandler.Search`.
2. Обработчик валидирует `types`, фильтрует дубликаты и запрещённые значения, ограничивает список шестью элементами и собирает числовые фильтры (категории, цены, рейтинги, сортировка, пагинация).【F:internal/handlers/global_search_handler.go†L26-L92】
3. Из JWT заголовка извлекается `user_id`; в случае отсутствия токена используется `0`.【F:internal/handlers/global_search_handler.go†L80-L101】【F:internal/handlers/global_search_handler.go†L177-L191】
4. Сформированный `GlobalSearchRequest` передаётся в `GlobalSearchService.Search`.【F:internal/handlers/global_search_handler.go†L82-L96】
5. Сервис определяет фактические `limit` и `page`, вычисляет `perTypeLimit = limit * page`, чтобы получить из каждого источника достаточно записей для текущей страницы (иначе объединённая выборка могла бы быть меньше).【F:internal/services/global_search_service.go†L35-L48】
6. Для каждого типа из `types` вызывается соответствующий репозиторий `<Type>Repo.Get...WithFilters`, который принимает все фильтры (включая `user_id`). Каждая запись оборачивается в `GlobalSearchItem` и дополняется метаинформацией о «топ»-активации и времени создания через `newGlobalSearchEntry`. Репозиторий может вернуть дополнительные поля (лайки, цены, адреса) — они передаются без изменений.【F:internal/services/global_search_service.go†L62-L135】【F:internal/services/global_search_service.go†L181-L187】
7. Полученные элементы сортируются: сначала активно продвигаемые (по `top`), затем по времени активации «топа», и только потом по дате создания. Алгоритм реализован в `sortGlobalSearchEntries` и `lessByTopState`, которые используют `models.ParseTopInfo` и `TopInfo.IsActive`. Это гарантирует приоритет платных размещений и свежих объявлений.【F:internal/services/global_search_service.go†L145-L216】【F:internal/models/top.go†L1-L106】
8. После сортировки применяется стандартная пагинация `limit/page`, вычисляется `total`, формируется итоговый массив `results` и возвращается клиенту.【F:internal/services/global_search_service.go†L141-L164】

## Поддерживаемые источники данных

| Тип (`types`) | Используемый репозиторий | Метод | Комментарии |
| --- | --- | --- | --- |
| `service` | `ServiceRepository` | `GetServicesWithFilters` | Передаются `user_id`, категории, подкатегории, ценовые границы, рейтинги, `sortOption`, а также фильтры `on_site` и `negotiable` для договорных цен.【F:internal/services/global_search_service.go†L62-L76】|
| `ad` | `AdRepository` | `GetAdWithFilters` | Аналогичные фильтры, включая `on_site` и `negotiable`, результат оборачивается в `GlobalSearchItem.Ad`.【F:internal/services/global_search_service.go†L76-L88】|
| `work` | `WorkRepository` | `GetWorksWithFilters` | Возвращает вакансии/резюме с возможностью фильтрации по `negotiable`.【F:internal/services/global_search_service.go†L88-L99】|
| `work_ad` | `WorkAdRepository` | `GetWorksAdWithFilters` | Предложения о поиске сотрудников с фильтром `negotiable`.【F:internal/services/global_search_service.go†L100-L111】|
| `rent` | `RentRepository` | `GetRentsWithFilters` | Объявления об аренде с фильтром `negotiable`.【F:internal/services/global_search_service.go†L112-L123】|
| `rent_ad` | `RentAdRepository` | `GetRentsAdWithFilters` | Запросы на аренду с фильтром `negotiable`.【F:internal/services/global_search_service.go†L124-L135】|

Если соответствующий репозиторий не сконфигурирован (nil), сервис немедленно возвращает ошибку `unsupported listing type`. Это защитный механизм при неполной инициализации зависимостей.【F:internal/services/global_search_service.go†L62-L136】

## Пример запроса

```
GET /search/global?types=service,ad,rent&categories=10,11&limit=10&page=1&price_from=0&price_to=50000&ratings=4,5 HTTP/1.1
Host: api.example.com
Authorization: Bearer <token>
```

В этом примере клиент просит первую страницу из трёх типов объявлений. Категории 10 и 11 будут применены ко всем источникам, будет возвращено до 10 элементов. Если у части записей активирован «топ», они окажутся в начале списка.

### Пример запроса с геофильтром и сортировкой по дистанции

```
GET /search/global?types=service,ad&limit=15&page=1&latitude=55.7522&longitude=37.6156&radius=10 HTTP/1.1
Host: api.example.com
```

* `latitude`/`longitude` задают точку в Москве; `radius=10` ограничивает выборку 10 км и включает сортировку по расстоянию.
* В ответе каждое объявление будет содержать поле `distance` (например, `1.3`), а ближайшие записи окажутся первыми.

### Полные примеры запросов по каждому типу

Ниже приведены готовые строки запросов с максимальным набором параметров для каждого поддерживаемого типа. Необязательные поля можно опускать — обработчик корректно обработает их отсутствие или пустые значения.

* **`service`** — услуги с фильтрами по категориям, цене, рейтингу, «договорной цене», «выезд на объект» и географией:

  ```
  GET /search/global?types=service&categories=10,11&subcategories=101,102&limit=20&page=1&price_from=500&price_to=50000&ratings=4,5&sort_option=0&on_site=yes&negotiable=no&latitude=55.75&longitude=37.62&radius=15 HTTP/1.1
  Authorization: Bearer <token>
  ```

* **`ad`** — объявления о товарах/услугах с теми же базовыми фильтрами, включая «выезд на объект»:

  ```
  GET /search/global?types=ad&categories=20,21&subcategories=201,202&limit=15&page=2&price_from=1000&price_to=250000&ratings=3,4,5&sort_option=1&on_site=no&negotiable=yes&latitude=59.93&longitude=30.33&radius=8 HTTP/1.1
  ```

* **`work`** — вакансии/резюме с фильтрами по опыту, графику, периоду оплаты, формату работы и языкам:

  ```
  GET /search/global?types=work&categories=30&subcategories=301,302&limit=25&page=1&price_from=80000&price_to=200000&ratings=4,5&sort_option=2&negotiable=yes&work_experience=middle,senior&work_schedules=full_time,part_time&payment_periods=month,week&remote=yes&languages=ru,en&educations=bachelor,master&latitude=55.04&longitude=82.93&radius=20 HTTP/1.1
  ```

* **`work_ad`** — объявления о поиске сотрудников с теми же кадровыми фильтрами, что и `work`:

  ```
  GET /search/global?types=work_ad&categories=31&subcategories=311&limit=10&page=1&price_from=60000&price_to=180000&ratings=4&sort_option=0&negotiable=no&work_experience=junior,senior&work_schedules=shift,remote&payment_periods=month&remote=no&languages=en,de&educations=college,bachelor&latitude=55.75&longitude=37.62&radius=12 HTTP/1.1
  ```

* **`rent`** — предложения аренды с фильтром по типам аренды и условиям залога:

  ```
  GET /search/global?types=rent&categories=40&subcategories=401,402&limit=30&page=1&price_from=15000&price_to=90000&ratings=4,5&sort_option=3&negotiable=yes&rent_types=apartment,room&deposits=deposit_free,one_month&latitude=43.12&longitude=131.89&radius=6 HTTP/1.1
  ```

* **`rent_ad`** — заявки на поиск аренды с аналогичными параметрами `rent`:

  ```
  GET /search/global?types=rent_ad&categories=41&subcategories=411&limit=20&page=2&price_from=20000&price_to=70000&ratings=3,4&sort_option=1&negotiable=no&rent_types=studio,apartment&deposits=deposit_free&latitude=56.84&longitude=35.89&radius=5 HTTP/1.1
  ```

### Подробные примеры полей и форматов

* **Массивы (CSV)**: `types=service,ad,work_ad`, `categories=1,2,3`, `ratings=4.5,5`.
* **Булевые**: `on_site=yes`, `negotiable=no`, `remote=1` (любое из значений `yes/no/да/нет/true/false/1/0`).
* **Цены**: `price_from=500` и `price_to=1500` (дробные допускаются: `price_to=199.99`).
* **Работа**: `work_experience=junior,senior`, `work_schedules=full_time,part_time`, `payment_periods=month,week`, `languages=en,ru`, `educations=bachelor,master`.
* **Аренда**: `rent_types=apartment,room`, `deposits=deposit_free,one_month`.
* **География**: передавайте оба параметра `latitude` и `longitude` в формате десятичных градусов (`55.75`, `37.61`). Радиус (`radius`) — положительное число в километрах.

Используйте `limit` и `page` для пагинации; безопасные значения по умолчанию — `20` и `1`. При необходимости комбинируйте фильтры: например, `types=rent&rent_types=apartment&price_to=50000&latitude=59.93&longitude=30.33&radius=5` вернёт ближайшие квартиры до 5 км от указанной точки с сортировкой по расстоянию.

## Пример успешного ответа

```json
{
  "results": [
    {
      "type": "service",
      "service": {
        "id": 345,
        "title": "Разработка мобильного приложения",
        "price": 150000,
        "top": { "activated_at": "2024-04-01T09:00:00Z" }
      }
    },
    {
      "type": "ad",
      "ad": {
        "id": 981,
        "title": "Продам ноутбук",
        "price": 250000
      }
    }
  ],
  "total": 32,
  "page": 1,
  "limit": 10
}
```

## Поведение при пустом результате

Если по всем типам не найдено ни одной записи, сервис вернёт пустой массив `results`, `total = 0`, а параметры `page` и `limit` останутся в ответе, что облегчает отображение состояния «ничего не найдено».【F:internal/services/global_search_service.go†L141-L164】
