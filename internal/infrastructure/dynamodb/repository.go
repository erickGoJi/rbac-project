package dynamodb

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	db "github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-xray-sdk-go/xray"
	"rbac-project/internal/domain"
)

type Client struct {
	db        *db.DynamoDB
	tableName string
}

func NewClient(region, tableName string) (*Client, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		return nil, err
	}
	client := db.New(sess)
	xray.AWS(client.Client)
	return &Client{db: client, tableName: tableName}, nil
}

func appPK(appID string) string     { return "APP#" + appID }
func appMetaSK() string             { return "META" }
func roleSK(roleID string) string   { return "ROLE#" + roleID }
func permSK(permID string) string   { return "PERM#" + permID }
func userPK(userID string) string   { return "USER#" + userID }
func userAppSK(appID string) string { return "APP#" + appID }

func isConditionalCheckFailure(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "ConditionalCheckFailedException")
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
	item := map[string]interface{}{
		"PK":          appPK(app.ID),
		"SK":          appMetaSK(),
		"EntityType":  "APPLICATION",
		"ID":          app.ID,
		"Name":        app.Name,
		"Description": app.Description,
		"CreatedAt":   app.CreatedAt.Format(time.RFC3339),
		"UpdatedAt":   app.UpdatedAt.Format(time.RFC3339),
	}
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return err
	}
	return xray.Capture(ctx, "DynamoDB.PutApplication", func(ctx context.Context) error {
		_, err = r.client.db.PutItemWithContext(ctx, &db.PutItemInput{
			TableName:           aws.String(r.client.tableName),
			Item:                av,
			ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
		})
		return err
	})
}

