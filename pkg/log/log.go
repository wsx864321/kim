package log

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	Options

	logger *zap.Logger
}

func NewLogger(opts ...Option) *Logger {
	opt := defaultOptions
	for _, o := range opts {
		o.apply(&opt)
	}
	log := &Logger{
		Options: opt,
	}

	fileWriteSyncer := log.getFileLogWriter()
	var core zapcore.Core
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	if opt.debug {
		core = zapcore.NewTee(
			zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel),
			zapcore.NewCore(encoder, fileWriteSyncer, zapcore.DebugLevel),
		)
	} else {
		core = zapcore.NewTee(
			zapcore.NewCore(encoder, fileWriteSyncer, zapcore.InfoLevel),
		)
	}
	log.logger = zap.New(core, zap.WithCaller(true), zap.AddCallerSkip(log.callerSkip))

	return log
}

func (l *Logger) getFileLogWriter() (writeSyncer zapcore.WriteSyncer) {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   fmt.Sprintf("%s/%s", l.logDir, l.filename),
		MaxSize:    l.maxSize,
		MaxBackups: l.maxBackups,
		MaxAge:     l.maxAge,
		Compress:   l.compress,
	}

	return zapcore.AddSync(lumberJackLogger)
}

// Debug debug日志
func (l *Logger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.Log(ctx, DebugLevel, msg, fields...)
}

// Info info日志
func (l *Logger) Info(ctx context.Context, msg string, fields ...Field) {
	l.Log(ctx, InfoLevel, msg, fields...)
}

// Warn warn日志
func (l *Logger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.Log(ctx, WarnLevel, msg, fields...)
}

// Error error日志
func (l *Logger) Error(ctx context.Context, msg string, fields ...Field) {
	l.Log(ctx, ErrorLevel, msg, fields...)
}

// Log 记录日志，自动从ctx中获取traceID
func (l *Logger) Log(ctx context.Context, level Level, msg string, fields ...Field) {
	traceID := TraceIDFromContext(ctx)
	if traceID != "" {
		fields = append([]Field{String(keyTraceID, traceID)}, fields...)
	}
	l.logger.Log(level, msg, fields...)
}
