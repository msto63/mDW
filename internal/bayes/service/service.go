package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/msto63/mDW/internal/bayes/store"
	"github.com/msto63/mDW/pkg/core/logging"
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

// LogStats contains aggregated log statistics
type LogStats struct {
	TotalEntries     int64
	EntriesByLevel   map[LogLevel]int64
	EntriesByService map[string]int64
	LastEntry        time.Time
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
}

// MetricDataPoint represents an aggregated metric data point
type MetricDataPoint struct {
	Timestamp time.Time
	Value     float64
	Labels    map[string]string
}

// Service is the Bayes logging service
type Service struct {
	logger        *logging.Logger
	logDir        string
	mu            sync.RWMutex
	entries       []*LogEntry
	maxSize       int
	fileOut       *os.File
	metricsMu     sync.RWMutex
	metrics       []*MetricEntry
	maxMetrics    int
	metricsFile   *os.File
	store         store.LogStore
}

// Config holds configuration for the Bayes service
type Config struct {
	LogDir            string
	MaxMemEntries     int
	MaxMetrics        int
	LogToFile         bool
	StorePath         string
	EnablePersistence bool
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		LogDir:            "logs",
		MaxMemEntries:     10000,
		MaxMetrics:        50000,
		LogToFile:         true,
		StorePath:         "./data/logs.db",
		EnablePersistence: true,
	}
}

// NewService creates a new Bayes logging service
func NewService(cfg Config) (*Service, error) {
	logger := logging.New("bayes")

	maxMetrics := cfg.MaxMetrics
	if maxMetrics == 0 {
		maxMetrics = 50000
	}

	svc := &Service{
		logger:     logger,
		logDir:     cfg.LogDir,
		entries:    make([]*LogEntry, 0, cfg.MaxMemEntries),
		maxSize:    cfg.MaxMemEntries,
		metrics:    make([]*MetricEntry, 0, maxMetrics),
		maxMetrics: maxMetrics,
	}

	// Initialize SQLite store if enabled
	if cfg.EnablePersistence {
		logStore, err := store.NewSQLiteLogStore(store.SQLiteLogConfig{
			Path: cfg.StorePath,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create log store: %w", err)
		}
		svc.store = logStore
		logger.Info("Log persistence enabled", "path", cfg.StorePath)
	}

	if cfg.LogToFile {
		if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		logFile := filepath.Join(cfg.LogDir, fmt.Sprintf("mdw-%s.jsonl", time.Now().Format("2006-01-02")))
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		svc.fileOut = file

		metricsFile := filepath.Join(cfg.LogDir, fmt.Sprintf("mdw-metrics-%s.jsonl", time.Now().Format("2006-01-02")))
		mFile, err := os.OpenFile(metricsFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open metrics file: %w", err)
		}
		svc.metricsFile = mFile
	}

	return svc, nil
}

// Log records a new log entry
func (s *Service) Log(ctx context.Context, entry *LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ID if not set
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Set timestamp if not set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Add to memory buffer
	s.entries = append(s.entries, entry)

	// Trim if exceeds max size (keep last half)
	if len(s.entries) > s.maxSize {
		s.entries = s.entries[s.maxSize/2:]
	}

	// Persist to SQLite store
	if s.store != nil {
		s.store.Log(ctx, toStoreLogEntry(entry))
	}

	// Write to file if enabled
	if s.fileOut != nil {
		data, err := json.Marshal(entry)
		if err != nil {
			s.logger.Error("Failed to marshal log entry", "error", err)
		} else {
			s.fileOut.Write(append(data, '\n'))
		}
	}

	// Log locally for debugging
	switch entry.Level {
	case LogLevelDebug:
		s.logger.Debug(entry.Message, "service", entry.Service, "request_id", entry.RequestID)
	case LogLevelInfo:
		s.logger.Info(entry.Message, "service", entry.Service, "request_id", entry.RequestID)
	case LogLevelWarning:
		s.logger.Warn(entry.Message, "service", entry.Service, "request_id", entry.RequestID)
	case LogLevelError:
		s.logger.Error(entry.Message, "service", entry.Service, "request_id", entry.RequestID)
	}

	return nil
}

// toStoreLogEntry converts service LogEntry to store LogEntry
func toStoreLogEntry(e *LogEntry) *store.LogEntry {
	return &store.LogEntry{
		ID:        e.ID,
		Timestamp: e.Timestamp,
		Service:   e.Service,
		Level:     store.LogLevel(e.Level),
		Message:   e.Message,
		RequestID: e.RequestID,
		Metadata:  e.Metadata,
	}
}

// Query retrieves log entries based on filter criteria
func (s *Service) Query(ctx context.Context, filter LogFilter) ([]*LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*LogEntry

	for _, entry := range s.entries {
		// Apply filters
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

	// Apply offset and limit
	if filter.Offset > 0 {
		if filter.Offset >= len(results) {
			return []*LogEntry{}, nil
		}
		results = results[filter.Offset:]
	}

	if filter.Limit > 0 && filter.Limit < len(results) {
		results = results[:filter.Limit]
	}

	return results, nil
}

// GetStats returns aggregated statistics
func (s *Service) GetStats(ctx context.Context) (*LogStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &LogStats{
		TotalEntries:     int64(len(s.entries)),
		EntriesByLevel:   make(map[LogLevel]int64),
		EntriesByService: make(map[string]int64),
	}

	for _, entry := range s.entries {
		stats.EntriesByLevel[entry.Level]++
		stats.EntriesByService[entry.Service]++
		if entry.Timestamp.After(stats.LastEntry) {
			stats.LastEntry = entry.Timestamp
		}
	}

	return stats, nil
}

// Stream returns a channel for real-time log streaming
func (s *Service) Stream(ctx context.Context, filter LogFilter) (<-chan *LogEntry, error) {
	ch := make(chan *LogEntry, 100)

	go func() {
		defer close(ch)

		// First send existing entries that match filter
		s.mu.RLock()
		for _, entry := range s.entries {
			if matchesFilter(entry, filter) {
				select {
				case ch <- entry:
				case <-ctx.Done():
					s.mu.RUnlock()
					return
				}
			}
		}
		s.mu.RUnlock()

		// Then watch for new entries
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		lastIdx := len(s.entries)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.mu.RLock()
				for i := lastIdx; i < len(s.entries); i++ {
					if matchesFilter(s.entries[i], filter) {
						select {
						case ch <- s.entries[i]:
						case <-ctx.Done():
							s.mu.RUnlock()
							return
						}
					}
				}
				lastIdx = len(s.entries)
				s.mu.RUnlock()
			}
		}
	}()

	return ch, nil
}

