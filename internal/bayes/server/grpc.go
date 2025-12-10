package server

import (
	"context"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	pb "github.com/msto63/mDW/api/gen/bayes"
	"github.com/msto63/mDW/internal/bayes/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Ensure Server implements BayesServiceServer
var _ pb.BayesServiceServer = (*Server)(nil)

// Log implements BayesServiceServer.Log
func (s *Server) Log(ctx context.Context, req *pb.LogRequest) (*common.Empty, error) {
	if req.Entry == nil {
		return nil, status.Error(codes.InvalidArgument, "entry is required")
	}
	if req.Entry.Service == "" {
		return nil, status.Error(codes.InvalidArgument, "service name is required")
	}
	if req.Entry.Message == "" {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	entry := &service.LogEntry{
		Service:   req.Entry.Service,
		Level:     convertProtoLevel(req.Entry.Level),
		Message:   req.Entry.Message,
		RequestID: req.Entry.RequestId,
		Metadata:  convertProtoFields(req.Entry.Fields),
	}

	if err := s.service.Log(ctx, entry); err != nil {
		s.logger.Error("Failed to log entry", "error", err)
		return nil, status.Error(codes.Internal, "failed to log entry")
	}

	return &common.Empty{}, nil
}

// LogBatch implements BayesServiceServer.LogBatch
func (s *Server) LogBatch(ctx context.Context, req *pb.LogBatchRequest) (*pb.LogBatchResponse, error) {
	if len(req.Entries) == 0 {
		return nil, status.Error(codes.InvalidArgument, "entries are required")
	}

	var accepted, rejected int32
	for _, entry := range req.Entries {
		if entry.Service == "" || entry.Message == "" {
			rejected++
			continue
		}

		svcEntry := &service.LogEntry{
			Service:   entry.Service,
			Level:     convertProtoLevel(entry.Level),
			Message:   entry.Message,
			RequestID: entry.RequestId,
			Metadata:  convertProtoFields(entry.Fields),
		}

		if err := s.service.Log(ctx, svcEntry); err != nil {
			rejected++
		} else {
			accepted++
		}
	}

	return &pb.LogBatchResponse{
		Accepted: accepted,
		Rejected: rejected,
	}, nil
}

// QueryLogs implements BayesServiceServer.QueryLogs
func (s *Server) QueryLogs(ctx context.Context, req *pb.QueryLogsRequest) (*pb.QueryLogsResponse, error) {
	filter := service.LogFilter{
		Service:   req.Service,
		Level:     convertProtoLevel(req.MinLevel),
		RequestID: req.RequestId,
		Limit:     int(req.Limit),
		Offset:    int(req.Offset),
	}

	if req.FromTimestamp > 0 {
		filter.StartTime = time.Unix(req.FromTimestamp, 0)
	}
	if req.ToTimestamp > 0 {
		filter.EndTime = time.Unix(req.ToTimestamp, 0)
	}

	entries, err := s.service.Query(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to query logs", "error", err)
		return nil, status.Error(codes.Internal, "failed to query logs")
	}

	pbEntries := make([]*pb.LogEntry, len(entries))
	for i, e := range entries {
		pbEntries[i] = &pb.LogEntry{
			Service:   e.Service,
			Level:     reverseConvertProtoLevel(e.Level),
			Message:   e.Message,
			Timestamp: e.Timestamp.Unix(),
			RequestId: e.RequestID,
			Fields:    reverseConvertProtoFields(e.Metadata),
		}
	}

	return &pb.QueryLogsResponse{
		Entries: pbEntries,
		Total:   int32(len(entries)),
		HasMore: false,
	}, nil
}

// StreamLogs implements BayesServiceServer.StreamLogs
func (s *Server) StreamLogs(req *pb.StreamLogsRequest, stream grpc.ServerStreamingServer[pb.LogEntry]) error {
	filter := service.LogFilter{
		Service: req.Service,
		Level:   convertProtoLevel(req.MinLevel),
	}

	ctx := stream.Context()
	ch, err := s.service.Stream(ctx, filter)
	if err != nil {
		return status.Error(codes.Internal, "failed to start stream")
	}

	for entry := range ch {
		pbEntry := &pb.LogEntry{
			Service:   entry.Service,
			Level:     reverseConvertProtoLevel(entry.Level),
			Message:   entry.Message,
			Timestamp: entry.Timestamp.Unix(),
			RequestId: entry.RequestID,
			Fields:    reverseConvertProtoFields(entry.Metadata),
		}
		if err := stream.Send(pbEntry); err != nil {
			return err
		}
	}

	return nil
}

// RecordMetric implements BayesServiceServer.RecordMetric
func (s *Server) RecordMetric(ctx context.Context, req *pb.MetricRequest) (*common.Empty, error) {
	if req.Entry == nil {
		return nil, status.Error(codes.InvalidArgument, "entry is required")
	}
	if req.Entry.Service == "" {
		return nil, status.Error(codes.InvalidArgument, "service name is required")
	}
	if req.Entry.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "metric name is required")
	}

	entry := &service.MetricEntry{
		Service:   req.Entry.Service,
		Name:      req.Entry.Name,
		Value:     req.Entry.Value,
		Type:      convertProtoMetricType(req.Entry.Type),
		Labels:    req.Entry.Labels,
	}
	if req.Entry.Timestamp > 0 {
		entry.Timestamp = time.Unix(req.Entry.Timestamp, 0)
	}

	if err := s.service.RecordMetric(ctx, entry); err != nil {
		s.logger.Error("Failed to record metric", "error", err)
		return nil, status.Error(codes.Internal, "failed to record metric")
	}

	return &common.Empty{}, nil
}

