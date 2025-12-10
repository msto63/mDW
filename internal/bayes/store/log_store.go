package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LogLevelDebug   LogLevel = "DEBUG"
	LogLevelInfo    LogLevel = "INFO"
	LogLevelWarning LogLevel = "WARNING"
	LogLevelError   LogLevel = "ERROR"
)

// LogEntry represents a single log entry
type LogEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Service   string                 `json:"service"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	RequestID string                 `json:"request_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// LogFilter defines criteria for filtering logs
type LogFilter struct {
	Service   string
	Level     LogLevel
	StartTime time.Time
	EndTime   time.Time
	RequestID string
	Limit     int
	Offset    int
}

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "COUNTER"
	MetricTypeGauge     MetricType = "GAUGE"
	MetricTypeHistogram MetricType = "HISTOGRAM"
)

// MetricEntry represents a single metric data point
type MetricEntry struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Service   string            `json:"service"`
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Type      MetricType        `json:"type"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// MetricFilter defines criteria for filtering metrics
type MetricFilter struct {
	Service       string
	Name          string
	StartTime     time.Time
	EndTime       time.Time
	Labels        map[string]string
	Aggregation   string
	BucketSeconds int
	Limit         int
}

// MetricDataPoint represents an aggregated metric data point
type MetricDataPoint struct {
	Timestamp time.Time
	Value     float64
	Labels    map[string]string
}

// LogStore defines the interface for log persistence
type LogStore interface {
	// Log operations
	Log(ctx context.Context, entry *LogEntry) error
	LogBatch(ctx context.Context, entries []*LogEntry) (int, int, error)
	Query(ctx context.Context, filter LogFilter) ([]*LogEntry, error)

	// Metric operations
	RecordMetric(ctx context.Context, entry *MetricEntry) error
	RecordMetricBatch(ctx context.Context, entries []*MetricEntry) (int, int, error)
	QueryMetrics(ctx context.Context, filter MetricFilter) ([]*MetricDataPoint, error)

	// Statistics
	GetLogStats(ctx context.Context) (map[string]interface{}, error)
	GetMetricStats(ctx context.Context) (map[string]interface{}, error)

	// Maintenance
	Vacuum(ctx context.Context) error
	Prune(ctx context.Context, olderThan time.Duration) (int64, error)
	Close() error
}

// SQLiteLogStore implements LogStore using SQLite
type SQLiteLogStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// SQLiteLogConfig holds configuration for SQLite store
type SQLiteLogConfig struct {
	Path string
}

// DefaultLogConfig returns default configuration
func DefaultLogConfig() SQLiteLogConfig {
	return SQLiteLogConfig{
		Path: "./data/logs.db",
	}
}

// NewSQLiteLogStore creates a new SQLite-based log store
func NewSQLiteLogStore(cfg SQLiteLogConfig) (*SQLiteLogStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database with WAL mode
	db, err := sql.Open("sqlite3", cfg.Path+"?_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteLogStore{db: db}

	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the necessary tables
func (s *SQLiteLogStore) initSchema() error {
	schema := `
	-- Log entries table
	CREATE TABLE IF NOT EXISTS logs (
		id TEXT PRIMARY KEY,
		timestamp DATETIME NOT NULL,
		service TEXT NOT NULL,
		level TEXT NOT NULL,
		message TEXT NOT NULL,
		request_id TEXT,
		metadata TEXT
	);

	-- Metric entries table
	CREATE TABLE IF NOT EXISTS metrics (
		id TEXT PRIMARY KEY,
		timestamp DATETIME NOT NULL,
		service TEXT NOT NULL,
		name TEXT NOT NULL,
		value REAL NOT NULL,
		type TEXT NOT NULL,
		labels TEXT
	);

	-- Indices for efficient querying
	CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_logs_service ON logs(service);
	CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level);
	CREATE INDEX IF NOT EXISTS idx_logs_request_id ON logs(request_id);
	CREATE INDEX IF NOT EXISTS idx_logs_service_level ON logs(service, level);

	CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_metrics_service ON metrics(service);
	CREATE INDEX IF NOT EXISTS idx_metrics_name ON metrics(name);
	CREATE INDEX IF NOT EXISTS idx_metrics_service_name ON metrics(service, name);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Log records a new log entry
