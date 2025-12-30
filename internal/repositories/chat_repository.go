package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

// GetBusinessWorkerChats returns base chats between a business account and its workers.
func (r *ChatRepository) GetBusinessWorkerChats(ctx context.Context, businessUserID int) ([]models.AdChats, error) {
	const query = `
WITH last_messages AS (
    SELECT m.chat_id, m.text, m.created_at
    FROM messages m
    JOIN (
        SELECT chat_id, MAX(created_at) AS max_created
        FROM messages
        GROUP BY chat_id
    ) t ON t.chat_id = m.chat_id AND t.max_created = m.created_at
)

SELECT bw.worker_user_id, bw.login, bw.status, bw.chat_id,
       u.name, u.surname, COALESCE(u.avatar_path, '') AS avatar_path, u.phone,
       COALESCE(lm.text, '') AS last_message, COALESCE(lm.created_at, c.created_at) AS last_message_at,
       c.created_at
FROM business_workers bw
JOIN chats c ON c.id = bw.chat_id
JOIN users u ON u.id = bw.worker_user_id
LEFT JOIN last_messages lm ON lm.chat_id = bw.chat_id
WHERE bw.business_user_id = ? AND (c.user1_id = ? OR c.user2_id = ?)
ORDER BY last_message_at DESC`

	rows, err := r.Db.QueryContext(ctx, query, businessUserID, businessUserID, businessUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []models.AdChats

	for rows.Next() {
		var workerUserID, chatID int
		var login, status, name, surname, avatarPath, phone, lastMessage string
		var lastMessageAt, createdAt sql.NullTime

		if err := rows.Scan(
			&workerUserID, &login, &status, &chatID,
			&name, &surname, &avatarPath, &phone,
			&lastMessage, &lastMessageAt,
			&createdAt,
		); err != nil {
			return nil, err
		}

		user := models.ChatUser{
			ID:            workerUserID,
			Name:          name,
			Surname:       surname,
			AvatarPath:    avatarPath,
			Phone:         phone,
			ProviderPhone: phone,
			ClientPhone:   phone,
			Price:         0,
			ChatID:        chatID,
			MyRole:        models.RoleBusinessWorker,
		}

		user.LastMessage = lastMessage
		if lastMessageAt.Valid {
			t := lastMessageAt.Time
			user.LastMessageAt = &t
		}

		if rating, err := getUserAverageRating(ctx, r.Db, workerUserID); err == nil {
			user.ReviewRating = rating
		}
		if count, err := getUserTotalReviews(ctx, r.Db, workerUserID); err == nil {
			user.ReviewsCount = count
		}

		performerID := workerUserID
		chat := models.AdChats{
			AdType:      "business_worker",
			AdName:      login,
			Status:      status,
			IsAuthor:    true,
			HidePhone:   false,
			PerformerID: &performerID,
			Users:       []models.ChatUser{user},
		}

		if createdAt.Valid {
			t := createdAt.Time
			chat.CreatedAt = &t
		}

		chats = append(chats, chat)
	}

	return chats, rows.Err()
}

// GetChatsByUserID retrieves chats grouped by advertisements for a specific author.
func (r *ChatRepository) GetChatsByUserID(ctx context.Context, userID int) ([]models.AdChats, error) {
	query := `
WITH target_users AS (
    SELECT ? AS id
    UNION
    SELECT worker_user_id FROM business_workers WHERE business_user_id = ?
),
last_messages AS (
    SELECT m.chat_id, m.text, m.created_at
    FROM messages m
    JOIN (
        SELECT chat_id, MAX(created_at) AS max_created
        FROM messages
        GROUP BY chat_id
    ) t ON t.chat_id = m.chat_id AND t.max_created = m.created_at
)

SELECT a.id, 'ad' AS ad_type, a.name, a.hide_phone, ac.status, CASE WHEN ac.confirmed THEN ac.performer_id END AS performer_id, 1 AS is_author,
       a.address, a.price, a.price_to, a.negotiable, a.on_site, a.description, a.work_time_from, a.work_time_to, a.latitude, a.longitude,
       '' AS rent_type, '' AS deposit, '' AS work_experience, '' AS schedule, '' AS distance_work, '' AS payment_period,
       a.images, a.videos, a.created_at,
       u.id, u.name, u.surname, COALESCE(u.avatar_path, '') AS avatar_path, u.phone,
       ar.price, ac.chat_id, COALESCE(lm.text, '') AS last_message, COALESCE(lm.created_at, c.created_at) AS last_message_at, 'customer' AS my_role,
       u.phone AS provider_phone, owner.phone AS client_phone
FROM ad a
JOIN ad_confirmations ac ON ac.ad_id = a.id
JOIN users u ON u.id = ac.performer_id
JOIN users owner ON owner.id = a.user_id
JOIN chats c ON c.id = ac.chat_id
JOIN ad_responses ar ON ar.ad_id = a.id AND ar.user_id = ac.performer_id
LEFT JOIN last_messages lm ON lm.chat_id = ac.chat_id
WHERE a.user_id IN (SELECT id FROM target_users)

UNION ALL

SELECT a.id, 'ad' AS ad_type, a.name, a.hide_phone, ac.status, CASE WHEN ac.confirmed THEN ac.performer_id END AS performer_id, 0 AS is_author,
       a.address, a.price, a.price_to, a.negotiable, a.on_site, a.description, a.work_time_from, a.work_time_to, a.latitude, a.longitude,
       '' AS rent_type, '' AS deposit, '' AS work_experience, '' AS schedule, '' AS distance_work, '' AS payment_period,
       a.images, a.videos, a.created_at,
       owner.id, owner.name, owner.surname, COALESCE(owner.avatar_path, '') AS avatar_path, owner.phone,
       ar.price, ac.chat_id, COALESCE(lm.text, '') AS last_message, COALESCE(lm.created_at, c.created_at) AS last_message_at, 'performer' AS my_role,
       performer.phone AS provider_phone, owner.phone AS client_phone
FROM ad a
JOIN ad_confirmations ac ON ac.ad_id = a.id
JOIN users owner ON owner.id = a.user_id
JOIN users performer ON performer.id = ac.performer_id
JOIN chats c ON c.id = ac.chat_id
JOIN ad_responses ar ON ar.ad_id = a.id AND ar.user_id = ac.performer_id
LEFT JOIN last_messages lm ON lm.chat_id = ac.chat_id
WHERE ac.performer_id IN (SELECT id FROM target_users)

UNION ALL

SELECT s.id, 'service' AS ad_type, s.name, s.hide_phone, sc.status, CASE WHEN sc.confirmed THEN sc.performer_id END AS performer_id, 1 AS is_author,
       s.address, s.price, s.price_to, s.negotiable, s.on_site, s.description, s.work_time_from, s.work_time_to, s.latitude, s.longitude,
       '' AS rent_type, '' AS deposit, '' AS work_experience, '' AS schedule, '' AS distance_work, '' AS payment_period,
       s.images, s.videos, s.created_at,
       u.id, u.name, u.surname, COALESCE(u.avatar_path, '') AS avatar_path, u.phone,
       sr.price, sc.chat_id, COALESCE(lm.text, '') AS last_message, COALESCE(lm.created_at, c.created_at) AS last_message_at, 'performer' AS my_role,
       owner.phone AS provider_phone, u.phone AS client_phone
FROM service s
JOIN service_confirmations sc ON sc.service_id = s.id
JOIN users u ON u.id = sc.client_id
JOIN users owner ON owner.id = s.user_id
JOIN chats c ON c.id = sc.chat_id
JOIN service_responses sr ON sr.service_id = s.id AND sr.user_id = sc.client_id
LEFT JOIN last_messages lm ON lm.chat_id = sc.chat_id
WHERE s.user_id IN (SELECT id FROM target_users)

UNION ALL

SELECT s.id, 'service' AS ad_type, s.name, s.hide_phone, sc.status, CASE WHEN sc.confirmed THEN sc.performer_id END AS performer_id, 0 AS is_author,
       s.address, s.price, s.price_to, s.negotiable, s.on_site, s.description, s.work_time_from, s.work_time_to, s.latitude, s.longitude,
       '' AS rent_type, '' AS deposit, '' AS work_experience, '' AS schedule, '' AS distance_work, '' AS payment_period,
       s.images, s.videos, s.created_at,
       owner.id, owner.name, owner.surname, COALESCE(owner.avatar_path, '') AS avatar_path, owner.phone,
       sr.price, sc.chat_id, COALESCE(lm.text, '') AS last_message, COALESCE(lm.created_at, c.created_at) AS last_message_at, 'customer' AS my_role,
       owner.phone AS provider_phone, client.phone AS client_phone
FROM service s
JOIN service_confirmations sc ON sc.service_id = s.id
JOIN users owner ON owner.id = s.user_id
JOIN users client ON client.id = sc.client_id
JOIN chats c ON c.id = sc.chat_id
JOIN service_responses sr ON sr.service_id = s.id AND sr.user_id = sc.client_id
LEFT JOIN last_messages lm ON lm.chat_id = sc.chat_id
WHERE sc.client_id IN (SELECT id FROM target_users)

UNION ALL

SELECT ra.id, 'rent_ad' AS ad_type, ra.name, ra.hide_phone, rac.status, CASE WHEN rac.confirmed THEN rac.performer_id END AS performer_id, 1 AS is_author,
       ra.address, ra.price, ra.price_to, ra.negotiable, NULL AS on_site, ra.description, ra.work_time_from, ra.work_time_to, ra.latitude, ra.longitude,
       ra.rent_type, ra.deposit, '' AS work_experience, '' AS schedule, '' AS distance_work, '' AS payment_period,
       ra.images, ra.videos, ra.created_at,
       u.id, u.name, u.surname, COALESCE(u.avatar_path, '') AS avatar_path, u.phone,
       rar.price, rac.chat_id, COALESCE(lm.text, '') AS last_message, COALESCE(lm.created_at, c.created_at) AS last_message_at, 'customer' AS my_role,
       u.phone AS provider_phone, owner.phone AS client_phone
FROM rent_ad ra
JOIN rent_ad_confirmations rac ON rac.rent_ad_id = ra.id
JOIN users u ON u.id = rac.performer_id
JOIN users owner ON owner.id = ra.user_id
JOIN chats c ON c.id = rac.chat_id
JOIN rent_ad_responses rar ON rar.rent_ad_id = ra.id AND rar.user_id = rac.performer_id
LEFT JOIN last_messages lm ON lm.chat_id = rac.chat_id
WHERE ra.user_id IN (SELECT id FROM target_users)

UNION ALL

SELECT ra.id, 'rent_ad' AS ad_type, ra.name, ra.hide_phone, rac.status, CASE WHEN rac.confirmed THEN rac.performer_id END AS performer_id, 0 AS is_author,
       ra.address, ra.price, ra.price_to, ra.negotiable, NULL AS on_site, ra.description, ra.work_time_from, ra.work_time_to, ra.latitude, ra.longitude,
       ra.rent_type, ra.deposit, '' AS work_experience, '' AS schedule, '' AS distance_work, '' AS payment_period,
       ra.images, ra.videos, ra.created_at,
       owner.id, owner.name, owner.surname, COALESCE(owner.avatar_path, '') AS avatar_path, owner.phone,
       rar.price, rac.chat_id, COALESCE(lm.text, '') AS last_message, COALESCE(lm.created_at, c.created_at) AS last_message_at, 'performer' AS my_role,
       performer.phone AS provider_phone, owner.phone AS client_phone
FROM rent_ad ra
JOIN rent_ad_confirmations rac ON rac.rent_ad_id = ra.id
JOIN users owner ON owner.id = ra.user_id
JOIN users performer ON performer.id = rac.performer_id
JOIN chats c ON c.id = rac.chat_id
JOIN rent_ad_responses rar ON rar.rent_ad_id = ra.id AND rar.user_id = rac.performer_id
LEFT JOIN last_messages lm ON lm.chat_id = rac.chat_id
WHERE rac.performer_id IN (SELECT id FROM target_users)

UNION ALL

SELECT wa.id, 'work_ad' AS ad_type, wa.name, wa.hide_phone, wac.status, CASE WHEN wac.confirmed THEN wac.performer_id END AS performer_id, 1 AS is_author,
       wa.address, wa.price, wa.price_to, wa.negotiable, NULL AS on_site, wa.description, wa.work_time_from, wa.work_time_to, wa.latitude, wa.longitude,
       '' AS rent_type, '' AS deposit, wa.work_experience, wa.schedule, wa.distance_work, wa.payment_period,
       wa.images, wa.videos, wa.created_at,
       u.id, u.name, u.surname, COALESCE(u.avatar_path, '') AS avatar_path, u.phone,
       war.price, wac.chat_id, COALESCE(lm.text, '') AS last_message, COALESCE(lm.created_at, c.created_at) AS last_message_at, 'customer' AS my_role,
       u.phone AS provider_phone, owner.phone AS client_phone
FROM work_ad wa
JOIN work_ad_confirmations wac ON wac.work_ad_id = wa.id
JOIN users u ON u.id = wac.performer_id
JOIN users owner ON owner.id = wa.user_id
JOIN chats c ON c.id = wac.chat_id
JOIN work_ad_responses war ON war.work_ad_id = wa.id AND war.user_id = wac.performer_id
LEFT JOIN last_messages lm ON lm.chat_id = wac.chat_id
WHERE wa.user_id IN (SELECT id FROM target_users)

UNION ALL

SELECT wa.id, 'work_ad' AS ad_type, wa.name, wa.hide_phone, wac.status, CASE WHEN wac.confirmed THEN wac.performer_id END AS performer_id, 0 AS is_author,
       wa.address, wa.price, wa.price_to, wa.negotiable, NULL AS on_site, wa.description, wa.work_time_from, wa.work_time_to, wa.latitude, wa.longitude,
       '' AS rent_type, '' AS deposit, wa.work_experience, wa.schedule, wa.distance_work, wa.payment_period,
       wa.images, wa.videos, wa.created_at,
       owner.id, owner.name, owner.surname, COALESCE(owner.avatar_path, '') AS avatar_path, owner.phone,
       war.price, wac.chat_id, COALESCE(lm.text, '') AS last_message, COALESCE(lm.created_at, c.created_at) AS last_message_at, 'performer' AS my_role,
       performer.phone AS provider_phone, owner.phone AS client_phone
FROM work_ad wa
JOIN work_ad_confirmations wac ON wac.work_ad_id = wa.id
JOIN users owner ON owner.id = wa.user_id
JOIN users performer ON performer.id = wac.performer_id
JOIN chats c ON c.id = wac.chat_id
JOIN work_ad_responses war ON war.work_ad_id = wa.id AND war.user_id = wac.performer_id
LEFT JOIN last_messages lm ON lm.chat_id = wac.chat_id
WHERE wac.performer_id IN (SELECT id FROM target_users)

UNION ALL

SELECT r.id, 'rent' AS ad_type, r.name, r.hide_phone, rc.status, CASE WHEN rc.confirmed THEN rc.performer_id END AS performer_id, 1 AS is_author,
       r.address, r.price, r.price_to, r.negotiable, NULL AS on_site, r.description, r.work_time_from, r.work_time_to, r.latitude, r.longitude,
       r.rent_type, r.deposit, '' AS work_experience, '' AS schedule, '' AS distance_work, '' AS payment_period,
       r.images, r.videos, r.created_at,
       u.id, u.name, u.surname, COALESCE(u.avatar_path, '') AS avatar_path, u.phone,
       rr.price, rc.chat_id, COALESCE(lm.text, '') AS last_message, COALESCE(lm.created_at, c.created_at) AS last_message_at, 'customer' AS my_role,
       u.phone AS provider_phone, owner.phone AS client_phone
FROM rent r
JOIN rent_confirmations rc ON rc.rent_id = r.id
JOIN users u ON u.id = rc.performer_id
JOIN users owner ON owner.id = r.user_id
JOIN chats c ON c.id = rc.chat_id
JOIN rent_responses rr ON rr.rent_id = r.id AND rr.user_id = rc.performer_id
LEFT JOIN last_messages lm ON lm.chat_id = rc.chat_id
WHERE r.user_id IN (SELECT id FROM target_users)

UNION ALL

SELECT r.id, 'rent' AS ad_type, r.name, r.hide_phone, rc.status, CASE WHEN rc.confirmed THEN rc.performer_id END AS performer_id, 0 AS is_author,
       r.address, r.price, r.price_to, r.negotiable, NULL AS on_site, r.description, r.work_time_from, r.work_time_to, r.latitude, r.longitude,
       r.rent_type, r.deposit, '' AS work_experience, '' AS schedule, '' AS distance_work, '' AS payment_period,
       r.images, r.videos, r.created_at,
       owner.id, owner.name, owner.surname, COALESCE(owner.avatar_path, '') AS avatar_path, owner.phone,
       rr.price, rc.chat_id, COALESCE(lm.text, '') AS last_message, COALESCE(lm.created_at, c.created_at) AS last_message_at, 'performer' AS my_role,
       performer.phone AS provider_phone, owner.phone AS client_phone
FROM rent r
JOIN rent_confirmations rc ON rc.rent_id = r.id
JOIN users owner ON owner.id = r.user_id
JOIN users performer ON performer.id = rc.performer_id
JOIN chats c ON c.id = rc.chat_id
JOIN rent_responses rr ON rr.rent_id = r.id AND rr.user_id = rc.performer_id
LEFT JOIN last_messages lm ON lm.chat_id = rc.chat_id
WHERE rc.performer_id IN (SELECT id FROM target_users)

UNION ALL

SELECT w.id, 'work' AS ad_type, w.name, w.hide_phone, wc.status, CASE WHEN wc.confirmed THEN wc.performer_id END AS performer_id, 1 AS is_author,
       w.address, w.price, w.price_to, w.negotiable, NULL AS on_site, w.description, w.work_time_from, w.work_time_to, w.latitude, w.longitude,
       '' AS rent_type, '' AS deposit, w.work_experience, w.schedule, w.distance_work, w.payment_period,
       w.images, w.videos, w.created_at,
       u.id, u.name, u.surname, COALESCE(u.avatar_path, '') AS avatar_path, u.phone,
       wr.price, wc.chat_id, COALESCE(lm.text, '') AS last_message, COALESCE(lm.created_at, c.created_at) AS last_message_at, 'performer' AS my_role,
       provider.phone AS provider_phone, u.phone AS client_phone
FROM work w
JOIN work_confirmations wc ON wc.work_id = w.id
JOIN users u ON u.id = wc.client_id
JOIN users provider ON provider.id = w.user_id
JOIN chats c ON c.id = wc.chat_id
JOIN work_responses wr ON wr.work_id = w.id AND wr.user_id = wc.client_id
LEFT JOIN last_messages lm ON lm.chat_id = wc.chat_id
WHERE w.user_id IN (SELECT id FROM target_users)

UNION ALL

SELECT w.id, 'work' AS ad_type, w.name, w.hide_phone, wc.status, CASE WHEN wc.confirmed THEN wc.performer_id END AS performer_id, 0 AS is_author,
       w.address, w.price, w.price_to, w.negotiable, NULL AS on_site, w.description, w.work_time_from, w.work_time_to, w.latitude, w.longitude,
       '' AS rent_type, '' AS deposit, w.work_experience, w.schedule, w.distance_work, w.payment_period,
       w.images, w.videos, w.created_at,
       owner.id, owner.name, owner.surname, COALESCE(owner.avatar_path, '') AS avatar_path, owner.phone,
       wr.price, wc.chat_id, COALESCE(lm.text, '') AS last_message, COALESCE(lm.created_at, c.created_at) AS last_message_at, 'customer' AS my_role,
       owner.phone AS provider_phone, client.phone AS client_phone
FROM work w
JOIN work_confirmations wc ON wc.work_id = w.id
JOIN users owner ON owner.id = w.user_id
JOIN users client ON client.id = wc.client_id
JOIN chats c ON c.id = wc.chat_id
JOIN work_responses wr ON wr.work_id = w.id AND wr.user_id = wc.client_id
LEFT JOIN last_messages lm ON lm.chat_id = wc.chat_id
WHERE wc.client_id IN (SELECT id FROM target_users)

ORDER BY last_message_at DESC
`

	rows, err := r.Db.QueryContext(ctx, query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.AdChats
	adIndex := make(map[string]int)

	for rows.Next() {
		var adID int
		var adType, adName, status string
		var address, description, workTimeFrom, workTimeTo, rentType, deposit, workExperience, schedule, distanceWork, paymentPeriod sql.NullString
		var latitude, longitude sql.NullString
		var price, priceTo sql.NullFloat64
		var negotiable, onSite sql.NullBool
		var imagesJSON, videosJSON []byte
		var createdAt sql.NullTime
		var hidePhone bool
		var confirmedPerformer sql.NullInt64
		var isAuthor bool
		var user models.ChatUser
		var lastMessageAt sql.NullTime

		if err := rows.Scan(
			&adID, &adType, &adName, &hidePhone, &status, &confirmedPerformer, &isAuthor,
			&address, &price, &priceTo, &negotiable, &onSite, &description, &workTimeFrom, &workTimeTo, &latitude, &longitude,
			&rentType, &deposit, &workExperience, &schedule, &distanceWork, &paymentPeriod,
			&imagesJSON, &videosJSON, &createdAt,
			&user.ID, &user.Name, &user.Surname, &user.AvatarPath, &user.Phone,
			&user.Price, &user.ChatID, &user.LastMessage, &lastMessageAt, &user.MyRole,
			&user.ProviderPhone, &user.ClientPhone,
		); err != nil {
			return nil, err
		}

		if lastMessageAt.Valid {
			t := lastMessageAt.Time
			user.LastMessageAt = &t
		}

		if rating, err := getUserAverageRating(ctx, r.Db, user.ID); err == nil {
			user.ReviewRating = rating
		}
		if count, err := getUserTotalReviews(ctx, r.Db, user.ID); err == nil {
			user.ReviewsCount = count
		}

		if review, err := getUserReviewForAd(ctx, r.Db, adType, adID, user.ID); err == nil {
			user.AdReview = review
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
				AdType:      adType,
				AdName:      adName,
				HidePhone:   hidePhone,
				Status:      status,
				IsAuthor:    isAuthor,
				PerformerID: performerID,
				Users:       []models.ChatUser{user},
			}

			if address.Valid {
				chatGroup.Address = address.String
			}
			if description.Valid {
				chatGroup.Description = description.String
			}
			if workTimeFrom.Valid {
				chatGroup.WorkTimeFrom = workTimeFrom.String
			}
			if workTimeTo.Valid {
				chatGroup.WorkTimeTo = workTimeTo.String
			}
			if rentType.Valid {
				chatGroup.RentType = rentType.String
			}
			if deposit.Valid {
				chatGroup.Deposit = deposit.String
			}
			if workExperience.Valid {
				chatGroup.WorkExperience = workExperience.String
			}
			if schedule.Valid {
				chatGroup.Schedule = schedule.String
			}
			if distanceWork.Valid {
				chatGroup.DistanceWork = distanceWork.String
			}
			if paymentPeriod.Valid {
				chatGroup.PaymentPeriod = paymentPeriod.String
			}

			if len(imagesJSON) > 0 {
				if err := json.Unmarshal(imagesJSON, &chatGroup.Images); err != nil {
					return nil, fmt.Errorf("decode images: %w", err)
				}
			}
			if len(videosJSON) > 0 {
				if err := json.Unmarshal(videosJSON, &chatGroup.Videos); err != nil {
					return nil, fmt.Errorf("decode videos: %w", err)
				}
			}
			if createdAt.Valid {
				t := createdAt.Time
				chatGroup.CreatedAt = &t
			}

			if price.Valid {
				chatGroup.Price = floatPtr(price.Float64)
			}
			if priceTo.Valid {
				chatGroup.PriceTo = floatPtr(priceTo.Float64)
			}
			if negotiable.Valid {
				chatGroup.Negotiable = negotiable.Bool
			}
			if onSite.Valid {
				chatGroup.OnSite = onSite.Bool
			}
			if latitude.Valid {
				chatGroup.Latitude = stringPtr(latitude.String)
			}
			if longitude.Valid {
				chatGroup.Longitude = stringPtr(longitude.String)
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

func floatPtr(v float64) *float64 {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

// getUserReviewForAd fetches a review left by a user for a specific advertisement type.
func getUserReviewForAd(ctx context.Context, db *sql.DB, adType string, adID, userID int) (*models.ChatUserReview, error) {
	var query string

	switch adType {
	case "ad":
		query = "SELECT user_id, rating, review FROM ad_reviews WHERE ad_id = ? AND user_id = ? LIMIT 1"
	case "service":
		query = "SELECT user_id, rating, review FROM reviews WHERE service_id = ? AND user_id = ? LIMIT 1"
	case "rent_ad":
		query = "SELECT user_id, rating, review FROM rent_ad_reviews WHERE rent_ad_id = ? AND user_id = ? LIMIT 1"
	case "work_ad":
		query = "SELECT user_id, rating, review FROM work_ad_reviews WHERE work_ad_id = ? AND user_id = ? LIMIT 1"
	case "rent":
		query = "SELECT user_id, rating, review FROM rent_reviews WHERE rent_id = ? AND user_id = ? LIMIT 1"
	case "work":
		query = "SELECT user_id, rating, review FROM work_reviews WHERE work_id = ? AND user_id = ? LIMIT 1"
	default:
		return nil, nil
	}

	var review models.ChatUserReview
	if err := db.QueryRowContext(ctx, query, adID, userID).Scan(&review.UserID, &review.Rating, &review.Review); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &review, nil
}
