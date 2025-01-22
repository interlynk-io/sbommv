// Copyright 2025 Interlynk.io
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger

type contextKey struct{}

// InitLogger initializes the logger with a specified log level and format (JSON or console).
func InitLogger(debug bool, jsonFormat bool) {
	if logger != nil {
		panic("logger already initialized")
	}

	var config zap.Config
	if debug {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	// Configure JSON or console output
	if jsonFormat {
		config.Encoding = "json"
	} else {
		config.Encoding = "console"
	}

	// Customize log output
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	l, err := config.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	logger = l.Sugar()
}

// WithLogger attaches the logger to the context.
func WithLogger(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext retrieves the logger from the context.
func FromContext(ctx context.Context) *zap.SugaredLogger {
	if l, ok := ctx.Value(contextKey{}).(*zap.SugaredLogger); ok {
		return l
	}
	return zap.NewNop().Sugar()
}

// Sync flushes any buffered log entries.
func Sync() {
	if logger != nil {
		_ = logger.Sync()
	}
}

// Example helper function for structured logging
func LogError(ctx context.Context, err error, msg string) {
	FromContext(ctx).Errorw(msg, "error", err)
}

// LogDebug logs debug messages if debug mode is enabled.
func LogDebug(ctx context.Context, msg string, keysAndValues ...interface{}) {
	FromContext(ctx).Debugw(msg, keysAndValues...)
}

// LogInfo logs informational messages.
func LogInfo(ctx context.Context, msg string, keysAndValues ...interface{}) {
	FromContext(ctx).Infow(msg, keysAndValues...)
}
