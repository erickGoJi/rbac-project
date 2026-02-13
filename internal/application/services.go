package application

import (
	"context"
	"errors"
	"rbac-project/internal/domain"
	"rbac-project/internal/ports"
	"slices"
	"time"
)

type noopLogger struct{}

func (noopLogger) Info(context.Context, string, ...any)  {}
func (noopLogger) Error(context.Context, string, ...any) {}
func (noopLogger) Warn(context.Context, string, ...any)  {}
func (noopLogger) Debug(context.Context, string, ...any) {}

func resolveLogger(logger []ports.Logger) ports.Logger {
	if len(logger) > 0 && logger[0] != nil {
		return logger[0]
	}
	return noopLogger{}
}

type ApplicationService struct {
	repo   ports.ApplicationRepository
	logger ports.Logger
}

func NewApplicationService(repo ports.ApplicationRepository, logger ...ports.Logger) *ApplicationService {
	return &ApplicationService{repo: repo, logger: resolveLogger(logger)}
}

func (s *ApplicationService) Create(ctx context.Context, app domain.Application) error {
	if app.ID == "" || app.Name == "" {
		s.logger.Warn(ctx, "invalid application create input", "app_id", app.ID)
		return domain.ErrInvalidInput
	}
	now := time.Now().UTC()
	app.CreatedAt = now
	app.UpdatedAt = now
	err := s.repo.Create(ctx, app)
	if err != nil {
		s.logger.Error(ctx, "failed to create application", "app_id", app.ID, "error", err)
		return err
	}
	s.logger.Info(ctx, "application created", "app_id", app.ID)
	return nil
}

func (s *ApplicationService) Update(ctx context.Context, app domain.Application) error {
	if app.ID == "" || app.Name == "" {
		s.logger.Warn(ctx, "invalid application update input", "app_id", app.ID)
		return domain.ErrInvalidInput
	}
	app.UpdatedAt = time.Now().UTC()
	err := s.repo.Update(ctx, app)
	if err != nil {
		s.logger.Error(ctx, "failed to update application", "app_id", app.ID, "error", err)
		return err
	}
	s.logger.Info(ctx, "application updated", "app_id", app.ID)
	return nil
}

func (s *ApplicationService) GetByID(ctx context.Context, appID string) (domain.Application, error) {
	if appID == "" {
		s.logger.Warn(ctx, "invalid application id", "app_id", appID)
		return domain.Application{}, domain.ErrInvalidInput
	}
	app, err := s.repo.GetByID(ctx, appID)
	if err != nil {
		s.logger.Error(ctx, "failed to get application", "app_id", appID, "error", err)
		return domain.Application{}, err
	}
	s.logger.Debug(ctx, "application fetched", "app_id", appID)
	return app, nil
}

type RoleService struct {
	repo   ports.RoleRepository
	logger ports.Logger
}

func NewRoleService(repo ports.RoleRepository, logger ...ports.Logger) *RoleService {
	return &RoleService{repo: repo, logger: resolveLogger(logger)}
}

func (s *RoleService) Create(ctx context.Context, role domain.Role) error {
	if role.AppID == "" || role.ID == "" || role.Name == "" {
		s.logger.Warn(ctx, "invalid role create input", "app_id", role.AppID, "role_id", role.ID)
		return domain.ErrInvalidInput
	}
	now := time.Now().UTC()
	role.CreatedAt = now
	role.UpdatedAt = now
	err := s.repo.Create(ctx, role)
	if err != nil {
		s.logger.Error(ctx, "failed to create role", "app_id", role.AppID, "role_id", role.ID, "error", err)
		return err
	}
	s.logger.Info(ctx, "role created", "app_id", role.AppID, "role_id", role.ID)
	return nil
}

func (s *RoleService) Update(ctx context.Context, role domain.Role) error {
	if role.AppID == "" || role.ID == "" || role.Name == "" {
		s.logger.Warn(ctx, "invalid role update input", "app_id", role.AppID, "role_id", role.ID)
		return domain.ErrInvalidInput
	}
	role.UpdatedAt = time.Now().UTC()
	err := s.repo.Update(ctx, role)
	if err != nil {
		s.logger.Error(ctx, "failed to update role", "app_id", role.AppID, "role_id", role.ID, "error", err)
		return err
	}
	s.logger.Info(ctx, "role updated", "app_id", role.AppID, "role_id", role.ID)
	return nil
}

func (s *RoleService) ListByAppID(ctx context.Context, appID string) ([]domain.Role, error) {
	if appID == "" {
		s.logger.Warn(ctx, "invalid role list app id", "app_id", appID)
		return nil, domain.ErrInvalidInput
	}
	roles, err := s.repo.ListByAppID(ctx, appID)
	if err != nil {
		s.logger.Error(ctx, "failed to list roles", "app_id", appID, "error", err)
		return nil, err
	}
	s.logger.Debug(ctx, "roles listed", "app_id", appID, "count", len(roles))
	return roles, nil
}

type PermissionService struct {
	repo   ports.PermissionRepository
	logger ports.Logger
}

func NewPermissionService(repo ports.PermissionRepository, logger ...ports.Logger) *PermissionService {
	return &PermissionService{repo: repo, logger: resolveLogger(logger)}
}

