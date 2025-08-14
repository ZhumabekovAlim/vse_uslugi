package repositories

import (
	"context"
	"database/sql"
	"naimuBack/internal/models"
)

type ChatRepository struct {
	Db *sql.DB
}

func (r *ChatRepository) CreateChat(ctx context.Context, chat models.Chat) (int, error) {
	query := `INSERT INTO chats (user1_id, user2_id) VALUES (?, ?)`
	result, err := r.Db.ExecContext(ctx, query, chat.User1ID, chat.User2ID)
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
              SELECT a.id, a.name, u.id, u.name, u.surname, ar.price, ac.chat_id
FROM ad a
JOIN ad_confirmations ac ON ac.ad_id = a.id
JOIN users u ON u.id = ac.performer_id
JOIN ad_responses ar ON ar.ad_id = a.id AND ar.user_id = ac.performer_id
WHERE a.user_id = ?

UNION ALL

SELECT a.id, a.name, owner.id, owner.name, owner.surname, ar.price, ac.chat_id
FROM ad a
JOIN ad_confirmations ac ON ac.ad_id = a.id
JOIN users owner ON owner.id = a.user_id
JOIN ad_responses ar ON ar.ad_id = a.id AND ar.user_id = ac.performer_id
WHERE ac.performer_id = ?

UNION ALL

SELECT s.id, s.name, u.id, u.name, u.surname, sr.price, sc.chat_id
FROM service s
JOIN service_confirmations sc ON sc.service_id = s.id
JOIN users u ON u.id = sc.performer_id
JOIN service_responses sr ON sr.service_id = s.id AND sr.user_id = sc.performer_id
WHERE s.user_id = ?

UNION ALL

SELECT s.id, s.name, owner.id, owner.name, owner.surname, sr.price, sc.chat_id
FROM service s
JOIN service_confirmations sc ON sc.service_id = s.id
JOIN users owner ON owner.id = s.user_id
JOIN service_responses sr ON sr.service_id = s.id AND sr.user_id = sc.performer_id
WHERE sc.performer_id = ?

UNION ALL

SELECT ra.id, ra.name, u.id, u.name, u.surname, rar.price, rac.chat_id
FROM rent_ad ra
JOIN rent_ad_confirmations rac ON rac.rent_ad_id = ra.id
JOIN users u ON u.id = rac.performer_id
JOIN rent_ad_responses rar ON rar.rent_ad_id = ra.id AND rar.user_id = rac.performer_id
WHERE ra.user_id = ?

UNION ALL

SELECT ra.id, ra.name, owner.id, owner.name, owner.surname, rar.price, rac.chat_id
FROM rent_ad ra
JOIN rent_ad_confirmations rac ON rac.rent_ad_id = ra.id
JOIN users owner ON owner.id = ra.user_id
JOIN rent_ad_responses rar ON rar.rent_ad_id = ra.id AND rar.user_id = rac.performer_id
WHERE rac.performer_id = ?

UNION ALL

SELECT wa.id, wa.name, u.id, u.name, u.surname, war.price, wac.chat_id
FROM work_ad wa
JOIN work_ad_confirmations wac ON wac.work_ad_id = wa.id
JOIN users u ON u.id = wac.performer_id
JOIN work_ad_responses war ON war.work_ad_id = wa.id AND war.user_id = wac.performer_id
WHERE wa.user_id = ?

UNION ALL

SELECT wa.id, wa.name, owner.id, owner.name, owner.surname, war.price, wac.chat_id
FROM work_ad wa
JOIN work_ad_confirmations wac ON wac.work_ad_id = wa.id
JOIN users owner ON owner.id = wa.user_id
JOIN work_ad_responses war ON war.work_ad_id = wa.id AND war.user_id = wac.performer_id
WHERE wac.performer_id = ?

UNION ALL

SELECT r.id, r.name, u.id, u.name, u.surname, rr.price, rc.chat_id
FROM rent r
JOIN rent_confirmations rc ON rc.rent_id = r.id
JOIN users u ON u.id = rc.performer_id
JOIN rent_responses rr ON rr.rent_id = r.id AND rr.user_id = rc.performer_id
WHERE r.user_id = ?

UNION ALL

SELECT r.id, r.name, owner.id, owner.name, owner.surname, rr.price, rc.chat_id
FROM rent r
JOIN rent_confirmations rc ON rc.rent_id = r.id
JOIN users owner ON owner.id = r.user_id
JOIN rent_responses rr ON rr.rent_id = r.id AND rr.user_id = rc.performer_id
WHERE rc.performer_id = ?

UNION ALL

SELECT w.id, w.name, u.id, u.name, u.surname, wr.price, wc.chat_id
FROM work w
JOIN work_confirmations wc ON wc.work_id = w.id
JOIN users u ON u.id = wc.performer_id
JOIN work_responses wr ON wr.work_id = w.id AND wr.user_id = wc.performer_id
WHERE w.user_id = ?

UNION ALL

SELECT w.id, w.name, owner.id, owner.name, owner.surname, wr.price, wc.chat_id
FROM work w
JOIN work_confirmations wc ON wc.work_id = w.id
JOIN users owner ON owner.id = w.user_id
JOIN work_responses wr ON wr.work_id = w.id AND wr.user_id = wc.performer_id
WHERE wc.performer_id = ?

ORDER BY 1`

	rows, err := r.Db.QueryContext(ctx, query, userID, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.AdChats
	adIndex := make(map[int]int)

	for rows.Next() {
		var adID int
		var adName string
		var user models.ChatUser
		if err := rows.Scan(&adID, &adName, &user.ID, &user.Name, &user.Surname, &user.Price, &user.ChatID); err != nil {
			return nil, err
		}

		rating, err := getUserAverageRating(ctx, r.Db, user.ID)
		if err == nil {
			user.ReviewRating = rating
		}

		count, err := getUserTotalReviews(ctx, r.Db, user.ID)
		if err == nil {
			user.ReviewsCount = count
		}

		if idx, ok := adIndex[adID]; ok {
			result[idx].Users = append(result[idx].Users, user)
		} else {
			result = append(result, models.AdChats{
				AdID:   adID,
				AdName: adName,
				Users:  []models.ChatUser{user},
			})
			adIndex[adID] = len(result) - 1
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
