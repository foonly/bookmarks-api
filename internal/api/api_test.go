package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/foonly/bookmarks-api/internal/models"
	"github.com/foonly/bookmarks-api/internal/store"
)

// setupTest initializes a router with an in-memory SQLite store for testing.
func setupTest(t *testing.T) (http.Handler, store.Store) {
	// Use :memory: for SQLite testing to ensure a clean slate and speed
	s, err := store.NewSQLiteStore(":memory:", 10)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	h := NewHandler(s)
	r := NewRouter(h)
	return r, s
}

func TestSyncAPI(t *testing.T) {
	router, _ := setupTest(t)
	syncID := "test-user-123"
	blobV1 := "encrypted-payload-v1"
	blobV2 := "encrypted-payload-v2"

	t.Run("GetNonExistentBlob", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/sync/"+syncID, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rr.Code)
		}
	})

	t.Run("UploadV1", func(t *testing.T) {
		body, _ := json.Marshal(models.SyncRequest{Data: blobV1})
		req := httptest.NewRequest("POST", "/api/v1/sync/"+syncID, bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", rr.Code)
		}
	})

	t.Run("GetLatestAfterV1", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/sync/"+syncID, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var res models.SyncBlob
		if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
			t.Fatal(err)
		}

		if res.Data != blobV1 {
			t.Errorf("expected data %s, got %s", blobV1, res.Data)
		}
	})

	t.Run("UploadV2", func(t *testing.T) {
		body, _ := json.Marshal(models.SyncRequest{Data: blobV2})
		req := httptest.NewRequest("POST", "/api/v1/sync/"+syncID, bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", rr.Code)
		}
	})

	t.Run("VerifyHistory", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/sync/"+syncID+"/history", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var history []models.SyncHistoryEntry
		if err := json.NewDecoder(rr.Body).Decode(&history); err != nil {
			t.Fatal(err)
		}

		if len(history) != 2 {
			t.Errorf("expected history size 2, got %d", len(history))
		}
	})

	t.Run("PayloadTooLarge", func(t *testing.T) {
		// Create a payload that exceeds MaxPayloadSize (1MB)
		largeData := strings.Repeat("a", MaxPayloadSize+1024)
		body, _ := json.Marshal(models.SyncRequest{Data: largeData})
		req := httptest.NewRequest("POST", "/api/v1/sync/large-id", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// MaxBytesReader error results in 400 Bad Request in our current implementation
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for oversized payload, got %d", rr.Code)
		}
	})

	t.Run("RateLimiting", func(t *testing.T) {
		limitID := "rate-limit-test"
		// 5 POSTs per minute are allowed
		for i := 0; i < 5; i++ {
			body, _ := json.Marshal(models.SyncRequest{Data: "data"})
			req := httptest.NewRequest("POST", "/api/v1/sync/"+limitID, bytes.NewBuffer(body))
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusCreated {
				t.Fatalf("expected 201 on attempt %d, got %d", i+1, rr.Code)
			}
		}

		// 6th POST should be rejected
		body, _ := json.Marshal(models.SyncRequest{Data: "data"})
		req := httptest.NewRequest("POST", "/api/v1/sync/"+limitID, bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("expected 429 for rate limit, got %d", rr.Code)
		}
	})
}

func TestFetchSpecificVersion(t *testing.T) {
	router, _ := setupTest(t)
	syncID := "version-test"

	// Upload one version
	body, _ := json.Marshal(models.SyncRequest{Data: "v1"})
	req := httptest.NewRequest("POST", "/api/v1/sync/"+syncID, bytes.NewBuffer(body))
	router.ServeHTTP(httptest.NewRecorder(), req)

	// Get latest to find timestamp
	req = httptest.NewRequest("GET", "/api/v1/sync/"+syncID, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var res models.SyncBlob
	json.NewDecoder(rr.Body).Decode(&res)
	ts := res.Timestamp

	// Fetch specific version by timestamp
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/sync/%s/%d", syncID, ts), nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 fetching specific version, got %d", rr.Code)
	}

	var verRes models.SyncBlob
	json.NewDecoder(rr.Body).Decode(&verRes)
	if verRes.Data != "v1" {
		t.Errorf("expected 'v1', got '%s'", verRes.Data)
	}
}
