package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_None(t *testing.T) {
	t.Setenv("AUTH_MODE", "none")

	mw, err := AuthMiddleware(nil)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	h := mw(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})

	err = h(c)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestAuthMiddleware_APIKey(t *testing.T) {
	t.Setenv("AUTH_MODE", "api_key")

	mw, err := AuthMiddleware(nil)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	h := mw(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})

	err = h(c)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestAuthMiddleware_Cognito(t *testing.T) {
	t.Setenv("AUTH_MODE", "cognito")

	cognitoCalled := false
	mockCognito := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cognitoCalled = true
			return next(c)
		}
	}

	mw, err := AuthMiddleware(mockCognito)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := mw(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	err = h(c)
	require.NoError(t, err)
	assert.True(t, cognitoCalled)
}

func TestAuthMiddleware_Invalid(t *testing.T) {
	t.Setenv("AUTH_MODE", "invalid")

	mw, err := AuthMiddleware(nil)
	assert.Nil(t, mw)
	assert.Error(t, err)
}

func TestParseAuthMode_DefaultsToNone(t *testing.T) {
	_ = os.Unsetenv("AUTH_MODE")
	mode, err := ParseAuthMode()
	require.NoError(t, err)
	assert.Equal(t, ModeNone, mode)
}
