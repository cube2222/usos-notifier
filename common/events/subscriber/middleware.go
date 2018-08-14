package subscriber

import (
	"context"
	"time"

	"github.com/cube2222/grpc-utils/logger"
	"github.com/cube2222/grpc-utils/requestid"
)

func WithLogger(log logger.Logger) func(next HandlerFunc) HandlerFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *Message) error {
			return next(logger.Inject(ctx, log), msg)
		}
	}
}

func WithLogging(keys ...string) func(next HandlerFunc) HandlerFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *Message) error {
			curRequestLogger := logger.FromContext(ctx)

			start := time.Now()
			err := next(ctx, msg)
			duration := time.Since(start)

			if err != nil {
				curRequestLogger = curRequestLogger.With(
					logger.NewField("err", err.Error()),
				)
			}

			curRequestLogger.With(
				logger.NewField("duration", duration),
			).Printf("Finished handling event.")

			return err
		}
	}
}

func WithRequestID(next HandlerFunc) HandlerFunc {
	return func(ctx context.Context, msg *Message) error {
		requestID, ok := msg.Attributes[requestid.Key]
		if !ok {
			requestID = requestid.GenerateRequestID()
		}
		ctx = context.WithValue(ctx, requestid.Key, requestID)

		return next(ctx, msg)
	}
}
