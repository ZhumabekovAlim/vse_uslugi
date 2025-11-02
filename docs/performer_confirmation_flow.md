# Performer Advertisement Order Lifecycle

This guide explains how the confirmation `status` fields added in migration 52 let performers keep their advertisements active while simultaneously tracking multiple customer orders. It also contains ready-to-use Postman examples that mirror the new workflow.

## Confirmation Status Values

| Status       | When it is assigned | Triggering endpoint |
|--------------|---------------------|---------------------|
| `active`     | A customer creates a response; confirmation rows are created with this default. | `POST /ad_responses`, `POST /rent_ad_responses`, `POST /work_ad_responses`, etc. |
| `in progress`| The performer accepts a specific customer response; only the accepted confirmation switches to this value. | `POST /ad/confirm`, `POST /rent_ad/confirm`, `POST /work_ad/confirm`, etc. |
| `cancelled`  | Either party cancels an accepted order; the confirmation is preserved for history with this status. | `POST /ad/cancel`, `POST /rent_ad/cancel`, `POST /work_ad/cancel`, etc. |
| `done`       | The performer marks the job complete; history views show the finished order. | `POST /ad/done`, `POST /rent_ad/done`, `POST /work_ad/done`, etc. |

All six confirmation tables received the new column with `DEFAULT 'active'`, so existing records automatically take part in the lifecycle without additional migrations.【F:db/migrations/000052_confirmation_status_columns.up.sql†L1-L17】

Repository methods now control the transitions instead of mutating the parent advertisement. For example, confirming an ad response updates the row to `in progress`, deletes competing responses, and leaves the advertisement row untouched, so new customers can still respond.【F:internal/repositories/ad_confirmation_repository.go†L14-L53】 When chats are fetched, the current confirmation `status` is returned and phone numbers are only exposed while a job is `in progress`, ensuring performers can juggle multiple deals from a single listing.【F:internal/repositories/chat_repository.go†L18-L125】【F:internal/repositories/chat_repository.go†L126-L166】

## Postman Walkthrough (Performer Ad)

Use the accompanying Postman collection at `docs/performer_confirmation_flow.postman_collection.json` for quick imports. It assumes a local API at `http://localhost:8080` and a bearer token stored in the `{{authToken}}` variable.

### 1. Customer responds to a performer advertisement

- **Endpoint:** `POST /ad_responses`
- **Body:**
```json
{
  "user_id": 42,
  "ad_id": 17,
  "price": 1500,
  "description": "Готов приступить завтра"
}
```
- **Sample response:**
```json
{
  "id": 501,
  "user_id": 42,
  "ad_id": 17,
  "chat_id": 9301,
  "client_id": 7,
  "performer_id": 42,
  "price": 1500,
  "description": "Готов приступить завтра",
  "created_at": "2024-05-15T09:42:11Z"
}
```

A confirmation row is created with `status = "active"`. The performer still has an `active` advertisement and can receive more responses because the ad itself is never archived during this flow.【F:internal/services/ad_responses_service.go†L19-L64】

### 2. Performer accepts a specific customer

- **Endpoint:** `POST /ad/confirm`
- **Body:**
```json
{
  "ad_id": 17,
  "performer_id": 42
}
```
- **Expected result:** HTTP 200 with an empty body.

The selected confirmation switches to `in progress`, rival responses are removed, and the subscription counter for the accepted performer is decremented.【F:internal/repositories/ad_confirmation_repository.go†L31-L53】 Other `active` confirmations (from different ads) remain untouched, so the performer can keep working on multiple jobs in parallel.

### 3. Tracking the order in chats/history

- **Endpoint:** `GET /api/chats/user/42`
- **Sample response snippet:**
```json
[
  {
    "ad_id": 17,
    "ad_name": "Настройка оборудования",
    "status": "in progress",
    "performer_id": 42,
    "users": [
      {
        "id": 7,
        "name": "Ирина",
        "surname": "Климова",
        "avatar_path": "",
        "phone": "+7-900-100-2020",
        "price": 1500,
        "chat_id": 9301,
        "lastMessage": "Отлично, приступаю завтра",
        "review_rating": 4.9,
        "reviews_count": 32,
        "my_role": "performer"
      }
    ]
  },
  {
    "ad_id": 18,
    "ad_name": "Настройка оборудования",
    "status": "active",
    "users": [
      {
        "id": 55,
        "name": "Максим",
        "surname": "Назаров",
        "avatar_path": "",
        "phone": "",
        "price": 1800,
        "chat_id": 9310,
        "lastMessage": "Готов обсудить детали",
        "review_rating": 4.7,
        "reviews_count": 11,
        "my_role": "performer"
      }
    ]
  }
]
```

The performer sees each order with its own status. Because the second job is still `active`, the customer's phone remains hidden (empty string). Once a job transitions to `done` or `cancelled`, it stays visible in history with that status while the advertisement itself continues to accept new responses.【F:internal/repositories/chat_repository.go†L18-L166】

### 4. Completing or cancelling the job

- **Mark done:** `POST /ad/done` with body `{"ad_id": 17}`
- **Cancel:** `POST /ad/cancel` with body `{"ad_id": 17}`

Both endpoints return HTTP 200 and update only the confirmation record (`status = 'done'` or `'cancelled'`). Subscription counters are restored when the client initiates the cancellation flow.【F:internal/repositories/ad_confirmation_repository.go†L57-L83】

## Extending to rentals and work ads

The same lifecycle applies to rental and work advertisements/services because their repositories follow the identical transition rules (`Create` → `Confirm` → `Cancel`/`Done`).【F:internal/repositories/work_ad_confirmation_repository.go†L14-L104】【F:internal/repositories/rent_ad_confirmation_repository.go†L14-L104】 Import their dedicated requests from the Postman collection to test these verticals without additional setup.

## Importing the Postman Collection

1. Open Postman → *File* → *Import*.
2. Select `docs/performer_confirmation_flow.postman_collection.json`.
3. Set the `baseUrl` and `authToken` variables in the collection or create an environment.
4. Execute the requests sequentially to reproduce the active → in progress → done/cancelled lifecycle.
