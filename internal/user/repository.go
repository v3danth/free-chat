package user

import (
	"context"
	"database/sql"
)

type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uint64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
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

	query := `
	SELECT
		id,
		user_type,
		username,
		gender,
		age,
		about,
		email,
		password_hash,
		created_at,
		updated_at
	FROM users
	WHERE username = ?
	LIMIT 1
	`

	var user User

	err := r.db.QueryRowContext(
		ctx,
		query,
		username,
	).Scan(
		&user.ID,
		&user.UserType,
		&user.Username,
		&user.Gender,
		&user.Age,
		&user.About,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *MySQLRepository) Create(
	ctx context.Context,
	user *User,
) error {

	query := `
	INSERT INTO users (
		user_type,
		username,
		gender,
		age,
		about,
		email,
		password_hash
	)
	VALUES (?, ?, ?, ?, ?, ?, ?)
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

	query := `
	SELECT
		id,
		user_type,
		username,
		gender,
		age,
		about,
		email,
		password_hash,
		created_at,
		updated_at
	FROM users
	WHERE id = ?
	`

	var user User

	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&user.ID,
		&user.UserType,
		&user.Username,
		&user.Gender,
		&user.Age,
		&user.About,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *MySQLRepository) GetByEmail(
	ctx context.Context,
	email string,
) (*User, error) {

	query := `
	SELECT
		id,
		user_type,
		username,
		gender,
		age,
		about,
		email,
		password_hash,
		created_at,
		updated_at
	FROM users
	WHERE email = ?
	LIMIT 1
	`

	var user User

	err := r.db.QueryRowContext(
		ctx,
		query,
		email,
	).Scan(
		&user.ID,
		&user.UserType,
		&user.Username,
		&user.Gender,
		&user.Age,
		&user.About,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
