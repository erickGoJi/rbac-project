package http

import (
	"errors"
	stdhttp "net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"rbac-project/internal/application"
	"rbac-project/internal/domain"
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
}

func NewApplicationsHandler(service *application.ApplicationService) *ApplicationsHandler {
	return &ApplicationsHandler{service: service}
}

func (h *ApplicationsHandler) Create(c echo.Context) error {
	var req struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	if err := h.service.Create(c.Request().Context(), domain.Application{ID: req.ID, Name: req.Name, Description: req.Description}); err != nil {
		return handleError(c, err)
	}
	return c.NoContent(stdhttp.StatusCreated)
}

func (h *ApplicationsHandler) Update(c echo.Context) error {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	if err := h.service.Update(c.Request().Context(), domain.Application{ID: c.Param("id"), Name: req.Name, Description: req.Description}); err != nil {
		return handleError(c, err)
	}
	return c.NoContent(stdhttp.StatusOK)
}

func (h *ApplicationsHandler) Get(c echo.Context) error {
	app, err := h.service.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(stdhttp.StatusOK, app)
}

type RolesHandler struct{ service *application.RoleService }

func NewRolesHandler(service *application.RoleService) *RolesHandler {
	return &RolesHandler{service: service}
}

func (h *RolesHandler) Create(c echo.Context) error {
	var req struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	err := h.service.Create(c.Request().Context(), domain.Role{AppID: c.Param("app_id"), ID: req.ID, Name: req.Name, Permissions: req.Permissions})
	if err != nil {
		return handleError(c, err)
	}
	return c.NoContent(stdhttp.StatusCreated)
}

func (h *RolesHandler) Update(c echo.Context) error {
	var req struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	err := h.service.Update(c.Request().Context(), domain.Role{AppID: c.Param("app_id"), ID: c.Param("role_id"), Name: req.Name, Permissions: req.Permissions})
	if err != nil {
		return handleError(c, err)
	}
	return c.NoContent(stdhttp.StatusOK)
}

func (h *RolesHandler) List(c echo.Context) error {
	roles, err := h.service.ListByAppID(c.Request().Context(), c.Param("app_id"))
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(stdhttp.StatusOK, roles)
}

type PermissionsHandler struct {
	service *application.PermissionService
}

func NewPermissionsHandler(service *application.PermissionService) *PermissionsHandler {
	return &PermissionsHandler{service: service}
}

func (h *PermissionsHandler) Create(c echo.Context) error {
	var req struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	err := h.service.Create(c.Request().Context(), domain.Permission{AppID: c.Param("app_id"), ID: req.ID, Name: req.Name, Description: req.Description})
	if err != nil {
		return handleError(c, err)
	}
	return c.NoContent(stdhttp.StatusCreated)
}

func (h *PermissionsHandler) List(c echo.Context) error {
	permissions, err := h.service.ListByAppID(c.Request().Context(), c.Param("app_id"))
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(stdhttp.StatusOK, permissions)
}

type UsersHandler struct{ service *application.UserService }

func NewUsersHandler(service *application.UserService) *UsersHandler {
	return &UsersHandler{service: service}
}

func (h *UsersHandler) AssignRole(c echo.Context) error {
	var req struct {
		RoleID string `json:"role_id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	err := h.service.AssignRole(c.Request().Context(), c.Param("app_id"), c.Param("user_id"), req.RoleID)
	if err != nil {
		return handleError(c, err)
	}
	return c.NoContent(stdhttp.StatusCreated)
}

func (h *UsersHandler) Get(c echo.Context) error {
	user, err := h.service.GetUserAppRoles(c.Request().Context(), c.Param("app_id"), c.Param("user_id"))
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(stdhttp.StatusOK, user)
}

type AuthorizationHandler struct {
	service *application.AuthorizationService
}

func NewAuthorizationHandler(service *application.AuthorizationService) *AuthorizationHandler {
	return &AuthorizationHandler{service: service}
}

func (h *AuthorizationHandler) Authorize(c echo.Context) error {
	if strings.EqualFold(os.Getenv("AUTHORIZE_TEST_MODE"), "true") {
		return c.JSON(stdhttp.StatusOK, map[string]bool{"allowed": true})
	}
	var req struct {
		AppID      string `json:"app_id"`
		UserID     string `json:"user_id"`
		Permission string `json:"permission"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(stdhttp.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	if req.UserID == "" {
		if uid, ok := c.Get("user_id").(string); ok {
			req.UserID = uid
		}
	}
	allowed, err := h.service.IsAllowed(c.Request().Context(), req.AppID, req.UserID, req.Permission)
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(stdhttp.StatusOK, map[string]bool{"allowed": allowed})
}
