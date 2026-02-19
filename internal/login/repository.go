package login

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrUserNotFound = errors.New("user not found")

type User struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

type Repository interface {
	GetByUsername(ctx context.Context, username string) (User, error)
	GetByID(ctx context.Context, id string) (User, error)
	Create(ctx context.Context, username, passwordHash string) (User, error)
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetByUsername(ctx context.Context, username string) (User, error) {
	const q = `SELECT id::text, username, password_hash, created_at FROM users WHERE username = $1`
	return r.scanUser(ctx, q, username)
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (User, error) {
	const q = `SELECT id::text, username, password_hash, created_at FROM users WHERE id = $1`
	return r.scanUser(ctx, q, id)
}

func (r *PostgresRepository) scanUser(ctx context.Context, q string, arg string) (User, error) {
	var user User
	err := r.db.QueryRowContext(ctx, q, arg).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	return user, err
}

func (r *PostgresRepository) Create(ctx context.Context, username, passwordHash string) (User, error) {
	id, err := newUUID()
	if err != nil {
		return User{}, err
	}
	const q = `
		INSERT INTO users (id, username, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id::text, username, password_hash, created_at`
	var user User
	err = r.db.QueryRowContext(ctx, q, id, username, passwordHash).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	return user, err
}

func (r *PostgresRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, userID, passwordHash)
	return err
}
