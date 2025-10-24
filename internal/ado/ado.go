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

type AdoClients struct {
	CoreClient core.Client
	GitClient  git.Client
}

func NewAdoClients(logger *zap.Logger, ctx context.Context) *AdoClients {
	var connection *azuredevops.Connection

	organizationUrl := config.AdoOrganizationUrl + config.AdoOrganization

	// Create a connection to your organization
	if config.AdoPat != "" && !config.UseLocalIdentity {
		connection = azuredevops.NewPatConnection(organizationUrl, config.AdoPat)
	} else if config.UseLocalIdentity {
		token, err := getAzureAccessToken(logger, config.AdoEnterpriseAppScope)
		if err != nil {
			logger.Error("Error", zap.Error(err))
			os.Exit(1)
		}
		logger.Debug("Token", zap.String("token", token))
		connection = azuredevops.NewPatConnection(organizationUrl, token)
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
	}
}

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
