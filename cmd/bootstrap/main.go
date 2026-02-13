package main

import (
	"errors"
	"log"
	"os"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/labstack/echo/v4"
	adapterauth "rbac-project/internal/adapters/http/middleware"
	"rbac-project/internal/application"
	"rbac-project/internal/infrastructure/auth"
	"rbac-project/internal/infrastructure/dynamodb"
	httpiface "rbac-project/internal/interfaces/http"
)

type config struct {
	TableName         string
	Region            string
	UserPoolID        string
	AuthMode          adapterauth.Mode
	AuthorizeTestMode string
	Port              string
}

func loadConfig() (config, error) {
	authMode, err := adapterauth.ParseAuthMode()
	if err != nil {
		return config{}, err
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	cfg := config{
		TableName:         os.Getenv("TABLE_NAME"),
		Region:            os.Getenv("AWS_REGION"),
		UserPoolID:        os.Getenv("COGNITO_USER_POOL_ID"),
		AuthMode:          authMode,
		AuthorizeTestMode: os.Getenv("AUTHORIZE_TEST_MODE"),
		Port:              port,
	}
	if cfg.TableName == "" || cfg.Region == "" {
		return config{}, errors.New("missing required environment variables")
	}
	if cfg.AuthMode == adapterauth.ModeCognito && cfg.UserPoolID == "" {
		return config{}, errors.New("COGNITO_USER_POOL_ID is required for cognito auth mode")
	}
	return cfg, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	xray.Configure(xray.Config{LogLevel: "error"})

	ddbClient, err := dynamodb.NewClient(cfg.Region, cfg.TableName)
	if err != nil {
		log.Fatal(err)
	}
	appRepo := dynamodb.NewApplicationRepository(ddbClient)
	roleRepo := dynamodb.NewRoleRepository(ddbClient)
	permRepo := dynamodb.NewPermissionRepository(ddbClient)
	userRepo := dynamodb.NewUserRoleRepository(ddbClient)

	appSvc := application.NewApplicationService(appRepo)
	roleSvc := application.NewRoleService(roleRepo)
	permSvc := application.NewPermissionService(permRepo)
	userSvc := application.NewUserService(userRepo, roleRepo)
	authorizationSvc := application.NewAuthorizationService(userRepo, roleRepo)

	var cognitoHandler echo.MiddlewareFunc
	if cfg.AuthMode == adapterauth.ModeCognito {
		cognitoHandler = auth.NewCognitoMiddleware(cfg.UserPoolID, cfg.Region).Handler
	}
	authMiddleware, err := adapterauth.AuthMiddleware(cognitoHandler)
	if err != nil {
		log.Fatal(err)
	}
	mw := httpiface.Middleware{Auth: authMiddleware}

	e := httpiface.NewMainRouter(
		httpiface.NewApplicationsHandler(appSvc),
		httpiface.NewRolesHandler(roleSvc),
		httpiface.NewPermissionsHandler(permSvc),
		httpiface.NewUsersHandler(userSvc),
		httpiface.NewAuthorizationHandler(authorizationSvc),
		mw,
	)
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