func (r *ApplicationRepository) Update(ctx context.Context, app domain.Application) error {
	return xray.Capture(ctx, "DynamoDB.UpdateApplication", func(ctx context.Context) error {
		_, err := r.client.db.UpdateItemWithContext(ctx, &db.UpdateItemInput{
			TableName: aws.String(r.client.tableName),
			Key: map[string]*db.AttributeValue{
				"PK": {S: aws.String(appPK(app.ID))},
				"SK": {S: aws.String(appMetaSK())},
			},
			UpdateExpression: aws.String("SET #n = :n, #d = :d, UpdatedAt = :u"),
			ExpressionAttributeNames: map[string]*string{
				"#n": aws.String("Name"),
				"#d": aws.String("Description"),
			},
			ExpressionAttributeValues: map[string]*db.AttributeValue{
				":n": {S: aws.String(app.Name)},
				":d": {S: aws.String(app.Description)},
				":u": {S: aws.String(app.UpdatedAt.Format(time.RFC3339))},
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
	var out *db.GetItemOutput
	err := xray.Capture(ctx, "DynamoDB.GetApplication", func(ctx context.Context) error {
		var e error
		out, e = r.client.db.GetItemWithContext(ctx, &db.GetItemInput{
			TableName: aws.String(r.client.tableName),
			Key: map[string]*db.AttributeValue{
				"PK": {S: aws.String(appPK(appID))},
				"SK": {S: aws.String(appMetaSK())},
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
	if err := dynamodbattribute.UnmarshalMap(out.Item, &raw); err != nil {
		return domain.Application{}, err
	}
	createdAt, _ := time.Parse(time.RFC3339, raw.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, raw.UpdatedAt)
	return domain.Application{ID: raw.ID, Name: raw.Name, Description: raw.Description, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func (r *RoleRepository) Create(ctx context.Context, role domain.Role) error {
	item := map[string]interface{}{
		"PK":          appPK(role.AppID),
		"SK":          roleSK(role.ID),
		"EntityType":  "ROLE",
		"ID":          role.ID,
		"Name":        role.Name,
		"Permissions": role.Permissions,
		"CreatedAt":   role.CreatedAt.Format(time.RFC3339),
		"UpdatedAt":   role.UpdatedAt.Format(time.RFC3339),
	}
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return err
	}
	return xray.Capture(ctx, "DynamoDB.PutRole", func(ctx context.Context) error {
		_, err = r.client.db.PutItemWithContext(ctx, &db.PutItemInput{
			TableName:           aws.String(r.client.tableName),
			Item:                av,
			ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
		})
		return err
	})
}

func (r *RoleRepository) Update(ctx context.Context, role domain.Role) error {
	permissionsAV, err := dynamodbattribute.Marshal(role.Permissions)
	if err != nil {
		return err
	}
	return xray.Capture(ctx, "DynamoDB.UpdateRole", func(ctx context.Context) error {
		_, err := r.client.db.UpdateItemWithContext(ctx, &db.UpdateItemInput{
			TableName: aws.String(r.client.tableName),
			Key: map[string]*db.AttributeValue{
				"PK": {S: aws.String(appPK(role.AppID))},
				"SK": {S: aws.String(roleSK(role.ID))},
			},
			UpdateExpression: aws.String("SET #n = :n, Permissions = :p, UpdatedAt = :u"),
			ExpressionAttributeNames: map[string]*string{
				"#n": aws.String("Name"),
			},
			ExpressionAttributeValues: map[string]*db.AttributeValue{
				":n": {S: aws.String(role.Name)},
				":p": permissionsAV,
				":u": {S: aws.String(role.UpdatedAt.Format(time.RFC3339))},
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
	var out *db.QueryOutput
	err := xray.Capture(ctx, "DynamoDB.QueryRoles", func(ctx context.Context) error {
		var e error
		out, e = r.client.db.QueryWithContext(ctx, &db.QueryInput{
			TableName:              aws.String(r.client.tableName),
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
			ExpressionAttributeValues: map[string]*db.AttributeValue{
				":pk": {S: aws.String(appPK(appID))},
				":sk": {S: aws.String("ROLE#")},
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
		if err := dynamodbattribute.UnmarshalMap(item, &raw); err != nil {
			return nil, err
		}
		createdAt, _ := time.Parse(time.RFC3339, raw.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339, raw.UpdatedAt)
		roles = append(roles, domain.Role{AppID: appID, ID: raw.ID, Name: raw.Name, Permissions: raw.Permissions, CreatedAt: createdAt, UpdatedAt: updatedAt})
	}
	return roles, nil
}

func (r *PermissionRepository) Create(ctx context.Context, permission domain.Permission) error {
	item := map[string]interface{}{
		"PK":          appPK(permission.AppID),
		"SK":          permSK(permission.ID),
		"EntityType":  "PERMISSION",
		"ID":          permission.ID,
		"Name":        permission.Name,
		"Description": permission.Description,
		"CreatedAt":   permission.CreatedAt.Format(time.RFC3339),
	}
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return err
	}
	return xray.Capture(ctx, "DynamoDB.PutPermission", func(ctx context.Context) error {
		_, err = r.client.db.PutItemWithContext(ctx, &db.PutItemInput{
			TableName:           aws.String(r.client.tableName),
			Item:                av,
			ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
		})
		return err
	})
}

func (r *PermissionRepository) ListByAppID(ctx context.Context, appID string) ([]domain.Permission, error) {
	var out *db.QueryOutput
	err := xray.Capture(ctx, "DynamoDB.QueryPermissions", func(ctx context.Context) error {
		var e error
		out, e = r.client.db.QueryWithContext(ctx, &db.QueryInput{
			TableName:              aws.String(r.client.tableName),
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
			ExpressionAttributeValues: map[string]*db.AttributeValue{
				":pk": {S: aws.String(appPK(appID))},
				":sk": {S: aws.String("PERM#")},
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
		if err := dynamodbattribute.UnmarshalMap(item, &raw); err != nil {
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
	rolesAV, err := dynamodbattribute.Marshal(roles)
	if err != nil {
		return err
	}
	return xray.Capture(ctx, "DynamoDB.PutUserRole", func(ctx context.Context) error {
		_, err := r.client.db.PutItemWithContext(ctx, &db.PutItemInput{
			TableName: aws.String(r.client.tableName),
			Item: map[string]*db.AttributeValue{
				"PK":         {S: aws.String(userPK(userID))},
				"SK":         {S: aws.String(userAppSK(appID))},
				"EntityType": {S: aws.String("USER_APP_ROLES")},
				"Roles":      rolesAV,
				"UpdatedAt":  {S: aws.String(time.Now().UTC().Format(time.RFC3339))},
			},
		})
		return err
	})
}

func (r *UserRoleRepository) GetByUserAndApp(ctx context.Context, appID, userID string) (domain.UserAppRoles, error) {
	var out *db.GetItemOutput
	err := xray.Capture(ctx, "DynamoDB.GetUserRoles", func(ctx context.Context) error {
		var e error
		out, e = r.client.db.GetItemWithContext(ctx, &db.GetItemInput{
			TableName: aws.String(r.client.tableName),
			Key: map[string]*db.AttributeValue{
				"PK": {S: aws.String(userPK(userID))},
				"SK": {S: aws.String(userAppSK(appID))},
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
	if err := dynamodbattribute.UnmarshalMap(out.Item, &raw); err != nil {
		return domain.UserAppRoles{}, err
	}
	updatedAt, _ := time.Parse(time.RFC3339, raw.UpdatedAt)
	return domain.UserAppRoles{UserID: userID, AppID: appID, Roles: raw.Roles, UpdatedAt: updatedAt}, nil
}
