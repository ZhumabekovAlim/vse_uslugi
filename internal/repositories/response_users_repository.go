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
	var includeReview func(*models.ResponseUser, sql.NullInt64, sql.NullFloat64, sql.NullString)

	switch itemType {
	case "service":
		includeReview = func(u *models.ResponseUser, userID sql.NullInt64, rating sql.NullFloat64, review sql.NullString) {
			if userID.Valid {
				u.ServiceReview = &models.ResponseUserReview{UserID: int(userID.Int64), Rating: rating.Float64, Review: review.String}
			}
		}
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

SELECT u.id, u.name, u.surname, u.avatar_path, u.review_rating, COALESCE(u.reviews_count, 0), u.phone,
       sr.price, sr.description, sr.created_at,
       COALESCE(sc.status, ''), COALESCE(sc.chat_id, 0), COALESCE(lm.text, ''), owner.phone, u.phone, 'performer' AS my_role,
       rv.user_id, rv.rating, rv.review
FROM service_responses sr
JOIN users u ON u.id = sr.user_id
JOIN service s ON s.id = sr.service_id
JOIN users owner ON owner.id = s.user_id
LEFT JOIN service_confirmations sc ON sc.service_id = sr.service_id AND sc.client_id = sr.user_id
LEFT JOIN last_messages lm ON lm.chat_id = sc.chat_id
 LEFT JOIN reviews rv ON rv.service_id = sr.service_id AND rv.user_id = sr.user_id
WHERE sr.service_id = ? AND (sc.status IS NULL OR sc.status = 'active')`
	case "ad":
		includeReview = func(u *models.ResponseUser, userID sql.NullInt64, rating sql.NullFloat64, review sql.NullString) {
			if userID.Valid {
				u.AdReview = &models.ResponseUserReview{UserID: int(userID.Int64), Rating: rating.Float64, Review: review.String}
			}
		}
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

SELECT u.id, u.name, u.surname, u.avatar_path, u.review_rating, COALESCE(u.reviews_count, 0), u.phone,
       ar.price, ar.description, ar.created_at,
       COALESCE(ac.status, ''), COALESCE(ac.chat_id, 0), COALESCE(lm.text, ''), u.phone, owner.phone, 'customer' AS my_role,
       arw.user_id, arw.rating, arw.review
FROM ad_responses ar
JOIN users u ON u.id = ar.user_id
JOIN ad a ON a.id = ar.ad_id
JOIN users owner ON owner.id = a.user_id
LEFT JOIN ad_confirmations ac ON ac.ad_id = ar.ad_id AND ac.performer_id = ar.user_id
LEFT JOIN last_messages lm ON lm.chat_id = ac.chat_id
 LEFT JOIN ad_reviews arw ON arw.ad_id = ar.ad_id AND arw.user_id = ar.user_id
WHERE ar.ad_id = ?`
	case "work":
		includeReview = func(u *models.ResponseUser, userID sql.NullInt64, rating sql.NullFloat64, review sql.NullString) {
			if userID.Valid {
				u.WorkReview = &models.ResponseUserReview{UserID: int(userID.Int64), Rating: rating.Float64, Review: review.String}
			}
		}
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

SELECT u.id, u.name, u.surname, u.avatar_path, u.review_rating, COALESCE(u.reviews_count, 0), u.phone,
       wr.price, wr.description, wr.created_at,
       COALESCE(wc.status, ''), COALESCE(wc.chat_id, 0), COALESCE(lm.text, ''), provider.phone, u.phone, 'performer' AS my_role,
       wrv.user_id, wrv.rating, wrv.review
FROM work_responses wr
JOIN users u ON u.id = wr.user_id
JOIN work w ON w.id = wr.work_id
JOIN users provider ON provider.id = w.user_id
LEFT JOIN work_confirmations wc ON wc.work_id = wr.work_id AND wc.client_id = wr.user_id
LEFT JOIN last_messages lm ON lm.chat_id = wc.chat_id
 LEFT JOIN work_reviews wrv ON wrv.work_id = wr.work_id AND wrv.user_id = wr.user_id
WHERE wr.work_id = ? AND (wc.status IS NULL OR wc.status = 'active')`
	case "work_ad":
		includeReview = func(u *models.ResponseUser, userID sql.NullInt64, rating sql.NullFloat64, review sql.NullString) {
			if userID.Valid {
				u.WorkAdReview = &models.ResponseUserReview{UserID: int(userID.Int64), Rating: rating.Float64, Review: review.String}
			}
		}
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

SELECT u.id, u.name, u.surname, u.avatar_path, u.review_rating, COALESCE(u.reviews_count, 0), u.phone,
       war.price, war.description, war.created_at,
       COALESCE(wac.status, ''), COALESCE(wac.chat_id, 0), COALESCE(lm.text, ''), u.phone, owner.phone, 'customer' AS my_role,
       wadr.user_id, wadr.rating, wadr.review
FROM work_ad_responses war
JOIN users u ON u.id = war.user_id
JOIN work_ad wa ON wa.id = war.work_ad_id
JOIN users owner ON owner.id = wa.user_id
LEFT JOIN work_ad_confirmations wac ON wac.work_ad_id = war.work_ad_id AND wac.performer_id = war.user_id
LEFT JOIN last_messages lm ON lm.chat_id = wac.chat_id
 LEFT JOIN work_ad_reviews wadr ON wadr.work_ad_id = war.work_ad_id AND wadr.user_id = war.user_id
WHERE war.work_ad_id = ?`
	case "rent":
		includeReview = func(u *models.ResponseUser, userID sql.NullInt64, rating sql.NullFloat64, review sql.NullString) {
			if userID.Valid {
				u.RentReview = &models.ResponseUserReview{UserID: int(userID.Int64), Rating: rating.Float64, Review: review.String}
			}
		}
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

SELECT u.id, u.name, u.surname, u.avatar_path, u.review_rating, COALESCE(u.reviews_count, 0), u.phone,
       rr.price, rr.description, rr.created_at,
       COALESCE(rc.status, ''), COALESCE(rc.chat_id, 0), COALESCE(lm.text, ''), u.phone, owner.phone, 'customer' AS my_role,
       rrv.user_id, rrv.rating, rrv.review
FROM rent_responses rr
JOIN users u ON u.id = rr.user_id
JOIN rent r ON r.id = rr.rent_id
JOIN users owner ON owner.id = r.user_id
LEFT JOIN rent_confirmations rc ON rc.rent_id = rr.rent_id AND rc.performer_id = rr.user_id
LEFT JOIN last_messages lm ON lm.chat_id = rc.chat_id
 LEFT JOIN rent_reviews rrv ON rrv.rent_id = rr.rent_id AND rrv.user_id = rr.user_id
WHERE rr.rent_id = ? AND (rc.status IS NULL OR rc.status = 'active')`
	case "rent_ad":
		includeReview = func(u *models.ResponseUser, userID sql.NullInt64, rating sql.NullFloat64, review sql.NullString) {
			if userID.Valid {
				u.RentAdReview = &models.ResponseUserReview{UserID: int(userID.Int64), Rating: rating.Float64, Review: review.String}
			}
		}
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

SELECT u.id, u.name, u.surname, u.avatar_path, u.review_rating, COALESCE(u.reviews_count, 0), u.phone,
       rar.price, rar.description, rar.created_at,
       COALESCE(rac.status, ''), COALESCE(rac.chat_id, 0), COALESCE(lm.text, ''), u.phone, owner.phone, 'customer' AS my_role,
       radr.user_id, radr.rating, radr.review
FROM rent_ad_responses rar
JOIN users u ON u.id = rar.user_id
JOIN rent_ad ra ON ra.id = rar.rent_ad_id
JOIN users owner ON owner.id = ra.user_id
LEFT JOIN rent_ad_confirmations rac ON rac.rent_ad_id = rar.rent_ad_id AND rac.performer_id = rar.user_id
LEFT JOIN last_messages lm ON lm.chat_id = rac.chat_id
 LEFT JOIN rent_ad_reviews radr ON radr.rent_ad_id = rar.rent_ad_id AND radr.user_id = rar.user_id
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
		var reviewUserID sql.NullInt64
		var reviewRating sql.NullFloat64
		var reviewText sql.NullString

		if err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Surname,
			&u.AvatarPath,
			&u.ReviewRating,
			&u.ReviewsCount,
			&u.Phone,
			&u.Price,
			&u.Description,
			&u.CreatedAt,
			&u.Status,
			&u.ChatID,
			&u.LastMessage,
			&u.ProviderPhone,
			&u.ClientPhone,
			&u.MyRole,
			&reviewUserID,
			&reviewRating,
			&reviewText,
		); err != nil {
			return nil, err
		}

		if includeReview != nil {
			includeReview(&u, reviewUserID, reviewRating, reviewText)
		}

		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

// GetItemInfo returns base info about an item for response aggregation.
func (r *ResponseUsersRepository) GetItemInfo(ctx context.Context, itemType string, itemID int) (models.ItemResponse, error) {
	var query string
	switch itemType {
	case "service":
		query = `SELECT s.id, s.name, sc.performer_id, COALESCE(sc.status, '')
                 FROM service s
                 LEFT JOIN service_confirmations sc ON sc.service_id = s.id
                 WHERE s.id = ?
                 ORDER BY sc.created_at DESC
                 LIMIT 1`
	case "ad":
		query = `SELECT a.id, a.name, ac.performer_id, COALESCE(ac.status, '')
                 FROM ad a
                 LEFT JOIN ad_confirmations ac ON ac.ad_id = a.id
                 WHERE a.id = ?
                 ORDER BY ac.created_at DESC
                 LIMIT 1`
	case "rent":
		query = `SELECT r.id, r.name, rc.performer_id, COALESCE(rc.status, '')
                 FROM rent r
                 LEFT JOIN rent_confirmations rc ON rc.rent_id = r.id
                 WHERE r.id = ?
                 ORDER BY rc.created_at DESC
                 LIMIT 1`
	case "work":
		query = `SELECT w.id, w.name, wc.performer_id, COALESCE(wc.status, '')
                 FROM work w
                 LEFT JOIN work_confirmations wc ON wc.work_id = w.id
                 WHERE w.id = ?
                 ORDER BY wc.created_at DESC
                 LIMIT 1`
	case "rent_ad":
		query = `SELECT ra.id, ra.name, rac.performer_id, COALESCE(rac.status, '')
                 FROM rent_ad ra
                 LEFT JOIN rent_ad_confirmations rac ON rac.rent_ad_id = ra.id
                 WHERE ra.id = ?
                 ORDER BY rac.created_at DESC
                 LIMIT 1`
	case "work_ad":
		query = `SELECT wa.id, wa.name, wac.performer_id, COALESCE(wac.status, '')
                 FROM work_ad wa
                 LEFT JOIN work_ad_confirmations wac ON wac.work_ad_id = wa.id
                 WHERE wa.id = ?
                 ORDER BY wac.created_at DESC
                 LIMIT 1`
	default:
		return models.ItemResponse{}, errors.New("unknown item type")
	}

	row := r.DB.QueryRowContext(ctx, query, itemID)
	var performerID sql.NullInt64
	var response models.ItemResponse
	response.ItemType = itemType

	if err := row.Scan(&response.ItemID, &response.ItemName, &performerID, &response.Status); err != nil {
		return models.ItemResponse{}, err
	}

	if performerID.Valid {
		value := int(performerID.Int64)
		response.PerformerID = &value
	}

	return response, nil
}
