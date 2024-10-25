package sqlitekv

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/mattn/go-sqlite3"
)

// SqliteStore represents a key-value store implemented with SQLite.
type SqliteStore struct {
	db *sql.DB
}

// NewStore initializes a new SqliteStore with the given database file path.
func NewStore(databasePath string) (*SqliteStore, error) {
	db, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		return nil, err
	}

	// Create the key-value table if it doesn't exist
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS kv (
            key TEXT PRIMARY KEY,
            value BLOB
        )
    `)
	if err != nil {
		return nil, err
	}

	return &SqliteStore{db: db}, nil
}

// Has checks if a key exists in the store.
func (ss *SqliteStore) Has(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := ss.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM kv WHERE key = ?)`, key).Scan(&exists)
	return exists, err
}

// Get retrieves the value associated with the key.
func (ss *SqliteStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	var value []byte
	err := ss.db.QueryRowContext(ctx, `SELECT value FROM kv WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	return value, err == nil, err
}

// Set inserts or updates a value associated with the key.
func (ss *SqliteStore) Set(ctx context.Context, key string, val []byte) error {
	_, err := ss.db.ExecContext(ctx, `INSERT INTO kv (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, val)
	return err
}

// Del deletes the key-value pair from the store.
func (ss *SqliteStore) Del(ctx context.Context, key string) (bool, error) {
	result, err := ss.db.ExecContext(ctx, `DELETE FROM kv WHERE key = ?`, key)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

// ForEach iterates over each key-value pair in the store.
func (ss *SqliteStore) ForEach(ctx context.Context, fn func(ctx context.Context, key string, value []byte) (ok bool)) (err error) {
	var rows *sql.Rows
	rows, err = ss.db.QueryContext(ctx, `SELECT key, value FROM kv`)
	if err != nil {
		return err
	}
	defer func() { err = rows.Close() }()

	for rows.Next() {
		var key string
		var value []byte
		if err = rows.Scan(&key, &value); err != nil {
			return err
		}

		if !fn(ctx, key, value) {
			break
		}
	}

	return rows.Err()
}
