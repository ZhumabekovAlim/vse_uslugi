# Руководство по приёму заказов (service, ad, work, work_ad, rent, rent_ad)

Документ описывает единый жизненный цикл заказа: **отклик → подтверждение → завершение/отмена → отзыв**. В каждой сущности участвуют две стороны:

- **Владелец карточки** — создатель объявления (исполнитель в `service/work/rent`, заказчик или работодатель в `ad/work_ad/rent_ad`).
- **Откликающийся** — пользователь, отправляющий предложение с ценой и комментариями. Отклик создаёт чат и запись в таблице подтверждений со статусом `active`.

## Статусы подтверждений

- `active` — создаётся вместе с откликом. Флаг `confirmed` = `false`.
- `active + confirmed=true` — после вызова `.../confirm` выбранное подтверждение фиксирует пару клиент/исполнитель. В объявлениях исполнителей (`ad/work_ad/rent_ad`) конкурирующие отклики удаляются, чтобы осталось одно «активное» направление работы.【F:internal/repositories/ad_confirmation_repository.go†L31-L52】【F:internal/repositories/work_ad_confirmation_repository.go†L32-L57】
- `archived` — финальное состояние после `.../cancel` или `.../done`; флаг `confirmed` сбрасывается при отмене. История сохраняется для отображения в чатах/заказах, но карточка остаётся активной для новых откликов.【F:internal/repositories/service_confirmation_repository.go†L56-L89】【F:internal/repositories/rent_confirmation_repository.go†L51-L80】

## Общие эффекты при создании отклика

- Списывается слот откликов подписки (`ConsumeResponse`) и восстанавливается при отмене отклика (`RestoreResponse`).【F:internal/services/ad_responses_service.go†L36-L43】【F:internal/services/service_response_service.go†L36-L44】
- Создаётся чат между сторонами, в него сразу отправляется первое сообщение с предложенной ценой.【F:internal/services/service_response_service.go†L70-L103】【F:internal/services/work_ad_responses_service.go†L74-L106】
- Записывается подтверждение с `status="active"` и `confirmed=false`. Для подтверждённых заказов отмена/завершение меняет статус на `archived`.【F:internal/repositories/ad_confirmation_repository.go†L14-L52】【F:internal/repositories/service_confirmation_repository.go†L19-L54】

## Маршруты по шагам

| Шаг | service | work | rent | ad | work_ad | rent_ad |
| --- | --- | --- | --- | --- | --- | --- |
| Отклик | `POST /responses` | `POST /work_responses` | `POST /rent_responses` | `POST /ad_responses` | `POST /work_ad_responses` | `POST /rent_ad_responses` |
| Отмена отклика | `DELETE /responses/:id` | `DELETE /work_responses/:id` | `DELETE /rent_responses/:id` | `DELETE /ad_responses/:id` | `DELETE /work_ad_responses/:id` | `DELETE /rent_ad_responses/:id` |
| Подтвердить пару | `POST /service/confirm` | `POST /work/confirm` | `POST /rent/confirm` | `POST /ad/confirm` | `POST /work_ad/confirm` | `POST /rent_ad/confirm` |
| Отменить подтверждённый заказ | `POST /service/cancel` | `POST /work/cancel` | `POST /rent/cancel` | `POST /ad/cancel` | `POST /work_ad/cancel` | `POST /rent_ad/cancel` |
| Завершить заказ | `POST /service/done` | `POST /work/done` | `POST /rent/done` | `POST /ad/done` | `POST /work_ad/done` | `POST /rent_ad/done` |
| Отзыв | `POST /review` | `POST /work_review` | `POST /rent_review` | `POST /ad_review` | `POST /work_ad_review` | `POST /rent_ad_review` |

> Все маршруты требуют авторизации, за исключением чтения отзывов. Эндпоинты указаны в `cmd/routes.go` (см. соответствующие блоки).【F:cmd/routes.go†L210-L241】【F:cmd/routes.go†L430-L508】【F:cmd/routes.go†L585-L608】

## Типы с владельцем-исполнителем (service, work, rent)

Владелец карточки — исполнитель. Откликается клиент, которому создаётся чат и выдаётся телефон исполнителя в ответе.

