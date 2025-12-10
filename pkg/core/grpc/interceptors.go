package grpc

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var interceptorLogger = logging.New("grpc")

// Context keys for request metadata
type contextKey string

const (
	RequestIDKey     contextKey = "request_id"
	RequestIDHeader  string     = "x-request-id"
)

// RecoveryInterceptor recovers from panics in gRPC handlers
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				interceptorLogger.Error("gRPC panic recovered", "panic", r, "stack", string(stack))
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// StreamRecoveryInterceptor recovers from panics in streaming gRPC handlers
func StreamRecoveryInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				interceptorLogger.Error("gRPC stream panic recovered", "panic", r, "stack", string(stack))
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(srv, ss)
	}
}

// LoggingInterceptor logs gRPC requests
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		requestID := GetRequestID(ctx)

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		interceptorLogger.Info("gRPC request",
			"request_id", requestID,
			"method", info.FullMethod,
			"status", statusCode.String(),
			"duration", duration,
		)

		return resp, err
	}
}

// StreamLoggingInterceptor logs gRPC streaming requests
func StreamLoggingInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := ss.Context()
		requestID := GetRequestID(ctx)

		err := handler(srv, ss)

		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		interceptorLogger.Info("gRPC stream request",
			"request_id", requestID,
			"method", info.FullMethod,
			"status", statusCode.String(),
			"duration", duration,
		)

		return err
	}
}

// RequestIDInterceptor adds a request ID to the context
func RequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestID := extractRequestID(ctx)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx = context.WithValue(ctx, RequestIDKey, requestID)

		// Add to outgoing metadata
		ctx = metadata.AppendToOutgoingContext(ctx, RequestIDHeader, requestID)

		return handler(ctx, req)
	}
}

// ClientRequestIDInterceptor propagates request ID to outgoing requests
func ClientRequestIDInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		requestID := GetRequestID(ctx)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx = metadata.AppendToOutgoingContext(ctx, RequestIDHeader, requestID)

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// ClientLoggingInterceptor logs outgoing gRPC requests
func ClientLoggingInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		interceptorLogger.Debug("gRPC client request",
			"method", method,
			"status", statusCode.String(),
			"duration", duration,
		)

		return err
	}
}

// ClientStreamLoggingInterceptor logs outgoing streaming gRPC requests
func ClientStreamLoggingInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		start := time.Now()

		stream, err := streamer(ctx, desc, cc, method, opts...)

		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		interceptorLogger.Debug("gRPC client stream request",
			"method", method,
			"status", statusCode.String(),
			"duration", duration,
		)

		return stream, err
	}
}

// GetRequestID extracts the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return extractRequestID(ctx)
}

// extractRequestID extracts request ID from incoming metadata
func extractRequestID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := md.Get(RequestIDHeader)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}
