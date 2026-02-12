package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"rbac-project/internal/domain"
)

type appRepoMock struct{ mock.Mock }

func (m *appRepoMock) Create(ctx context.Context, app domain.Application) error {
	args := m.Called(ctx, app)
	return args.Error(0)
}

func (m *appRepoMock) Update(ctx context.Context, app domain.Application) error {
	args := m.Called(ctx, app)
	return args.Error(0)
}

func (m *appRepoMock) GetByID(ctx context.Context, appID string) (domain.Application, error) {
	args := m.Called(ctx, appID)
	return args.Get(0).(domain.Application), args.Error(1)
}

type roleRepoMock struct{ mock.Mock }

func (m *roleRepoMock) Create(ctx context.Context, role domain.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *roleRepoMock) Update(ctx context.Context, role domain.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *roleRepoMock) ListByAppID(ctx context.Context, appID string) ([]domain.Role, error) {
	args := m.Called(ctx, appID)
	return args.Get(0).([]domain.Role), args.Error(1)
}

type permissionRepoMock struct{ mock.Mock }

func (m *permissionRepoMock) Create(ctx context.Context, permission domain.Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *permissionRepoMock) ListByAppID(ctx context.Context, appID string) ([]domain.Permission, error) {
	args := m.Called(ctx, appID)
	return args.Get(0).([]domain.Permission), args.Error(1)
}

type userRoleRepoMock struct{ mock.Mock }

func (m *userRoleRepoMock) AssignRole(ctx context.Context, appID, userID, roleID string) error {
	args := m.Called(ctx, appID, userID, roleID)
	return args.Error(0)
}

func (m *userRoleRepoMock) GetByUserAndApp(ctx context.Context, appID, userID string) (domain.UserAppRoles, error) {
	args := m.Called(ctx, appID, userID)
	return args.Get(0).(domain.UserAppRoles), args.Error(1)
}

