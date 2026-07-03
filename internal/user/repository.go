package user

import (
	"context"
	"database/sql"
	"time"
)

type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uint64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)

	// Guest-specific methods
	IsGuestUsernameAvailable(ctx context.Context, username string) (bool, error)
	SetInactive(ctx context.Context, id uint64) error
	DeleteInactiveGuests(ctx context.Context, olderThan time.Duration) (int64, error)
}

type MySQLRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &MySQLRepository{
		db: db,
	}
}

func (r *MySQLRepository) GetByUsername(
	ctx context.Context,
	username string,
) (*User, error) {

	const query = `
    SELECT 
		id, 
		user_type, 
		status, 
		username,
		gender,
		age,
		about,
		email,
		password_hash,
		last_seen_at,
		created_at,
		updated_at
        FROM users
        WHERE username = ?
        LIMIT 1
    `

	return r.scanUser(ctx, query, username)
}

func (r *MySQLRepository) Create(
	ctx context.Context,
	user *User,
) error {

	const query = `
        INSERT INTO users (
		user_type, 
		username, 
		gender, 
		age, 
		about, 
		email, 
		password_hash,
		status
		)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `

	result, err := r.db.ExecContext(
		ctx,
		query,
		user.UserType,
		user.Username,
		user.Gender,
		user.Age,
		user.About,
		user.Email,
		user.PasswordHash,
		user.Status,
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	user.ID = uint64(id)

	return nil
}

func (r *MySQLRepository) GetByID(
	ctx context.Context,
	id uint64,
) (*User, error) {

	const query = `
        SELECT 
			id, 
			user_type, 
			status, 
			username, 
			gender, 
			age, 
			about,
            email, 
			password_hash, 
			last_seen_at, 
			created_at, 
			updated_at
        FROM users
        WHERE id = ?
    `

	return r.scanUser(ctx, query, id)
}

func (r *MySQLRepository) GetByEmail(
	ctx context.Context,
	email string,
) (*User, error) {

	const query = `
    SELECT 
		id, 
		user_type, 
		status, 
		username,
		gender,
		age,
		about,
		email,
		password_hash,
		last_seen_at,
		created_at,
		updated_at
        FROM users
        WHERE email = ?
        LIMIT 1
    `

	return r.scanUser(ctx, query, email)
}

func (r *MySQLRepository) IsGuestUsernameAvailable(ctx context.Context, username string) (bool, error) {
	const query = `
        SELECT COUNT(*) 
        FROM users 
        WHERE username = ? 
          AND user_type = 'guest' 
          AND status = 'active'
    `

	var count int
	if err := r.db.QueryRowContext(ctx, query, username).Scan(&count); err != nil {
		return false, err
	}

	// Also check if username is taken by registered user
	const regQuery = `
        SELECT COUNT(*) 
        FROM users 
        WHERE username = ? 
          AND user_type = 'registered'
    `

	var regCount int
	if err := r.db.QueryRowContext(ctx, regQuery, username).Scan(&regCount); err != nil {
		return false, err
	}

	return count == 0 && regCount == 0, nil
}

func (r *MySQLRepository) SetInactive(ctx context.Context, id uint64) error {
	const query = `
        UPDATE users 
        SET status = 'inactive', last_seen_at = NOW() 
        WHERE id = ? AND user_type = 'guest'
    `

	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *MySQLRepository) DeleteInactiveGuests(
	ctx context.Context,
	olderThan time.Duration,
) (int64, error) {
	const query = `
        DELETE FROM users 
        WHERE user_type = 'guest' 
          AND status = 'inactive'
          AND last_seen_at < DATE_SUB(NOW(), INTERVAL ? SECOND)
    `

	result, err := r.db.ExecContext(ctx, query, int64(olderThan.Seconds()))
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (r *MySQLRepository) scanUser(ctx context.Context, query string, args ...interface{}) (*User, error) {
	var user User

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.UserType,
		&user.Status,
		&user.Username,
		&user.Gender,
		&user.Age,
		&user.About,
		&user.Email,
		&user.PasswordHash,
		&user.LastSeenAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}
