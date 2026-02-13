package logger

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-xray-sdk-go/xray"
	"log/slog"
)

func TestSlogLogger_LogsJSONWithoutTraceContext(t *testing.T) {
	buf := new(bytes.Buffer)
	l := NewWithWriter(buf, slog.LevelDebug)

	l.Info(context.Background(), "test message", "k", "v")

	output := buf.String()
	if !strings.Contains(output, "\"msg\":\"test message\"") {
		t.Fatalf("expected message in output: %s", output)
	}
	if strings.Contains(output, "trace_id") {
		t.Fatalf("did not expect trace_id without segment: %s", output)
	}
}

func TestSlogLogger_LogsWithTraceIDWhenSegmentExists(t *testing.T) {
	buf := new(bytes.Buffer)
	l := NewWithWriter(buf, slog.LevelDebug)

	ctx, seg := xray.BeginSegment(context.Background(), "test-segment")
	defer seg.Close(nil)

	l.Info(ctx, "trace message")

	output := buf.String()
	if !strings.Contains(output, "trace_id") {
		t.Fatalf("expected trace_id in output: %s", output)
	}
}