func TestApplicationService_Create(t *testing.T) {
	repo := new(appRepoMock)
	svc := NewApplicationService(repo)

	repo.On("Create", mock.Anything, mock.MatchedBy(func(app domain.Application) bool {
		return app.ID == "app-1" && app.Name == "MyApp" && !app.CreatedAt.IsZero() && !app.UpdatedAt.IsZero()
	})).Return(nil)

	err := svc.Create(context.Background(), domain.Application{ID: "app-1", Name: "MyApp"})
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestApplicationService_GetByID(t *testing.T) {
	repo := new(appRepoMock)
	svc := NewApplicationService(repo)
	expected := domain.Application{ID: "app-1", Name: "my app"}
	repo.On("GetByID", mock.Anything, "app-1").Return(expected, nil)

	got, err := svc.GetByID(context.Background(), "app-1")
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestApplicationService_Update(t *testing.T) {
	repo := new(appRepoMock)
	svc := NewApplicationService(repo)

	repo.On("Update", mock.Anything, mock.MatchedBy(func(app domain.Application) bool {
		return app.ID == "app-1" && app.Name == "my app"
	})).Return(nil)

	err := svc.Update(context.Background(), domain.Application{ID: "app-1", Name: "my app", Description: "desc"})
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestApplicationService_InvalidInput(t *testing.T) {
	repo := new(appRepoMock)
	svc := NewApplicationService(repo)

	err := svc.Create(context.Background(), domain.Application{ID: "", Name: ""})
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestRoleService_Create(t *testing.T) {
	repo := new(roleRepoMock)
	svc := NewRoleService(repo)
	repo.On("Create", mock.Anything, mock.MatchedBy(func(role domain.Role) bool {
		return role.AppID == "a1" && role.ID == "r1" && role.Name == "admin"
	})).Return(nil)

	err := svc.Create(context.Background(), domain.Role{AppID: "a1", ID: "r1", Name: "admin"})
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestRoleService_UpdateAndList(t *testing.T) {
	repo := new(roleRepoMock)
	svc := NewRoleService(repo)

	repo.On("Update", mock.Anything, mock.MatchedBy(func(role domain.Role) bool {
		return role.AppID == "a1" && role.ID == "r1"
	})).Return(nil)
	repo.On("ListByAppID", mock.Anything, "a1").Return([]domain.Role{{AppID: "a1", ID: "r1"}}, nil)

	err := svc.Update(context.Background(), domain.Role{AppID: "a1", ID: "r1", Name: "admin"})
	require.NoError(t, err)

	got, err := svc.ListByAppID(context.Background(), "a1")
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestPermissionService_Create(t *testing.T) {
	repo := new(permissionRepoMock)
	svc := NewPermissionService(repo)
	repo.On("Create", mock.Anything, mock.MatchedBy(func(p domain.Permission) bool {
		return p.AppID == "a1" && p.ID == "p1" && p.Name == "read"
	})).Return(nil)

	err := svc.Create(context.Background(), domain.Permission{AppID: "a1", ID: "p1", Name: "read"})
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestPermissionService_List(t *testing.T) {
	repo := new(permissionRepoMock)
	svc := NewPermissionService(repo)
	repo.On("ListByAppID", mock.Anything, "a1").Return([]domain.Permission{{AppID: "a1", ID: "p1"}}, nil)

	got, err := svc.ListByAppID(context.Background(), "a1")
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestUserService_AssignRole(t *testing.T) {
	userRepo := new(userRoleRepoMock)
	roleRepo := new(roleRepoMock)
	svc := NewUserService(userRepo, roleRepo)

	roleRepo.On("ListByAppID", mock.Anything, "a1").Return([]domain.Role{{AppID: "a1", ID: "r1", Name: "admin"}}, nil)
	userRepo.On("AssignRole", mock.Anything, "a1", "u1", "r1").Return(nil)

	err := svc.AssignRole(context.Background(), "a1", "u1", "r1")
	require.NoError(t, err)
	userRepo.AssertExpectations(t)
}

func TestUserService_GetUserAppRoles(t *testing.T) {
	userRepo := new(userRoleRepoMock)
	roleRepo := new(roleRepoMock)
	svc := NewUserService(userRepo, roleRepo)

	userRepo.On("GetByUserAndApp", mock.Anything, "a1", "u1").Return(domain.UserAppRoles{AppID: "a1", UserID: "u1", Roles: []string{"r1"}}, nil)
	out, err := svc.GetUserAppRoles(context.Background(), "a1", "u1")
	require.NoError(t, err)
	assert.Equal(t, "u1", out.UserID)
}

func TestUserService_AssignRoleNotFound(t *testing.T) {
	userRepo := new(userRoleRepoMock)
	roleRepo := new(roleRepoMock)
	svc := NewUserService(userRepo, roleRepo)
	roleRepo.On("ListByAppID", mock.Anything, "a1").Return([]domain.Role{{ID: "other"}}, nil)

	err := svc.AssignRole(context.Background(), "a1", "u1", "r1")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAuthorizationService_Allowed(t *testing.T) {
	userRepo := new(userRoleRepoMock)
	roleRepo := new(roleRepoMock)
	svc := NewAuthorizationService(userRepo, roleRepo)

	userRepo.On("GetByUserAndApp", mock.Anything, "a1", "u1").Return(domain.UserAppRoles{AppID: "a1", UserID: "u1", Roles: []string{"admin"}}, nil)
	roleRepo.On("ListByAppID", mock.Anything, "a1").Return([]domain.Role{{ID: "admin", Permissions: []string{"perm:write"}}}, nil)

	allowed, err := svc.IsAllowed(context.Background(), "a1", "u1", "perm:write")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestAuthorizationService_InvalidInput(t *testing.T) {
	userRepo := new(userRoleRepoMock)
	roleRepo := new(roleRepoMock)
	svc := NewAuthorizationService(userRepo, roleRepo)

	allowed, err := svc.IsAllowed(context.Background(), "", "u1", "perm:read")
	assert.False(t, allowed)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestAuthorizationService_Denied(t *testing.T) {
	userRepo := new(userRoleRepoMock)
	roleRepo := new(roleRepoMock)
	svc := NewAuthorizationService(userRepo, roleRepo)

	userRepo.On("GetByUserAndApp", mock.Anything, "a1", "u1").Return(domain.UserAppRoles{AppID: "a1", UserID: "u1", Roles: []string{"viewer"}}, nil)
	roleRepo.On("ListByAppID", mock.Anything, "a1").Return([]domain.Role{{ID: "viewer", Permissions: []string{"perm:read"}}}, nil)

	allowed, err := svc.IsAllowed(context.Background(), "a1", "u1", "perm:write")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestAuthorizationService_UserWithoutRole(t *testing.T) {
	userRepo := new(userRoleRepoMock)
	roleRepo := new(roleRepoMock)
	svc := NewAuthorizationService(userRepo, roleRepo)

	userRepo.On("GetByUserAndApp", mock.Anything, "a1", "u1").Return(domain.UserAppRoles{}, domain.ErrNotFound)

	allowed, err := svc.IsAllowed(context.Background(), "a1", "u1", "perm:read")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestAuthorizationService_PropagatesErrors(t *testing.T) {
	userRepo := new(userRoleRepoMock)
	roleRepo := new(roleRepoMock)
	svc := NewAuthorizationService(userRepo, roleRepo)

	expectedErr := errors.New("db down")
	userRepo.On("GetByUserAndApp", mock.Anything, "a1", "u1").Return(domain.UserAppRoles{}, expectedErr)

	allowed, err := svc.IsAllowed(context.Background(), "a1", "u1", "perm:read")
	assert.False(t, allowed)
	assert.ErrorIs(t, err, expectedErr)
}
