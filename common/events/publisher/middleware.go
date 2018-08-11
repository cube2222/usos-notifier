package publisher

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

func WithRequestID(next PublishEventFunc) PublishEventFunc {
	return func(ctx context.Context, eventType string, metadata map[string]string, message string) error {
		var requestID string

		value := ctx.Value("request-id")
		if value, ok := value.(string); ok {
			requestID = value
		} else {
			ctxzap.Extract(ctx).Error("No request-id when publishing event", zap.String("eventType", eventType))
			requestID = "no request-id"
		}

		metadata["request-id"] = requestID

		return next(ctx, eventType, metadata, message)
	}
}