func (s *SQLiteLogStore) Log(ctx context.Context, entry *LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry.ID == "" {
		entry.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	var metadataJSON []byte
	if entry.Metadata != nil {
		metadataJSON, _ = json.Marshal(entry.Metadata)
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO logs (id, timestamp, service, level, message, request_id, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, entry.Timestamp, entry.Service, entry.Level, entry.Message, entry.RequestID, metadataJSON)

	if err != nil {
		return fmt.Errorf("failed to insert log entry: %w", err)
	}

	return nil
}

// LogBatch records multiple log entries
func (s *SQLiteLogStore) LogBatch(ctx context.Context, entries []*LogEntry) (int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, len(entries), fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO logs (id, timestamp, service, level, message, request_id, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, len(entries), fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	var accepted, rejected int
	for _, entry := range entries {
		if entry.ID == "" {
			entry.ID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}

		var metadataJSON []byte
		if entry.Metadata != nil {
			metadataJSON, _ = json.Marshal(entry.Metadata)
		}

		_, err := stmt.ExecContext(ctx, entry.ID, entry.Timestamp, entry.Service, entry.Level,
			entry.Message, entry.RequestID, metadataJSON)
		if err != nil {
			rejected++
		} else {
			accepted++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, len(entries), fmt.Errorf("failed to commit transaction: %w", err)
	}

	return accepted, rejected, nil
}

// Query retrieves log entries based on filter criteria
func (s *SQLiteLogStore) Query(ctx context.Context, filter LogFilter) ([]*LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, timestamp, service, level, message, request_id, metadata FROM logs WHERE 1=1`
	var args []interface{}

	if filter.Service != "" {
		query += " AND service = ?"
		args = append(args, filter.Service)
	}
	if filter.Level != "" {
		query += " AND level = ?"
		args = append(args, filter.Level)
	}
	if !filter.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.EndTime)
	}
	if filter.RequestID != "" {
		query += " AND request_id = ?"
		args = append(args, filter.RequestID)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var entries []*LogEntry
	for rows.Next() {
		var entry LogEntry
		var metadataJSON sql.NullString
		var requestID sql.NullString

		if err := rows.Scan(&entry.ID, &entry.Timestamp, &entry.Service, &entry.Level,
			&entry.Message, &requestID, &metadataJSON); err != nil {
			return nil, fmt.Errorf("failed to scan log entry: %w", err)
		}

		if requestID.Valid {
			entry.RequestID = requestID.String
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &entry.Metadata)
		}

		entries = append(entries, &entry)
	}

	return entries, nil
}

// RecordMetric records a single metric entry
func (s *SQLiteLogStore) RecordMetric(ctx context.Context, entry *MetricEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry.ID == "" {
		entry.ID = fmt.Sprintf("m%d", time.Now().UnixNano())
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	var labelsJSON []byte
	if entry.Labels != nil {
		labelsJSON, _ = json.Marshal(entry.Labels)
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO metrics (id, timestamp, service, name, value, type, labels)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, entry.Timestamp, entry.Service, entry.Name, entry.Value, entry.Type, labelsJSON)

	if err != nil {
		return fmt.Errorf("failed to insert metric: %w", err)
	}

	return nil
}

// RecordMetricBatch records multiple metric entries
func (s *SQLiteLogStore) RecordMetricBatch(ctx context.Context, entries []*MetricEntry) (int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, len(entries), fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO metrics (id, timestamp, service, name, value, type, labels)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, len(entries), fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	var accepted, rejected int
	for _, entry := range entries {
		if entry.Service == "" || entry.Name == "" {
			rejected++
			continue
		}

		if entry.ID == "" {
			entry.ID = fmt.Sprintf("m%d", time.Now().UnixNano())
		}
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}

		var labelsJSON []byte
		if entry.Labels != nil {
			labelsJSON, _ = json.Marshal(entry.Labels)
		}

		_, err := stmt.ExecContext(ctx, entry.ID, entry.Timestamp, entry.Service, entry.Name,
			entry.Value, entry.Type, labelsJSON)
		if err != nil {
			rejected++
		} else {
			accepted++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, len(entries), fmt.Errorf("failed to commit transaction: %w", err)
	}

	return accepted, rejected, nil
}

// QueryMetrics retrieves metric data points based on filter criteria
func (s *SQLiteLogStore) QueryMetrics(ctx context.Context, filter MetricFilter) ([]*MetricDataPoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build base query
	query := `SELECT timestamp, value, labels FROM metrics WHERE 1=1`
	var args []interface{}

	if filter.Service != "" {
		query += " AND service = ?"
		args = append(args, filter.Service)
	}
	if filter.Name != "" {
		query += " AND name = ?"
		args = append(args, filter.Name)
	}
	if !filter.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.EndTime)
	}

	query += " ORDER BY timestamp ASC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var points []*MetricDataPoint
	for rows.Next() {
		var point MetricDataPoint
		var labelsJSON sql.NullString

		if err := rows.Scan(&point.Timestamp, &point.Value, &labelsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		if labelsJSON.Valid {
			json.Unmarshal([]byte(labelsJSON.String), &point.Labels)
		}

		// Filter by labels if specified
		if filter.Labels != nil {
			match := true
			for k, v := range filter.Labels {
				if point.Labels[k] != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		points = append(points, &point)
	}

	// Apply aggregation if specified
	if filter.Aggregation != "" && filter.Aggregation != "NONE" && filter.BucketSeconds > 0 {
		points = aggregateMetrics(points, filter.Aggregation, filter.BucketSeconds)
	}

	return points, nil
}

// aggregateMetrics aggregates metric data points into buckets
func aggregateMetrics(points []*MetricDataPoint, aggregation string, bucketSeconds int) []*MetricDataPoint {
	if len(points) == 0 {
		return points
	}

	// Group by bucket
	buckets := make(map[int64][]float64)
	for _, p := range points {
		bucket := p.Timestamp.Unix() / int64(bucketSeconds) * int64(bucketSeconds)
		buckets[bucket] = append(buckets[bucket], p.Value)
	}

	// Aggregate each bucket
	var results []*MetricDataPoint
	for bucket, values := range buckets {
		var aggregatedValue float64
		switch aggregation {
		case "SUM":
			for _, v := range values {
				aggregatedValue += v
			}
		case "AVG":
			for _, v := range values {
				aggregatedValue += v
			}
			aggregatedValue /= float64(len(values))
		case "MIN":
			aggregatedValue = values[0]
			for _, v := range values[1:] {
				if v < aggregatedValue {
					aggregatedValue = v
				}
			}
		case "MAX":
			aggregatedValue = values[0]
			for _, v := range values[1:] {
				if v > aggregatedValue {
					aggregatedValue = v
				}
			}
		case "COUNT":
			aggregatedValue = float64(len(values))
		default:
			aggregatedValue = values[0]
		}

		results = append(results, &MetricDataPoint{
			Timestamp: time.Unix(bucket, 0),
			Value:     aggregatedValue,
		})
	}

	return results
}

// GetLogStats returns log statistics
func (s *SQLiteLogStore) GetLogStats(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	// Total entries
	var total int64
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM logs`).Scan(&total)
	stats["total_entries"] = total

	// Entries by level
	levelCounts := make(map[string]int64)
	rows, _ := s.db.QueryContext(ctx, `SELECT level, COUNT(*) FROM logs GROUP BY level`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var level string
			var count int64
			rows.Scan(&level, &count)
			levelCounts[level] = count
		}
	}
	stats["entries_by_level"] = levelCounts

	// Entries by service
	serviceCounts := make(map[string]int64)
	rows2, _ := s.db.QueryContext(ctx, `SELECT service, COUNT(*) FROM logs GROUP BY service`)
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var service string
			var count int64
			rows2.Scan(&service, &count)
			serviceCounts[service] = count
		}
	}
	stats["entries_by_service"] = serviceCounts

	// Last entry time
	var lastEntry sql.NullTime
	s.db.QueryRowContext(ctx, `SELECT MAX(timestamp) FROM logs`).Scan(&lastEntry)
	if lastEntry.Valid {
		stats["last_entry"] = lastEntry.Time
	}

	return stats, nil
}

