# Security and Best Practices Audit - Foonblob API

This document provides a comprehensive audit of the `foonblob-api` project, covering security vulnerabilities, architectural best practices, and actionable recommendations.

## 1. Security Analysis

### Findings: High Priority

- **CORS Configuration**: The project now implements **Dynamic CORS**. When an identity is registered, its `Origin` is captured (or explicitly provided) and stored. Subsequent requests for that sync ID are restricted to that specific origin via a custom middleware.
- **Plaintext Secret Storage**: The `signing_secret` (used for HMAC verification) is stored in plaintext in the SQLite database. If the database file is compromised, all user signatures can be forged. _Note: Since this is used for HMAC, the plaintext is required for verification; however, encryption at rest for the database is recommended._
- **Unauthenticated Read Access**: GET requests to `/api/v1/sync/{id}` do not require a signature or authentication. If a sync ID is leaked or guessed, any party can read the stored blob. While the blob is intended to be client-side encrypted, this still represents a significant privacy risk and potential metadata leak.

### Findings: Medium Priority

- **Public Statistics Endpoint**: The `/api/v1/stats` endpoint is publicly accessible without any authentication. This leaks metadata about the system usage (total identities, blob counts, storage size).
- **In-Memory Rate Limiting**: The rate limiter is in-memory. If the server restarts, all rate limit buckets are reset. In a multi-node deployment (if ever scaled), rate limits would not be shared.
- **Rate Limiter Memory Leak**: The `RateLimiter` map in `internal/api/ratelimit.go` grows indefinitely. Every unique sync ID that interacts with the API creates a new entry in the map that is never pruned, which could lead to gradual memory exhaustion over time.
- **Information Leakage in Logs**: The `Logger` middleware is enabled. Ensure that sensitive headers or body content are not logged in production environments.

### Findings: Low Priority

- **Timing Attacks**: Signature verification in `handlers.go` uses `hmac.Equal`, which is good as it prevents timing attacks.
- **Replay Protection**: The system implements a 5-minute window and checks for strictly increasing timestamps. This is a solid implementation for replay protection.
- **Resource Exhaustion**: `MaxBytesReader` is correctly used to limit upload sizes to 1MB, preventing OOM (Out of Memory) attacks via large payloads.

---

## 2. Best Practices Analysis

### Code Quality & Architecture

- **Project Structure**: The project follows the standard Go project layout (`cmd/`, `internal/`), which is excellent for maintainability.
- **Concurrency**: SQLite is correctly configured with `MaxOpenConns(1)` and WAL mode to handle concurrent reads/writes safely.
- **Error Handling**: Uses Go 1.13+ error wrapping (`%w`) and `errors.Is` for reliable error checking.
- **Graceful Shutdown**: `main.go` correctly handles `SIGINT` and `SIGTERM` to shut down the HTTP server and close database connections.
- **Go Version Anomaly**: The `go.mod` file specifies `go 1.26.1`. As of current stable releases (Go 1.24), this version does not exist and should be corrected to a supported version.

### Database Management

- **Ad-hoc Migrations**: Migrations are currently performed using manual `ALTER TABLE` statements in `NewSQLiteStore`. As the schema grows, this will become difficult to manage.
- **Cleanup Logic**: The background cleanup worker is a good addition for maintaining a slim database.

---

## 3. Actionable Recommendations

### Phase 1: Immediate Security Fixes

- [x] **Restrict CORS**: Implemented Dynamic CORS that locks sync IDs to their registration origin.
- [ ] **Protect Stats**: Add a simple API key requirement or restrict `/api/v1/stats` to internal IP ranges.
- [ ] **Authenticate Reads**: Consider requiring an HMAC signature for GET requests as well, similar to the POST implementation, to prevent unauthorized access to blobs.

### Phase 2: Architectural Improvements

- [ ] **Structured Logging**: Replace the standard `log` package with `log/slog` (available in Go 1.21+) to produce JSON logs for better observability.
- [ ] **Configuration Management**: Move configuration (Port, DSN, History Limit) from flags to environment variables or a configuration file (e.g., using `viper` or `caarlos0/env`) to support containerized deployments (Docker/K8s).
- [ ] **Migration Tooling**: Integrate a migration tool like `golang-migrate` or `pressly/goose` to manage database schema versions formally.
- [ ] **Fix Rate Limiter Leak**: Implement a TTL-based eviction or a LRU cache for the rate limiter map to prevent unbounded memory growth.
- [x] **Fix Go Version**: Correct the `go.mod` version to a stable release (e.g., `1.24.0`).

### Phase 3: Robustness & Scaling

- [ ] **Request Contexts**: Ensure all database operations strictly respect the `r.Context()` for cancellation, especially for long-running queries (already mostly implemented).
- [ ] **Database Encryption**: If sensitive data is stored, consider using a SQLite extension like SQLCipher to encrypt the database file at rest.

---

## Conclusion

The `foonblob-api` is well-structured and implements core security patterns correctly (HMAC, Replay Protection, Body Limits). With the implementation of Dynamic CORS, the risk of cross-origin data leakage is significantly reduced while maintaining flexibility for multiple client applications. Remaining risks are primarily related to the exposure of the stats endpoint and the lack of structured configuration for production environments.
