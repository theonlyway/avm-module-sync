package cmd

import (
	"context"
	"flag"
	"os"

	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"github.com/theonlyway/avm-module-sync/internal/ado"
	"github.com/theonlyway/avm-module-sync/internal/avmmodules"
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

func Main() {
	var logger *zap.Logger
	var sugaredLogger *zap.SugaredLogger

	flag.BoolVar(&config.ProcessResourceModules, "process-resource", true, "Process resource modules")
	flag.BoolVar(&config.ProcessPatternModules, "process-pattern", true, "Process pattern modules")
	flag.BoolVar(&config.ProcessUtilityModules, "process-utility", true, "Process utility modules")
	flag.BoolVar(&config.CleanTempModulesDir, "clean-temp", true, "Clean temporary modules directory before processing")
	flag.StringVar(&config.AdoOrganization, "ado-organization", "", "The ADO organization")
	flag.StringVar(&config.AdoProject, "ado-project", "", "The ADO project")
	flag.StringVar(&config.AdoRepo, "ado-repo", "", "The ADO repository")
	flag.StringVar(&config.AdoPat, "ado-pat", "", "The ADO personal access token")
	flag.BoolVar(&config.UseLocalIdentity, "use-local-identity", false, "Use the local identity")
	flag.BoolVar(&config.ReadLocalCsvFile, "read-local-csv", true, "Read module CSV files from local disk instead of downloading")
	flag.BoolVar(&config.PullRemoteTerraformRepository, "pull-remote-repo", true, "Pull the remote Terraform repository to get existing modules")
	flag.BoolVar(&config.DebugMode, "debug", false, "Enable debug mode")
	flag.Parse()

	if config.DebugMode {
		logger, _ = zap.NewDevelopment()
		sugaredLogger = logger.Sugar()
		defer logger.Sync()

	} else {
		logger, _ = zap.NewProduction()
		sugaredLogger = logger.Sugar()
		defer logger.Sync()
	}

	processor := avmmodules.ModuleProcessor{Logger: logger, SugaredLogger: sugaredLogger}

	processor.CleanupTempRepos()

	ctx := context.Background()
	clients := ado.NewAdoClients(logger, ctx)

	projectValue, err := clients.CoreClient.GetProject(ctx, core.GetProjectArgs{
		ProjectId: &config.AdoProject,
	})
	if err != nil {
		logger.Error("Failed to get project", zap.Error(err))
		os.Exit(1)
	}
	var webURL string
	if links, ok := projectValue.Links.(map[string]interface{}); ok {
		if web, ok := links["web"].(map[string]interface{}); ok {
			if href, ok := web["href"].(string); ok {
				webURL = href
			}
		}
	}
	logger.Debug("Project", zap.Any("response", projectValue))
	logger.Info("Looked up project", zap.String("project", *projectValue.Name), zap.Any("id", *projectValue.Id), zap.String("url", webURL))

	repoValue, err := clients.GitClient.GetRepository(ctx, git.GetRepositoryArgs{
		RepositoryId: &config.AdoRepo,
		Project:      projectValue.Name,
	})
	if err != nil {
		logger.Error("Failed to get repository", zap.Error(err))
		os.Exit(1)
	}
	logger.Debug("Repository", zap.Any("response", repoValue))
	logger.Info("Looked up repository", zap.String("repo", *repoValue.Name), zap.Any("id", *repoValue.Id), zap.String("url", *repoValue.WebUrl))

	if config.ProcessResourceModules {
		sugaredLogger.Infow("Processing resource modules")
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
		sugaredLogger.Infow("Processing pattern modules")
		err := processor.ProcessPatternModules(func(module avmmodules.PatternModulesStruct) {
			sugaredLogger.Infow(
				"processing pattern module",
				"module", module.ModuleName,
				"status", module.ModuleStatus,
				"firstPublishedIn", module.FirstPublishedIn,
			)
		})
		if err != nil {
			sugaredLogger.Error("error processing pattern modules:", zap.Error(err))
		}
	}

	if config.ProcessUtilityModules {
		sugaredLogger.Infow("Processing utility modules")
		err := processor.ProcessUtilityModules(func(module avmmodules.UtilityModulesStruct) {
			sugaredLogger.Infow(
				"processing utility module",
				"module", module.ModuleName,
				"status", module.ModuleStatus,
				"firstPublishedIn", module.FirstPublishedIn,
			)
		})
		if err != nil {
			sugaredLogger.Error("error processing utility modules:", zap.Error(err))
		}
	}
}

func validateRequiredFlags(logger *zap.Logger) {
	var missingFlags []string
	flag.VisitAll(func(f *flag.Flag) {
		if f.Value.String() == "" {
			missingFlags = append(missingFlags, f.Name)
		}
	})

	if len(missingFlags) > 0 {
		for _, flagName := range missingFlags {
			logger.Error("Missing required flag", zap.String("flag", flagName))
		}
		os.Exit(1)
	}
}
