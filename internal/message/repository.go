package message

import (
	"context"
	"database/sql"
)

type Repository interface {
	Create(ctx context.Context, msg *Message) error
	GetByRoom(ctx context.Context, roomID uint64, limit int) ([]*Message, error)
	GetByRoomAfter(ctx context.Context, roomID uint64, afterID uint64, limit int) ([]*Message, error)
	GetByUser(ctx context.Context, userID uint64, limit int) ([]*Message, error)
}

type MySQLRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) Create(ctx context.Context, msg *Message) error {
	const query = `
        INSERT INTO messages (
			room_id, 
			sender_id, 
			content, 
			media_id)
	VALUES (?, ?, ?, ?)
    `

	result, err := r.db.ExecContext(
		ctx,
		query,
		msg.RoomID,
		msg.SenderID,
		msg.Content,
		msg.MediaID,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	msg.ID = uint64(id)
	return nil
}

func (r *MySQLRepository) GetByRoom(
	ctx context.Context,
	roomID uint64,
	limit int,
) ([]*Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	const query = `
        SELECT id, room_id, sender_id, content, media_id, created_at
        FROM messages
        WHERE room_id = ?
        ORDER BY created_at DESC
        LIMIT ?
    `

	return r.scanMessages(ctx, query, roomID, limit)
}

func (r *MySQLRepository) GetByRoomAfter(
	ctx context.Context,
	roomID uint64,
	afterID uint64,
	limit int,
) ([]*Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	const query = `
        SELECT id, room_id, sender_id, content, media_id, created_at
        FROM messages
        WHERE room_id = ? AND id > ?
        ORDER BY id ASC
        LIMIT ?
    `

	return r.scanMessages(ctx, query, roomID, afterID, limit)
}

func (r *MySQLRepository) GetByUser(
	ctx context.Context,
	userID uint64,
	limit int,
) ([]*Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	const query = `
        SELECT id, room_id, sender_id, content, media_id, created_at
        FROM messages
        WHERE sender_id = ?
        ORDER BY created_at DESC
        LIMIT ?
    `

	return r.scanMessages(ctx, query, userID, limit)
}

func (r *MySQLRepository) scanMessages(
	ctx context.Context,
	query string,
	args ...interface{},
) ([]*Message, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(
			&msg.ID,
			&msg.RoomID,
			&msg.SenderID,
			&msg.Content,
			&msg.MediaID,
			&msg.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}

	return messages, rows.Err()
}
