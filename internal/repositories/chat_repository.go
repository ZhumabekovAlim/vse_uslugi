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
               SELECT a.id, a.name,
                      u.id, u.name, u.surname,
                      ar.price,
                      c.id
               FROM ad a
               JOIN ad_responses ar ON ar.ad_id = a.id
               JOIN users u ON u.id = ar.user_id
               JOIN chats c ON ((c.user1_id = a.user_id AND c.user2_id = u.id) OR (c.user1_id = u.id AND c.user2_id = a.user_id))
               WHERE a.user_id = ?
               ORDER BY a.id`

	rows, err := r.Db.QueryContext(ctx, query, userID)
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
