package zitadel

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/zitadel/oidc/v2/pkg/oidc"
	"go.uber.org/zap"

	"github.com/zitadel/zitadel-go/v3/pkg/client/management"
	"github.com/zitadel/zitadel-go/v3/pkg/client/middleware"

	// userv2 "github.com/zitadel/zitadel-go/v3/pkg/client/user/v2"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/action"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/app"
	managementpb "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/project"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	logger  *zap.Logger
	issuer  = cmp.Or(os.Getenv("ZITADEL_ISSUER"), "http://iam.127-0-0-1.sslip.io")
	api     = cmp.Or(os.Getenv("ZITADEL_API"), "iam.127-0-0-1.sslip.io:80")
	keyPath = cmp.Or(os.Getenv("ZITADEL_KEY_PATH"), "__zitadel-machinekey/zitadel-admin-sa.json")
)

const (
	EDGE_PROJECT_NAME       = "edge"
	OIDC_CLIENT_EDGE        = "edge-ui"
	OIDC_CLIENT_OAUTH2PROXY = "oauth2-proxy"
	OIDC_CLIENT_MINIO       = "minio"
	OIDC_CLIENT_S3          = "seaweedfs"
)

func init() {
	flag.Parse()

	// Initialize the zap logger
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()
}

func Configure() {
	logger.Info("ZITADEL configuration",
		zap.String("issuer", issuer),
		zap.String("api", api),
		zap.String("keyPath", keyPath))

	client, err := createZitadelClient(issuer, api)
	if err != nil {
		logger.Fatal("failed to create ZITADEL client", zap.Error(err))
	}
	defer func() {
		_ = client.Connection.Close()
	}()

	ctx := context.Background()

	createdProject, err := ensureProject(ctx, client)
	if err != nil {
		logger.Fatal("failed to ensure project", zap.Error(err))
	}
	logger.Info("ensured project", zap.Any("project", createdProject))

	createdConsoleApp, err := ensureConsoleApp(ctx, client, createdProject.Id)
	if err != nil {
		logger.Fatal("failed to ensure OIDC app", zap.Error(err))
	}
	logger.Info("ensured OIDC app", zap.Any("app", createdConsoleApp))

	actions := ensurePolicyActions(ctx, client)
	if actions == nil {
		logger.Fatal("failed to ensure actions", zap.Error(err))
	}
	logger.Info("ensured actions", zap.Any("actions", actions))

	oauth2ProxyApp, err := ensureOAuth2ProxyApp(ctx, client, createdProject.Id)
	if err != nil {
		logger.Fatal("failed to ensure OAuth2 Proxy app", zap.Error(err))
	}
	logger.Info("ensured OAuth2 Proxy app", zap.Any("app", oauth2ProxyApp))

	minioClientApp, err := ensureMinioClientApp(ctx, client, createdProject.Id)
	if err != nil {
		logger.Fatal("failed to ensure MinIO client app", zap.Error(err))
	}
	logger.Info("ensured MinIO client app", zap.Any("app", minioClientApp))
}

