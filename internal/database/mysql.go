package database

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

// Open builds the DSN with the driver's own formatter (so credentials with
// special characters are escaped correctly) and verifies connectivity.
func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	dsn := mysql.NewConfig()
	dsn.Net = "tcp"
	dsn.Addr = net.JoinHostPort(cfg.Host, cfg.Port)
	dsn.User = cfg.User
	dsn.Passwd = cfg.Password
	dsn.DBName = cfg.Name
	dsn.ParseTime = true
	dsn.Loc = time.UTC
	// NOW() and DATETIME columns are UTC, matching dsn.Loc.
	dsn.Params = map[string]string{"time_zone": "'+00:00'"}
	// RowsAffected counts matched rows, so "0" reliably means "no such row"
	// rather than "row already had these values".
	dsn.ClientFoundRows = true

	db, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(8 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

const (
	errDuplicateKey = 1062
	errMissingRef   = 1452
)

func IsDuplicateKey(err error) bool { return hasCode(err, errDuplicateKey) }

// IsDuplicateKeyOn reports a duplicate on the named unique index.
func IsDuplicateKeyOn(err error, index string) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == errDuplicateKey && strings.Contains(me.Message, index)
}

// IsMissingReference reports a foreign-key violation on insert/update.
func IsMissingReference(err error) bool { return hasCode(err, errMissingRef) }

func hasCode(err error, code uint16) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == code
}

// In builds "(?, ?, ...)" and its args for a non-empty slice.
func In[T any](vals []T) (string, []any) {
	args := make([]any, len(vals))
	for i, v := range vals {
		args[i] = v
	}
	return "(" + strings.TrimSuffix(strings.Repeat("?, ", len(vals)), ", ") + ")", args
}

// Collect scans a single-column result set.
func Collect[T any](ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, query string, args ...any) ([]T, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []T
	for rows.Next() {
		var v T
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// TimePtr converts a nullable time for scanning targets.
func TimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
