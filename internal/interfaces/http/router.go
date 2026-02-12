package http

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Middleware struct {
	Auth echo.MiddlewareFunc
}

func newEcho(m Middleware) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	if m.Auth != nil {
		e.Use(m.Auth)
	}
	return e
}

func NewApplicationsRouter(h *ApplicationsHandler, m Middleware) *echo.Echo {
	e := newEcho(m)
	e.POST("/applications", h.Create)
	e.PUT("/applications/:id", h.Update)
	e.GET("/applications/:id", h.Get)
	return e
}

func NewRolesRouter(h *RolesHandler, m Middleware) *echo.Echo {
	e := newEcho(m)
	e.POST("/applications/:app_id/roles", h.Create)
	e.PUT("/applications/:app_id/roles/:role_id", h.Update)
	e.GET("/applications/:app_id/roles", h.List)
	return e
}

func NewPermissionsRouter(h *PermissionsHandler, m Middleware) *echo.Echo {
	e := newEcho(m)
	e.POST("/applications/:app_id/permissions", h.Create)
	e.GET("/applications/:app_id/permissions", h.List)
	return e
}

func NewUsersRouter(h *UsersHandler, m Middleware) *echo.Echo {
	e := newEcho(m)
	e.POST("/applications/:app_id/users/:user_id/roles", h.AssignRole)
	e.GET("/applications/:app_id/users/:user_id", h.Get)
	return e
}

func NewAuthorizationRouter(h *AuthorizationHandler, m Middleware) *echo.Echo {
	e := newEcho(m)
	e.POST("/authorize", h.Authorize)
	return e
}
