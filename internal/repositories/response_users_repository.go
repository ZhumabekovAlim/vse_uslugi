package repositories

import (
	"context"
	"database/sql"
	"errors"

	"naimuBack/internal/models"
)

// ResponseUsersRepository retrieves users who responded to items.
type ResponseUsersRepository struct {
	DB *sql.DB
}

// GetUsersByItemID returns users who responded to a specific item type.
func (r *ResponseUsersRepository) GetUsersByItemID(ctx context.Context, itemType string, itemID int) ([]models.ResponseUser, error) {
	var query string
	switch itemType {
	case "service":
		query = `
WITH last_messages AS (
    SELECT m.chat_id, m.text
    FROM messages m
    JOIN (
        SELECT chat_id, MAX(created_at) AS max_created
        FROM messages
        GROUP BY chat_id
    ) t ON t.chat_id = m.chat_id AND t.max_created = m.created_at
)

SELECT u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0), sr.price, sr.description, sr.created_at,
       COALESCE(sc.status, ''), sc.chat_id, COALESCE(lm.text, ''), owner.phone, u.phone, 'performer' AS my_role
FROM service_responses sr
JOIN users u ON u.id = sr.user_id
JOIN service s ON s.id = sr.service_id
JOIN users owner ON owner.id = s.user_id
LEFT JOIN service_confirmations sc ON sc.service_id = sr.service_id AND sc.client_id = sr.user_id
LEFT JOIN last_messages lm ON lm.chat_id = sc.chat_id
WHERE sr.service_id = ? AND (sc.status IS NULL OR sc.status = 'active')`
	case "ad":
		query = `
WITH last_messages AS (
    SELECT m.chat_id, m.text
    FROM messages m
    JOIN (
        SELECT chat_id, MAX(created_at) AS max_created
        FROM messages
        GROUP BY chat_id
    ) t ON t.chat_id = m.chat_id AND t.max_created = m.created_at
)

SELECT u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0), ar.price, ar.description, ar.created_at,
       COALESCE(ac.status, ''), ac.chat_id, COALESCE(lm.text, ''), u.phone, owner.phone, 'customer' AS my_role
FROM ad_responses ar
JOIN users u ON u.id = ar.user_id
JOIN ad a ON a.id = ar.ad_id
JOIN users owner ON owner.id = a.user_id
LEFT JOIN ad_confirmations ac ON ac.ad_id = ar.ad_id AND ac.performer_id = ar.user_id
LEFT JOIN last_messages lm ON lm.chat_id = ac.chat_id
WHERE ar.ad_id = ?`
	case "work":
		query = `
WITH last_messages AS (
    SELECT m.chat_id, m.text
    FROM messages m
    JOIN (
        SELECT chat_id, MAX(created_at) AS max_created
        FROM messages
        GROUP BY chat_id
    ) t ON t.chat_id = m.chat_id AND t.max_created = m.created_at
)

SELECT u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0), wr.price, wr.description, wr.created_at,
       COALESCE(wc.status, ''), wc.chat_id, COALESCE(lm.text, ''), provider.phone, u.phone, 'performer' AS my_role
FROM work_responses wr
JOIN users u ON u.id = wr.user_id
JOIN work w ON w.id = wr.work_id
JOIN users provider ON provider.id = w.user_id
LEFT JOIN work_confirmations wc ON wc.work_id = wr.work_id AND wc.client_id = wr.user_id
LEFT JOIN last_messages lm ON lm.chat_id = wc.chat_id
WHERE wr.work_id = ? AND (wc.status IS NULL OR wc.status = 'active')`
	case "work_ad":
		query = `
WITH last_messages AS (
    SELECT m.chat_id, m.text
    FROM messages m
    JOIN (
        SELECT chat_id, MAX(created_at) AS max_created
        FROM messages
        GROUP BY chat_id
    ) t ON t.chat_id = m.chat_id AND t.max_created = m.created_at
)

SELECT u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0), war.price, war.description, war.created_at,
       COALESCE(wac.status, ''), wac.chat_id, COALESCE(lm.text, ''), u.phone, owner.phone, 'customer' AS my_role
FROM work_ad_responses war
JOIN users u ON u.id = war.user_id
JOIN work_ad wa ON wa.id = war.work_ad_id
JOIN users owner ON owner.id = wa.user_id
LEFT JOIN work_ad_confirmations wac ON wac.work_ad_id = war.work_ad_id AND wac.performer_id = war.user_id
LEFT JOIN last_messages lm ON lm.chat_id = wac.chat_id
WHERE war.work_ad_id = ?`
	case "rent":
		query = `
WITH last_messages AS (
    SELECT m.chat_id, m.text
    FROM messages m
    JOIN (
        SELECT chat_id, MAX(created_at) AS max_created
        FROM messages
        GROUP BY chat_id
    ) t ON t.chat_id = m.chat_id AND t.max_created = m.created_at
)

SELECT u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0), rr.price, rr.description, rr.created_at,
       COALESCE(rc.status, ''), rc.chat_id, COALESCE(lm.text, ''), u.phone, owner.phone, 'customer' AS my_role
FROM rent_responses rr
JOIN users u ON u.id = rr.user_id
JOIN rent r ON r.id = rr.rent_id
JOIN users owner ON owner.id = r.user_id
LEFT JOIN rent_confirmations rc ON rc.rent_id = rr.rent_id AND rc.performer_id = rr.user_id
LEFT JOIN last_messages lm ON lm.chat_id = rc.chat_id
WHERE rr.rent_id = ? AND (rc.status IS NULL OR rc.status = 'active')`
	case "rent_ad":
		query = `
WITH last_messages AS (
    SELECT m.chat_id, m.text
    FROM messages m
    JOIN (
        SELECT chat_id, MAX(created_at) AS max_created
        FROM messages
        GROUP BY chat_id
    ) t ON t.chat_id = m.chat_id AND t.max_created = m.created_at
)

SELECT u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0), rar.price, rar.description, rar.created_at,
       COALESCE(rac.status, ''), rac.chat_id, COALESCE(lm.text, ''), u.phone, owner.phone, 'customer' AS my_role
FROM rent_ad_responses rar
JOIN users u ON u.id = rar.user_id
JOIN rent_ad ra ON ra.id = rar.rent_ad_id
JOIN users owner ON owner.id = ra.user_id
LEFT JOIN rent_ad_confirmations rac ON rac.rent_ad_id = rar.rent_ad_id AND rac.performer_id = rar.user_id
LEFT JOIN last_messages lm ON lm.chat_id = rac.chat_id
WHERE rar.rent_ad_id = ?`
	default:
		return nil, errors.New("unknown item type")
	}

	rows, err := r.DB.QueryContext(ctx, query, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.ResponseUser
	for rows.Next() {
		var u models.ResponseUser
		if err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Surname,
			&u.AvatarPath,
			&u.Rating,
			&u.Price,
			&u.Description,
			&u.CreatedAt,
			&u.Status,
			&u.ChatID,
			&u.LastMessage,
			&u.ProviderPhone,
			&u.ClientPhone,
			&u.MyRole,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}
