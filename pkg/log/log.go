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

	var core zapcore.Core
	if opt.debug { // Debug 模式：控制台使用带颜色的编码器
		consoleEncoder := zapcore.NewConsoleEncoder(
			zapcore.EncoderConfig{
				TimeKey:       "T",
				LevelKey:      "L",
				NameKey:       "N",
				CallerKey:     "C",
				MessageKey:    "M",
				StacktraceKey: "S",
				LineEnding:    zapcore.DefaultLineEnding,
				EncodeLevel:   zapcore.LevelEncoder(func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) { enc.AppendString(levelColor(l)) }),
				EncodeTime:    zapcore.ISO8601TimeEncoder,
				EncodeCaller:  zapcore.ShortCallerEncoder,
			})
		core = zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel),
		)
	} else { // 非 Debug 模式：只输出到文件，使用 InfoLevel
		// 文件输出使用 JSON 格式
		fileEncoderConfig := zap.NewProductionEncoderConfig()
		fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)
		fileWriteSyncer := log.getFileLogWriter()
		core = zapcore.NewTee(
			zapcore.NewCore(fileEncoder, fileWriteSyncer, zapcore.InfoLevel),
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

// levelColor 返回带颜色的日志级别字符串
func levelColor(l zapcore.Level) string {
	switch l {
	case zapcore.DebugLevel:
		return "\033[36mDEBUG\033[0m" // 青色
	case zapcore.InfoLevel:
		return "\033[32mINFO\033[0m" // 绿色
	case zapcore.WarnLevel:
		return "\033[33mWARN\033[0m" // 黄色
	case zapcore.ErrorLevel:
		return "\033[31mERROR\033[0m" // 红色
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return "\033[35mFATAL\033[0m" // 紫色
	default:
		return l.String()
	}
}
