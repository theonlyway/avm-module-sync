package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/theonlyway/avm-module-sync/internal/avmmodules"
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

func Main() {

	flag.BoolVar(&config.ProcessResourceModules, "process-resource", true, "Process resource modules")
	flag.BoolVar(&config.ProcessPatternModules, "process-pattern", false, "Process pattern modules")
	flag.BoolVar(&config.ProcessUtilityModules, "process-utility", false, "Process utility modules")
	flag.BoolVar(&config.CleanTempModulesDir, "clean-temp", false, "Clean temporary modules directory before processing")
	flag.StringVar(&config.AdoOrganization, "ado-organization", "", "The ADO organization")
	flag.StringVar(&config.AdoProject, "ado-project", "", "The ADO project")
	flag.StringVar(&config.AdoRepo, "ado-repo", "", "The ADO repository")
	flag.StringVar(&config.AdoPat, "ado-pat", "", "The ADO personal access token")
	flag.BoolVar(&config.UseLocalIdentity, "use-local-identity", false, "Use the local identity")
	flag.BoolVar(&config.DebugMode, "debug", true, "Enable debug mode")
	flag.Parse()
	var logger *zap.Logger
	var sugaredLogger *zap.SugaredLogger

	if config.DebugMode {
		logger, _ = zap.NewDevelopment()
		sugaredLogger = logger.Sugar()
		defer logger.Sync()

	} else {
		logger, _ = zap.NewProduction()
		sugaredLogger = logger.Sugar()
		defer logger.Sync()
	}

	flag.VisitAll(func(f *flag.Flag) {
		logger.Debug("Flag value", zap.String("name", f.Name), zap.String("value", f.Value.String()))
	})

	processor := avmmodules.ModuleProcessor{Logger: logger, SugaredLogger: sugaredLogger}

	ctx := context.Background()
	coreClient := adoClient(logger, ctx)

	coreClient.GetProjects(ctx, core.GetProjectsArgs{})

	if config.ProcessResourceModules {
		sugaredLogger.Infow("resource modules:")
		err := processor.ProcessResourceModules(func(module avmmodules.ResourceModulesStruct) {
			sugaredLogger.Infow(
				"processing resource module",
				"module", module.ModuleName,
				"status", module.ModuleStatus,
				"firstPublishedIn", module.FirstPublishedIn,
			)
		})
		if err != nil {
			logger.Error("error processing resource modules:", zap.Error(err))
		}
	}

	if config.ProcessPatternModules {
		fmt.Println("Pattern Modules:")
		err := avmmodules.ProcessPatternModules(func(module avmmodules.PatternModulesStruct) {
			fmt.Printf("Module: %s, Status: %s, First Published In: %s\n", module.ModuleName, module.ModuleStatus, module.FirstPublishedIn)
		})
		if err != nil {
			fmt.Println("Error processing pattern modules:", err)
		}
	}

	if config.ProcessUtilityModules {
		fmt.Println("Utility Modules:")
		err := avmmodules.ProcessUtilityModules(func(module avmmodules.UtilityModulesStruct) {
			fmt.Printf("Module: %s, Status: %s, First Published In: %s\n", module.ModuleName, module.ModuleStatus, module.FirstPublishedIn)
		})
		if err != nil {
			fmt.Println("Error processing utility modules:", err)
		}
	}
}

func adoClient(logger *zap.Logger, ctx context.Context) core.Client {
	var connection *azuredevops.Connection

	organizationUrl := config.AdoOrganizationUrl + config.AdoOrganization

	// Create a connection to your organization
	if config.AdoPat != "" && !config.UseLocalIdentity {
		connection = azuredevops.NewPatConnection(organizationUrl, config.AdoPat)
	} else if config.UseLocalIdentity {
		token, err := getAzureAccessToken(logger, config.AdoEnterpriseAppScope)
		if err != nil {
			logger.Error("Error", zap.Error(err))
		}
		logger.Debug("Token", zap.String("token", token))
	}

	// Create a client to interact with the Core area
	coreClient, err := core.NewClient(ctx, connection)
	if err != nil {
		logger.Error("Failed to create client", zap.Error(err))
		os.Exit(1)
	}

	return coreClient
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
