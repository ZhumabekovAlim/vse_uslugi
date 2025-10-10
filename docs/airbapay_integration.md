# Инструкция по подключению Airbapay

Данный документ описывает процесс настройки и запуска интеграции платежной системы Airbapay в проекте `naimuBack`.

## 1. Создание и настройка аккаунта Airbapay

1. Перейдите в личный кабинет Airbapay по адресу [https://pc.airbapay.kz/auth/login](https://pc.airbapay.kz/auth/login).
2. Пройдите регистрацию компании. После регистрации передайте номер телефона менеджеру Airbapay, чтобы аккаунту присвоили роль «Владелец».
3. В разделе «Настройки» личного кабинета добавьте сотрудников, если требуется.
4. После активации аккаунта получите боевые данные (логин, пароль, `terminal_id`). Эти значения необходимо сохранить: их нужно будет указать в переменных окружения приложения.

## 2. Настройка окружения приложения

Интеграция Airbapay конфигурируется через переменные окружения, которые считываются при старте приложения (`cmd/initializer.go`).

| Переменная | Назначение | Значение по умолчанию |
|------------|------------|------------------------|
| `AIRBAPAY_USERNAME` | Логин для Basic Auth запросов в API. | `VSEUSLUGI` |
| `AIRBAPAY_PASSWORD` | Пароль для Basic Auth. | `v(A3Z!_zua%V&%a` |
| `AIRBAPAY_TERMINAL_ID` | Идентификатор терминала из кабинета Airbapay. | `68e73c28a36bcb28994f2061` |
| `AIRBAPAY_BASE_URL` | Базовый URL API. Для прод окружения — `https://ps.airbapay.kz/acquiring-api`. | `https://ps.airbapay.kz/acquiring-api` |
| `AIRBAPAY_CREATE_INVOICE_URI` | Дополнительный путь или абсолютный URL для создания инвойса. Можно использовать для указания точного endpoint, если он отличается от значения по умолчанию. | `/v1/invoice/create` |
| `AIRBAPAY_SUCCESS_URL` | URL на стороне вашего сервиса, куда Airbapay перенаправит клиента после успешной оплаты. | пусто |
| `AIRBAPAY_FAILURE_URL` | URL для редиректа после неуспешной оплаты. | пусто |

> **Важно:** убедитесь, что значения в продакшене совпадают с данными, выданными Airbapay. В тестовом окружении можно использовать тестовые реквизиты.

Пример `.env` файла:

```env
AIRBAPAY_USERNAME="VSEUSLUGI"
AIRBAPAY_PASSWORD="v(A3Z!_zua%V&%a"
AIRBAPAY_TERMINAL_ID="68e73c28a36bcb28994f2061"
AIRBAPAY_BASE_URL="https://ps.airbapay.kz/acquiring-api"
AIRBAPAY_CREATE_INVOICE_URI="/v1/invoice/create"
AIRBAPAY_SUCCESS_URL="https://example.com/payments/airbapay/success"
AIRBAPAY_FAILURE_URL="https://example.com/payments/airbapay/failure"
```

Приложение автоматически подставит значения из `.env`/переменных окружения при инициализации сервиса Airbapay (`services.NewAirbapayService`).

## 3. Конфигурация success/failure callback URL

Airbapay требует зарегистрировать адреса, куда будет перенаправлять клиента после оплаты. Укажите боевые URL в личном кабинете и сообщите их менеджеру Airbapay, чтобы добавить в whitelist. На стороне приложения маршруты уже настроены:

- `GET /airbapay/success` — возвращает JSON `{ "status": "success" }`.
- `GET /airbapay/failure` — возвращает JSON `{ "status": "failure" }`.

При необходимости замените заглушки собственными страницами/редиректами.

## 4. API-эндпоинты приложения

После старта приложения доступны следующие маршруты для работы с платежами (`cmd/routes.go`):

| Метод и путь | Описание |
|--------------|---------|
| `POST /airbapay/pay` | Создание инвойса и получение `payment_url`. Требует JSON: `{ "user_id": number, "amount": float, "description": string }`. Возвращает `inv_id`, `order_id`, `invoice_id`, `payment_url`, `status`. |
| `POST /airbapay/callback` | Callback от Airbapay. Принимает JSON payload с `order_id`, `status` и др. Полностью обрабатывается сервером, в ответ возвращает `{ "status": "ok", "order_id": "..." }`. |
| `GET /airbapay/history/:user_id` | История платежей пользователя. Требует авторизации. |
| `GET /airbapay/success` | Ответ при успешной оплате (используется в качестве redirect URL). |
| `GET /airbapay/failure` | Ответ при неуспешной оплате. |

### Создание платежа

1. Клиент вызывает `POST /airbapay/pay` с телом:
   ```json
   {
     "user_id": 123,
     "amount": 1000.00,
     "description": "Оплата заказа #123"
   }
   ```
2. Сервер создаёт запись в таблице `invoices`, отправляет запрос в Airbapay (по умолчанию `/v1/invoice/create`, конечная точка может быть переопределена через `AIRBAPAY_CREATE_INVOICE_URI`) и возвращает ссылку для оплаты.
3. Клиент перенаправляет пользователя на полученный `payment_url`.

### Обработка callback

1. Airbapay отправляет POST запрос на `/airbapay/callback` с JSON, содержащим `order_id`, `status`, `invoice_id` и т.д.
2. Сервер проверяет подпись (пока — на непустое значение), сопоставляет `order_id` с внутренним инвойсом и обновляет статус в БД (`paid`, `failed`, либо фактический статус из payload).
3. В ответ сервер возвращает JSON `{ "status": "ok", "order_id": "..." }`.

## 5. Схема работы с базой данных

Инвойсы сохраняются в таблице `invoices`. Репозиторий (`internal/repositories/invoice.go`) предоставляет методы:

- `CreateInvoice` — вставляет запись со статусом `pending`.
- `MarkPaid` — помечает инвойс как `paid`.
- `UpdateStatus` — обновляет произвольный статус.
- `GetByUser` — возвращает историю платежей пользователя.

Убедитесь, что таблица `invoices` существует и содержит поля `inv_id`, `user_id`, `amount`, `description`, `status`, `created_at`.

## 6. Локальный запуск и проверка

1. Настройте файл `.env` или переменные окружения с реквизитами Airbapay.
2. Запустите сервис командой:
   ```bash
   go run ./cmd
   ```
3. Создайте тестовый инвойс через `POST /airbapay/pay` (используйте Postman/HTTP client).
4. Откройте возвращённый `payment_url` и выполните тестовую оплату.
5. Проверьте, что Airbapay вызывает ваш `/airbapay/callback` (в тестовом окружении можно сымитировать запрос вручную) и что статус в таблице `invoices` обновился.

## 7. Подготовка к продакшену

- Сообщите Airbapay боевые `success_callback` и `failure_callback` URL для добавления в whitelist.
- Убедитесь, что сервер доступен по HTTPS и принимает входящие запросы от Airbapay.
- Добавьте алертинг/логирование для мониторинга ошибок в обработчике `Callback`.
- При необходимости реализуйте строгую проверку подписи коллбэка, когда Airbapay предоставит точный алгоритм формирования `signature`.

Следуя указанным шагам, вы сможете подключить и запустить платежную систему Airbapay в проекте.
