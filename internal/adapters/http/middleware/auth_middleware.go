package middleware

import (
	"errors"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

type Mode string

const (
	ModeNone    Mode = "none"
	ModeAPIKey  Mode = "api_key"
	ModeCognito Mode = "cognito"
)

func ParseAuthMode() (Mode, error) {
	mode := Mode(os.Getenv("AUTH_MODE"))
	switch mode {
	case "", ModeNone, ModeAPIKey, ModeCognito:
		if mode == "" {
			return ModeNone, nil
		}
		return mode, nil
	default:
		return "", errors.New("invalid auth mode")
	}
}

func AuthMiddleware(cognito echo.MiddlewareFunc) (echo.MiddlewareFunc, error) {
	mode, err := ParseAuthMode()
	if err != nil {
		return nil, err
	}
	if mode == ModeCognito && cognito == nil {
		return nil, errors.New("cognito middleware is required when AUTH_MODE=cognito")
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			switch mode {
			case ModeNone:
				return next(c)
			case ModeAPIKey:
				return next(c)
			case ModeCognito:
				return cognito(next)(c)
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "invalid auth mode")
			}
		}
	}, nil
}
