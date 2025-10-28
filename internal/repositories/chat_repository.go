package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"naimuBack/internal/models"
)

type ChatRepository struct {
	Db *sql.DB
}

func (r *ChatRepository) CreateChat(ctx context.Context, chat models.Chat) (int, error) {
	insertQuery := `INSERT INTO chats (user1_id, user2_id) VALUES (?, ?)`
	result, err := r.Db.ExecContext(ctx, insertQuery, chat.User1ID, chat.User2ID)
	if err != nil {
		return 0, err
	}

	chatID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(chatID), nil
}

func (r *ChatRepository) GetChatByID(ctx context.Context, id int) (models.Chat, error) {
	var chat models.Chat
	query := `SELECT id, user1_id, user2_id, created_at FROM chats WHERE id = ?`
	err := r.Db.QueryRowContext(ctx, query, id).Scan(&chat.ID, &chat.User1ID, &chat.User2ID, &chat.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Chat{}, nil // Chat not found
		}
		return models.Chat{}, err
	}
	return chat, nil
}

func (r *ChatRepository) GetAllChats(ctx context.Context) ([]models.Chat, error) {
	var chats []models.Chat
	query := `
               SELECT c.id,
                      c.user1_id, u1.name, u1.surname, u1.avatar_path,
                      c.user2_id, u2.name, u2.surname, u2.avatar_path,
                      c.created_at
               FROM chats c
               JOIN users u1 ON c.user1_id = u1.id
               JOIN users u2 ON c.user2_id = u2.id
       `

	rows, err := r.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var chat models.Chat
		err := rows.Scan(
			&chat.ID,
			&chat.User1ID, &chat.User1.Name, &chat.User1.Surname, &chat.User1.AvatarPath,
			&chat.User2ID, &chat.User2.Name, &chat.User2.Surname, &chat.User2.AvatarPath,
			&chat.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return chats, nil
}

func (r *ChatRepository) DeleteChat(ctx context.Context, id int) error {
	query := `DELETE FROM chats WHERE id=?`
	_, err := r.Db.ExecContext(ctx, query, id)
	return err
}

// GetChatsByUserID retrieves chats grouped by advertisements for a specific author.
func (r *ChatRepository) GetChatsByUserID(ctx context.Context, userID int) ([]models.AdChats, error) {
	query := `
WITH last_messages AS (
    SELECT m.chat_id, m.text, m.created_at
    FROM messages m
    JOIN (
        SELECT chat_id, MAX(created_at) AS max_created
        FROM messages
        GROUP BY chat_id
    ) t ON t.chat_id = m.chat_id AND t.max_created = m.created_at
)

SELECT a.id, 'ad' AS ad_type, a.name, a.status, c.performer_id,
       u.id, u.name, u.surname, COALESCE(u.avatar_path, '') AS avatar_path, u.phone,
       ar.price, ac.chat_id, COALESCE(lm.text, '') AS last_message, 'customer' AS my_role
FROM ad a
JOIN ad_confirmations ac ON ac.ad_id = a.id
JOIN users u ON u.id = ac.performer_id
JOIN ad_responses ar ON ar.ad_id = a.id AND ar.user_id = ac.performer_id
LEFT JOIN ad_confirmations c ON c.ad_id = a.id AND c.confirmed = true
LEFT JOIN last_messages lm ON lm.chat_id = ac.chat_id
WHERE a.user_id = ?

UNION ALL

SELECT a.id, 'ad' AS ad_type, a.name, a.status, c.performer_id,
       owner.id, owner.name, owner.surname, COALESCE(owner.avatar_path, '') AS avatar_path, owner.phone,
       ar.price, ac.chat_id, COALESCE(lm.text, '') AS last_message, 'performer' AS my_role
FROM ad a
JOIN ad_confirmations ac ON ac.ad_id = a.id
JOIN users owner ON owner.id = a.user_id
JOIN ad_responses ar ON ar.ad_id = a.id AND ar.user_id = ac.performer_id
LEFT JOIN ad_confirmations c ON c.ad_id = a.id AND c.confirmed = true
LEFT JOIN last_messages lm ON lm.chat_id = ac.chat_id
WHERE ac.performer_id = ?

UNION ALL

SELECT s.id, 'service' AS ad_type, s.name, s.status, c.performer_id,
       u.id, u.name, u.surname, COALESCE(u.avatar_path, '') AS avatar_path, u.phone,
       sr.price, sc.chat_id, COALESCE(lm.text, '') AS last_message, 'customer' AS my_role
FROM service s
JOIN service_confirmations sc ON sc.service_id = s.id
JOIN users u ON u.id = sc.performer_id
JOIN service_responses sr ON sr.service_id = s.id AND sr.user_id = sc.performer_id
LEFT JOIN service_confirmations c ON c.service_id = s.id AND c.confirmed = true
LEFT JOIN last_messages lm ON lm.chat_id = sc.chat_id
WHERE s.user_id = ?

UNION ALL

SELECT s.id, 'service' AS ad_type, s.name, s.status, c.performer_id,
       owner.id, owner.name, owner.surname, COALESCE(owner.avatar_path, '') AS avatar_path, owner.phone,
       sr.price, sc.chat_id, COALESCE(lm.text, '') AS last_message, 'performer' AS my_role
FROM service s
JOIN service_confirmations sc ON sc.service_id = s.id
JOIN users owner ON owner.id = s.user_id
JOIN service_responses sr ON sr.service_id = s.id AND sr.user_id = sc.performer_id
LEFT JOIN service_confirmations c ON c.service_id = s.id AND c.confirmed = true
LEFT JOIN last_messages lm ON lm.chat_id = sc.chat_id
WHERE sc.performer_id = ?

UNION ALL

SELECT ra.id, 'rent_ad' AS ad_type, ra.name, ra.status, c.performer_id,
       u.id, u.name, u.surname, COALESCE(u.avatar_path, '') AS avatar_path, u.phone,
       rar.price, rac.chat_id, COALESCE(lm.text, '') AS last_message, 'customer' AS my_role
FROM rent_ad ra
JOIN rent_ad_confirmations rac ON rac.rent_ad_id = ra.id
JOIN users u ON u.id = rac.performer_id
JOIN rent_ad_responses rar ON rar.rent_ad_id = ra.id AND rar.user_id = rac.performer_id
LEFT JOIN rent_ad_confirmations c ON c.rent_ad_id = ra.id AND c.confirmed = true
LEFT JOIN last_messages lm ON lm.chat_id = rac.chat_id
WHERE ra.user_id = ?

UNION ALL

SELECT ra.id, 'rent_ad' AS ad_type, ra.name, ra.status, c.performer_id,
       owner.id, owner.name, owner.surname, COALESCE(owner.avatar_path, '') AS avatar_path, owner.phone,
       rar.price, rac.chat_id, COALESCE(lm.text, '') AS last_message, 'performer' AS my_role
FROM rent_ad ra
JOIN rent_ad_confirmations rac ON rac.rent_ad_id = ra.id
JOIN users owner ON owner.id = ra.user_id
JOIN rent_ad_responses rar ON rar.rent_ad_id = ra.id AND rar.user_id = rac.performer_id
LEFT JOIN rent_ad_confirmations c ON c.rent_ad_id = ra.id AND c.confirmed = true
LEFT JOIN last_messages lm ON lm.chat_id = rac.chat_id
WHERE rac.performer_id = ?

UNION ALL

SELECT wa.id, 'work_ad' AS ad_type, wa.name, wa.status, c.performer_id,
       u.id, u.name, u.surname, COALESCE(u.avatar_path, '') AS avatar_path, u.phone,
       war.price, wac.chat_id, COALESCE(lm.text, '') AS last_message, 'customer' AS my_role
FROM work_ad wa
JOIN work_ad_confirmations wac ON wac.work_ad_id = wa.id
JOIN users u ON u.id = wac.performer_id
JOIN work_ad_responses war ON war.work_ad_id = wa.id AND war.user_id = wac.performer_id
LEFT JOIN work_ad_confirmations c ON c.work_ad_id = wa.id AND c.confirmed = true
LEFT JOIN last_messages lm ON lm.chat_id = wac.chat_id
WHERE wa.user_id = ?

UNION ALL

SELECT wa.id, 'work_ad' AS ad_type, wa.name, wa.status, c.performer_id,
       owner.id, owner.name, owner.surname, COALESCE(owner.avatar_path, '') AS avatar_path, owner.phone,
       war.price, wac.chat_id, COALESCE(lm.text, '') AS last_message, 'performer' AS my_role
FROM work_ad wa
JOIN work_ad_confirmations wac ON wac.work_ad_id = wa.id
JOIN users owner ON owner.id = wa.user_id
JOIN work_ad_responses war ON war.work_ad_id = wa.id AND war.user_id = wac.performer_id
LEFT JOIN work_ad_confirmations c ON c.work_ad_id = wa.id AND c.confirmed = true
LEFT JOIN last_messages lm ON lm.chat_id = wac.chat_id
WHERE wac.performer_id = ?

UNION ALL

SELECT r.id, 'rent' AS ad_type, r.name, r.status, c.performer_id,
       u.id, u.name, u.surname, COALESCE(u.avatar_path, '') AS avatar_path, u.phone,
       rr.price, rc.chat_id, COALESCE(lm.text, '') AS last_message, 'customer' AS my_role
FROM rent r
JOIN rent_confirmations rc ON rc.rent_id = r.id
JOIN users u ON u.id = rc.performer_id
JOIN rent_responses rr ON rr.rent_id = r.id AND rr.user_id = rc.performer_id
LEFT JOIN rent_confirmations c ON c.rent_id = r.id AND c.confirmed = true
LEFT JOIN last_messages lm ON lm.chat_id = rc.chat_id
WHERE r.user_id = ?

UNION ALL

SELECT r.id, 'rent' AS ad_type, r.name, r.status, c.performer_id,
       owner.id, owner.name, owner.surname, COALESCE(owner.avatar_path, '') AS avatar_path, owner.phone,
       rr.price, rc.chat_id, COALESCE(lm.text, '') AS last_message, 'performer' AS my_role
FROM rent r
JOIN rent_confirmations rc ON rc.rent_id = r.id
JOIN users owner ON owner.id = r.user_id
JOIN rent_responses rr ON rr.rent_id = r.id AND rr.user_id = rc.performer_id
LEFT JOIN rent_confirmations c ON c.rent_id = r.id AND c.confirmed = true
LEFT JOIN last_messages lm ON lm.chat_id = rc.chat_id
WHERE rc.performer_id = ?

UNION ALL

SELECT w.id, 'work' AS ad_type, w.name, w.status, c.performer_id,
       u.id, u.name, u.surname, COALESCE(u.avatar_path, '') AS avatar_path, u.phone,
       wr.price, wc.chat_id, COALESCE(lm.text, '') AS last_message, 'customer' AS my_role
FROM work w
JOIN work_confirmations wc ON wc.work_id = w.id
JOIN users u ON u.id = wc.performer_id
JOIN work_responses wr ON wr.work_id = w.id AND wr.user_id = wc.performer_id
LEFT JOIN work_confirmations c ON c.work_id = w.id AND c.confirmed = true
LEFT JOIN last_messages lm ON lm.chat_id = wc.chat_id
WHERE w.user_id = ?

UNION ALL

SELECT w.id, 'work' AS ad_type, w.name, w.status, c.performer_id,
       owner.id, owner.name, owner.surname, COALESCE(owner.avatar_path, '') AS avatar_path, owner.phone,
       wr.price, wc.chat_id, COALESCE(lm.text, '') AS last_message, 'performer' AS my_role
FROM work w
JOIN work_confirmations wc ON wc.work_id = w.id
JOIN users owner ON owner.id = w.user_id
JOIN work_responses wr ON wr.work_id = w.id AND wr.user_id = wc.performer_id
LEFT JOIN work_confirmations c ON c.work_id = w.id AND c.confirmed = true
LEFT JOIN last_messages lm ON lm.chat_id = wc.chat_id
WHERE wc.performer_id = ?

ORDER BY 1
`

	rows, err := r.Db.QueryContext(
		ctx, query,
		userID, userID, // ad
		userID, userID, // service
		userID, userID, // rent_ad
		userID, userID, // work_ad
		userID, userID, // rent
		userID, userID, // work
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.AdChats
	adIndex := make(map[string]int)

	for rows.Next() {
		var adID int
		var adType, adName, status string
		var confirmedPerformer sql.NullInt64
		var user models.ChatUser

		if err := rows.Scan(
			&adID, &adType, &adName, &status, &confirmedPerformer,
			&user.ID, &user.Name, &user.Surname, &user.AvatarPath, &user.Phone,
			&user.Price, &user.ChatID, &user.LastMessage, &user.MyRole, // ← тут новое поле
		); err != nil {
			return nil, err
		}

		if status != "in progress" {
			user.Phone = ""
		}

		if rating, err := getUserAverageRating(ctx, r.Db, user.ID); err == nil {
			user.ReviewRating = rating
		}
		if count, err := getUserTotalReviews(ctx, r.Db, user.ID); err == nil {
			user.ReviewsCount = count
		}

		var performerID *int
		if confirmedPerformer.Valid {
			pid := int(confirmedPerformer.Int64)
			performerID = &pid
		}

		key := fmt.Sprintf("%s:%d", adType, adID)
		if idx, ok := adIndex[key]; ok {
			result[idx].Users = append(result[idx].Users, user)
		} else {
			chatGroup := models.AdChats{
				AdName:      adName,
				Status:      status,
				PerformerID: performerID,
				Users:       []models.ChatUser{user},
			}
			chatGroup.SetIDByType(adType, adID)
			result = append(result, chatGroup)
			adIndex[key] = len(result) - 1
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil

}
