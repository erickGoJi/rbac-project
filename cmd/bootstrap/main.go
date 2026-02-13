package main

import (
	"context"
	"errors"
	"os"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/labstack/echo/v4"
	adaptermiddleware "rbac-project/internal/adapters/http/middleware"
	adapterlogger "rbac-project/internal/adapters/logger"
	"rbac-project/internal/application"
	"rbac-project/internal/infrastructure/auth"
	"rbac-project/internal/infrastructure/dynamodb"
	httpiface "rbac-project/internal/interfaces/http"
)

type config struct {
	TableName         string
	Region            string
	UserPoolID        string
	AuthMode          adaptermiddleware.Mode
	AuthorizeTestMode string
	Port              string
}

func loadConfig() (config, error) {
	authMode, err := adaptermiddleware.ParseAuthMode()
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
	if cfg.AuthMode == adaptermiddleware.ModeCognito && cfg.UserPoolID == "" {
		return config{}, errors.New("COGNITO_USER_POOL_ID is required for cognito auth mode")
	}
	return cfg, nil
}

func main() {
	logger := adapterlogger.New()

	cfg, err := loadConfig()
	if err != nil {
		logger.Error(context.Background(), "configuration error", "error", err)
		os.Exit(1)
	}
	xray.Configure(xray.Config{LogLevel: "error"})

	ddbClient, err := dynamodb.NewClient(context.Background(), cfg.Region, cfg.TableName)
	if err != nil {
		logger.Error(context.Background(), "failed to initialize dynamodb client", "error", err)
		os.Exit(1)
	}
	appRepo := dynamodb.NewApplicationRepository(ddbClient)
	roleRepo := dynamodb.NewRoleRepository(ddbClient)
	permRepo := dynamodb.NewPermissionRepository(ddbClient)
	userRepo := dynamodb.NewUserRoleRepository(ddbClient)

	appSvc := application.NewApplicationService(appRepo, logger)
	roleSvc := application.NewRoleService(roleRepo, logger)
	permSvc := application.NewPermissionService(permRepo, logger)
	userSvc := application.NewUserService(userRepo, roleRepo, logger)
	authorizationSvc := application.NewAuthorizationService(userRepo, roleRepo, logger)

	var cognitoHandler echo.MiddlewareFunc
	if cfg.AuthMode == adaptermiddleware.ModeCognito {
		cognitoHandler = auth.NewCognitoMiddleware(cfg.UserPoolID, cfg.Region).Handler
	}
	authMiddleware, err := adaptermiddleware.AuthMiddleware(cognitoHandler)
	if err != nil {
		logger.Error(context.Background(), "failed to initialize auth middleware", "error", err)
		os.Exit(1)
	}
	mw := httpiface.Middleware{
		Auth:          authMiddleware,
		XRay:          adaptermiddleware.XRayMiddleware("rbac-http"),
		RequestLogger: adaptermiddleware.RequestLogger(logger),
	}

	e := httpiface.NewMainRouter(
		httpiface.NewApplicationsHandler(appSvc, logger),
		httpiface.NewRolesHandler(roleSvc, logger),
		httpiface.NewPermissionsHandler(permSvc, logger),
		httpiface.NewUsersHandler(userSvc, logger),
		httpiface.NewAuthorizationHandler(authorizationSvc, logger),
		mw,
	)
	logger.Info(context.Background(), "starting http server", "port", cfg.Port)
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