func createZitadelClient(issuer, api string) (*management.Client, error) {
	ctx := context.Background()
	client, err := management.NewClient(
		ctx,
		issuer,
		api,
		[]string{oidc.ScopeOpenID, zitadel.ScopeZitadelAPI()},
		zitadel.WithJWTProfileTokenSource(middleware.JWTProfileFromPath(ctx, keyPath)),
		zitadel.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	/*
		// load self-signed certificate
		caCert, err := os.ReadFile("gw.tls.crt")
		if err != nil {
			return nil, fmt.Errorf("error reading CA certificate: %v", err)
		}

		// create a certificate pool and add the cert
		certPool := x509.NewCertPool()
		if ok := certPool.AppendCertsFromPEM(caCert); !ok {
			return nil, fmt.Errorf("failed to append certificate to pool")
		}

		// create TLS credentials with the custom cert pool
		creds := credentials.NewTLS(&tls.Config{
			RootCAs: certPool,
		})

		// create the client with custom dial options including the TLS credentials
		client, err = management.NewClient(
			ctx,
			issuer,
			api,
			[]string{oidc.ScopeOpenID, zitadel.ScopeZitadelAPI()},
			zitadel.WithJWTProfileTokenSource(middleware.JWTProfileFromPath(ctx, keyPath)),
			zitadel.WithDialOptions(
				grpc.WithTransportCredentials(creds),
			),
		)
	*/

	return client, nil
}

func listProjects(ctx context.Context, client *management.Client) ([]*project.Project, error) {
	projects, err := client.ListProjects(ctx, &managementpb.ListProjectsRequest{})
	if err != nil {
		return nil, err
	}
	return projects.Result, nil
}

func ensureProject(ctx context.Context, client *management.Client) (*project.Project, error) {
	// Check if project already exists
	projects, err := listProjects(ctx, client)
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		if project.Name == EDGE_PROJECT_NAME {
			return project, nil
		}
	}

	// Create project if it doesn't exist
	createdProject, err := client.AddProject(ctx, &managementpb.AddProjectRequest{
		Name:                   EDGE_PROJECT_NAME,
		ProjectRoleAssertion:   true,
		ProjectRoleCheck:       false,
		HasProjectCheck:        false,
		PrivateLabelingSetting: *project.PrivateLabelingSetting_PRIVATE_LABELING_SETTING_UNSPECIFIED.Enum(),
	})
	if err != nil {
		return nil, err
	}

	proj, err := client.GetProjectByID(ctx, &managementpb.GetProjectByIDRequest{
		Id: createdProject.Id,
	})
	if err != nil {
		return nil, err
	}

	return proj.Project, nil
}

func listApps(ctx context.Context, client *management.Client, projectID string) ([]*app.App, error) {
	apps, err := client.ListApps(ctx, &managementpb.ListAppsRequest{
		ProjectId: projectID,
	})
	if err != nil {
		return nil, err
	}
	return apps.Result, nil
}

func ensureConsoleApp(ctx context.Context, client *management.Client, projectID string) (*app.App, error) {
	apps, err := listApps(ctx, client, projectID)
	if err != nil {
		return nil, err
	}

	for _, app := range apps {
		if app.Name == OIDC_CLIENT_EDGE {
			return app, nil
		}
	}

	// Create OIDC app
	createdOIDCApp, err := client.AddOIDCApp(ctx, &managementpb.AddOIDCAppRequest{
		ProjectId: projectID,
		Name:      OIDC_CLIENT_EDGE,
		RedirectUris: []string{
			"http://localhost:4200/signin/callback",
		},
		ResponseTypes: []app.OIDCResponseType{},
		GrantTypes: []app.OIDCGrantType{
			app.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE,
			app.OIDCGrantType_OIDC_GRANT_TYPE_REFRESH_TOKEN,
		},
		AppType:        app.OIDCAppType_OIDC_APP_TYPE_USER_AGENT,
		AuthMethodType: app.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_NONE,
		PostLogoutRedirectUris: []string{
			"http://localhost:4200/signout/callback",
		},
		Version:                  0,
		DevMode:                  false,
		AccessTokenType:          app.OIDCTokenType_OIDC_TOKEN_TYPE_JWT,
		AccessTokenRoleAssertion: true,
		IdTokenRoleAssertion:     true,
		IdTokenUserinfoAssertion: true,
		ClockSkew:                &durationpb.Duration{Seconds: 1},
		AdditionalOrigins: []string{
			"scheme://localhost:8080",
		},
		SkipNativeAppSuccessPage: true,
	})
	if err != nil {
		return nil, err
	}

	createdApp, err := client.GetAppByID(ctx, &managementpb.GetAppByIDRequest{
		ProjectId: projectID,
		AppId:     createdOIDCApp.AppId,
	})
	if err != nil {
		return nil, err
	}

	fmt.Println("createdApp", createdApp.App)

	return createdApp.App, nil
}