### Service
1. **Отклик** — `POST /responses`
   ```json
   {
     "service_id": 12,
     "price": 1500,
     "description": "Могу завтра в 10:00"
   }
   ```
   **Ответ**: `201 Created`
   ```json
   {
     "id": 501,
     "user_id": 42,
     "service_id": 12,
     "chat_id": 9100,
     "client_id": 42,
     "performer_id": 7,
     "price": 1500,
     "description": "Могу завтра в 10:00",
     "phone": "+7-900-100-2020",
     "created_at": "2024-05-15T09:42:11Z"
   }
   ```
   - Создаётся чат `client_id ↔ performer_id`, подтверждение `service_confirmations` со статусом `active`, отправляется приветственное сообщение от клиента к исполнителю.【F:internal/services/service_response_service.go†L70-L103】 
2. **Подтверждение** — `POST /service/confirm` с телом `{"service_id":12,"client_id":42}`. Обновляет строку подтверждения: `confirmed=true`, статус остаётся `active`. Ответ `200 OK`.【F:internal/repositories/service_confirmation_repository.go†L36-L54】
3. **Отмена подтверждённого заказа** — `POST /service/cancel` с `{"service_id":12}`. Сбрасывает `confirmed=false`, ставит статус `archived`. Ответ `200 OK`.【F:internal/repositories/service_confirmation_repository.go†L56-L75】
4. **Завершение** — `POST /service/done` с `{"service_id":12}` переводит подтверждение в `archived`. Ответ `200 OK`.【F:internal/repositories/service_confirmation_repository.go†L77-L89】
5. **Отзыв** — `POST /review`
   ```json
   {
     "service_id": 12,
     "user_id": 42,
     "rating": 5,
     "review": "Работа сделана в срок"
   }
   ```
   Создание отзыва автоматически вызывает `Done` для подтверждения, если оно ещё активно.【F:internal/services/review_service.go†L12-L27】

### Work
- Шаги и тела запросов идентичны `service`, но используются эндпоинты `/work_responses`, `/work/confirm|cancel|done`, `/work_review`. Ответы включают `work_id` вместо `service_id`. Создание отклика сразу отправляет сообщение от клиента к исполнителю и сохраняет телефон исполнителя в ответе.【F:internal/services/work_response_service.go†L46-L105】【F:cmd/routes.go†L489-L503】 Отзыв завершает подтверждение через `work_confirmations.Done`.【F:internal/services/work_rreview_service.go†L14-L24】

### Rent
- Аналогично `service`, с полями `rent_id` и маршрутами `/rent_responses`, `/rent/confirm|cancel|done`, `/rent_review`. Отмена/завершение работают через `rent_confirmations` с переходом в `archived`. Отзыв вызывает `Done`.【F:internal/repositories/rent_confirmation_repository.go†L14-L80】【F:internal/services/rent_review_service.go†L14-L24】

## Типы с владельцем-заказчиком (ad, work_ad, rent_ad)

Здесь откликается исполнитель. Успешный отклик даёт исполнителю доступ к чату и номеру телефона заказчика. Подтверждение выбирает конкретного исполнителя и удаляет конкурирующие отклики в объявлениях исполнителя.

### Ad
1. **Отклик** — `POST /ad_responses`
   ```json
   {
     "ad_id": 17,
     "price": 1800,
     "description": "Готов приступить завтра"
   }
   ```
   **Ответ**: `201 Created` с полями `id, ad_id, chat_id, client_id, performer_id, phone`. Первое сообщение отправляется от исполнителя (откликающегося) клиенту-заказчику.【F:internal/services/ad_responses_service.go†L74-L108】
2. **Подтверждение** — `POST /ad/confirm` с `{"ad_id":17,"performer_id":42}`. Помечает строку подтверждения как подтверждённую и удаляет все остальные отклики по этому объявлению. Ответ `200 OK`.【F:internal/repositories/ad_confirmation_repository.go†L31-L52】
3. **Отмена** — `POST /ad/cancel` с `{"ad_id":17}`. Сбрасывает подтверждение, статус `archived`. Ответ `200 OK`.【F:internal/repositories/ad_confirmation_repository.go†L54-L70】
4. **Завершение** — `POST /ad/done` с `{"ad_id":17}` ставит статус `archived`. Ответ `200 OK`.【F:internal/repositories/ad_confirmation_repository.go†L72-L84】
5. **Отзыв** — `POST /ad_review` с полями `ad_id, user_id, rating, review`. Создание отзыва также вызывает `Done` для подтверждения.【F:internal/services/ad_reviews_service.go†L9-L27】

