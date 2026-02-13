package dynamodb

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	awsv2dynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awsv2types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awsv2xray "github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	"rbac-project/internal/domain"
)

type Client struct {
	db        *awsv2dynamodb.Client
	tableName string
}

func NewClient(ctx context.Context, region, tableName string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	awsv2xray.AWSV2Instrumentor(&cfg.APIOptions)
	client := awsv2dynamodb.NewFromConfig(cfg)
	return &Client{db: client, tableName: tableName}, nil
}

func appPK(appID string) string     { return "APP#" + appID }
func appMetaSK() string             { return "META" }
func roleSK(roleID string) string   { return "ROLE#" + roleID }
func permSK(permID string) string   { return "PERM#" + permID }
func userPK(userID string) string   { return "USER#" + userID }
func userAppSK(appID string) string { return "APP#" + appID }

func isConditionalCheckFailure(err error) bool {
	var condErr *awsv2types.ConditionalCheckFailedException
	return errors.As(err, &condErr)
}

type ApplicationRepository struct{ client *Client }

type RoleRepository struct{ client *Client }

type PermissionRepository struct{ client *Client }

type UserRoleRepository struct{ client *Client }

func NewApplicationRepository(client *Client) *ApplicationRepository {
	return &ApplicationRepository{client: client}
}

func NewRoleRepository(client *Client) *RoleRepository {
	return &RoleRepository{client: client}
}

func NewPermissionRepository(client *Client) *PermissionRepository {
	return &PermissionRepository{client: client}
}

func NewUserRoleRepository(client *Client) *UserRoleRepository {
	return &UserRoleRepository{client: client}
}

func (r *ApplicationRepository) Create(ctx context.Context, app domain.Application) error {
	item := map[string]any{
		"PK":          appPK(app.ID),
		"SK":          appMetaSK(),
		"EntityType":  "APPLICATION",
		"ID":          app.ID,
		"Name":        app.Name,
		"Description": app.Description,
		"CreatedAt":   app.CreatedAt.Format(time.RFC3339),
		"UpdatedAt":   app.UpdatedAt.Format(time.RFC3339),
	}
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}
	return xray.Capture(ctx, "DynamoDB.PutApplication", func(ctx context.Context) error {
		_, err = r.client.db.PutItem(ctx, &awsv2dynamodb.PutItemInput{
			TableName:           aws.String(r.client.tableName),
			Item:                av,
			ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
		})
		return err
	})
}

func (r *ApplicationRepository) Update(ctx context.Context, app domain.Application) error {
	return xray.Capture(ctx, "DynamoDB.UpdateApplication", func(ctx context.Context) error {
		_, err := r.client.db.UpdateItem(ctx, &awsv2dynamodb.UpdateItemInput{
			TableName: aws.String(r.client.tableName),
			Key: map[string]awsv2types.AttributeValue{
				"PK": &awsv2types.AttributeValueMemberS{Value: appPK(app.ID)},
				"SK": &awsv2types.AttributeValueMemberS{Value: appMetaSK()},
			},
			UpdateExpression: aws.String("SET #n = :n, #d = :d, UpdatedAt = :u"),
			ExpressionAttributeNames: map[string]string{
				"#n": "Name",
				"#d": "Description",
			},
			ExpressionAttributeValues: map[string]awsv2types.AttributeValue{
				":n": &awsv2types.AttributeValueMemberS{Value: app.Name},
				":d": &awsv2types.AttributeValueMemberS{Value: app.Description},
				":u": &awsv2types.AttributeValueMemberS{Value: app.UpdatedAt.Format(time.RFC3339)},
			},
			ConditionExpression: aws.String("attribute_exists(PK)"),
		})
		if isConditionalCheckFailure(err) {
			return domain.ErrNotFound
		}
		return err
	})
}

