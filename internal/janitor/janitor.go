// Package janitor enforces retention: old messages, guests who left long
// ago (with their image files), and expired IP bans.
package janitor

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/v3danth/free-chat/internal/database"
	"github.com/v3danth/free-chat/internal/id"
)

const purgeBatch = 5000

type Store interface {
	// PurgeMessages deletes up to batch messages with id < cutoff.
	PurgeMessages(ctx context.Context, cutoff uint64, batch int) (int64, error)
	// ExpireGuests deletes guests last seen before cutoff and returns their
	// ids and the file keys of their images, whose rows the delete cascades away.
	ExpireGuests(ctx context.Context, cutoff time.Time, online []uint64) (guests []uint64, keys []string, err error)
	PurgeExpiredIPBans(ctx context.Context) (int64, error)
}

type Files interface {
	DeleteFiles(key string)
}

// Live is the chat hub: who is online, and its in-memory room history,
// which must forget what the sweep deletes.
type Live interface {
	OnlineIDs() []uint64
	Forget(beforeID uint64, senders []uint64)
}

type Config struct {
	Retention time.Duration
	Interval  time.Duration
}

// Run sweeps every cfg.Interval until ctx is cancelled.
func Run(ctx context.Context, store Store, files Files, online Live, cfg Config) {
	log.Printf("janitor started: retention=%v interval=%v", cfg.Retention, cfg.Interval)
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			Sweep(ctx, store, files, online, cfg.Retention, time.Now())
		}
	}
}

func Sweep(ctx context.Context, store Store, files Files, online Live, retention time.Duration, now time.Time) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cutoff := now.Add(-retention)

	// Ids are time-ordered, so retention is a primary-key range delete,
	// done in batches to keep each transaction small.
	var purged int64
	for {
		n, err := store.PurgeMessages(ctx, id.Floor(cutoff), purgeBatch)
		if err != nil {
			log.Printf("janitor: purge messages: %v", err)
			break
		}
		purged += n
		if n < purgeBatch {
			break
		}
	}

	guests, keys, err := store.ExpireGuests(ctx, cutoff, online.OnlineIDs())
	if err != nil {
		log.Printf("janitor: expire guests: %v", err)
	}
	for _, k := range keys {
		files.DeleteFiles(k)
	}
	online.Forget(id.Floor(cutoff), guests)

	if _, err := store.PurgeExpiredIPBans(ctx); err != nil {
		log.Printf("janitor: purge ip bans: %v", err)
	}
	if purged > 0 || len(keys) > 0 {
		log.Printf("janitor: purged %d messages, %d guest images", purged, len(keys))
	}
}

// MySQLStore implements Store.
type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(db *sql.DB) *MySQLStore { return &MySQLStore{db: db} }

func (s *MySQLStore) PurgeMessages(ctx context.Context, cutoff uint64, batch int) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE id < ? ORDER BY id LIMIT ?`, cutoff, batch)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ExpireGuests reads the image keys and deletes the users in one
// transaction, under row locks, so no key is lost to the cascade.
func (s *MySQLStore) ExpireGuests(ctx context.Context, cutoff time.Time, online []uint64) ([]uint64, []string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	query := `SELECT id FROM users WHERE kind = 'guest' AND last_seen_at < ?`
	args := []any{cutoff}
	if len(online) > 0 {
		in, ids := database.In(online)
		query += ` AND id NOT IN ` + in
		args = append(args, ids...)
	}
	guests, err := database.Collect[uint64](ctx, tx, query+` LIMIT 1000 FOR UPDATE`, args...)
	if err != nil || len(guests) == 0 {
		return nil, nil, err
	}

	in, ids := database.In(guests)
	keys, err := database.Collect[string](ctx, tx, `SELECT file_key FROM media WHERE owner_id IN `+in, ids...)
	if err != nil {
		return nil, nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id IN `+in, ids...); err != nil {
		return nil, nil, err
	}
	return guests, keys, tx.Commit()
}

func (s *MySQLStore) PurgeExpiredIPBans(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM ip_bans WHERE expires_at <= NOW()`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
