package certificate

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"
	"log/slog"

	_ "github.com/mattn/go-sqlite3"
	"errors"
)

type SqliteStore struct {
	db *sql.DB
}

func NewSqliteStore(dbPath string) (*SqliteStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	store := &SqliteStore{db: db}
	if err := store.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return store, nil
}

func (s *SqliteStore) Init() error {
	// Certificates table
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS certificates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			raw_proto BLOB NOT NULL,
			received_at DATETIME NOT NULL,
			processed_at DATETIME,
			metadata TEXT
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create certificates table: %w", err)
	}

	// Configuration table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS configuration (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create configuration table: %w", err)
	}

	// Credentials table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS credentials (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create credentials table: %w", err)
	}

	// Scheduler status table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS scheduler_status (
			id INTEGER PRIMARY KEY,
			is_active BOOLEAN NOT NULL DEFAULT 1,
			last_updated DATETIME NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create scheduler_status table: %w", err)
	}

	// Kill switch attempts table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS kill_switch_attempts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			attempt_type TEXT NOT NULL,
			attempted_at DATETIME NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create kill_switch_attempts table: %w", err)
	}

	// Initialize scheduler status if not exists
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM scheduler_status").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check scheduler status: %w", err)
	}
	if count == 0 {
		_, err = s.db.Exec("INSERT INTO scheduler_status (id, is_active, last_updated) VALUES (1, 1, ?)", time.Now())
		if err != nil {
			return fmt.Errorf("failed to initialize scheduler status: %w", err)
		}
	}

	// Check if we need to migrate from delay_hours to delay_seconds
	var delayHours string
	err = s.db.QueryRow("SELECT value FROM configuration WHERE key = 'delay_hours'").Scan(&delayHours)
	if err == nil && delayHours != "" {
		// Migrate to delay_seconds
		hours, _ := strconv.Atoi(delayHours)
		seconds := hours * 3600
		_, err = s.db.Exec("INSERT OR REPLACE INTO configuration (key, value) VALUES ('delay_seconds', ?)", strconv.Itoa(seconds))
		if err == nil {
			// Remove old delay_hours
			s.db.Exec("DELETE FROM configuration WHERE key = 'delay_hours'")
		}
	}

	// Set default delay if not present
	var count2 int
	err = s.db.QueryRow("SELECT COUNT(*) FROM configuration WHERE key = 'delay_seconds'").Scan(&count2)
	if err != nil {
		return fmt.Errorf("failed to check for delay configuration: %w", err)
	}

	if count2 == 0 {
		// Default: 48 hours = 172800 seconds
		_, err = s.db.Exec("INSERT INTO configuration (key, value) VALUES ('delay_seconds', '172800')")
		if err != nil {
			return fmt.Errorf("failed to insert default delay configuration: %w", err)
		}
	}

	// Set default delayed chains if not present
	err = s.db.QueryRow("SELECT COUNT(*) FROM configuration WHERE key = 'delayed_chains'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for delayed chains configuration: %w", err)
	}

	if count == 0 {
		// Default: delay chains 1 and 137 (Ethereum mainnet and Polygon)
		_, err = s.db.Exec("INSERT INTO configuration (key, value) VALUES ('delayed_chains', '1,137')")
		if err != nil {
			return fmt.Errorf("failed to insert default delayed chains configuration: %w", err)
		}
	}

	return nil
}

func (s *SqliteStore) Close() error {
	return s.db.Close()
}

// StoreCertificate stores a new certificate
func (s *SqliteStore) StoreCertificate(rawProto []byte, metadata string) error {
	_, err := s.db.Exec(
		"INSERT INTO certificates (raw_proto, received_at, metadata) VALUES (?, ?, ?)",
		rawProto, time.Now(), metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to store certificate: %w", err)
	}
	return nil
}

// GetProcessableCertificates retrieves certificates that are older than the configured delay and not yet processed.
func (s *SqliteStore) GetProcessableCertificates() ([]Certificate, error) {
	var delaySeconds int
	err := s.db.QueryRow("SELECT value FROM configuration WHERE key = 'delay_seconds'").Scan(&delaySeconds)
	if err != nil {
		// Check old delay_hours for backward compatibility
		var delayHours int
		err = s.db.QueryRow("SELECT value FROM configuration WHERE key = 'delay_hours'").Scan(&delayHours)
		if err != nil {
			return nil, fmt.Errorf("failed to get delay configuration: %w", err)
		}
		delaySeconds = delayHours * 3600
	}

	delay := time.Duration(delaySeconds) * time.Second
	cutoff := time.Now().Add(-delay)

	rows, err := s.db.Query("SELECT id, raw_proto, received_at, processed_at, metadata FROM certificates WHERE processed_at IS NULL AND received_at <= ?", cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to query for processable certificates: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			slog.Error("Failed to close certificates query: %v\n", closeErr)
		}
	}()

	var certs []Certificate
	for rows.Next() {
		var cert Certificate
		if err := rows.Scan(&cert.ID, &cert.RawProto, &cert.ReceivedAt, &cert.ProcessedAt, &cert.Metadata); err != nil {
			return nil, fmt.Errorf("failed to scan certificate row: %w", err)
		}
		certs = append(certs, cert)
	}

	return certs, nil
}

