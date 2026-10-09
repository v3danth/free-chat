package user

import (
	"context"
	"database/sql"
	"encoding/json"
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
	// UpdateCard saves the editable card fields (tags, colour, about,
	// location); photoID must belong to the user and not be removed, or
	// ErrPhotoNotYours.
	UpdateCard(ctx context.Context, id uint64, card Profile, photoID *uint64) error
	Touch(ctx context.Context, id uint64) error
	SetRole(ctx context.Context, id uint64, role Role) error
	// SetBan sets or clears a ban; banning also revokes every token.
	SetBan(ctx context.Context, id uint64, until *time.Time) error
	SetMute(ctx context.Context, id uint64, until *time.Time) error
	// ListStaff returns every moderator and admin, admins first.
	ListStaff(ctx context.Context) ([]User, error)
}

type MySQLRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

const selectUser = `
	SELECT u.id, u.kind, u.role, u.name, u.gender, u.age, u.tags, u.name_color, u.about, u.location,
	       u.country_code, u.photo_media_id, m.file_key, u.email, u.password_hash,
	       u.token_version, u.banned_until, u.muted_until, u.ip_hash, u.last_seen_at, u.created_at
	FROM users u
	LEFT JOIN media m ON m.id = u.photo_media_id AND m.removed_at IS NULL `

func (r *MySQLRepository) Create(ctx context.Context, u User) (User, error) {
	const query = `
		INSERT INTO users (kind, role, name, gender, age, tags, name_color, about, location,
		                   country_code, email, password_hash, ip_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	p := u.Profile
	if p.Color == "" {
		p.Color = DefaultColor // accounts made in code (the admin) pick no colour
	}
	res, err := r.db.ExecContext(ctx, query, u.Kind, u.Role, p.Name, p.Gender, p.Age, tagsJSON(p.Tags), p.Color,
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

func (r *MySQLRepository) UpdateCard(ctx context.Context, id uint64, card Profile, photoID *uint64) error {
	const query = `
		UPDATE users SET tags = ?, name_color = ?, about = ?, location = ?, photo_media_id = ?
		WHERE id = ? AND (? IS NULL OR EXISTS (
			SELECT 1 FROM media WHERE id = ? AND owner_id = ? AND removed_at IS NULL))`

	res, err := r.db.ExecContext(ctx, query, tagsJSON(card.Tags), card.Color, card.About, card.Location, photoID, id, photoID, photoID, id)
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

func (r *MySQLRepository) ListStaff(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, selectUser+`WHERE u.role IN ('admin', 'moderator') ORDER BY u.role = 'admin' DESC, u.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var staff []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		staff = append(staff, u)
	}
	return staff, rows.Err()
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
	u, err := scanUser(r.db.QueryRowContext(ctx, query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func scanUser(row interface{ Scan(...any) error }) (User, error) {
	var (
		u             User
		country       sql.NullString
		banned, muted sql.NullTime
		tags          []byte
	)
	err := row.Scan(
		&u.ID, &u.Kind, &u.Role, &u.Profile.Name, &u.Profile.Gender, &u.Profile.Age,
		&tags, &u.Profile.Color, &u.Profile.About, &u.Profile.Location, &country, &u.PhotoID,
		&u.PhotoKey, &u.Email, &u.PasswordHash, &u.TokenVersion, &banned, &muted,
		&u.IPHash, &u.LastSeenAt, &u.CreatedAt,
	)
	if err == nil {
		err = json.Unmarshal(tags, &u.Profile.Tags)
	}
	u.Country = country.String
	u.BannedUntil = database.TimePtr(banned)
	u.MutedUntil = database.TimePtr(muted)
	return u, err
}

// tagsJSON stores tags as a JSON array; never null, so reads always decode.
func tagsJSON(tags []string) string {
	if tags == nil {
		tags = []string{}
	}
	data, _ := json.Marshal(tags) // a []string always marshals
	return string(data)
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
