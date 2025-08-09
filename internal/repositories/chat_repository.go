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
