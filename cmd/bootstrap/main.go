package main

import (
	"errors"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/labstack/echo/v4"
	adapterauth "rbac-project/internal/adapters/http/middleware"
	"rbac-project/internal/application"
	"rbac-project/internal/infrastructure/auth"
	"rbac-project/internal/infrastructure/dynamodb"
	httpiface "rbac-project/internal/interfaces/http"
	platformlambda "rbac-project/internal/platform/lambda"
)

type config struct {
	TableName  string
	Region     string
	UserPoolID string
	Handler    string
	AuthMode   adapterauth.Mode
}

func loadConfig() (config, error) {
	authMode, err := adapterauth.ParseAuthMode()
	if err != nil {
		return config{}, err
	}
	cfg := config{
		TableName:  os.Getenv("TABLE_NAME"),
		Region:     os.Getenv("AWS_REGION"),
		UserPoolID: os.Getenv("COGNITO_USER_POOL_ID"),
		Handler:    os.Getenv("LAMBDA_HANDLER"),
		AuthMode:   authMode,
	}
	if cfg.TableName == "" || cfg.Region == "" || cfg.Handler == "" {
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

	applicationsHandler := httpiface.NewApplicationsHandler(appSvc)
	rolesHandler := httpiface.NewRolesHandler(roleSvc)
	permissionsHandler := httpiface.NewPermissionsHandler(permSvc)
	usersHandler := httpiface.NewUsersHandler(userSvc)
	authorizeHandler := httpiface.NewAuthorizationHandler(authorizationSvc)

	var lambdaHandler platformlambda.LambdaHandler
	switch cfg.Handler {
	case "applications":
		lambdaHandler = platformlambda.NewLambdaHandler(httpiface.NewApplicationsRouter(applicationsHandler, mw))
	case "roles":
		lambdaHandler = platformlambda.NewLambdaHandler(httpiface.NewRolesRouter(rolesHandler, mw))
	case "permissions":
		lambdaHandler = platformlambda.NewLambdaHandler(httpiface.NewPermissionsRouter(permissionsHandler, mw))
	case "users":
		lambdaHandler = platformlambda.NewLambdaHandler(httpiface.NewUsersRouter(usersHandler, mw))
	case "authorization":
		lambdaHandler = platformlambda.NewLambdaHandler(httpiface.NewAuthorizationRouter(authorizeHandler, mw))
	default:
		log.Fatalf("unknown LAMBDA_HANDLER: %s", cfg.Handler)
	}

	lambda.Start(lambdaHandler)
}
