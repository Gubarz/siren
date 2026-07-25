// Package journal implements journal.Store on SQLite via modernc.org/sqlite
// (pure Go — no cgo, keeps wails build and CI cross-compiles working).
package journal

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	journalv1 "sliver-gui/internal/journal"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS entries (
	id TEXT PRIMARY KEY,
	time INTEGER NOT NULL,
	connection_id TEXT NOT NULL DEFAULT '',
	actor_kind TEXT NOT NULL DEFAULT '',
	rule_id TEXT NOT NULL DEFAULT '',
	rule_name TEXT NOT NULL DEFAULT '',
	verb TEXT NOT NULL,
	command_line TEXT NOT NULL DEFAULT '',
	target_id TEXT NOT NULL DEFAULT '',
	target_kind TEXT NOT NULL DEFAULT '',
	hostname TEXT NOT NULL DEFAULT '',
	panel TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT '',
	err TEXT NOT NULL DEFAULT '',
	duration_ms INTEGER NOT NULL DEFAULT 0,
	correlation_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_entries_time ON entries(time);
CREATE INDEX IF NOT EXISTS idx_entries_verb ON entries(verb);
CREATE INDEX IF NOT EXISTS idx_entries_target ON entries(target_id);
CREATE INDEX IF NOT EXISTS idx_entries_conn ON entries(connection_id);
`

const insertSQL = `INSERT INTO entries (
	id, time, connection_id, actor_kind, rule_id, rule_name, verb,
	command_line, target_id, target_kind, hostname, panel, status, err,
	duration_ms, correlation_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dataDir string) (*SQLiteStore, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.Join(dataDir, "journal.db") +
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("journal schema: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) InsertBatch(ctx context.Context, entries []journalv1.Entry) error {
	if len(entries) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit
	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, e := range entries {
		if _, err := stmt.ExecContext(ctx,
			e.ID, e.Time, e.ConnectionID, e.ActorKind, e.RuleID, e.RuleName,
			e.Verb, e.CommandLine, e.TargetID, e.TargetKind, e.Hostname,
			e.Panel, e.Status, e.Err, e.DurationMs, e.CorrelationID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func whereClause(f journalv1.Filter) (string, []any) {
	var clauses []string
	var args []any
	equal := func(column, value string) {
		if value != "" {
			clauses = append(clauses, column+" = ?")
			args = append(args, value)
		}
	}
	equal("connection_id", f.ConnectionID)
	equal("target_id", f.TargetID)
	equal("verb", f.Verb)
	equal("actor_kind", f.ActorKind)
	if f.Since != 0 {
		clauses = append(clauses, "time >= ?")
		args = append(args, f.Since)
	}
	if f.Until != 0 {
		clauses = append(clauses, "time <= ?")
		args = append(args, f.Until)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (s *SQLiteStore) Query(ctx context.Context, f journalv1.Filter) ([]journalv1.Entry, int, error) {
	where, args := whereClause(f)
	var total int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM entries"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limit := f.Limit
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, time, connection_id, actor_kind, rule_id, rule_name, verb, "+
			"command_line, target_id, target_kind, hostname, panel, status, err, "+
			"duration_ms, correlation_id FROM entries"+where+
			" ORDER BY time DESC LIMIT ? OFFSET ?",
		append(args, limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var entries []journalv1.Entry
	for rows.Next() {
		var e journalv1.Entry
		if err := rows.Scan(
			&e.ID, &e.Time, &e.ConnectionID, &e.ActorKind, &e.RuleID, &e.RuleName,
			&e.Verb, &e.CommandLine, &e.TargetID, &e.TargetKind, &e.Hostname,
			&e.Panel, &e.Status, &e.Err, &e.DurationMs, &e.CorrelationID,
		); err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

func (s *SQLiteStore) VerbCounts(ctx context.Context, f journalv1.Filter) (map[string]int64, error) {
	where, args := whereClause(f)
	rows, err := s.db.QueryContext(ctx,
		"SELECT verb, COUNT(*) FROM entries"+where+" GROUP BY verb", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int64{}
	for rows.Next() {
		var verb string
		var n int64
		if err := rows.Scan(&verb, &n); err != nil {
			return nil, err
		}
		counts[verb] = n
	}
	return counts, rows.Err()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
