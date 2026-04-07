package store

import (
	"context"
	"errors"

	"github.com/foonly/bookmarks-api/internal/models"
)

// ErrNotFound is returned when the requested sync ID or version does not exist.
var ErrNotFound = errors.New("sync data not found")

// Store defines the interface for persisting encrypted bookmark blobs and their history.
type Store interface {
	// SaveBlob stores a new encrypted blob for the given ID and handles pruning of old versions.
	SaveBlob(ctx context.Context, id string, data string) error

	// GetLatestBlob retrieves the most recent blob for the given ID.
	GetLatestBlob(ctx context.Context, id string) (*models.SyncBlob, error)

	// GetHistory retrieves a list of timestamps for available historical versions.
	GetHistory(ctx context.Context, id string) ([]models.SyncHistoryEntry, error)

	// GetBlobAtTimestamp retrieves a specific historical blob by its timestamp.
	GetBlobAtTimestamp(ctx context.Context, id string, ts int64) (*models.SyncBlob, error)

	// Close closes the underlying storage connection.
	Close() error
}