// MarkCertificateProcessed marks a certificate as processed.
func (s *SqliteStore) MarkCertificateProcessed(id int64) error {
	_, err := s.db.Exec("UPDATE certificates SET processed_at = ? WHERE id = ?", time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to mark certificate as processed: %w", err)
	}
	return nil
}

// GetCertificates retrieves all certificates.
func (s *SqliteStore) GetCertificates() ([]Certificate, error) {
	rows, err := s.db.Query("SELECT id, raw_proto, received_at, processed_at, metadata FROM certificates ORDER BY received_at DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to query for certificates: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			slog.Error("Failed to close certificates query: %v\n", closeErr)
		}
	}()

	var certs []Certificate
	for rows.Next() {
		var cert Certificate
		if err := rows.Scan(&cert.ID, &cert.RawProto, &cert.ReceivedAt, &cert.ProcessedAt, &cert.Metadata); err != nil {
			return nil, fmt.Errorf("failed to scan certificate row: %w", err)
		}
		certs = append(certs, cert)
	}

	return certs, nil
}

// GetConfigValue retrieves a configuration value.
func (s *SqliteStore) GetConfigValue(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM configuration WHERE key = ?", key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get config value for key %s: %w", key, err)
	}
	return value, nil
}

// SetConfigValue sets a configuration value.
func (s *SqliteStore) SetConfigValue(key, value string) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO configuration (key, value) VALUES (?, ?)", key, value)
	if err != nil {
		return fmt.Errorf("failed to set config value for key %s: %w", key, err)
	}
	return nil
}

// GetCredential retrieves a credential value.
func (s *SqliteStore) GetCredential(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM credentials WHERE key = ?", key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get credential for key %s: %w", key, err)
	}
	return value, nil
}

// SetCredential sets a credential value.
func (s *SqliteStore) SetCredential(key, value string) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO credentials (key, value) VALUES (?, ?)", key, value)
	if err != nil {
		return fmt.Errorf("failed to set credential for key %s: %w", key, err)
	}
	return nil
}

// GetSchedulerStatus retrieves the scheduler status.
func (s *SqliteStore) GetSchedulerStatus() (bool, error) {
	var isActive bool
	err := s.db.QueryRow("SELECT is_active FROM scheduler_status WHERE id = 1").Scan(&isActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// If no status exists, default to active
			return true, nil
		}
		return false, fmt.Errorf("failed to get scheduler status: %w", err)
	}
	return isActive, nil
}

// SetSchedulerStatus sets the scheduler status.
func (s *SqliteStore) SetSchedulerStatus(isActive bool) error {
	_, err := s.db.Exec("UPDATE scheduler_status SET is_active = ?, last_updated = ? WHERE id = 1", isActive, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set scheduler status: %w", err)
	}
	return nil
}

// RecordKillSwitchAttempt records a kill switch attempt.
func (s *SqliteStore) RecordKillSwitchAttempt(attemptType string) error {
	_, err := s.db.Exec("INSERT INTO kill_switch_attempts (attempt_type, attempted_at) VALUES (?, ?)", attemptType, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record kill switch attempt: %w", err)
	}
	return nil
}

// GetRecentKillSwitchAttempts retrieves recent kill switch attempts within the specified duration.
func (s *SqliteStore) GetRecentKillSwitchAttempts(attemptType string, duration time.Duration) (int, error) {
	cutoff := time.Now().Add(-duration)
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM kill_switch_attempts WHERE attempt_type = ? AND attempted_at >= ?",
		attemptType, cutoff,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get recent kill switch attempts: %w", err)
	}
	return count, nil
}

// CleanupOldKillSwitchAttempts removes old kill switch attempts.
func (s *SqliteStore) CleanupOldKillSwitchAttempts(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	_, err := s.db.Exec("DELETE FROM kill_switch_attempts WHERE attempted_at < ?", cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup old kill switch attempts: %w", err)
	}
	return nil
}
