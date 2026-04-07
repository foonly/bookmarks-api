package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/foonly/bookmarks-api/internal/models"
	_ "modernc.org/sqlite"
)

type sqliteStore struct {
	db           *sql.DB
	historyLimit int
}

// NewSQLiteStore initializes a new SQLite database and creates the necessary tables.
func NewSQLiteStore(dsn string, historyLimit int) (Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// Create table for storing blobs
	query := `
	CREATE TABLE IF NOT EXISTS sync_blobs (
		id TEXT NOT NULL,
		blob TEXT NOT NULL,
		timestamp INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_sync_id_ts ON sync_blobs (id, timestamp DESC);
	`
	if _, err := db.Exec(query); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &sqliteStore{
		db:           db,
		historyLimit: historyLimit,
	}, nil
}

func (s *sqliteStore) SaveBlob(ctx context.Context, id string, data string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ts := time.Now().UnixMilli()

	// Insert new blob
	_, err = tx.ExecContext(ctx, "INSERT INTO sync_blobs (id, blob, timestamp) VALUES (?, ?, ?)", id, data, ts)
	if err != nil {
		return fmt.Errorf("failed to insert blob: %w", err)
	}

	// Prune old versions: Keep only the latest N versions
	// Using a subquery to find the timestamps to delete
	pruneQuery := `
	DELETE FROM sync_blobs
	WHERE id = ? AND timestamp NOT IN (
		SELECT timestamp FROM sync_blobs
		WHERE id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	)`
	_, err = tx.ExecContext(ctx, pruneQuery, id, id, s.historyLimit)
	if err != nil {
		return fmt.Errorf("failed to prune history: %w", err)
	}

	return tx.Commit()
}

func (s *sqliteStore) GetLatestBlob(ctx context.Context, id string) (*models.SyncBlob, error) {
	var blob models.SyncBlob
	query := "SELECT id, blob, timestamp FROM sync_blobs WHERE id = ? ORDER BY timestamp DESC LIMIT 1"
	err := s.db.QueryRowContext(ctx, query, id).Scan(&blob.ID, &blob.Data, &blob.Timestamp)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &blob, nil
}

func (s *sqliteStore) GetHistory(ctx context.Context, id string) ([]models.SyncHistoryEntry, error) {
	query := "SELECT timestamp FROM sync_blobs WHERE id = ? ORDER BY timestamp DESC"
	rows, err := s.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.SyncHistoryEntry
	for rows.Next() {
		var entry models.SyncHistoryEntry
		if err := rows.Scan(&entry.Timestamp); err != nil {
			return nil, err
		}
		history = append(history, entry)
	}

	if len(history) == 0 {
		return nil, ErrNotFound
	}

	return history, nil
}

func (s *sqliteStore) GetBlobAtTimestamp(ctx context.Context, id string, ts int64) (*models.SyncBlob, error) {
	var blob models.SyncBlob
	query := "SELECT id, blob, timestamp FROM sync_blobs WHERE id = ? AND timestamp = ?"
	err := s.db.QueryRowContext(ctx, query, id, ts).Scan(&blob.ID, &blob.Data, &blob.Timestamp)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &blob, nil
}

func (s *sqliteStore) Close() error {
	return s.db.Close()
}
