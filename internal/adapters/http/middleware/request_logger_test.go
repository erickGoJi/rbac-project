package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/labstack/echo/v4"
)

type mockLogger struct {
	lastCtx  context.Context
	lastMsg  string
	lastArgs []any
}

func (m *mockLogger) Info(ctx context.Context, msg string, args ...any) {
	m.lastCtx = ctx
	m.lastMsg = msg
	m.lastArgs = args
}

func (m *mockLogger) Error(context.Context, string, ...any) {}
func (m *mockLogger) Warn(context.Context, string, ...any)  {}
func (m *mockLogger) Debug(context.Context, string, ...any) {}

func TestRequestLogger_LogsExpectedFields(t *testing.T) {
	logger := &mockLogger{}
	mw := RequestLogger(logger)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/applications", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/applications")

	h := mw(func(c echo.Context) error {
		return c.NoContent(http.StatusCreated)
	})

	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if logger.lastMsg != "http request" {
		t.Fatalf("unexpected log message: %s", logger.lastMsg)
	}

	keys := map[string]bool{}
	for i := 0; i < len(logger.lastArgs)-1; i += 2 {
		k, ok := logger.lastArgs[i].(string)
		if ok {
			keys[k] = true
		}
	}

	for _, expected := range []string{"method", "path", "status", "duration"} {
		if !keys[expected] {
			t.Fatalf("missing expected key %s in args: %v", expected, logger.lastArgs)
		}
	}
}

func TestRequestLogger_PassesContextWithXRaySegment(t *testing.T) {
	logger := &mockLogger{}
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/applications", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	ctx, seg := xray.BeginSegment(req.Context(), "http-test")
	defer seg.Close(nil)
	c.SetRequest(req.Clone(ctx))

	h := RequestLogger(logger)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if xray.GetSegment(logger.lastCtx) == nil {
		t.Fatalf("expected xray segment in logged context")
	}
}