func (r *ApplicationRepository) GetByID(ctx context.Context, appID string) (domain.Application, error) {
	var out *awsv2dynamodb.GetItemOutput
	err := xray.Capture(ctx, "DynamoDB.GetApplication", func(ctx context.Context) error {
		var e error
		out, e = r.client.db.GetItem(ctx, &awsv2dynamodb.GetItemInput{
			TableName: aws.String(r.client.tableName),
			Key: map[string]awsv2types.AttributeValue{
				"PK": &awsv2types.AttributeValueMemberS{Value: appPK(appID)},
				"SK": &awsv2types.AttributeValueMemberS{Value: appMetaSK()},
			},
		})
		return e
	})
	if err != nil {
		return domain.Application{}, err
	}
	if out.Item == nil {
		return domain.Application{}, domain.ErrNotFound
	}
	raw := struct {
		ID          string `dynamodbav:"ID"`
		Name        string `dynamodbav:"Name"`
		Description string `dynamodbav:"Description"`
		CreatedAt   string `dynamodbav:"CreatedAt"`
		UpdatedAt   string `dynamodbav:"UpdatedAt"`
	}{}
	if err := attributevalue.UnmarshalMap(out.Item, &raw); err != nil {
		return domain.Application{}, err
	}
	createdAt, _ := time.Parse(time.RFC3339, raw.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, raw.UpdatedAt)
	return domain.Application{ID: raw.ID, Name: raw.Name, Description: raw.Description, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func (r *RoleRepository) Create(ctx context.Context, role domain.Role) error {
	item := map[string]any{
		"PK":          appPK(role.AppID),
		"SK":          roleSK(role.ID),
		"EntityType":  "ROLE",
		"ID":          role.ID,
		"Name":        role.Name,
		"Permissions": role.Permissions,
		"CreatedAt":   role.CreatedAt.Format(time.RFC3339),
		"UpdatedAt":   role.UpdatedAt.Format(time.RFC3339),
	}
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}
	return xray.Capture(ctx, "DynamoDB.PutRole", func(ctx context.Context) error {
		_, err = r.client.db.PutItem(ctx, &awsv2dynamodb.PutItemInput{
			TableName:           aws.String(r.client.tableName),
			Item:                av,
			ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
		})
		return err
	})
}

func (r *RoleRepository) Update(ctx context.Context, role domain.Role) error {
	permissionsAV, err := attributevalue.Marshal(role.Permissions)
	if err != nil {
		return err
	}
	return xray.Capture(ctx, "DynamoDB.UpdateRole", func(ctx context.Context) error {
		_, err := r.client.db.UpdateItem(ctx, &awsv2dynamodb.UpdateItemInput{
			TableName: aws.String(r.client.tableName),
			Key: map[string]awsv2types.AttributeValue{
				"PK": &awsv2types.AttributeValueMemberS{Value: appPK(role.AppID)},
				"SK": &awsv2types.AttributeValueMemberS{Value: roleSK(role.ID)},
			},
			UpdateExpression: aws.String("SET #n = :n, Permissions = :p, UpdatedAt = :u"),
			ExpressionAttributeNames: map[string]string{
				"#n": "Name",
			},
			ExpressionAttributeValues: map[string]awsv2types.AttributeValue{
				":n": &awsv2types.AttributeValueMemberS{Value: role.Name},
				":p": permissionsAV,
				":u": &awsv2types.AttributeValueMemberS{Value: role.UpdatedAt.Format(time.RFC3339)},
			},
			ConditionExpression: aws.String("attribute_exists(PK)"),
		})
		if isConditionalCheckFailure(err) {
			return domain.ErrNotFound
		}
		return err
	})
}

func (r *RoleRepository) ListByAppID(ctx context.Context, appID string) ([]domain.Role, error) {
	var out *awsv2dynamodb.QueryOutput
	err := xray.Capture(ctx, "DynamoDB.QueryRoles", func(ctx context.Context) error {
		var e error
		out, e = r.client.db.Query(ctx, &awsv2dynamodb.QueryInput{
			TableName:              aws.String(r.client.tableName),
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
			ExpressionAttributeValues: map[string]awsv2types.AttributeValue{
				":pk": &awsv2types.AttributeValueMemberS{Value: appPK(appID)},
				":sk": &awsv2types.AttributeValueMemberS{Value: "ROLE#"},
			},
		})
		return e
	})
	if err != nil {
		return nil, err
	}
	roles := make([]domain.Role, 0, len(out.Items))
	for _, item := range out.Items {
		raw := struct {
			ID          string   `dynamodbav:"ID"`
			Name        string   `dynamodbav:"Name"`
			Permissions []string `dynamodbav:"Permissions"`
			CreatedAt   string   `dynamodbav:"CreatedAt"`
			UpdatedAt   string   `dynamodbav:"UpdatedAt"`
		}{}
		if err := attributevalue.UnmarshalMap(item, &raw); err != nil {
			return nil, err
		}
		createdAt, _ := time.Parse(time.RFC3339, raw.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339, raw.UpdatedAt)
		roles = append(roles, domain.Role{AppID: appID, ID: raw.ID, Name: raw.Name, Permissions: raw.Permissions, CreatedAt: createdAt, UpdatedAt: updatedAt})
	}
	return roles, nil
}

func (r *PermissionRepository) Create(ctx context.Context, permission domain.Permission) error {
	item := map[string]any{
		"PK":          appPK(permission.AppID),
		"SK":          permSK(permission.ID),
		"EntityType":  "PERMISSION",
		"ID":          permission.ID,
		"Name":        permission.Name,
		"Description": permission.Description,
		"CreatedAt":   permission.CreatedAt.Format(time.RFC3339),
	}
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}
	return xray.Capture(ctx, "DynamoDB.PutPermission", func(ctx context.Context) error {
		_, err = r.client.db.PutItem(ctx, &awsv2dynamodb.PutItemInput{
			TableName:           aws.String(r.client.tableName),
			Item:                av,
			ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
		})
		return err
	})
}

