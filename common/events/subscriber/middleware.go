package subscriber

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/satori/go.uuid"
)

func WithRequestID(next HandlerFunc) HandlerFunc {
	return func(ctx context.Context, msg *Message) error {
		requestID, ok := msg.Attributes["request-id"]
		if !ok {
			id, err := uuid.NewV4()
			if err != nil {
				ctxzap.Extract(ctx).Error(fmt.Sprintf("couldn't generate uuid %s", err))
				requestID = "err"
			}
			requestID = id.String()
		}

		ctx = context.WithValue(ctx, "request-id", requestID)

		return next(ctx, msg)
	}
}
