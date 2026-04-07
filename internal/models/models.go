package models

// SyncIdentity represents the registration and authentication info for a sync ID.
type SyncIdentity struct {
	ID            string `json:"id"`
	SigningSecret string `json:"signing_secret"`
	LastTimestamp int64  `json:"last_timestamp"`
}

// SyncBlob represents the encrypted data stored for a specific sync ID.
type SyncBlob struct {
	ID        string `json:"id,omitempty"`
	Data      string `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

// SyncHistoryEntry represents a summary of a historical version.
type SyncHistoryEntry struct {
	Timestamp int64 `json:"timestamp"`
}

// SyncRequest represents the payload for uploading a new blob.
type SyncRequest struct {
	Data               string `json:"data"`
	RegistrationSecret string `json:"registration_secret,omitempty"`
}
