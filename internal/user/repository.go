package user

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/database"
)

var (
	ErrNotFound      = apperr.New(apperr.NotFound, "user not found")
	ErrEmailTaken    = apperr.New(apperr.Conflict, "email already registered")
	ErrNameTaken     = apperr.New(apperr.Conflict, "that name belongs to a member")
	ErrPhotoNotYours = apperr.New(apperr.Forbidden, "you can only use a photo you uploaded")
)

const memberNameIndex = "uniq_member_name"

type Repository interface {
	// Create inserts u and returns the stored row, including DB defaults.
	Create(ctx context.Context, u User) (User, error)
	GetByID(ctx context.Context, id uint64) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	// UpdateCard changes the editable card fields; photoID must belong to
	// the user and not be removed, or ErrPhotoNotYours.
	UpdateCard(ctx context.Context, id uint64, intent Intent, about, location string, photoID *uint64) error
	Touch(ctx context.Context, id uint64) error
	SetRole(ctx context.Context, id uint64, role Role) error
	// SetBan sets or clears a ban; banning also revokes every token.
	SetBan(ctx context.Context, id uint64, until *time.Time) error
	SetMute(ctx context.Context, id uint64, until *time.Time) error
}

type MySQLRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

const selectUser = `
	SELECT u.id, u.kind, u.role, u.name, u.gender, u.age, u.intent, u.about, u.location,
	       u.country_code, u.photo_media_id, m.file_key, u.email, u.password_hash,
	       u.token_version, u.banned_until, u.muted_until, u.ip_hash, u.last_seen_at, u.created_at
	FROM users u
	LEFT JOIN media m ON m.id = u.photo_media_id AND m.removed_at IS NULL `

func (r *MySQLRepository) Create(ctx context.Context, u User) (User, error) {
	const query = `
		INSERT INTO users (kind, role, name, gender, age, intent, about, location,
		                   country_code, email, password_hash, ip_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	p := u.Profile
	res, err := r.db.ExecContext(ctx, query, u.Kind, u.Role, p.Name, p.Gender, p.Age, p.Intent,
		p.About, p.Location, nullString(u.Country), u.Email, u.PasswordHash, u.IPHash)
	switch {
	case database.IsDuplicateKeyOn(err, memberNameIndex):
		return User{}, ErrNameTaken
	case database.IsDuplicateKey(err):
		return User{}, ErrEmailTaken
	case err != nil:
		return User{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return User{}, err
	}
	return r.GetByID(ctx, uint64(id))
}

func (r *MySQLRepository) GetByID(ctx context.Context, id uint64) (User, error) {
	return r.one(ctx, selectUser+`WHERE u.id = ?`, id)
}

func (r *MySQLRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	return r.one(ctx, selectUser+`WHERE u.email = ?`, email)
}

func (r *MySQLRepository) UpdateCard(ctx context.Context, id uint64, intent Intent, about, location string, photoID *uint64) error {
	const query = `
		UPDATE users SET intent = ?, about = ?, location = ?, photo_media_id = ?
		WHERE id = ? AND (? IS NULL OR EXISTS (
			SELECT 1 FROM media WHERE id = ? AND owner_id = ? AND removed_at IS NULL))`

	res, err := r.db.ExecContext(ctx, query, intent, about, location, photoID, id, photoID, photoID, id)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		if photoID != nil {
			return ErrPhotoNotYours
		}
		return ErrNotFound
	}
	return nil
}

func (r *MySQLRepository) Touch(ctx context.Context, id uint64) error {
	return r.exec(ctx, `UPDATE users SET last_seen_at = NOW() WHERE id = ?`, id)
}

func (r *MySQLRepository) SetRole(ctx context.Context, id uint64, role Role) error {
	return r.exec(ctx, `UPDATE users SET role = ? WHERE id = ? AND kind = 'member'`, role, id)
}

func (r *MySQLRepository) SetBan(ctx context.Context, id uint64, until *time.Time) error {
	return r.exec(ctx, `
		UPDATE users SET banned_until = ?,
		       token_version = token_version + IF(? IS NULL, 0, 1)
		WHERE id = ?`, until, until, id)
}

func (r *MySQLRepository) SetMute(ctx context.Context, id uint64, until *time.Time) error {
	return r.exec(ctx, `UPDATE users SET muted_until = ? WHERE id = ?`, until, id)
}

func (r *MySQLRepository) exec(ctx context.Context, query string, args ...any) error {
	res, err := r.db.ExecContext(ctx, query, args...)
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

func (r *MySQLRepository) one(ctx context.Context, query string, args ...any) (User, error) {
	var (
		u             User
		country       sql.NullString
		banned, muted sql.NullTime
	)
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&u.ID, &u.Kind, &u.Role, &u.Profile.Name, &u.Profile.Gender, &u.Profile.Age,
		&u.Profile.Intent, &u.Profile.About, &u.Profile.Location, &country, &u.PhotoID,
		&u.PhotoKey, &u.Email, &u.PasswordHash, &u.TokenVersion, &banned, &muted,
		&u.IPHash, &u.LastSeenAt, &u.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	u.Country = country.String
	u.BannedUntil = database.TimePtr(banned)
	u.MutedUntil = database.TimePtr(muted)
	return u, err
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
