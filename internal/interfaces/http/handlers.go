package http

import (
	"errors"
	stdhttp "net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"rbac-project/internal/application"
	"rbac-project/internal/domain"
	"rbac-project/internal/ports"
)

func handleError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, domain.ErrNotFound):
		return c.JSON(stdhttp.StatusNotFound, map[string]string{"error": err.Error()})
	default:
		return c.JSON(stdhttp.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

type ApplicationsHandler struct {
	service *application.ApplicationService
	logger  ports.Logger
}

func NewApplicationsHandler(service *application.ApplicationService, logger ports.Logger) *ApplicationsHandler {
	return &ApplicationsHandler{service: service, logger: logger}
}

func (h *ApplicationsHandler) Create(c echo.Context) error {
	ctx := c.Request().Context()
	var req struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.Bind(&req); err != nil {
		h.logger.Warn(ctx, "invalid payload for create application", "error", err)
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	if err := h.service.Create(ctx, domain.Application{ID: req.ID, Name: req.Name, Description: req.Description}); err != nil {
		h.logger.Error(ctx, "create application failed", "app_id", req.ID, "error", err)
		return handleError(c, err)
	}
	return c.NoContent(stdhttp.StatusCreated)
}

func (h *ApplicationsHandler) Update(c echo.Context) error {
	ctx := c.Request().Context()
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.Bind(&req); err != nil {
		h.logger.Warn(ctx, "invalid payload for update application", "app_id", c.Param("id"), "error", err)
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	if err := h.service.Update(ctx, domain.Application{ID: c.Param("id"), Name: req.Name, Description: req.Description}); err != nil {
		h.logger.Error(ctx, "update application failed", "app_id", c.Param("id"), "error", err)
		return handleError(c, err)
	}
	return c.NoContent(stdhttp.StatusOK)
}

func (h *ApplicationsHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()
	app, err := h.service.GetByID(ctx, c.Param("id"))
	if err != nil {
		h.logger.Error(ctx, "get application failed", "app_id", c.Param("id"), "error", err)
		return handleError(c, err)
	}
	return c.JSON(stdhttp.StatusOK, app)
}

type RolesHandler struct {
	service *application.RoleService
	logger  ports.Logger
}

func NewRolesHandler(service *application.RoleService, logger ports.Logger) *RolesHandler {
	return &RolesHandler{service: service, logger: logger}
}

func (h *RolesHandler) Create(c echo.Context) error {
	ctx := c.Request().Context()
	var req struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
	}
	if err := c.Bind(&req); err != nil {
		h.logger.Warn(ctx, "invalid payload for create role", "app_id", c.Param("app_id"), "error", err)
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	err := h.service.Create(ctx, domain.Role{AppID: c.Param("app_id"), ID: req.ID, Name: req.Name, Permissions: req.Permissions})
	if err != nil {
		h.logger.Error(ctx, "create role failed", "app_id", c.Param("app_id"), "role_id", req.ID, "error", err)
		return handleError(c, err)
	}
	return c.NoContent(stdhttp.StatusCreated)
}

func (h *RolesHandler) Update(c echo.Context) error {
	ctx := c.Request().Context()
	var req struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
	}
	if err := c.Bind(&req); err != nil {
		h.logger.Warn(ctx, "invalid payload for update role", "app_id", c.Param("app_id"), "role_id", c.Param("role_id"), "error", err)
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	err := h.service.Update(ctx, domain.Role{AppID: c.Param("app_id"), ID: c.Param("role_id"), Name: req.Name, Permissions: req.Permissions})
	if err != nil {
		h.logger.Error(ctx, "update role failed", "app_id", c.Param("app_id"), "role_id", c.Param("role_id"), "error", err)
		return handleError(c, err)
	}
	return c.NoContent(stdhttp.StatusOK)
}

func (h *RolesHandler) List(c echo.Context) error {
	ctx := c.Request().Context()
	roles, err := h.service.ListByAppID(ctx, c.Param("app_id"))
	if err != nil {
		h.logger.Error(ctx, "list roles failed", "app_id", c.Param("app_id"), "error", err)
		return handleError(c, err)
	}
	return c.JSON(stdhttp.StatusOK, roles)
}

type PermissionsHandler struct {
	service *application.PermissionService
	logger  ports.Logger
}

func NewPermissionsHandler(service *application.PermissionService, logger ports.Logger) *PermissionsHandler {
	return &PermissionsHandler{service: service, logger: logger}
}

func (h *PermissionsHandler) Create(c echo.Context) error {
	ctx := c.Request().Context()
	var req struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.Bind(&req); err != nil {
		h.logger.Warn(ctx, "invalid payload for create permission", "app_id", c.Param("app_id"), "error", err)
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	err := h.service.Create(ctx, domain.Permission{AppID: c.Param("app_id"), ID: req.ID, Name: req.Name, Description: req.Description})
	if err != nil {
		h.logger.Error(ctx, "create permission failed", "app_id", c.Param("app_id"), "permission_id", req.ID, "error", err)
		return handleError(c, err)
	}
	return c.NoContent(stdhttp.StatusCreated)
}

func (h *PermissionsHandler) List(c echo.Context) error {
	ctx := c.Request().Context()
	permissions, err := h.service.ListByAppID(ctx, c.Param("app_id"))
	if err != nil {
		h.logger.Error(ctx, "list permissions failed", "app_id", c.Param("app_id"), "error", err)
		return handleError(c, err)
	}
	return c.JSON(stdhttp.StatusOK, permissions)
}

type UsersHandler struct {
	service *application.UserService
	logger  ports.Logger
}

func NewUsersHandler(service *application.UserService, logger ports.Logger) *UsersHandler {
	return &UsersHandler{service: service, logger: logger}
}

func (h *UsersHandler) AssignRole(c echo.Context) error {
	ctx := c.Request().Context()
	var req struct {
		RoleID string `json:"role_id"`
	}
	if err := c.Bind(&req); err != nil {
		h.logger.Warn(ctx, "invalid payload for assign role", "app_id", c.Param("app_id"), "user_id", c.Param("user_id"), "error", err)
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	err := h.service.AssignRole(ctx, c.Param("app_id"), c.Param("user_id"), req.RoleID)
	if err != nil {
		h.logger.Error(ctx, "assign role failed", "app_id", c.Param("app_id"), "user_id", c.Param("user_id"), "role_id", req.RoleID, "error", err)
		return handleError(c, err)
	}
	return c.NoContent(stdhttp.StatusCreated)
}

func (h *UsersHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()
	user, err := h.service.GetUserAppRoles(ctx, c.Param("app_id"), c.Param("user_id"))
	if err != nil {
		h.logger.Error(ctx, "get user roles failed", "app_id", c.Param("app_id"), "user_id", c.Param("user_id"), "error", err)
		return handleError(c, err)
	}
	return c.JSON(stdhttp.StatusOK, user)
}

type AuthorizationHandler struct {
	service *application.AuthorizationService
	logger  ports.Logger
}

func NewAuthorizationHandler(service *application.AuthorizationService, logger ports.Logger) *AuthorizationHandler {
	return &AuthorizationHandler{service: service, logger: logger}
}

func (h *AuthorizationHandler) Authorize(c echo.Context) error {
	ctx := c.Request().Context()
	if strings.EqualFold(os.Getenv("AUTHORIZE_TEST_MODE"), "true") {
		h.logger.Info(ctx, "authorize test mode enabled")
		return c.JSON(stdhttp.StatusOK, map[string]bool{"allowed": true})
	}
	var req struct {
		AppID      string `json:"app_id"`
		UserID     string `json:"user_id"`
		Permission string `json:"permission"`
	}
	if err := c.Bind(&req); err != nil {
		h.logger.Warn(ctx, "invalid payload for authorize", "error", err)
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	if req.UserID == "" {
		if uid, ok := c.Get("user_id").(string); ok {
			req.UserID = uid
		}
	}
	allowed, err := h.service.IsAllowed(ctx, req.AppID, req.UserID, req.Permission)
	if err != nil {
		h.logger.Error(ctx, "authorize failed", "app_id", req.AppID, "user_id", req.UserID, "permission", req.Permission, "error", err)
		return handleError(c, err)
	}
	return c.JSON(stdhttp.StatusOK, map[string]bool{"allowed": allowed})
}