func ensureOAuth2ProxyApp(ctx context.Context, client *management.Client, projectID string) (*app.App, error) {
	apps, err := listApps(ctx, client, projectID)
	if err != nil {
		return nil, err
	}

	for _, app := range apps {
		if app.Name == OIDC_CLIENT_OAUTH2PROXY {
			return app, nil
		}
	}

	// Create OIDC app for OAuth2 Proxy
	createdOAuth2ProxyApp, err := client.AddOIDCApp(ctx, &managementpb.AddOIDCAppRequest{
		ProjectId: projectID,
		Name:      OIDC_CLIENT_OAUTH2PROXY,
		RedirectUris: []string{
			"http://localhost:4200/signin/callback",
		},
		ResponseTypes:            []app.OIDCResponseType{app.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE},
		GrantTypes:               []app.OIDCGrantType{app.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE, app.OIDCGrantType_OIDC_GRANT_TYPE_REFRESH_TOKEN, app.OIDCGrantType_OIDC_GRANT_TYPE_TOKEN_EXCHANGE},
		AppType:                  app.OIDCAppType_OIDC_APP_TYPE_WEB,
		AuthMethodType:           app.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC,
		PostLogoutRedirectUris:   []string{"http://localhost:4200/signout"},
		Version:                  0,
		DevMode:                  false,
		AccessTokenType:          app.OIDCTokenType_OIDC_TOKEN_TYPE_BEARER,
		AccessTokenRoleAssertion: true,
		IdTokenRoleAssertion:     true,
		IdTokenUserinfoAssertion: true,
		ClockSkew:                &durationpb.Duration{Seconds: 5},
		AdditionalOrigins:        []string{"scheme://localhost:8080"},
		SkipNativeAppSuccessPage: true,
	})
	if err != nil {
		return nil, err
	}

	createdApp, err := client.GetAppByID(ctx, &managementpb.GetAppByIDRequest{
		ProjectId: projectID,
		AppId:     createdOAuth2ProxyApp.AppId,
	})
	if err != nil {
		return nil, err
	}

	return createdApp.App, nil
}

func listActions(ctx context.Context, client *management.Client) ([]*action.Action, error) {
	ations, err := client.ListActions(ctx, &managementpb.ListActionsRequest{})
	if err != nil {
		return nil, err
	}
	return ations.Result, nil
}

// ensurePolicyActions ensures that the setPolicies and setMinioPolicy actions exist
func ensurePolicyActions(ctx context.Context, client *management.Client) []*action.Action {
	// Check if the setPolicies and setMinioPolicy actions already exist
	actions, err := listActions(ctx, client)
	if err != nil {
		logger.Fatal("failed to list actions", zap.Error(err))
	}

	setPoliciesExists := false
	setMinioPolicyExists := false

	for _, action := range actions {
		if action.Name == "setPolicies" {
			setPoliciesExists = true
		}
		if action.Name == "setMinioPolicy" {
			setMinioPolicyExists = true
		}
	}

	if setPoliciesExists && setMinioPolicyExists {
		// Both actions exist, return the current list of actions
		return actions
	}

	var actionIds []string

	// Create setPolicies action if it doesn't exist
	if !setPoliciesExists {
		setPoliciesAction, err := createActionSetPolicies(ctx, client)
		if err != nil {
			logger.Fatal("failed to create setPolicies action", zap.Error(err))
		}
		actionIds = append(actionIds, setPoliciesAction.Id)
		actions = append(actions, setPoliciesAction)
	}

	// Create setMinioPolicy action if it doesn't exist
	if !setMinioPolicyExists {
		setMinioPolicyAction, err := createActionSetMinioPolicy(ctx, client)
		if err != nil {
			logger.Fatal("failed to create setMinioPolicy action", zap.Error(err))
		}
		actionIds = append(actionIds, setMinioPolicyAction.Id)
		actions = append(actions, setMinioPolicyAction)
	}

	// Set triggers for all created actions
	if len(actionIds) > 0 {
		if err := setTriggers(ctx, client, actionIds); err != nil {
			logger.Fatal("failed to set triggers for actions", zap.Error(err))
		}
	}

	return actions
}

