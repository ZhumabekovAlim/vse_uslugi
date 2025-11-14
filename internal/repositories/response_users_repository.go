package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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

            SELECT u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0), sr.price, sr.description, sr.created_at, sc.chat_id

            FROM service_responses sr
            JOIN users u ON u.id = sr.user_id
            LEFT JOIN service_confirmations sc ON sc.service_id = sr.service_id 
            WHERE sr.service_id = ? AND sc.status != 'done' AND sc.status != 'in progress'`
	case "ad":
		query = `

            SELECT u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0), ar.price, ar.description, ar.created_at, ac.chat_id

            FROM ad_responses ar
            JOIN users u ON u.id = ar.user_id
            LEFT JOIN ad_confirmations ac ON ac.ad_id = ar.ad_id AND ac.client_id = ar.user_id
            WHERE ar.ad_id = ? `
	case "work":
		query = `

            SELECT u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0), wr.price, wr.description, wr.created_at, wc.chat_id

            FROM work_responses wr
            JOIN users u ON u.id = wr.user_id
            LEFT JOIN work_confirmations wc ON wc.work_id = wr.work_id
            WHERE wr.work_id = ? AND wc.status != 'done' AND wc.status != 'in progress'`
	case "work_ad":
		query = `
            SELECT u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0), war.price, war.description, war.created_at, wac.chat_id

            FROM work_ad_responses war
            JOIN users u ON u.id = war.user_id
            LEFT JOIN work_ad_confirmations wac ON wac.work_ad_id = war.work_ad_id AND wac.client_id = war.user_id
            WHERE war.work_ad_id = ?`
	case "rent":
		query = `

            SELECT u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0), rr.price, rr.description, rr.created_at, rc.chat_id

            FROM rent_responses rr
            JOIN users u ON u.id = rr.user_id
            LEFT JOIN rent_confirmations rc ON rc.rent_id = rr.rent_id
            WHERE rr.rent_id = ? AND rc.status != 'done' AND rc.status != 'in progress'`
	case "rent_ad":
		query = `

            SELECT u.id, u.name, u.surname, COALESCE(u.avatar_path, ''), COALESCE(u.review_rating, 0), rar.price, rar.description, rar.created_at, rac.chat_id

            FROM rent_ad_responses rar
            JOIN users u ON u.id = rar.user_id
            LEFT JOIN rent_ad_confirmations rac ON rac.rent_ad_id = rar.rent_ad_id
            WHERE rar.rent_ad_id = ?`
	default:
		return nil, errors.New("unknown item type")
	}

	rows, err := r.DB.QueryContext(ctx, query, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fmt.Println("test: ", rows)

	var users []models.ResponseUser
	for rows.Next() {
		var u models.ResponseUser
		if err := rows.Scan(&u.ID, &u.Name, &u.Surname, &u.AvatarPath, &u.Rating, &u.Price, &u.Description, &u.CreatedAt, &u.ChatID); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	fmt.Println("Fetched users:", users)
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}
