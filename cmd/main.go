package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/theonlyway/avm-module-sync/internal/avmmodules"
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

func Execute() {

	flag.BoolVar(&config.ProcessResourceModules, "process-resource", true, "Process resource modules")
	flag.BoolVar(&config.ProcessPatternModules, "process-pattern", false, "Process pattern modules")
	flag.BoolVar(&config.ProcessUtilityModules, "process-utility", false, "Process utility modules")
	flag.BoolVar(&config.CleanTempModulesDir, "clean-temp", false, "Clean temporary modules directory before processing")
	flag.StringVar(&config.AdoOrganization, "ado-organization", "", "The ADO organization")
	flag.StringVar(&config.AdoProject, "ado-project", "", "The ADO project")
	flag.StringVar(&config.AdoRepo, "ado-repo", "", "The ADO repository")
	flag.StringVar(&config.AdoPat, "ado-pat", "", "The ADO personal access token")

	flag.BoolVar(&config.DebugMode, "debug", false, "Enable debug mode")

	logger, _ := zap.NewProduction()
	sugaredLogger := logger.Sugar()
	defer logger.Sync()
	flag.Parse()
	parseRequiredFlags()

	processor := avmmodules.ModuleProcessor{Logger: logger, SugaredLogger: sugaredLogger}

	organizationUrl := config.AdoOrganizationUrl + config.AdoOrganization

	// Create a connection to your organization
	connection := azuredevops.NewPatConnection(organizationUrl, config.AdoPat)

	ctx := context.Background()

	// Create a client to interact with the Core area
	coreClient, err := core.NewClient(ctx, connection)
	if err != nil {
		logger.Error("Failed to create client", zap.Error(err))
		os.Exit(1)
	}

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

func parseRequiredFlags() {
	if config.AdoOrganization == "" {
		panic("ADO organization is required. Use -ado-organization to specify it.")
	}
	if config.AdoProject == "" {
		panic("ADO project is required. Use -ado-project to specify it.")
	}
	if config.AdoRepo == "" {
		panic("ADO repository is required. Use -ado-repo to specify it.")
	}
	if config.AdoPat == "" {
		panic("ADO personal access token is required. Use -ado-pat to specify it.")
	}
}
