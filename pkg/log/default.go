package log

import (
	"context"
	"sync"
)

var (
	defaultLogger     *Logger
	defaultLoggerOnce sync.Once
)

// InitLogger 初始化默认 logger，应该在配置加载后调用
func InitLogger(opts ...Option) {
	opts = append(opts, WithCallerSkip(3))
	defaultLoggerOnce.Do(func() {
		defaultLogger = NewLogger(opts...)
	})
}

// getDefaultLogger 获取默认 logger，如果未初始化则使用默认配置
func getDefaultLogger() *Logger {
	if defaultLogger == nil {
		defaultLoggerOnce.Do(func() {
			defaultLogger = NewLogger()
		})
	}
	return defaultLogger
}

func Debug(ctx context.Context, msg string, fields ...Field) {
	getDefaultLogger().Debug(ctx, msg, fields...)
}

func Info(ctx context.Context, msg string, fields ...Field) {
	getDefaultLogger().Info(ctx, msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...Field) {
	getDefaultLogger().Warn(ctx, msg, fields...)
}

func Error(ctx context.Context, msg string, fields ...Field) {
	getDefaultLogger().Error(ctx, msg, fields...)
}