// GetMetricStats returns metric statistics
func (s *SQLiteLogStore) GetMetricStats(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	// Total metrics
	var total int64
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM metrics`).Scan(&total)
	stats["total_metrics"] = total

	// Metrics by service
	serviceCounts := make(map[string]int64)
	rows, _ := s.db.QueryContext(ctx, `SELECT service, COUNT(*) FROM metrics GROUP BY service`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var service string
			var count int64
			rows.Scan(&service, &count)
			serviceCounts[service] = count
		}
	}
	stats["metrics_by_service"] = serviceCounts

	// Metrics by name
	nameCounts := make(map[string]int64)
	rows2, _ := s.db.QueryContext(ctx, `SELECT name, COUNT(*) FROM metrics GROUP BY name ORDER BY COUNT(*) DESC LIMIT 20`)
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var name string
			var count int64
			rows2.Scan(&name, &count)
			nameCounts[name] = count
		}
	}
	stats["top_metrics"] = nameCounts

	return stats, nil
}

// Vacuum optimizes the database
func (s *SQLiteLogStore) Vacuum(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `VACUUM`)
	return err
}

// Prune removes entries older than the specified duration
func (s *SQLiteLogStore) Prune(ctx context.Context, olderThan time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)

	// Delete old logs
	result1, err := s.db.ExecContext(ctx, `DELETE FROM logs WHERE timestamp < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to prune logs: %w", err)
	}
	logsDeleted, _ := result1.RowsAffected()

	// Delete old metrics
	result2, err := s.db.ExecContext(ctx, `DELETE FROM metrics WHERE timestamp < ?`, cutoff)
	if err != nil {
		return logsDeleted, fmt.Errorf("failed to prune metrics: %w", err)
	}
	metricsDeleted, _ := result2.RowsAffected()

	return logsDeleted + metricsDeleted, nil
}

