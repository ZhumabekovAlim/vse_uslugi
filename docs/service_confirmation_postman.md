# Service Confirmation via Postman

This document demonstrates how to create a service response and confirm a performer using the API.

## Create Service Response

- **Endpoint:** `POST /responses`
- **Body (JSON):**
```json
{
  "user_id": 5,
  "service_id": 12,
  "price": 1000,
  "description": "Готов выполнить работу"
}
```
- **Response (JSON):**
```json
{
  "id": 3,
  "user_id": 5,
  "service_id": 12,
  "chat_id": 7,
  "client_id": 2,
  "performer_id": 5,
  "price": 1000,
  "description": "Готов выполнить работу",
  "created_at": "2024-05-15T10:00:00Z"
}
```

`client_id` corresponds to the service author, while `performer_id` is the user who responded.

## Confirm Performer

- **Endpoint:** `POST /service/confirm`
- **Body (JSON):**
```json
{
  "service_id": 12,
  "performer_id": 5
}
```
- **Expected Result:** HTTP 200 OK. The service status changes to `active`, all other responses are deleted, and the chat remains for further communication.

In Postman, ensure you include the user's JWT token in the Authorization header (`Bearer <token>`).