### Work_ad (резюме исполнителя)
- Шаги повторяют `ad`, но с полями `work_ad_id` и маршрутами `/work_ad_responses`, `/work_ad/confirm|cancel|done`, `/work_ad_review`.
- Подтверждение выбирает конкретного исполнителя (владельца резюме) и очищает остальные отклики работодателей по этому резюме. Статусы меняются так же: `active` → `archived`.【F:internal/repositories/work_ad_confirmation_repository.go†L32-L104】
- Ответ отклика содержит `client_id` (работодатель), `performer_id` (исполнитель-резюме), `phone` заказчика и `chat_id`; первое сообщение отправляется от исполнителя-откликающегося работодателю.【F:internal/services/work_ad_responses_service.go†L74-L106】 Отзыв завершает подтверждение через `WorkAdReviewService`.【F:internal/services/work_ad_review_service.go†L14-L24】

### Rent_ad
- Полностью аналогично `work_ad`, но с полями `rent_ad_id` и маршрутами `/rent_ad_responses`, `/rent_ad/confirm|cancel|done`, `/rent_ad_review`.
- Подтверждение: выбор исполнителя, статус `active` → `archived`, удаление конкурирующих откликов при подтверждении. Отмена/завершение архивируют запись. Отзыв закрывает подтверждение автоматически.【F:internal/services/rent_ad_responses_service.go†L85-L108】【F:internal/repositories/rent_ad_confirmation_repository.go†L32-L80】【F:internal/services/rent_ad_reviews_service.go†L14-L24】

## Что видно в чатах и историях
- Чат создаётся на этапе отклика, поэтому участники сразу могут переписываться. Телефон владельца карточки возвращается в теле отклика (`phone`).【F:internal/models/service_response.go†L7-L18】【F:internal/models/ad_responses.go†L7-L19】
- История заказов строится по таблицам подтверждений; даже после `cancel`/`done` записи остаются со статусом `archived`, что позволяет показать завершённые и отменённые сделки без блокировки новых откликов на карточку.【F:internal/repositories/service_confirmation_repository.go†L56-L89】【F:internal/repositories/ad_confirmation_repository.go†L54-L84】

## Сводка по заказам пользователя

### История заказов
- **Маршрут:** `GET /user/orders/:user_id` (требует авторизации).【F:cmd/routes.go†L216-L219】
- **Что передать:** `:user_id` — идентификатор пользователя (путь). Без тела.
- **Что возвращает:** список объединённых заказов по всем типам (`service`, `work`, `rent`, `ad`, `work_ad`, `rent_ad`) с полями `id`, `name`, `price`, `description`, `created_at`, `status`, `type`. В выборку попадают строки подтверждений, где пользователь участвует как клиент или исполнитель, со статусом `active` или `archived`. Сортировка — по дате создания подтверждения (DESC).【F:internal/handlers/user_items_handler.go†L29-L51】【F:internal/repositories/user_items_repository.go†L34-L104】
- **Пример ответа:**
  ```json
  [
    {
      "id": 91,
      "name": "Монтаж кондиционера",
      "price": 4500,
      "description": "Нужна установка сплит-системы",
      "created_at": "2024-06-10T12:41:00Z",
      "status": "archived",
      "type": "service"
    },
    {
      "id": 17,
      "name": "Уборка офиса",
      "price": 1800,
      "description": "Готов приступить завтра",
      "created_at": "2024-06-11T09:20:00Z",
      "status": "active",
      "type": "ad"
    }
  ]
  ```

### Активные заказы исполнителя
- **Маршрут:** `GET /user/active_orders/:user_id` (требует авторизации).【F:cmd/routes.go†L216-L219】
- **Что передать:** `:user_id` — идентификатор исполнителя (путь). Без тела.
- **Что возвращает:** подтверждённые (`confirmed = true`) заказы со статусом `active`, где пользователь выступает исполнителем, по всем типам (`service`, `work`, `rent`, `ad`, `work_ad`, `rent_ad`). Поля — те же, что в истории. Сортировка — по дате создания подтверждения (DESC).【F:internal/handlers/user_items_handler.go†L53-L76】【F:internal/repositories/user_items_repository.go†L106-L150】
