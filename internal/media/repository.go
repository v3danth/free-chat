package media

import (
	"context"
	"database/sql"
	"errors"

	"github.com/v3danth/free-chat/internal/database"
)

type Repository interface {
	Create(ctx context.Context, m Media) (Media, error)
	GetByID(ctx context.Context, id uint64) (Media, error)
	// MarkRemoved soft-deletes the row and returns it as it was.
	MarkRemoved(ctx context.Context, id uint64) (Media, error)
	BanHash(ctx context.Context, sha []byte, actorID uint64) error
	IsHashBanned(ctx context.Context, sha []byte) (bool, error)
}

type MySQLRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) Create(ctx context.Context, m Media) (Media, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO media (owner_id, file_key, sha256, width, height, bytes)
		VALUES (?, ?, ?, ?, ?, ?)`, m.OwnerID, m.Key, m.SHA256, m.Width, m.Height, m.Bytes)
	if err != nil {
		return Media{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Media{}, err
	}
	return r.GetByID(ctx, uint64(id))
}

func (r *MySQLRepository) GetByID(ctx context.Context, id uint64) (Media, error) {
	var (
		m       Media
		removed sql.NullTime
	)
	err := r.db.QueryRowContext(ctx, `
		SELECT id, owner_id, file_key, sha256, width, height, bytes, removed_at, created_at
		FROM media WHERE id = ?`, id).Scan(
		&m.ID, &m.OwnerID, &m.Key, &m.SHA256, &m.Width, &m.Height, &m.Bytes, &removed, &m.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Media{}, ErrNotFound
	}
	m.RemovedAt = database.TimePtr(removed)
	return m, err
}

func (r *MySQLRepository) MarkRemoved(ctx context.Context, id uint64) (Media, error) {
	m, err := r.GetByID(ctx, id)
	if err != nil {
		return Media{}, err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE media SET removed_at = NOW() WHERE id = ? AND removed_at IS NULL`, id)
	return m, err
}

func (r *MySQLRepository) BanHash(ctx context.Context, sha []byte, actorID uint64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO banned_media_hashes (sha256, created_by) VALUES (?, NULLIF(?, 0))
		ON DUPLICATE KEY UPDATE sha256 = sha256`, sha, actorID)
	return err
}

func (r *MySQLRepository) IsHashBanned(ctx context.Context, sha []byte) (bool, error) {
	var banned bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM banned_media_hashes WHERE sha256 = ?)`, sha).Scan(&banned)
	return banned, err
}
