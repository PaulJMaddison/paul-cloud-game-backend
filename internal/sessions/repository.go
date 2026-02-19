package sessions

import (
	"context"
	"database/sql"
)

type Repository interface {
	CreateSession(ctx context.Context, ownerUserID, status string, members []string) (Session, error)
	IsMember(ctx context.Context, sessionID, userID string) (bool, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateSession(ctx context.Context, ownerUserID, status string, members []string) (Session, error) {
	sessionID, err := newUUID()
	if err != nil {
		return Session{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, err
	}
	defer func() { _ = tx.Rollback() }()

	const qInsertSession = `INSERT INTO sessions (id, owner_user_id, status) VALUES ($1, $2, $3) RETURNING id::text, owner_user_id::text, status, created_at`
	var out Session
	if err := tx.QueryRowContext(ctx, qInsertSession, sessionID, ownerUserID, status).Scan(&out.ID, &out.OwnerUserID, &out.Status, &out.CreatedAt); err != nil {
		return Session{}, err
	}

	const qInsertMember = `INSERT INTO session_members (session_id, user_id, role) VALUES ($1, $2, $3)`
	for _, userID := range members {
		role := "player"
		if userID == ownerUserID {
			role = "owner"
		}
		if _, err := tx.ExecContext(ctx, qInsertMember, sessionID, userID, role); err != nil {
			return Session{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Session{}, err
	}
	return out, nil
}

func (r *PostgresRepository) IsMember(ctx context.Context, sessionID, userID string) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM session_members WHERE session_id = $1 AND user_id = $2)`
	var exists bool
	err := r.db.QueryRowContext(ctx, q, sessionID, userID).Scan(&exists)
	return exists, err
}