// Close closes the service and releases resources
func (s *Service) Close() error {
	var errs []error
	if s.fileOut != nil {
		if err := s.fileOut.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.metricsFile != nil {
		if err := s.metricsFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.store != nil {
		if err := s.store.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// RecordMetric records a single metric entry
func (s *Service) RecordMetric(ctx context.Context, entry *MetricEntry) error {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()

	// Generate ID if not set
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("m%d", time.Now().UnixNano())
	}

	// Set timestamp if not set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Add to memory buffer
	s.metrics = append(s.metrics, entry)

	// Trim if exceeds max size (keep last half)
	if len(s.metrics) > s.maxMetrics {
		s.metrics = s.metrics[s.maxMetrics/2:]
	}

	// Persist to SQLite store
	if s.store != nil {
		s.store.RecordMetric(ctx, toStoreMetricEntry(entry))
	}

	// Write to file if enabled
	if s.metricsFile != nil {
		data, err := json.Marshal(entry)
		if err != nil {
			s.logger.Error("Failed to marshal metric entry", "error", err)
		} else {
			s.metricsFile.Write(append(data, '\n'))
		}
	}

	s.logger.Debug("Metric recorded",
		"service", entry.Service,
		"name", entry.Name,
		"value", entry.Value,
		"type", entry.Type,
	)

	return nil
}

// toStoreMetricEntry converts service MetricEntry to store MetricEntry
func toStoreMetricEntry(e *MetricEntry) *store.MetricEntry {
	return &store.MetricEntry{
		ID:        e.ID,
		Timestamp: e.Timestamp,
		Service:   e.Service,
		Name:      e.Name,
		Value:     e.Value,
		Type:      store.MetricType(e.Type),
		Labels:    e.Labels,
	}
}

// RecordMetricBatch records multiple metric entries
func (s *Service) RecordMetricBatch(ctx context.Context, entries []*MetricEntry) (int, int, error) {
	var accepted, rejected int

	for _, entry := range entries {
		if entry.Service == "" || entry.Name == "" {
			rejected++
			continue
		}

		if err := s.RecordMetric(ctx, entry); err != nil {
			rejected++
		} else {
			accepted++
		}
	}

	return accepted, rejected, nil
}

// QueryMetrics retrieves metric data points based on filter criteria
func (s *Service) QueryMetrics(ctx context.Context, filter MetricFilter) ([]*MetricDataPoint, error) {
	s.metricsMu.RLock()
	defer s.metricsMu.RUnlock()

	var results []*MetricDataPoint

	for _, entry := range s.metrics {
		// Apply filters
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
		if filter.Labels != nil {
			match := true
			for k, v := range filter.Labels {
				if entry.Labels[k] != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		results = append(results, &MetricDataPoint{
			Timestamp: entry.Timestamp,
			Value:     entry.Value,
			Labels:    entry.Labels,
		})
	}

	// Apply aggregation if specified
	if filter.Aggregation != "" && filter.Aggregation != "NONE" && filter.BucketSeconds > 0 {
		results = s.aggregateMetrics(results, filter.Aggregation, filter.BucketSeconds)
	}

	return results, nil
}

// aggregateMetrics aggregates metric data points into buckets
func (s *Service) aggregateMetrics(points []*MetricDataPoint, aggregation string, bucketSeconds int) []*MetricDataPoint {
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
			// No aggregation, use first value
			aggregatedValue = values[0]
		}

		results = append(results, &MetricDataPoint{
			Timestamp: time.Unix(bucket, 0),
			Value:     aggregatedValue,
		})
	}

	return results
}

// GetMetricsCount returns the total number of metrics in memory
func (s *Service) GetMetricsCount() int64 {
	s.metricsMu.RLock()
	defer s.metricsMu.RUnlock()
	return int64(len(s.metrics))
}

// matchesFilter checks if an entry matches the given filter
func matchesFilter(entry *LogEntry, filter LogFilter) bool {
	if filter.Service != "" && entry.Service != filter.Service {
		return false
	}
	if filter.Level != "" && entry.Level != filter.Level {
		return false
	}
	if !filter.StartTime.IsZero() && entry.Timestamp.Before(filter.StartTime) {
		return false
	}
	if !filter.EndTime.IsZero() && entry.Timestamp.After(filter.EndTime) {
		return false
	}
	if filter.RequestID != "" && entry.RequestID != filter.RequestID {
		return false
	}
	return true
}
