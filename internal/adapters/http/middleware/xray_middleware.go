package middleware

import (
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/labstack/echo/v4"
)

func XRayMiddleware(segmentName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, seg := xray.BeginSegment(c.Request().Context(), segmentName)
			defer seg.Close(nil)
			req := c.Request().Clone(ctx)
			c.SetRequest(req)
			return next(c)
		}
	}
}
