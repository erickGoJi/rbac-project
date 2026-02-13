package logger

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/aws/aws-xray-sdk-go/xray"
)

type SlogLogger struct {
	logger *slog.Logger
}

func New() *SlogLogger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	return &SlogLogger{logger: slog.New(h)}
}

func NewWithWriter(w io.Writer, level slog.Leveler) *SlogLogger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return &SlogLogger{logger: slog.New(h)}
}

func (l *SlogLogger) enrichWithTraceID(ctx context.Context, args []any) []any {
	if ctx == nil {
		return args
	}
	seg := xray.GetSegment(ctx)
	if seg == nil || seg.TraceID == "" {
		return args
	}
	return append(args, "trace_id", seg.TraceID)
}

func (l *SlogLogger) Info(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, l.enrichWithTraceID(ctx, args)...)
}

func (l *SlogLogger) Error(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, l.enrichWithTraceID(ctx, args)...)
}

func (l *SlogLogger) Warn(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, msg, l.enrichWithTraceID(ctx, args)...)
}

func (l *SlogLogger) Debug(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, l.enrichWithTraceID(ctx, args)...)
}