func (s *PermissionService) Create(ctx context.Context, permission domain.Permission) error {
	if permission.AppID == "" || permission.ID == "" || permission.Name == "" {
		s.logger.Warn(ctx, "invalid permission create input", "app_id", permission.AppID, "permission_id", permission.ID)
		return domain.ErrInvalidInput
	}
	permission.CreatedAt = time.Now().UTC()
	err := s.repo.Create(ctx, permission)
	if err != nil {
		s.logger.Error(ctx, "failed to create permission", "app_id", permission.AppID, "permission_id", permission.ID, "error", err)
		return err
	}
	s.logger.Info(ctx, "permission created", "app_id", permission.AppID, "permission_id", permission.ID)
	return nil
}

func (s *PermissionService) ListByAppID(ctx context.Context, appID string) ([]domain.Permission, error) {
	if appID == "" {
		s.logger.Warn(ctx, "invalid permission list app id", "app_id", appID)
		return nil, domain.ErrInvalidInput
	}
	permissions, err := s.repo.ListByAppID(ctx, appID)
	if err != nil {
		s.logger.Error(ctx, "failed to list permissions", "app_id", appID, "error", err)
		return nil, err
	}
	s.logger.Debug(ctx, "permissions listed", "app_id", appID, "count", len(permissions))
	return permissions, nil
}

type UserService struct {
	userRepo ports.UserRoleRepository
	roleRepo ports.RoleRepository
	logger   ports.Logger
}

func NewUserService(userRepo ports.UserRoleRepository, roleRepo ports.RoleRepository, logger ...ports.Logger) *UserService {
	return &UserService{userRepo: userRepo, roleRepo: roleRepo, logger: resolveLogger(logger)}
}

func (s *UserService) AssignRole(ctx context.Context, appID, userID, roleID string) error {
	if appID == "" || userID == "" || roleID == "" {
		s.logger.Warn(ctx, "invalid assign role input", "app_id", appID, "user_id", userID, "role_id", roleID)
		return domain.ErrInvalidInput
	}
	roles, err := s.roleRepo.ListByAppID(ctx, appID)
	if err != nil {
		s.logger.Error(ctx, "failed to list roles for assignment", "app_id", appID, "error", err)
		return err
	}
	found := false
	for _, role := range roles {
		if role.ID == roleID {
			found = true
			break
		}
	}
	if !found {
		s.logger.Warn(ctx, "role not found for assignment", "app_id", appID, "user_id", userID, "role_id", roleID)
		return domain.ErrNotFound
	}
	if err := s.userRepo.AssignRole(ctx, appID, userID, roleID); err != nil {
		s.logger.Error(ctx, "failed to assign role", "app_id", appID, "user_id", userID, "role_id", roleID, "error", err)
		return err
	}
	s.logger.Info(ctx, "role assigned", "app_id", appID, "user_id", userID, "role_id", roleID)
	return nil
}

func (s *UserService) GetUserAppRoles(ctx context.Context, appID, userID string) (domain.UserAppRoles, error) {
	if appID == "" || userID == "" {
		s.logger.Warn(ctx, "invalid user roles query", "app_id", appID, "user_id", userID)
		return domain.UserAppRoles{}, domain.ErrInvalidInput
	}
	userRoles, err := s.userRepo.GetByUserAndApp(ctx, appID, userID)
	if err != nil {
		s.logger.Error(ctx, "failed to get user roles", "app_id", appID, "user_id", userID, "error", err)
		return domain.UserAppRoles{}, err
	}
	s.logger.Debug(ctx, "user roles fetched", "app_id", appID, "user_id", userID, "roles", len(userRoles.Roles))
	return userRoles, nil
}

type AuthorizationService struct {
	userRepo ports.UserRoleRepository
	roleRepo ports.RoleRepository
	logger   ports.Logger
}

func NewAuthorizationService(userRepo ports.UserRoleRepository, roleRepo ports.RoleRepository, logger ...ports.Logger) *AuthorizationService {
	return &AuthorizationService{userRepo: userRepo, roleRepo: roleRepo, logger: resolveLogger(logger)}
}

func (s *AuthorizationService) IsAllowed(ctx context.Context, appID, userID, permission string) (bool, error) {
	if appID == "" || userID == "" || permission == "" {
		s.logger.Warn(ctx, "invalid authorize input", "app_id", appID, "user_id", userID, "permission", permission)
		return false, domain.ErrInvalidInput
	}
	userRoles, err := s.userRepo.GetByUserAndApp(ctx, appID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.logger.Info(ctx, "user has no roles", "app_id", appID, "user_id", userID)
			return false, nil
		}
		s.logger.Error(ctx, "failed to get user roles for authorization", "app_id", appID, "user_id", userID, "error", err)
		return false, err
	}
	if len(userRoles.Roles) == 0 {
		s.logger.Info(ctx, "authorization denied: empty roles", "app_id", appID, "user_id", userID)
		return false, nil
	}
	roles, err := s.roleRepo.ListByAppID(ctx, appID)
	if err != nil {
		s.logger.Error(ctx, "failed to list roles for authorization", "app_id", appID, "error", err)
		return false, err
	}
	rolePerms := map[string][]string{}
	for _, role := range roles {
		rolePerms[role.ID] = role.Permissions
	}
	for _, userRole := range userRoles.Roles {
		if perms, ok := rolePerms[userRole]; ok && slices.Contains(perms, permission) {
			s.logger.Info(ctx, "authorization allowed", "app_id", appID, "user_id", userID, "permission", permission)
			return true, nil
		}
	}
	s.logger.Info(ctx, "authorization denied", "app_id", appID, "user_id", userID, "permission", permission)
	return false, nil
}
