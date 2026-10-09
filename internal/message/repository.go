package message

import (
	"context"
	"database/sql"
	"strings"
)

type Repository interface {
	InsertBatch(ctx context.Context, msgs []Message) error
	// Recent returns up to limit visible messages of a room, newest first.
	Recent(ctx context.Context, roomID uint64, limit int) ([]Entry, error)
	Get(ctx context.Context, id uint64) (Message, error)
	// Before returns up to n messages of m's conversation sent before it,
	// newest first; used as context for moderators.
	Before(ctx context.Context, m Message, n int) ([]Message, error)
	BySender(ctx context.Context, senderID uint64, limit int) ([]Message, error)
	Hide(ctx context.Context, id uint64) error
	LiveRooms(ctx context.Context) ([]Room, error)
}

type MySQLRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

const selectMessage = `
	SELECT id, COALESCE(room_id, 0), COALESCE(recipient_id, 0), sender_id, sender_name, body, COALESCE(media_id, 0)
	FROM messages `

// InsertBatch writes every message in one statement: one round trip and
// one commit for the whole batch.
func (r *MySQLRepository) InsertBatch(ctx context.Context, msgs []Message) error {
	if len(msgs) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO messages (id, room_id, recipient_id, sender_id, sender_name, body, media_id) VALUES `)
	args := make([]any, 0, len(msgs)*7)
	for i, m := range msgs {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("(?, NULLIF(?, 0), NULLIF(?, 0), ?, ?, ?, NULLIF(?, 0))")
		args = append(args, m.ID, m.RoomID, m.RecipientID, m.SenderID, m.SenderName, m.Body, m.MediaID)
	}
	_, err := r.db.ExecContext(ctx, b.String(), args...)
	return err
}

func (r *MySQLRepository) Recent(ctx context.Context, roomID uint64, limit int) ([]Entry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.room_id, m.sender_id, m.sender_name, m.body, COALESCE(m.media_id, 0), COALESCE(md.file_key, '')
		FROM messages m
		LEFT JOIN media md ON md.id = m.media_id AND md.removed_at IS NULL
		WHERE m.room_id = ? AND m.hidden_at IS NULL
		ORDER BY m.id DESC
		LIMIT ?`, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.RoomID, &e.SenderID, &e.SenderName, &e.Body, &e.MediaID, &e.MediaKey); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *MySQLRepository) Get(ctx context.Context, id uint64) (Message, error) {
	msgs, err := r.list(ctx, selectMessage+`WHERE id = ?`, id)
	if err != nil {
		return Message{}, err
	}
	if len(msgs) == 0 {
		return Message{}, ErrNotFound
	}
	return msgs[0], nil
}

func (r *MySQLRepository) Before(ctx context.Context, m Message, n int) ([]Message, error) {
	if !m.IsDirect() {
		return r.list(ctx, selectMessage+`WHERE room_id = ? AND id < ? ORDER BY id DESC LIMIT ?`, m.RoomID, m.ID, n)
	}
	return r.list(ctx, selectMessage+`
		WHERE ((sender_id = ? AND recipient_id = ?) OR (sender_id = ? AND recipient_id = ?)) AND id < ?
		ORDER BY id DESC LIMIT ?`, m.SenderID, m.RecipientID, m.RecipientID, m.SenderID, m.ID, n)
}

func (r *MySQLRepository) BySender(ctx context.Context, senderID uint64, limit int) ([]Message, error) {
	return r.list(ctx, selectMessage+`WHERE sender_id = ? ORDER BY id DESC LIMIT ?`, senderID, limit)
}

func (r *MySQLRepository) Hide(ctx context.Context, id uint64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE messages SET hidden_at = COALESCE(hidden_at, NOW()) WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MySQLRepository) LiveRooms(ctx context.Context) ([]Room, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, slug, name FROM rooms WHERE is_live ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Room
	for rows.Next() {
		var room Room
		if err := rows.Scan(&room.ID, &room.Slug, &room.Name); err != nil {
			return nil, err
		}
		out = append(out, room)
	}
	return out, rows.Err()
}

func (r *MySQLRepository) list(ctx context.Context, query string, args ...any) ([]Message, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.RoomID, &m.RecipientID, &m.SenderID, &m.SenderName, &m.Body, &m.MediaID); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
