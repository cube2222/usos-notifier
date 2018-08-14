package publisher

import (
	"context"

	"github.com/cube2222/grpc-utils/logger"
	"github.com/cube2222/grpc-utils/requestid"
)

func WithRequestID(next PublishEventFunc) PublishEventFunc {
	return func(ctx context.Context, eventType string, metadata map[string]string, message string) error {
		if requestID, ok := ctx.Value(requestid.Key).(string); ok {
			metadata[requestid.Key] = requestID
		} else {
			logger.FromContext(ctx).Errorf("No request id when publishing event of type %v", eventType)
		}

		return next(ctx, eventType, metadata, message)
	}
}