// createActionSetMinioPolicy creates the setMinioPolicy action
func createActionSetMinioPolicy(ctx context.Context, client *management.Client) (*action.Action, error) {
	createdAction, err := client.CreateAction(ctx, &managementpb.CreateActionRequest{
		Name: "setMinioPolicy",
		Script: `function setMinioPolicy(ctx, api) {  
  api.v1.claims.setClaim('policy_minio', "readonly")
}
`,
		Timeout:       &durationpb.Duration{Seconds: 10},
		AllowedToFail: false,
	})
	if err != nil {
		return nil, err
	}

	retrievedAction, err := client.GetAction(ctx, &managementpb.GetActionRequest{
		Id: createdAction.Id,
	})
	if err != nil {
		return nil, err
	}

	return retrievedAction.Action, nil
}

// createActionSetPolicies creates the setPolicies action
func createActionSetPolicies(ctx context.Context, client *management.Client) (*action.Action, error) {
	createdAction, err := client.CreateAction(ctx, &managementpb.CreateActionRequest{
		Name: "setPolicies",
		Script: `function setPolicies(ctx, api) {
  policy = {
    'pgrole': 'authn',
	'postgres': 'authn',
    'mqtt': '',
    'minio': 'readwrite'
  }
  
  api.v1.claims.setClaim('policy', policy)
}
`,
		Timeout:       &durationpb.Duration{Seconds: 10},
		AllowedToFail: false,
	})
	if err != nil {
		return nil, err
	}

	retrievedAction, err := client.GetAction(ctx, &managementpb.GetActionRequest{
		Id: createdAction.Id,
	})
	if err != nil {
		return nil, err
	}

	return retrievedAction.Action, nil
}

// setTriggers sets triggers for the specified action
func setTriggers(ctx context.Context, client *management.Client, actionIds []string) error {
	// Implement your logic to set triggers
	_, err := client.SetTriggerActions(ctx, &managementpb.SetTriggerActionsRequest{
		FlowType:    "2",
		TriggerType: "4",
		ActionIds:   actionIds,
	})
	if err != nil {
		return err
	}

	// accessTokenTrigger
	_, err = client.SetTriggerActions(ctx, &managementpb.SetTriggerActionsRequest{
		FlowType:    "2",
		TriggerType: "5",
		ActionIds:   actionIds,
	})
	if err != nil {
		return err
	}
	return nil
}

func ensureMinioClientApp(ctx context.Context, client *management.Client, projectID string) (*app.App, error) {
	apps, err := listApps(ctx, client, projectID)
	if err != nil {
		return nil, err
	}

	for _, app := range apps {
		if app.Name == OIDC_CLIENT_MINIO {
			return app, nil
		}
	}

	// Create OIDC app for MinIO
	createdMinioApp, err := client.AddOIDCApp(ctx, &managementpb.AddOIDCAppRequest{
		ProjectId: projectID,
		Name:      OIDC_CLIENT_MINIO,
		RedirectUris: []string{
			// TODO: don't hardcode
			"http://127.0.0.1:9001/oauth_callback",
			"http://minio.127-0-0-1.sslip.io/oauth_callback",
		},
		ResponseTypes: []app.OIDCResponseType{app.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE},
		GrantTypes: []app.OIDCGrantType{
			app.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE,
			app.OIDCGrantType_OIDC_GRANT_TYPE_REFRESH_TOKEN,
		},
		AppType: app.OIDCAppType_OIDC_APP_TYPE_WEB,

		AuthMethodType: app.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC,
	})
	if err != nil {
		return nil, err
	}

	createdApp, err := client.GetAppByID(ctx, &managementpb.GetAppByIDRequest{
		ProjectId: projectID,
		AppId:     createdMinioApp.AppId,
	})
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return createdApp.App, nil
}

/*
// setPasswordChangeRequired sets ChangeRequired=false for the admin user
func setPasswordChangeRequired(ctx context.Context, client *userv2.Client) (*user.SetPasswordResponse, error) {
	res, err := client.SetPassword(context.Background(), &user.SetPasswordRequest{
		UserId: "",
		NewPassword: &user.Password{
			// Password:       "",
			ChangeRequired: false,
		},
	})
	if err != nil {
		return &user.SetPasswordResponse{}, err
	}

	return res, nil
}
*/