// RecordMetricBatch implements BayesServiceServer.RecordMetricBatch
func (s *Server) RecordMetricBatch(ctx context.Context, req *pb.MetricBatchRequest) (*common.Empty, error) {
	if len(req.Entries) == 0 {
		return nil, status.Error(codes.InvalidArgument, "entries are required")
	}

	entries := make([]*service.MetricEntry, 0, len(req.Entries))
	for _, e := range req.Entries {
		entry := &service.MetricEntry{
			Service: e.Service,
			Name:    e.Name,
			Value:   e.Value,
			Type:    convertProtoMetricType(e.Type),
			Labels:  e.Labels,
		}
		if e.Timestamp > 0 {
			entry.Timestamp = time.Unix(e.Timestamp, 0)
		}
		entries = append(entries, entry)
	}

	accepted, rejected, err := s.service.RecordMetricBatch(ctx, entries)
	if err != nil {
		s.logger.Error("Failed to record metric batch", "error", err)
		return nil, status.Error(codes.Internal, "failed to record metric batch")
	}

	s.logger.Debug("Metric batch recorded", "accepted", accepted, "rejected", rejected)
	return &common.Empty{}, nil
}

// QueryMetrics implements BayesServiceServer.QueryMetrics
func (s *Server) QueryMetrics(ctx context.Context, req *pb.QueryMetricsRequest) (*pb.QueryMetricsResponse, error) {
	filter := service.MetricFilter{
		Service:       req.Service,
		Name:          req.Name,
		Labels:        req.Labels,
		Aggregation:   convertProtoAggregation(req.Aggregation),
		BucketSeconds: int(req.BucketSeconds),
	}

	if req.FromTimestamp > 0 {
		filter.StartTime = time.Unix(req.FromTimestamp, 0)
	}
	if req.ToTimestamp > 0 {
		filter.EndTime = time.Unix(req.ToTimestamp, 0)
	}

	dataPoints, err := s.service.QueryMetrics(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to query metrics", "error", err)
		return nil, status.Error(codes.Internal, "failed to query metrics")
	}

	pbDataPoints := make([]*pb.MetricDataPoint, len(dataPoints))
	for i, dp := range dataPoints {
		pbDataPoints[i] = &pb.MetricDataPoint{
			Timestamp: dp.Timestamp.Unix(),
			Value:     dp.Value,
			Labels:    dp.Labels,
		}
	}

	return &pb.QueryMetricsResponse{
		DataPoints: pbDataPoints,
		Service:    req.Service,
		Name:       req.Name,
	}, nil
}

