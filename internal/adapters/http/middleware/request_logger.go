package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"rbac-project/internal/ports"
)

func RequestLogger(logger ports.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			started := time.Now()
			err := next(c)
			duration := time.Since(started)
			ctx := c.Request().Context()
			logger.Info(ctx, "http request",
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"route_pattern", c.Path(),
				"status", c.Response().Status,
				"duration", duration.String(),
			)
			return err
		}
	}
}
