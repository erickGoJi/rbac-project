package ports

import (
	"context"
	"rbac-project/internal/domain"
)

type ApplicationRepository interface {
	Create(ctx context.Context, app domain.Application) error
	Update(ctx context.Context, app domain.Application) error
	GetByID(ctx context.Context, appID string) (domain.Application, error)
}

type RoleRepository interface {
	Create(ctx context.Context, role domain.Role) error
	Update(ctx context.Context, role domain.Role) error
	ListByAppID(ctx context.Context, appID string) ([]domain.Role, error)
}

type PermissionRepository interface {
	Create(ctx context.Context, permission domain.Permission) error
	ListByAppID(ctx context.Context, appID string) ([]domain.Permission, error)
}

type UserRoleRepository interface {
	AssignRole(ctx context.Context, appID, userID, roleID string) error
	GetByUserAndApp(ctx context.Context, appID, userID string) (domain.UserAppRoles, error)
}
