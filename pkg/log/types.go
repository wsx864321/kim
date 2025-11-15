package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DebugLevel = zapcore.DebugLevel
	InfoLevel  = zapcore.InfoLevel
	WarnLevel  = zapcore.WarnLevel
	ErrorLevel = zapcore.ErrorLevel
)

var (
	Bool     = zap.Bool
	Int      = zap.Int
	String   = zap.String
	Strings  = zap.Strings
	Any      = zap.Any
	Duration = zap.Duration
	Float64  = zap.Float64
	Int64    = zap.Int64
	Uint64   = zap.Uint64
	Binary   = zap.Binary
	Stack    = zap.Stack
	Object   = zap.Object
	Array    = zap.Array
)

type (
	Level = zapcore.Level
	Field = zap.Field
)
