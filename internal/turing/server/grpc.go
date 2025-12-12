package server

import (
	"context"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	pb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/internal/turing/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Ensure Server implements TuringServiceServer
var _ pb.TuringServiceServer = (*Server)(nil)

// Chat implements TuringServiceServer.Chat
func (s *Server) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
	if len(req.Messages) == 0 {
		return nil, status.Error(codes.InvalidArgument, "messages are required")
	}

	messages := make([]service.Message, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = service.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	svcReq := &service.ChatRequest{
		Messages:    messages,
		Model:       req.Model,
		MaxTokens:   int(req.MaxTokens),
		Temperature: float64(req.Temperature),
		TopP:        float64(req.TopP),
	}

	resp, err := s.service.Chat(ctx, svcReq)
	if err != nil {
		s.logger.Error("Chat failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.ChatResponse{
		Content:          resp.Message.Content,
		Model:            resp.Model,
		PromptTokens:     int32(resp.PromptTokens),
		CompletionTokens: int32(resp.OutputTokens),
		TotalTokens:      int32(resp.PromptTokens + resp.OutputTokens),
		FinishReason:     "stop",
	}, nil
}

// StreamChat implements TuringServiceServer.StreamChat
func (s *Server) StreamChat(req *pb.ChatRequest, stream grpc.ServerStreamingServer[pb.ChatChunk]) error {
	if len(req.Messages) == 0 {
		return status.Error(codes.InvalidArgument, "messages are required")
	}

	messages := make([]service.Message, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = service.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	svcReq := &service.ChatRequest{
		Messages:    messages,
		Model:       req.Model,
		MaxTokens:   int(req.MaxTokens),
		Temperature: float64(req.Temperature),
		TopP:        float64(req.TopP),
		Stream:      true,
	}

	ctx := stream.Context()
	respCh, errCh := s.service.ChatStream(ctx, svcReq)

	for {
		select {
		case resp, ok := <-respCh:
			if !ok {
				return nil
			}
			chunk := &pb.ChatChunk{
				Delta:            resp.Message.Content,
				Done:             resp.Done,
				PromptTokens:     int32(resp.PromptTokens),
				CompletionTokens: int32(resp.OutputTokens),
			}
			if resp.Done {
				chunk.FinishReason = "stop"
			}
			if err := stream.Send(chunk); err != nil {
				return err
			}
		case err := <-errCh:
			if err != nil {
				return status.Error(codes.Internal, err.Error())
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Embed implements TuringServiceServer.Embed
func (s *Server) Embed(ctx context.Context, req *pb.EmbedRequest) (*pb.EmbedResponse, error) {
	if req.Input == "" {
		return nil, status.Error(codes.InvalidArgument, "input is required")
	}

	svcReq := &service.EmbeddingRequest{
		Input: []string{req.Input},
		Model: req.Model,
	}

	resp, err := s.service.Embed(ctx, svcReq)
	if err != nil {
		s.logger.Error("Embed failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(resp.Embeddings) == 0 {
		return nil, status.Error(codes.Internal, "no embeddings returned")
	}

	// Convert float64 to float32
	embedding := make([]float32, len(resp.Embeddings[0]))
	for i, v := range resp.Embeddings[0] {
		embedding[i] = float32(v)
	}

	return &pb.EmbedResponse{
		Embedding:  embedding,
		Dimensions: int32(len(embedding)),
		Model:      resp.Model,
	}, nil
}

// BatchEmbed implements TuringServiceServer.BatchEmbed
func (s *Server) BatchEmbed(ctx context.Context, req *pb.BatchEmbedRequest) (*pb.BatchEmbedResponse, error) {
	if len(req.Inputs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "inputs are required")
	}

	svcReq := &service.EmbeddingRequest{
		Input: req.Inputs,
		Model: req.Model,
	}

	resp, err := s.service.Embed(ctx, svcReq)
	if err != nil {
		s.logger.Error("BatchEmbed failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	results := make([]*pb.EmbedResult, len(resp.Embeddings))
	var totalTokens int32
	for i, emb := range resp.Embeddings {
		embedding := make([]float32, len(emb))
		for j, v := range emb {
			embedding[j] = float32(v)
		}
		results[i] = &pb.EmbedResult{
			Embedding: embedding,
			Index:     int32(i),
		}
	}

	return &pb.BatchEmbedResponse{
		Embeddings:  results,
		TotalTokens: totalTokens,
		Model:       resp.Model,
	}, nil
}

// ListModels implements TuringServiceServer.ListModels
func (s *Server) ListModels(ctx context.Context, _ *common.Empty) (*pb.ModelListResponse, error) {
	models, err := s.service.ListModels(ctx)
	if err != nil {
		s.logger.Error("ListModels failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbModels := make([]*pb.ModelInfo, len(models))
	for i, m := range models {
		pbModels[i] = &pb.ModelInfo{
			Name:      m.Name,
			Provider:  "ollama",
			Size:      m.Size,
			Available: true,
			Details: map[string]string{
				"parameter_size": m.ParameterSize,
				"family":         m.Family,
			},
		}
	}

	return &pb.ModelListResponse{
		Models: pbModels,
	}, nil
}

// GetModel implements TuringServiceServer.GetModel
func (s *Server) GetModel(ctx context.Context, req *pb.GetModelRequest) (*pb.ModelInfo, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	models, err := s.service.ListModels(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, m := range models {
		if m.Name == req.Name {
			return &pb.ModelInfo{
				Name:      m.Name,
				Provider:  "ollama",
				Size:      m.Size,
				Available: true,
				Details: map[string]string{
					"parameter_size": m.ParameterSize,
					"family":         m.Family,
				},
			}, nil
		}
	}

	return nil, status.Error(codes.NotFound, "model not found")
}

// PullModel implements TuringServiceServer.PullModel
func (s *Server) PullModel(req *pb.PullModelRequest, stream grpc.ServerStreamingServer[pb.PullProgress]) error {
	if req.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}

	s.logger.Info("Pulling model", "name", req.Name)

	ctx := stream.Context()
	progressCh, errCh := s.service.PullModel(ctx, req.Name)

	for {
		select {
		case progress, ok := <-progressCh:
			if !ok {
				// Channel closed, send final completion
				return stream.Send(&pb.PullProgress{
					Status:    "success",
					Completed: 100,
					Total:     100,
					Percent:   100.0,
				})
			}

			// Calculate percentage
			var percent float32
			if progress.Total > 0 {
				percent = float32(progress.Completed) / float32(progress.Total) * 100
			}

			if err := stream.Send(&pb.PullProgress{
				Status:    progress.Status,
				Completed: progress.Completed,
				Total:     progress.Total,
				Percent:   percent,
			}); err != nil {
				return err
			}

		case err := <-errCh:
			if err != nil {
				s.logger.Error("Pull failed", "error", err)
				return status.Error(codes.Internal, err.Error())
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// HealthCheck implements TuringServiceServer.HealthCheck
func (s *Server) HealthCheck(ctx context.Context, _ *common.HealthCheckRequest) (*common.HealthCheckResponse, error) {
	result := s.health.Check(ctx)

	details := make(map[string]string)
	for _, check := range result.Checks {
		details[check.Name] = string(check.Status)
	}

	return &common.HealthCheckResponse{
		Status:        string(result.Status),
		Service:       "turing",
		Version:       "1.0.0",
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		Details:       details,
	}, nil
}

// GetConfig implements TuringServiceServer.GetConfig
func (s *Server) GetConfig(ctx context.Context, _ *pb.GetConfigRequest) (*pb.GetConfigResponse, error) {
	return &pb.GetConfigResponse{
		DefaultModel:       s.config.DefaultModel,
		DefaultProvider:    "ollama",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2048,
		OllamaUrl:          s.config.OllamaURL,
	}, nil
}