// Close closes the database connection
func (s *SQLiteLogStore) Close() error {
	return s.db.Close()
}

// MemoryLogStore is an in-memory implementation for testing
type MemoryLogStore struct {
	mu      sync.RWMutex
	logs    []*LogEntry
	metrics []*MetricEntry
}

// NewMemoryLogStore creates a new in-memory log store
func NewMemoryLogStore() *MemoryLogStore {
	return &MemoryLogStore{
		logs:    make([]*LogEntry, 0),
		metrics: make([]*MetricEntry, 0),
	}
}

// Log records a new log entry
func (s *MemoryLogStore) Log(ctx context.Context, entry *LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry.ID == "" {
		entry.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	s.logs = append(s.logs, entry)
	return nil
}

// LogBatch records multiple log entries
func (s *MemoryLogStore) LogBatch(ctx context.Context, entries []*LogEntry) (int, int, error) {
	for _, entry := range entries {
		s.Log(ctx, entry)
	}
	return len(entries), 0, nil
}

// Query retrieves log entries based on filter criteria
func (s *MemoryLogStore) Query(ctx context.Context, filter LogFilter) ([]*LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*LogEntry
	for _, entry := range s.logs {
		if filter.Service != "" && entry.Service != filter.Service {
			continue
		}
		if filter.Level != "" && entry.Level != filter.Level {
			continue
		}
		if !filter.StartTime.IsZero() && entry.Timestamp.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && entry.Timestamp.After(filter.EndTime) {
			continue
		}
		if filter.RequestID != "" && entry.RequestID != filter.RequestID {
			continue
		}
		results = append(results, entry)
	}

	if filter.Offset > 0 && filter.Offset < len(results) {
		results = results[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(results) {
		results = results[:filter.Limit]
	}

	return results, nil
}

// RecordMetric records a single metric entry
func (s *MemoryLogStore) RecordMetric(ctx context.Context, entry *MetricEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry.ID == "" {
		entry.ID = fmt.Sprintf("m%d", time.Now().UnixNano())
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	s.metrics = append(s.metrics, entry)
	return nil
}

// RecordMetricBatch records multiple metric entries
func (s *MemoryLogStore) RecordMetricBatch(ctx context.Context, entries []*MetricEntry) (int, int, error) {
	var accepted, rejected int
	for _, entry := range entries {
		if entry.Service == "" || entry.Name == "" {
			rejected++
			continue
		}
		s.RecordMetric(ctx, entry)
		accepted++
	}
	return accepted, rejected, nil
}

// QueryMetrics retrieves metric data points
func (s *MemoryLogStore) QueryMetrics(ctx context.Context, filter MetricFilter) ([]*MetricDataPoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*MetricDataPoint
	for _, entry := range s.metrics {
		if filter.Service != "" && entry.Service != filter.Service {
			continue
		}
		if filter.Name != "" && entry.Name != filter.Name {
			continue
		}
		if !filter.StartTime.IsZero() && entry.Timestamp.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && entry.Timestamp.After(filter.EndTime) {
			continue
		}
		results = append(results, &MetricDataPoint{
			Timestamp: entry.Timestamp,
			Value:     entry.Value,
			Labels:    entry.Labels,
		})
	}

	if filter.Aggregation != "" && filter.BucketSeconds > 0 {
		results = aggregateMetrics(results, filter.Aggregation, filter.BucketSeconds)
	}

	return results, nil
}

// GetLogStats returns log statistics
func (s *MemoryLogStore) GetLogStats(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	levelCounts := make(map[string]int64)
	serviceCounts := make(map[string]int64)
	for _, entry := range s.logs {
		levelCounts[string(entry.Level)]++
		serviceCounts[entry.Service]++
	}

	return map[string]interface{}{
		"total_entries":      len(s.logs),
		"entries_by_level":   levelCounts,
		"entries_by_service": serviceCounts,
	}, nil
}

// GetMetricStats returns metric statistics
func (s *MemoryLogStore) GetMetricStats(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	serviceCounts := make(map[string]int64)
	for _, entry := range s.metrics {
		serviceCounts[entry.Service]++
	}

	return map[string]interface{}{
		"total_metrics":      len(s.metrics),
		"metrics_by_service": serviceCounts,
	}, nil
}

// Vacuum is a no-op for memory store
func (s *MemoryLogStore) Vacuum(ctx context.Context) error {
	return nil
}

// Prune removes old entries
func (s *MemoryLogStore) Prune(ctx context.Context, olderThan time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	var deleted int64

	// Prune logs
	newLogs := make([]*LogEntry, 0)
	for _, entry := range s.logs {
		if entry.Timestamp.After(cutoff) {
			newLogs = append(newLogs, entry)
		} else {
			deleted++
		}
	}
	s.logs = newLogs

	// Prune metrics
	newMetrics := make([]*MetricEntry, 0)
	for _, entry := range s.metrics {
		if entry.Timestamp.After(cutoff) {
			newMetrics = append(newMetrics, entry)
		} else {
			deleted++
		}
	}
	s.metrics = newMetrics

	return deleted, nil
}

// Close is a no-op for memory store
func (s *MemoryLogStore) Close() error {
	return nil
}
