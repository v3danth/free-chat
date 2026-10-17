// Package letters is slow mail between members. A letter is written in an
// open chat and arrives hours later, which gives a conversation a pause to
// look forward to. Delivery needs the recipient online, so a loop hands due
// letters to whoever is connected and leaves the rest for the next pass.
package letters

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/database"
	"github.com/v3danth/free-chat/internal/id"
)

// ErrPending stops a second letter while the first is still on its way,
// so a letter stays special and cannot be used to flood someone.
var ErrPending = apperr.New(apperr.Conflict, "your last letter to them has not arrived yet")

type Letter struct {
	ID          uint64
	SenderID    uint64
	RecipientID uint64
	SenderName  string
	Body        string
	DeliverAt   time.Time
}

// SentAt is when the letter was written; ids are time-ordered.
func (l Letter) SentAt() time.Time { return id.Time(l.ID) }

type MySQLStore struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *MySQLStore { return &MySQLStore{db: db} }

// Send stores l unless the sender already has one on its way to the same
// person. The check and the insert are one statement, so two quick sends
// cannot both get through.
func (s *MySQLStore) Send(ctx context.Context, l Letter) error {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO letters (id, sender_id, recipient_id, sender_name, body, deliver_at)
		SELECT ?, ?, ?, ?, ?, ? FROM DUAL
		WHERE NOT EXISTS (
			SELECT 1 FROM letters WHERE sender_id = ? AND recipient_id = ? AND delivered_at IS NULL)`,
		l.ID, l.SenderID, l.RecipientID, l.SenderName, l.Body, l.DeliverAt.UTC(), l.SenderID, l.RecipientID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrPending
	}
	return nil
}

// Due returns up to limit letters whose time has come for the given
// recipients, oldest first. Asking only for people online means letters to
// the offline are not read every pass, and cannot fill the batch and hold
// back letters for people who are here.
func (s *MySQLStore) Due(ctx context.Context, now time.Time, recipients []uint64, limit int) ([]Letter, error) {
	if len(recipients) == 0 {
		return nil, nil
	}
	in, args := database.In(recipients)
	args = append(args, now.UTC(), limit)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, sender_id, recipient_id, sender_name, body, deliver_at
		FROM letters WHERE recipient_id IN `+in+` AND delivered_at IS NULL AND deliver_at <= ?
		ORDER BY deliver_at LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	due := make([]Letter, 0, limit)
	for rows.Next() {
		var l Letter
		if err := rows.Scan(&l.ID, &l.SenderID, &l.RecipientID, &l.SenderName, &l.Body, &l.DeliverAt); err != nil {
			return nil, err
		}
		due = append(due, l)
	}
	return due, rows.Err()
}

// MarkDelivered records that the letters reached their recipients.
func (s *MySQLStore) MarkDelivered(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	in, args := database.In(ids)
	_, err := s.db.ExecContext(ctx, `UPDATE letters SET delivered_at = NOW() WHERE id IN `+in, args...)
	return err
}

// Store is what delivery needs from storage.
type Store interface {
	Due(ctx context.Context, now time.Time, recipients []uint64, limit int) ([]Letter, error)
	MarkDelivered(ctx context.Context, ids []uint64) error
}

// Postbox hands a letter to its recipient. Done is true when the letter
// is finished with: delivered, or dropped because the two blocked each other.
type Postbox interface {
	DeliverLetter(l Letter) (done bool)
	OnlineIDs() []uint64
}

const batch = 200

// Run delivers due letters every interval until ctx ends.
func Run(ctx context.Context, store Store, box Postbox, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := Deliver(ctx, store, box, time.Now()); err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("deliver letters", "err", err)
			}
		}
	}
}

// Deliver makes one pass: every due letter whose recipient is online is
// handed over and marked, the others wait for the next pass.
func Deliver(ctx context.Context, store Store, box Postbox, now time.Time) error {
	online := box.OnlineIDs()
	if len(online) == 0 {
		return nil
	}
	due, err := store.Due(ctx, now, online, batch)
	if err != nil {
		return err
	}
	done := make([]uint64, 0, len(due))
	for _, l := range due {
		if box.DeliverLetter(l) {
			done = append(done, l.ID)
		}
	}
	return store.MarkDelivered(ctx, done)
}
