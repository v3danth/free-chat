package media

import (
	"context"
	"database/sql"
)

type Repository interface {
	CreateMedia(ctx context.Context, m *Media) error
	GetMediaByID(ctx context.Context, id uint64) (*Media, error)
	GetMediaByUser(ctx context.Context, userID uint64, limit int) ([]*Media, error)
	DeleteMedia(ctx context.Context, id uint64) error
	IncrementFlagCount(ctx context.Context, id uint64) error
	SetFlagged(ctx context.Context, id uint64, flagged bool) error

	CreateFlag(ctx context.Context, flag *Flag) error
	GetFlagByMediaAndReporter(ctx context.Context, mediaID, reporterID uint64) (*Flag, error)
	HasUserFlagged(ctx context.Context, mediaID, reporterID uint64) (bool, error)
}

type MySQLRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) CreateMedia(ctx context.Context, m *Media) error {
	const query = `
        INSERT INTO media (
            user_id, media_type, file_name, file_path, file_size,
            mime_type, width, height, duration_ms
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	result, err := r.db.ExecContext(
		ctx,
		query,
		m.UserID,
		m.MediaType,
		m.FileName,
		m.FilePath,
		m.FileSize,
		m.MimeType,
		m.Width,
		m.Height,
		m.DurationMs,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	m.ID = uint64(id)
	return nil
}

func (r *MySQLRepository) GetMediaByID(ctx context.Context, id uint64) (*Media, error) {
	const query = `
        SELECT id, user_id, media_type, file_name, file_path, file_size,
               mime_type, width, height, duration_ms, flag_count, is_flagged, created_at
        FROM media
        WHERE id = ?
    `

	var m Media
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&m.ID,
		&m.UserID,
		&m.MediaType,
		&m.FileName,
		&m.FilePath,
		&m.FileSize,
		&m.MimeType,
		&m.Width,
		&m.Height,
		&m.DurationMs,
		&m.FlagCount,
		&m.IsFlagged,
		&m.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (r *MySQLRepository) GetMediaByUser(
	ctx context.Context,
	userID uint64,
	limit int,
) ([]*Media, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	const query = `
        SELECT id, user_id, media_type, file_name, file_path, file_size,
               mime_type, width, height, duration_ms, flag_count, is_flagged, created_at
        FROM media
        WHERE user_id = ?
        ORDER BY created_at DESC
        LIMIT ?
    `

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var media []*Media
	for rows.Next() {
		var m Media
		if err := rows.Scan(
			&m.ID,
			&m.UserID,
			&m.MediaType,
			&m.FileName,
			&m.FilePath,
			&m.FileSize,
			&m.MimeType,
			&m.Width,
			&m.Height,
			&m.DurationMs,
			&m.FlagCount,
			&m.IsFlagged,
			&m.CreatedAt,
		); err != nil {
			return nil, err
		}
		media = append(media, &m)
	}

	return media, rows.Err()
}

func (r *MySQLRepository) DeleteMedia(ctx context.Context, id uint64) error {
	const query = `DELETE FROM media WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *MySQLRepository) IncrementFlagCount(ctx context.Context, id uint64) error {
	const query = `UPDATE media SET flag_count = flag_count + 1 WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *MySQLRepository) SetFlagged(ctx context.Context, id uint64, flagged bool) error {
	const query = `UPDATE media SET is_flagged = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, flagged, id)
	return err
}

func (r *MySQLRepository) CreateFlag(ctx context.Context, flag *Flag) error {
	const query = `
        INSERT INTO media_flags (media_id, reporter_id, reason)
        VALUES (?, ?, ?)
    `

	result, err := r.db.ExecContext(ctx, query, flag.MediaID, flag.ReporterID, flag.Reason)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	flag.ID = uint64(id)
	return nil
}

func (r *MySQLRepository) GetFlagByMediaAndReporter(
	ctx context.Context,
	mediaID,
	reporterID uint64,
) (*Flag, error) {
	const query = `
        SELECT id, media_id, reporter_id, reason, created_at
        FROM media_flags
        WHERE media_id = ? AND reporter_id = ?
    `

	var flag Flag
	err := r.db.QueryRowContext(ctx, query, mediaID, reporterID).Scan(
		&flag.ID,
		&flag.MediaID,
		&flag.ReporterID,
		&flag.Reason,
		&flag.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &flag, nil
}

func (r *MySQLRepository) HasUserFlagged(
	ctx context.Context,
	mediaID,
	reporterID uint64,
) (bool, error) {
	const query = `
        SELECT COUNT(*) FROM media_flags
        WHERE media_id = ? AND reporter_id = ?
    `

	var count int
	err := r.db.QueryRowContext(ctx, query, mediaID, reporterID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