func (r *PermissionRepository) ListByAppID(ctx context.Context, appID string) ([]domain.Permission, error) {
	var out *awsv2dynamodb.QueryOutput
	err := xray.Capture(ctx, "DynamoDB.QueryPermissions", func(ctx context.Context) error {
		var e error
		out, e = r.client.db.Query(ctx, &awsv2dynamodb.QueryInput{
			TableName:              aws.String(r.client.tableName),
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
			ExpressionAttributeValues: map[string]awsv2types.AttributeValue{
				":pk": &awsv2types.AttributeValueMemberS{Value: appPK(appID)},
				":sk": &awsv2types.AttributeValueMemberS{Value: "PERM#"},
			},
		})
		return e
	})
	if err != nil {
		return nil, err
	}
	permissions := make([]domain.Permission, 0, len(out.Items))
	for _, item := range out.Items {
		raw := struct {
			ID          string `dynamodbav:"ID"`
			Name        string `dynamodbav:"Name"`
			Description string `dynamodbav:"Description"`
			CreatedAt   string `dynamodbav:"CreatedAt"`
		}{}
		if err := attributevalue.UnmarshalMap(item, &raw); err != nil {
			return nil, err
		}
		createdAt, _ := time.Parse(time.RFC3339, raw.CreatedAt)
		permissions = append(permissions, domain.Permission{AppID: appID, ID: raw.ID, Name: raw.Name, Description: raw.Description, CreatedAt: createdAt})
	}
	return permissions, nil
}

func (r *UserRoleRepository) AssignRole(ctx context.Context, appID, userID, roleID string) error {
	current, err := r.GetByUserAndApp(ctx, appID, userID)
	if err != nil && err != domain.ErrNotFound {
		return err
	}
	roles := current.Roles
	for _, role := range roles {
		if role == roleID {
			return nil
		}
	}
	roles = append(roles, roleID)
	rolesAV, err := attributevalue.Marshal(roles)
	if err != nil {
		return err
	}
	return xray.Capture(ctx, "DynamoDB.PutUserRole", func(ctx context.Context) error {
		_, err := r.client.db.PutItem(ctx, &awsv2dynamodb.PutItemInput{
			TableName: aws.String(r.client.tableName),
			Item: map[string]awsv2types.AttributeValue{
				"PK":         &awsv2types.AttributeValueMemberS{Value: userPK(userID)},
				"SK":         &awsv2types.AttributeValueMemberS{Value: userAppSK(appID)},
				"EntityType": &awsv2types.AttributeValueMemberS{Value: "USER_APP_ROLES"},
				"Roles":      rolesAV,
				"UpdatedAt":  &awsv2types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
			},
		})
		return err
	})
}

func (r *UserRoleRepository) GetByUserAndApp(ctx context.Context, appID, userID string) (domain.UserAppRoles, error) {
	var out *awsv2dynamodb.GetItemOutput
	err := xray.Capture(ctx, "DynamoDB.GetUserRoles", func(ctx context.Context) error {
		var e error
		out, e = r.client.db.GetItem(ctx, &awsv2dynamodb.GetItemInput{
			TableName: aws.String(r.client.tableName),
			Key: map[string]awsv2types.AttributeValue{
				"PK": &awsv2types.AttributeValueMemberS{Value: userPK(userID)},
				"SK": &awsv2types.AttributeValueMemberS{Value: userAppSK(appID)},
			},
		})
		return e
	})
	if err != nil {
		return domain.UserAppRoles{}, err
	}
	if out.Item == nil {
		return domain.UserAppRoles{}, domain.ErrNotFound
	}
	raw := struct {
		Roles     []string `dynamodbav:"Roles"`
		UpdatedAt string   `dynamodbav:"UpdatedAt"`
	}{}
	if err := attributevalue.UnmarshalMap(out.Item, &raw); err != nil {
		return domain.UserAppRoles{}, err
	}
	updatedAt, _ := time.Parse(time.RFC3339, raw.UpdatedAt)
	return domain.UserAppRoles{UserID: userID, AppID: appID, Roles: raw.Roles, UpdatedAt: updatedAt}, nil
}
