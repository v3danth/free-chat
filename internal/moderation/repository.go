package moderation

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/v3danth/free-chat/internal/database"
	"github.com/v3danth/free-chat/internal/filter"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

// --- banned words ---

func (r *MySQLRepository) ListWords(ctx context.Context) ([]Word, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, word, action, created_at FROM banned_words ORDER BY word`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Word
	for rows.Next() {
		var w Word
		if err := rows.Scan(&w.ID, &w.Word, &w.Action, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *MySQLRepository) AddWord(ctx context.Context, word string, action filter.Action, actor uint64) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO banned_words (word, action, created_by) VALUES (?, ?, NULLIF(?, 0))`, word, action, actor)
	if database.IsDuplicateKey(err) {
		return ErrWordExists
	}
	return err
}

func (r *MySQLRepository) RemoveWord(ctx context.Context, id uint64) (string, error) {
	var word string
	err := r.db.QueryRowContext(ctx, `SELECT word FROM banned_words WHERE id = ?`, id).Scan(&word)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrWordNotFound
	}
	if err != nil {
		return "", err
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM banned_words WHERE id = ?`, id)
	return word, err
}

// --- IP bans ---

func (r *MySQLRepository) BanIP(ctx context.Context, ipHash []byte, until time.Time, reason string, actor uint64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ip_bans (ip_hash, expires_at, reason, created_by) VALUES (?, ?, ?, NULLIF(?, 0))
		ON DUPLICATE KEY UPDATE expires_at = GREATEST(expires_at, VALUES(expires_at)), reason = VALUES(reason)`,
		ipHash, until, reason, actor)
	return err
}

func (r *MySQLRepository) IsIPBanned(ctx context.Context, ipHash []byte) (bool, error) {
	var banned bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM ip_bans WHERE ip_hash = ? AND expires_at > NOW())`, ipHash).Scan(&banned)
	return banned, err
}

// --- reports ---

const selectReport = `
	SELECT id, reporter_id, target_type, target_id, target_user_id, reason, note, evidence,
	       status, handled_by, handled_at, created_at
	FROM reports `

func (r *MySQLRepository) CreateReport(ctx context.Context, rep Report) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO reports (reporter_id, target_type, target_id, target_user_id, reason, note, evidence)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		rep.ReporterID, rep.TargetType, rep.TargetID, rep.TargetUserID, rep.Reason, rep.Note, []byte(rep.Evidence))
	if database.IsDuplicateKey(err) {
		return ErrAlreadyReported
	}
	return err
}

func (r *MySQLRepository) CountOpen(ctx context.Context, t TargetType, targetID uint64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM reports WHERE target_type = ? AND target_id = ? AND status = 'open'`, t, targetID).Scan(&n)
	return n, err
}

// ListReports returns reports with status, oldest first (a queue).
func (r *MySQLRepository) ListReports(ctx context.Context, status Status, limit int) ([]Report, error) {
	// The open queue is worked oldest first; closed reports are history,
	// where the latest matter most.
	order := "id"
	if status != StatusOpen {
		order = "id DESC"
	}
	rows, err := r.db.QueryContext(ctx, selectReport+`WHERE status = ? ORDER BY `+order+` LIMIT ?`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Report
	for rows.Next() {
		rep, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rep)
	}
	return out, rows.Err()
}

func (r *MySQLRepository) SetReportStatus(ctx context.Context, id uint64, status Status, actor uint64) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE reports SET status = ?, handled_by = NULLIF(?, 0), handled_at = NOW()
		WHERE id = ? AND status = 'open'`, status, actor, id)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrReportNotFound
	}
	return nil
}

func (r *MySQLRepository) GetReport(ctx context.Context, id uint64) (Report, error) {
	rep := Report{ID: id}
	err := r.db.QueryRowContext(ctx, `SELECT target_type, target_id, status FROM reports WHERE id = ?`, id).
		Scan(&rep.TargetType, &rep.TargetID, &rep.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return Report{}, ErrReportNotFound
	}
	return rep, err
}

func (r *MySQLRepository) WasActioned(ctx context.Context, t TargetType, targetID uint64) (bool, error) {
	var done bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM reports WHERE target_type = ? AND target_id = ? AND status = 'actioned')`,
		t, targetID).Scan(&done)
	return done, err
}

// ResolveTarget closes every open report about one target.
func (r *MySQLRepository) ResolveTarget(ctx context.Context, t TargetType, targetID uint64, status Status, actor uint64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE reports SET status = ?, handled_by = NULLIF(?, 0), handled_at = NOW()
		WHERE target_type = ? AND target_id = ? AND status = 'open'`, status, actor, t, targetID)
	return err
}

func scanReport(row interface{ Scan(...any) error }) (Report, error) {
	var (
		rep       Report
		evidence  []byte
		handledAt sql.NullTime
	)
	err := row.Scan(&rep.ID, &rep.ReporterID, &rep.TargetType, &rep.TargetID, &rep.TargetUserID, &rep.Reason,
		&rep.Note, &evidence, &rep.Status, &rep.HandledBy, &handledAt, &rep.CreatedAt)
	rep.Evidence = evidence
	rep.HandledAt = database.TimePtr(handledAt)
	return rep, err
}

// --- audit log ---

func (r *MySQLRepository) Log(ctx context.Context, a Action) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO mod_actions (actor_id, action, target_user_id, target_name, target_id, detail)
		VALUES (NULLIF(?, 0), ?, NULLIF(?, 0), ?, NULLIF(?, 0), ?)`,
		a.ActorID, a.Action, a.TargetUserID, a.TargetName, a.TargetID, a.Detail)
	return err
}

func (r *MySQLRepository) ListActions(ctx context.Context, limit int) ([]Action, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(actor_id, 0), action, COALESCE(target_user_id, 0), target_name,
		       COALESCE(target_id, 0), detail, created_at
		FROM mod_actions ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Action
	for rows.Next() {
		var a Action
		if err := rows.Scan(&a.ID, &a.ActorID, &a.Action, &a.TargetUserID, &a.TargetName, &a.TargetID,
			&a.Detail, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