// HealthCheck implements BayesServiceServer.HealthCheck
func (s *Server) HealthCheck(ctx context.Context, _ *common.HealthCheckRequest) (*common.HealthCheckResponse, error) {
	result := s.health.Check(ctx)

	details := make(map[string]string)
	for _, check := range result.Checks {
		details[check.Name] = string(check.Status)
	}

	return &common.HealthCheckResponse{
		Status:        string(result.Status),
		Service:       "bayes",
		Version:       "1.0.0",
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		Details:       details,
	}, nil
}

// GetStats implements BayesServiceServer.GetStats
func (s *Server) GetStats(ctx context.Context, _ *common.Empty) (*pb.LogStats, error) {
	stats, err := s.service.GetStats(ctx)
	if err != nil {
		s.logger.Error("Failed to get stats", "error", err)
		return nil, status.Error(codes.Internal, "failed to get stats")
	}

	levelCounts := make(map[string]int64)
	for level, count := range stats.EntriesByLevel {
		levelCounts[string(level)] = count
	}

	return &pb.LogStats{
		TotalLogs:     stats.TotalEntries,
		TotalMetrics:  s.service.GetMetricsCount(),
		StorageBytes:  0, // TODO: Implement file size calculation
		LogsByLevel:   levelCounts,
		LogsByService: stats.EntriesByService,
		OldestLog:     0, // TODO: Implement oldest log tracking
		NewestLog:     stats.LastEntry.Unix(),
	}, nil
}

// Helper functions for proto type conversion

func convertProtoLevel(level pb.LogLevel) service.LogLevel {
	switch level {
	case pb.LogLevel_LOG_LEVEL_DEBUG:
		return service.LogLevelDebug
	case pb.LogLevel_LOG_LEVEL_INFO:
		return service.LogLevelInfo
	case pb.LogLevel_LOG_LEVEL_WARN:
		return service.LogLevelWarning
	case pb.LogLevel_LOG_LEVEL_ERROR:
		return service.LogLevelError
	case pb.LogLevel_LOG_LEVEL_FATAL:
		return service.LogLevelError // Map FATAL to ERROR
	default:
		return service.LogLevelInfo
	}
}

func reverseConvertProtoLevel(level service.LogLevel) pb.LogLevel {
	switch level {
	case service.LogLevelDebug:
		return pb.LogLevel_LOG_LEVEL_DEBUG
	case service.LogLevelInfo:
		return pb.LogLevel_LOG_LEVEL_INFO
	case service.LogLevelWarning:
		return pb.LogLevel_LOG_LEVEL_WARN
	case service.LogLevelError:
		return pb.LogLevel_LOG_LEVEL_ERROR
	default:
		return pb.LogLevel_LOG_LEVEL_INFO
	}
}

func convertProtoFields(fields map[string]string) map[string]interface{} {
	if fields == nil {
		return nil
	}
	result := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		result[k] = v
	}
	return result
}

func reverseConvertProtoFields(m map[string]interface{}) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}

func convertProtoMetricType(t pb.MetricType) service.MetricType {
	switch t {
	case pb.MetricType_METRIC_TYPE_COUNTER:
		return service.MetricTypeCounter
	case pb.MetricType_METRIC_TYPE_GAUGE:
		return service.MetricTypeGauge
	case pb.MetricType_METRIC_TYPE_HISTOGRAM:
		return service.MetricTypeHistogram
	default:
		return service.MetricTypeGauge
	}
}

func convertProtoAggregation(a pb.AggregationType) string {
	switch a {
	case pb.AggregationType_AGGREGATION_TYPE_SUM:
		return "SUM"
	case pb.AggregationType_AGGREGATION_TYPE_AVG:
		return "AVG"
	case pb.AggregationType_AGGREGATION_TYPE_MIN:
		return "MIN"
	case pb.AggregationType_AGGREGATION_TYPE_MAX:
		return "MAX"
	case pb.AggregationType_AGGREGATION_TYPE_COUNT:
		return "COUNT"
	default:
		return "NONE"
	}
}
