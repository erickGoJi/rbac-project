package application

import (
	"context"
	"errors"
	"rbac-project/internal/domain"
	"rbac-project/internal/ports"
	"slices"
	"time"
)

type ApplicationService struct {
	repo ports.ApplicationRepository
}

func NewApplicationService(repo ports.ApplicationRepository) *ApplicationService {
	return &ApplicationService{repo: repo}
}

func (s *ApplicationService) Create(ctx context.Context, app domain.Application) error {
	if app.ID == "" || app.Name == "" {
		return domain.ErrInvalidInput
	}
	now := time.Now().UTC()
	app.CreatedAt = now
	app.UpdatedAt = now
	return s.repo.Create(ctx, app)
}

func (s *ApplicationService) Update(ctx context.Context, app domain.Application) error {
	if app.ID == "" || app.Name == "" {
		return domain.ErrInvalidInput
	}
	app.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, app)
}

func (s *ApplicationService) GetByID(ctx context.Context, appID string) (domain.Application, error) {
	if appID == "" {
		return domain.Application{}, domain.ErrInvalidInput
	}
	return s.repo.GetByID(ctx, appID)
}

type RoleService struct {
	repo ports.RoleRepository
}

func NewRoleService(repo ports.RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

func (s *RoleService) Create(ctx context.Context, role domain.Role) error {
	if role.AppID == "" || role.ID == "" || role.Name == "" {
		return domain.ErrInvalidInput
	}
	now := time.Now().UTC()
	role.CreatedAt = now
	role.UpdatedAt = now
	return s.repo.Create(ctx, role)
}

func (s *RoleService) Update(ctx context.Context, role domain.Role) error {
	if role.AppID == "" || role.ID == "" || role.Name == "" {
		return domain.ErrInvalidInput
	}
	role.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, role)
}

func (s *RoleService) ListByAppID(ctx context.Context, appID string) ([]domain.Role, error) {
	if appID == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.ListByAppID(ctx, appID)
}

type PermissionService struct {
	repo ports.PermissionRepository
}

func NewPermissionService(repo ports.PermissionRepository) *PermissionService {
	return &PermissionService{repo: repo}
}

func (s *PermissionService) Create(ctx context.Context, permission domain.Permission) error {
	if permission.AppID == "" || permission.ID == "" || permission.Name == "" {
		return domain.ErrInvalidInput
	}
	permission.CreatedAt = time.Now().UTC()
	return s.repo.Create(ctx, permission)
}

func (s *PermissionService) ListByAppID(ctx context.Context, appID string) ([]domain.Permission, error) {
	if appID == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.ListByAppID(ctx, appID)
}

type UserService struct {
	userRepo ports.UserRoleRepository
	roleRepo ports.RoleRepository
}

func NewUserService(userRepo ports.UserRoleRepository, roleRepo ports.RoleRepository) *UserService {
	return &UserService{userRepo: userRepo, roleRepo: roleRepo}
}

func (s *UserService) AssignRole(ctx context.Context, appID, userID, roleID string) error {
	if appID == "" || userID == "" || roleID == "" {
		return domain.ErrInvalidInput
	}
	roles, err := s.roleRepo.ListByAppID(ctx, appID)
	if err != nil {
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
		return domain.ErrNotFound
	}
	return s.userRepo.AssignRole(ctx, appID, userID, roleID)
}

func (s *UserService) GetUserAppRoles(ctx context.Context, appID, userID string) (domain.UserAppRoles, error) {
	if appID == "" || userID == "" {
		return domain.UserAppRoles{}, domain.ErrInvalidInput
	}
	return s.userRepo.GetByUserAndApp(ctx, appID, userID)
}

type AuthorizationService struct {
	userRepo ports.UserRoleRepository
	roleRepo ports.RoleRepository
}

func NewAuthorizationService(userRepo ports.UserRoleRepository, roleRepo ports.RoleRepository) *AuthorizationService {
	return &AuthorizationService{userRepo: userRepo, roleRepo: roleRepo}
}

func (s *AuthorizationService) IsAllowed(ctx context.Context, appID, userID, permission string) (bool, error) {
	if appID == "" || userID == "" || permission == "" {
		return false, domain.ErrInvalidInput
	}
	userRoles, err := s.userRepo.GetByUserAndApp(ctx, appID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if len(userRoles.Roles) == 0 {
		return false, nil
	}
	roles, err := s.roleRepo.ListByAppID(ctx, appID)
	if err != nil {
		return false, err
	}
	rolePerms := map[string][]string{}
	for _, role := range roles {
		rolePerms[role.ID] = role.Permissions
	}
	for _, userRole := range userRoles.Roles {
		if perms, ok := rolePerms[userRole]; ok && slices.Contains(perms, permission) {
			return true, nil
		}
	}
	return false, nil
}
