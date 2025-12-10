package server

import (
	"context"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	pb "github.com/msto63/mDW/api/gen/babbage"
	"github.com/msto63/mDW/internal/babbage/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Ensure Server implements BabbageServiceServer
var _ pb.BabbageServiceServer = (*Server)(nil)

// Analyze implements BabbageServiceServer.Analyze
func (s *Server) Analyze(ctx context.Context, req *pb.AnalyzeRequest) (*pb.AnalyzeResponse, error) {
	if req.Text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}

	result, err := s.service.Analyze(ctx, req.Text)
	if err != nil {
		s.logger.Error("Analyze failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert entities
	pbEntities := make([]*pb.Entity, len(result.Entities))
	for i, e := range result.Entities {
		pbEntities[i] = &pb.Entity{
			Text:  e.Text,
			Type:  convertEntityType(e.Type),
			Start: int32(e.Start),
			End:   int32(e.End),
		}
	}

	// Convert keywords (result.Keywords is []string)
	pbKeywords := make([]*pb.Keyword, len(result.Keywords))
	for i, k := range result.Keywords {
		pbKeywords[i] = &pb.Keyword{
			Word:  k,
			Score: 1.0,
		}
	}

	// Convert sentiment
	var pbSentiment *pb.SentimentResult
	if result.Sentiment != nil {
		pbSentiment = &pb.SentimentResult{
			Sentiment:  convertSentiment(string(result.Sentiment.Label)),
			Confidence: float32(result.Sentiment.Score),
		}
	}

	return &pb.AnalyzeResponse{
		Language:  result.Language,
		Entities:  pbEntities,
		Keywords:  pbKeywords,
		Sentiment: pbSentiment,
	}, nil
}

// ExtractEntities implements BabbageServiceServer.ExtractEntities
func (s *Server) ExtractEntities(ctx context.Context, req *pb.ExtractRequest) (*pb.EntityResponse, error) {
	if req.Text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}

	result, err := s.service.Analyze(ctx, req.Text)
	if err != nil {
		s.logger.Error("ExtractEntities failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbEntities := make([]*pb.Entity, len(result.Entities))
	for i, e := range result.Entities {
		pbEntities[i] = &pb.Entity{
			Text:  e.Text,
			Type:  convertEntityType(e.Type),
			Start: int32(e.Start),
			End:   int32(e.End),
		}
	}

	return &pb.EntityResponse{
		Entities: pbEntities,
	}, nil
}

// ExtractKeywords implements BabbageServiceServer.ExtractKeywords
func (s *Server) ExtractKeywords(ctx context.Context, req *pb.ExtractRequest) (*pb.KeywordResponse, error) {
	if req.Text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}

	maxKeywords := 10 // Default value

	keywords, err := s.service.ExtractKeywords(ctx, req.Text, maxKeywords)
	if err != nil {
		s.logger.Error("ExtractKeywords failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbKeywords := make([]*pb.Keyword, len(keywords))
	for i, k := range keywords {
		pbKeywords[i] = &pb.Keyword{
			Word:  k,
			Score: 1.0,
		}
	}

	return &pb.KeywordResponse{
		Keywords: pbKeywords,
	}, nil
}

// DetectLanguage implements BabbageServiceServer.DetectLanguage
func (s *Server) DetectLanguage(ctx context.Context, req *pb.DetectLanguageRequest) (*pb.LanguageResponse, error) {
	if req.Text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}

	lang, err := s.service.DetectLanguage(ctx, req.Text)
	if err != nil {
		s.logger.Error("DetectLanguage failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.LanguageResponse{
		Language:   lang,
		Confidence: 0.9,
	}, nil
}

// Summarize implements BabbageServiceServer.Summarize
func (s *Server) Summarize(ctx context.Context, req *pb.SummarizeRequest) (*pb.SummarizeResponse, error) {
	if req.Text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}

	maxLength := int(req.MaxLength)
	if maxLength <= 0 {
		maxLength = 200
	}

	// Convert style enum to string
	style := "brief"
	switch req.Style {
	case pb.SummarizationStyle_SUMMARIZATION_STYLE_DETAILED:
		style = "detailed"
	case pb.SummarizationStyle_SUMMARIZATION_STYLE_BULLET_POINTS:
		style = "bullet"
	case pb.SummarizationStyle_SUMMARIZATION_STYLE_HEADLINE:
		style = "headline"
	}

	svcReq := &service.SummarizeRequest{
		Text:      req.Text,
		MaxLength: maxLength,
		Style:     style,
	}

	summary, err := s.service.Summarize(ctx, svcReq)
	if err != nil {
		s.logger.Error("Summarize failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.SummarizeResponse{
		Summary:        summary,
		OriginalLength: int32(len(req.Text)),
		SummaryLength:  int32(len(summary)),
	}, nil
}

// Translate implements BabbageServiceServer.Translate
func (s *Server) Translate(ctx context.Context, req *pb.TranslateRequest) (*pb.TranslateResponse, error) {
	if req.Text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}
	if req.TargetLanguage == "" {
		return nil, status.Error(codes.InvalidArgument, "target_language is required")
	}

	svcReq := &service.TranslateRequest{
		Text:           req.Text,
		SourceLanguage: req.SourceLanguage,
		TargetLanguage: req.TargetLanguage,
	}

	result, err := s.service.Translate(ctx, svcReq)
	if err != nil {
		s.logger.Error("Translate failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.TranslateResponse{
		TranslatedText: result.TranslatedText,
		SourceLanguage: result.SourceLanguage,
		TargetLanguage: result.TargetLanguage,
	}, nil
}

// Classify implements BabbageServiceServer.Classify
func (s *Server) Classify(ctx context.Context, req *pb.ClassifyRequest) (*pb.ClassifyResponse, error) {
	if req.Text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}
	if len(req.Categories) == 0 {
		return nil, status.Error(codes.InvalidArgument, "categories are required")
	}

	svcReq := &service.ClassifyRequest{
		Text:   req.Text,
		Labels: req.Categories,
	}

	result, err := s.service.Classify(ctx, svcReq)
	if err != nil {
		s.logger.Error("Classify failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Create scores for all categories (service returns single result)
	pbScores := make([]*pb.CategoryScore, len(req.Categories))
	for i, cat := range req.Categories {
		score := float32(0.0)
		if cat == result.Label {
			score = float32(result.Score)
		}
		pbScores[i] = &pb.CategoryScore{
			Category: cat,
			Score:    score,
		}
	}

	return &pb.ClassifyResponse{
		Category:   result.Label,
		Confidence: float32(result.Score),
		Scores:     pbScores,
	}, nil
}

// AnalyzeSentiment implements BabbageServiceServer.AnalyzeSentiment
func (s *Server) AnalyzeSentiment(ctx context.Context, req *pb.SentimentRequest) (*pb.SentimentResponse, error) {
	if req.Text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}

	result, err := s.service.Analyze(ctx, req.Text)
	if err != nil {
		s.logger.Error("AnalyzeSentiment failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	var pbSentiment pb.Sentiment
	var confidence float32
	if result.Sentiment != nil {
		pbSentiment = convertSentiment(string(result.Sentiment.Label))
		confidence = float32(result.Sentiment.Score)
	}

	return &pb.SentimentResponse{
		Result: &pb.SentimentResult{
			Sentiment:  pbSentiment,
			Confidence: confidence,
		},
	}, nil
}

// HealthCheck implements BabbageServiceServer.HealthCheck
func (s *Server) HealthCheck(ctx context.Context, _ *common.HealthCheckRequest) (*common.HealthCheckResponse, error) {
	result := s.health.Check(ctx)

	details := make(map[string]string)
	for _, check := range result.Checks {
		details[check.Name] = string(check.Status)
	}

	return &common.HealthCheckResponse{
		Status:        string(result.Status),
		Service:       "babbage",
		Version:       "1.0.0",
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		Details:       details,
	}, nil
}

// Helper function to convert entity type
func convertEntityType(t string) pb.EntityType {
	switch t {
	case "PERSON":
		return pb.EntityType_ENTITY_TYPE_PERSON
	case "ORGANIZATION", "ORG":
		return pb.EntityType_ENTITY_TYPE_ORGANIZATION
	case "LOCATION", "LOC", "GPE":
		return pb.EntityType_ENTITY_TYPE_LOCATION
	case "DATE":
		return pb.EntityType_ENTITY_TYPE_DATE
	case "TIME":
		return pb.EntityType_ENTITY_TYPE_TIME
	case "MONEY":
		return pb.EntityType_ENTITY_TYPE_MONEY
	case "PERCENT":
		return pb.EntityType_ENTITY_TYPE_PERCENT
	default:
		return pb.EntityType_ENTITY_TYPE_UNKNOWN
	}
}

// Helper function to convert sentiment
func convertSentiment(sentiment string) pb.Sentiment {
	switch sentiment {
	case "positive":
		return pb.Sentiment_SENTIMENT_POSITIVE
	case "negative":
		return pb.Sentiment_SENTIMENT_NEGATIVE
	case "neutral":
		return pb.Sentiment_SENTIMENT_NEUTRAL
	case "mixed":
		return pb.Sentiment_SENTIMENT_MIXED
	default:
		return pb.Sentiment_SENTIMENT_UNKNOWN
	}
}
