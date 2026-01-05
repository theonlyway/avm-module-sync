package ado

import (
	"context"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

// AdoClients holds the Azure DevOps API clients and authentication token.
type AdoClients struct {
	CoreClient core.Client
	GitClient  git.Client
	Token      string
}

// TokenResponse represents the structure for Azure access token responses.
type TokenResponse struct {
	Token string `json:"token"`
}

// NewAdoClients creates and initializes a new AdoClients instance with authentication.
// It supports both session token and local Azure identity authentication methods.
func NewAdoClients(logger *zap.Logger, ctx context.Context) *AdoClients {
	var connection *azuredevops.Connection
	var token string

	organizationUrl := config.AdoOrganizationUrl + config.AdoOrganization

	// Create a connection to your organization
	if config.AdoSessionToken != "" && !config.UseLocalIdentity {
		logger.Info("Creating ADO client with session token")
		connection = azuredevops.NewPatConnection(organizationUrl, config.AdoSessionToken)
	} else if config.UseLocalIdentity {
		logger.Info("Using local identity")
		token, err := getAzureAccessToken(logger, config.AdoEnterpriseAppScope)
		if err != nil {
			logger.Error("Error", zap.Error(err))
			os.Exit(1)
		}
		logger.Debug("Token", zap.String("token", token))
		connection = azuredevops.NewPatConnection(organizationUrl, token)
	} else {
		logger.Error("Unknown auth mechanism")
		panic("Unknown auth mechanism")
	}

	// Ensure connection is initialized
	if connection == nil {
		logger.Error("ADO connection was not initialized. Please provide either a PAT or enable local identity.")
		os.Exit(1)
	}

	coreClient, err := core.NewClient(ctx, connection)
	if err != nil {
		logger.Error("Failed to create client", zap.Error(err))
		os.Exit(1)
	}
	gitClient, err := git.NewClient(ctx, connection)
	if err != nil {
		logger.Error("Failed to create git client", zap.Error(err))
		os.Exit(1)
	}

	return &AdoClients{
		CoreClient: coreClient,
		GitClient:  gitClient,
		Token:      token,
	}
}

// getAzureAccessToken retrieves an Azure access token using the default Azure credential chain.
func getAzureAccessToken(logger *zap.Logger, resource string) (string, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error("Error", zap.Error(err))
		os.Exit(1)
	}

	token, err := cred.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{resource},
	})
	if err != nil {
		logger.Error("Error", zap.Error(err))
		os.Exit(1)
	}
	return token.Token, nil
}
