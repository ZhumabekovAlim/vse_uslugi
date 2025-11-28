package repositories

import (
	"context"
	"database/sql"
	"errors"
	"naimuBack/internal/models"
	"strconv"
	"time"
)

type MessageRepository struct {
	Db *sql.DB
}

func (r *MessageRepository) CreateMessage(ctx context.Context, message models.Message) (string, error) {
	chatID := message.ChatID
	if chatID == 0 {
		// Пытаемся найти существующий чат между двумя пользователями
		queryChat := `
                SELECT id
                FROM chats
                WHERE (user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)
                ORDER BY id ASC
                LIMIT 1`
		err := r.Db.QueryRowContext(ctx, queryChat, message.SenderID, message.ReceiverID, message.ReceiverID, message.SenderID).Scan(&chatID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Если чат не найден, создаём новый
				createChatQuery := `
                                INSERT INTO chats (user1_id, user2_id, created_at)
                                VALUES (?, ?, ?)`
				res, err := r.Db.ExecContext(ctx, createChatQuery, message.SenderID, message.ReceiverID, time.Now())
				if err != nil {
					return "", err
				}
				newChatID, err := res.LastInsertId()
				if err != nil {
					return "", err
				}
				chatID = int(newChatID)
			} else {
				return "", err
			}
		}
	}

	// Теперь вставляем сообщение, используя найденный или созданный chat_id
	insertMessageQuery := `
        INSERT INTO messages (sender_id, receiver_id, text, created_at, chat_id)
        VALUES (?, ?, ?, ?, ?)`
	res, err := r.Db.ExecContext(ctx, insertMessageQuery, message.SenderID, message.ReceiverID, message.Text, time.Now(), chatID)
	if err != nil {
		return "", err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

func (r *MessageRepository) GetMessagesForChat(ctx context.Context, chatID, page, pageSize int) ([]models.Message, error) {
	var messages []models.Message
	offset := (page - 1) * pageSize

	query := `SELECT id, sender_id, receiver_id, text, created_at, chat_id FROM messages WHERE chat_id = ? ORDER BY created_at ASC LIMIT ? OFFSET ?`

	rows, err := r.Db.QueryContext(ctx, query, chatID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var message models.Message
		err := rows.Scan(&message.ID, &message.SenderID, &message.ReceiverID, &message.Text, &message.CreatedAt, &message.ChatID)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *MessageRepository) DeleteMessage(ctx context.Context, messageID int) error {
	query := `DELETE FROM messages WHERE id=?`
	_, err := r.Db.ExecContext(ctx, query, messageID)
	return err
}

func (r *MessageRepository) GetMessagesByUserIDs(ctx context.Context, user1ID, user2ID, page, pageSize int) ([]models.Message, error) {
	offset := (page - 1) * pageSize
	query := `
        SELECT id, sender_Id, receiver_Id, text, created_at, chat_id
        FROM messages
        WHERE (sender_Id = ? AND receiver_Id = ?) OR (sender_Id = ? AND receiver_Id = ?)
        ORDER BY created_at DESC
        LIMIT ? OFFSET ?`

	rows, err := r.Db.QueryContext(ctx, query, user1ID, user2ID, user2ID, user1ID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []models.Message{}
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.ID, &msg.SenderID, &msg.ReceiverID, &msg.Text, &msg.CreatedAt, &msg.ChatID); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// GetChatParticipants returns user ids for the chat.
func (r *MessageRepository) GetChatParticipants(ctx context.Context, chatID int) (int, int, error) {
	var user1ID, user2ID int
	err := r.Db.QueryRowContext(ctx, `SELECT user1_id, user2_id FROM chats WHERE id = ?`, chatID).Scan(&user1ID, &user2ID)
	if err != nil {
		return 0, 0, err
	}
	return user1ID, user2ID, nil
}
