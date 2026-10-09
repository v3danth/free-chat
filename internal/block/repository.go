// Package block stores who has blocked whom. A block works both ways: once
// either side blocks, neither can reach the other.
package block

import (
	"context"
	"database/sql"

	"github.com/v3danth/free-chat/internal/database"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

// Between returns everyone user has blocked or been blocked by.
func (r *MySQLRepository) Between(ctx context.Context, user uint64) (map[uint64]struct{}, error) {
	ids, err := database.Collect[uint64](ctx, r.db, `
		SELECT blocked_id FROM blocks WHERE blocker_id = ?
		UNION
		SELECT blocker_id FROM blocks WHERE blocked_id = ?`, user, user)
	if err != nil {
		return nil, err
	}
	set := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set, nil
}

func (r *MySQLRepository) Add(ctx context.Context, blocker, blocked uint64) error {
	_, err := r.db.ExecContext(ctx, `INSERT IGNORE INTO blocks (blocker_id, blocked_id) VALUES (?, ?)`, blocker, blocked)
	return err
}

func (r *MySQLRepository) Remove(ctx context.Context, blocker, blocked uint64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM blocks WHERE blocker_id = ? AND blocked_id = ?`, blocker, blocked)
	return err
}
