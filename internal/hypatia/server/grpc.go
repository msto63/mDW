package server

import (
	"context"
	"io"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	pb "github.com/msto63/mDW/api/gen/hypatia"
	"github.com/msto63/mDW/internal/hypatia/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Ensure Server implements HypatiaServiceServer
var _ pb.HypatiaServiceServer = (*Server)(nil)

// Search implements HypatiaServiceServer.Search
func (s *Server) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}

	topK := int(req.TopK)
	if topK == 0 {
		topK = s.config.DefaultTopK
	}
	minScore := float64(req.MinScore)
	if minScore == 0 {
		minScore = s.config.MinRelevance
	}

	svcReq := &service.SearchRequest{
		Query:      req.Query,
		Collection: req.Collection,
		TopK:       topK,
		MinScore:   minScore,
	}

	results, err := s.service.Search(ctx, svcReq)
	if err != nil {
		s.logger.Error("Search failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbResults := make([]*pb.SearchResult, len(results))
	for i, r := range results {
		pbResults[i] = &pb.SearchResult{
			DocumentId: r.ID,
			Content:    r.Content,
			Score:      float32(r.Score),
		}
	}

	return &pb.SearchResponse{
		Results: pbResults,
	}, nil
}

// HybridSearch implements HypatiaServiceServer.HybridSearch
func (s *Server) HybridSearch(ctx context.Context, req *pb.HybridSearchRequest) (*pb.SearchResponse, error) {
	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}

	topK := int(req.TopK)
	if topK == 0 {
		topK = s.config.DefaultTopK
	}

	vectorWeight := float64(req.VectorWeight)
	keywordWeight := float64(req.KeywordWeight)
	if vectorWeight == 0 && keywordWeight == 0 {
		vectorWeight = 0.7
		keywordWeight = 0.3
	}

	svcReq := &service.SearchRequest{
		Query:      req.Query,
		Collection: req.Collection,
		TopK:       topK,
		MinScore:   float64(req.MinScore),
	}

	results, err := s.service.HybridSearch(ctx, svcReq, vectorWeight, keywordWeight)
	if err != nil {
		s.logger.Error("HybridSearch failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbResults := make([]*pb.SearchResult, len(results))
	for i, r := range results {
		pbResults[i] = &pb.SearchResult{
			DocumentId: r.ID,
			Content:    r.Content,
			Score:      float32(r.Score),
		}
	}

	return &pb.SearchResponse{
		Results: pbResults,
	}, nil
}

// IngestDocument implements HypatiaServiceServer.IngestDocument
func (s *Server) IngestDocument(ctx context.Context, req *pb.IngestDocumentRequest) (*pb.IngestResponse, error) {
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	// Use Title as ID if provided, otherwise generate one
	docID := req.Title
	if docID == "" {
		docID = req.Source
	}
	if docID == "" {
		docID = time.Now().Format("20060102150405")
	}

	// Build metadata with title and source
	metadata := make(map[string]string)
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			metadata[k] = v
		}
	}
	if req.Title != "" {
		metadata["title"] = req.Title
	}
	if req.Source != "" {
		metadata["source"] = req.Source
	}

	indexReq := &service.IndexRequest{
		ID:         docID,
		Content:    req.Content,
		Collection: req.Collection,
		Metadata:   metadata,
	}

	if err := s.service.Index(ctx, indexReq); err != nil {
		s.logger.Error("IngestDocument failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.IngestResponse{
		DocumentId: docID,
		Success:    true,
	}, nil
}

// IngestFile implements HypatiaServiceServer.IngestFile
func (s *Server) IngestFile(stream grpc.ClientStreamingServer[pb.FileChunk, pb.IngestResponse]) error {
	var content []byte
	var isFirst = true
	var docID string

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}

		if isFirst {
			docID = time.Now().Format("20060102150405")
			isFirst = false
		}
		content = append(content, chunk.Data...)
	}

	if len(content) == 0 {
		return status.Error(codes.InvalidArgument, "content is required")
	}

	indexReq := &service.IndexRequest{
		ID:      docID,
		Content: string(content),
	}

	if err := s.service.Index(stream.Context(), indexReq); err != nil {
		s.logger.Error("IngestFile failed", "error", err)
		return status.Error(codes.Internal, err.Error())
	}

	return stream.SendAndClose(&pb.IngestResponse{
		DocumentId: docID,
		Success:    true,
	})
}

