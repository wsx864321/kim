package log

import (
	"context"
)

var defaultLogger = NewLogger()

func Debug(ctx context.Context, msg string, fields ...Field) {
	defaultLogger.Debug(ctx, msg, fields...)
}

func Info(ctx context.Context, msg string, fields ...Field) {
	defaultLogger.Info(ctx, msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...Field) {
	defaultLogger.Warn(ctx, msg, fields...)
}

func Error(ctx context.Context, msg string, fields ...Field) {
	defaultLogger.Error(ctx, msg, fields...)
}