// DeleteDocument implements HypatiaServiceServer.DeleteDocument
func (s *Server) DeleteDocument(ctx context.Context, req *pb.DeleteDocumentRequest) (*common.Empty, error) {
	if req.DocumentId == "" {
		return nil, status.Error(codes.InvalidArgument, "document_id is required")
	}

	if err := s.service.Delete(ctx, req.DocumentId); err != nil {
		s.logger.Error("DeleteDocument failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &common.Empty{}, nil
}

// GetDocument implements HypatiaServiceServer.GetDocument
func (s *Server) GetDocument(ctx context.Context, req *pb.GetDocumentRequest) (*pb.DocumentInfo, error) {
	if req.DocumentId == "" {
		return nil, status.Error(codes.InvalidArgument, "document_id is required")
	}

	doc, err := s.service.GetDocument(ctx, req.DocumentId)
	if err != nil {
		s.logger.Error("GetDocument failed", "error", err)
		return nil, status.Error(codes.NotFound, err.Error())
	}

	title := ""
	source := ""
	var chunkCount int32
	if doc.Metadata != nil {
		title = doc.Metadata["title"]
		source = doc.Metadata["source"]
		if cc := doc.Metadata["_chunk_count"]; cc != "" {
			// Parse chunk count from metadata
			var n int
			for _, c := range cc {
				if c >= '0' && c <= '9' {
					n = n*10 + int(c-'0')
				}
			}
			chunkCount = int32(n)
		}
	}

	return &pb.DocumentInfo{
		Id:         doc.ID,
		Title:      title,
		Source:     source,
		Collection: doc.Collection,
		ChunkCount: chunkCount,
		CreatedAt:  time.Now().Unix(),
		Metadata: &pb.DocumentMetadata{
			Title:  title,
			Source: source,
			Custom: doc.Metadata,
		},
	}, nil
}

// ListDocuments implements HypatiaServiceServer.ListDocuments
func (s *Server) ListDocuments(ctx context.Context, req *pb.ListDocumentsRequest) (*pb.DocumentListResponse, error) {
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}

	docs, total, err := s.service.ListDocuments(ctx, req.Collection, page, pageSize)
	if err != nil {
		s.logger.Error("ListDocuments failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbDocs := make([]*pb.DocumentInfo, len(docs))
	for i, doc := range docs {
		pbDocs[i] = &pb.DocumentInfo{
			Id:         doc.ID,
			Title:      doc.Title,
			Source:     doc.Source,
			Collection: doc.Collection,
			ChunkCount: int32(doc.ChunkCount),
			CreatedAt:  doc.CreatedAt.Unix(),
		}
	}

	totalPages := int32(total) / int32(pageSize)
	if int32(total)%int32(pageSize) > 0 {
		totalPages++
	}

	return &pb.DocumentListResponse{
		Documents: pbDocs,
		Pagination: &common.Pagination{
			Page:       int32(page),
			PageSize:   int32(pageSize),
			Total:      int32(total),
			TotalPages: totalPages,
		},
	}, nil
}

// CreateCollection implements HypatiaServiceServer.CreateCollection
func (s *Server) CreateCollection(ctx context.Context, req *pb.CreateCollectionRequest) (*pb.CollectionInfo, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if err := s.service.CreateCollection(ctx, req.Name); err != nil {
		s.logger.Error("CreateCollection failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.CollectionInfo{
		Name:          req.Name,
		DocumentCount: 0,
		CreatedAt:     time.Now().Unix(),
	}, nil
}

// DeleteCollection implements HypatiaServiceServer.DeleteCollection
func (s *Server) DeleteCollection(ctx context.Context, req *pb.DeleteCollectionRequest) (*common.Empty, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if err := s.service.DeleteCollection(ctx, req.Name); err != nil {
		s.logger.Error("DeleteCollection failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &common.Empty{}, nil
}

// ListCollections implements HypatiaServiceServer.ListCollections
func (s *Server) ListCollections(ctx context.Context, _ *common.Empty) (*pb.CollectionListResponse, error) {
	collections, err := s.service.ListCollections(ctx)
	if err != nil {
		s.logger.Error("ListCollections failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbCollections := make([]*pb.CollectionInfo, len(collections))
	for i, c := range collections {
		pbCollections[i] = &pb.CollectionInfo{
			Name:          c.Name,
			DocumentCount: int32(c.DocumentCount),
		}
	}

	return &pb.CollectionListResponse{
		Collections: pbCollections,
	}, nil
}

// GetCollectionStats implements HypatiaServiceServer.GetCollectionStats
func (s *Server) GetCollectionStats(ctx context.Context, req *pb.GetCollectionStatsRequest) (*pb.CollectionStats, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	stats, err := s.service.GetCollectionStats(ctx, req.Name)
	if err != nil {
		s.logger.Error("GetCollectionStats failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.CollectionStats{
		Name:          stats.Name,
		DocumentCount: int32(stats.DocumentCount),
		ChunkCount:    int32(stats.ChunkCount),
		TotalTokens:   stats.TotalTokens,
		StorageBytes:  stats.StorageBytes,
	}, nil
}

// AugmentPrompt implements HypatiaServiceServer.AugmentPrompt
func (s *Server) AugmentPrompt(ctx context.Context, req *pb.AugmentPromptRequest) (*pb.AugmentPromptResponse, error) {
	if req.Prompt == "" {
		return nil, status.Error(codes.InvalidArgument, "prompt is required")
	}

	topK := int(req.TopK)
	if topK == 0 {
		topK = s.config.DefaultTopK
	}

	svcReq := &service.SearchRequest{
		Query:      req.Prompt,
		Collection: req.Collection,
		TopK:       topK,
		MinScore:   s.config.MinRelevance,
	}

	results, err := s.service.Search(ctx, svcReq)
	if err != nil {
		s.logger.Error("AugmentPrompt search failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	var contextText string
	sources := make([]*pb.SearchResult, len(results))
	for i, r := range results {
		contextText += "\n---\n" + r.Content + "\n"
		sources[i] = &pb.SearchResult{
			DocumentId: r.ID,
			Content:    r.Content,
			Score:      float32(r.Score),
		}
	}

	augmentedPrompt := req.Prompt
	if len(results) > 0 {
		augmentedPrompt = "Context:\n" + contextText + "\n---\n\nQuestion: " + req.Prompt
	}

	return &pb.AugmentPromptResponse{
		AugmentedPrompt: augmentedPrompt,
		Sources:         sources,
	}, nil
}

// HealthCheck implements HypatiaServiceServer.HealthCheck
func (s *Server) HealthCheck(ctx context.Context, _ *common.HealthCheckRequest) (*common.HealthCheckResponse, error) {
	result := s.health.Check(ctx)

	details := make(map[string]string)
	for _, check := range result.Checks {
		details[check.Name] = string(check.Status)
	}

	return &common.HealthCheckResponse{
		Status:        string(result.Status),
		Service:       "hypatia",
		Version:       "1.0.0",
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		Details:       details,
	}, nil
}
