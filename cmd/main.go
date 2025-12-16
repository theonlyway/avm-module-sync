package cmd

import (
	"context"
	"flag"
	"strings"

	"github.com/google/uuid"
	"github.com/theonlyway/avm-module-sync/internal/ado"
	"github.com/theonlyway/avm-module-sync/internal/avmmodules"
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

func maskToken(token string) string {
	if token == "" {
		return "<empty>"
	}
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func logFlags(logger *zap.SugaredLogger) {
	sensitiveFlags := map[string]bool{
		"ado-session-token": true,
		"ado-pat":           true,
	}

	logger.Info("Configuration flags:")
	flag.VisitAll(func(f *flag.Flag) {
		value := f.Value.String()
		if sensitiveFlags[f.Name] && value != "" {
			value = maskToken(value)
		}
		logger.Infof("--%s = %s", f.Name, value)
	})
}

func Main() {
	var logger *zap.Logger
	var sugaredLogger *zap.SugaredLogger

	flag.BoolVar(&config.ProcessResourceModules, "process-resource", true, "Process resource modules")
	flag.BoolVar(&config.ProcessPatternModules, "process-pattern", false, "Process pattern modules")
	flag.BoolVar(&config.ProcessUtilityModules, "process-utility", true, "Process utility modules")
	flag.BoolVar(&config.CleanTempDirs, "cleanup-temp-dirs", false, "Clean temporary directories before processing")
	flag.StringVar(&config.AdoOrganization, "ado-organization", "", "The ADO organization")
	flag.StringVar(&config.AdoProject, "ado-project", "", "The ADO project")
	flag.StringVar(&config.AdoRepoId, "ado-repo-id", "", "The ADO repository ID")
	flag.StringVar(&config.AdoSessionToken, "ado-session-token", "", "The ADO session token. The session token is used for API calls")
	flag.StringVar(&config.AdoPat, "ado-pat", "", "The ADO personal access token. The PAT is used for git operations if the pipeline isn't automatically confgiruing the git credentials")
	flag.StringVar(&config.ModuleSyncAuthorName, "module-sync-author-name", "AVM Module Sync", "The author name for commits")
	flag.StringVar(&config.ModuleSyncAuthorEmail, "module-sync-author-email", "avm-module-sync@example.com", "The author email for commits")
	flag.StringVar(&config.ModuleSyncSourceRepoChildPath, "module-sync-source-repo-child-path", "", "The child path within the source repo where modules are to be copied")
	flag.BoolVar(&config.UseLocalIdentity, "use-local-identity", false, "Use the local identity")
	flag.BoolVar(&config.ReadLocalCsvFile, "read-local-csv", false, "Read module CSV files from local disk instead of downloading")
	flag.BoolVar(&config.PullRemoteTerraformRepository, "pull-remote-repo", true, "Pull the remote Terraform repository to get existing modules")
	flag.StringVar(&config.TempAvmModuleRepoPath, "temp-avm-module-repo-path", "./avm_modules", "The temporary path for the AVM module repository")
	flag.StringVar(&config.SourceRepoPath, "source-repo-path", "", "The path to copy the AVM modules into")
	flag.BoolVar(&config.DebugMode, "debug", false, "Enable debug mode")
	config.AllowedStatuses = []string{"Available"}
	flag.Func("allowed-statuses", "Comma-separated list of allowed module statuses (Available, Proposed, Orphaned, Deprecated, Provisional, Planned)", func(val string) error {
		statuses := strings.Split(val, ",")
		for i := range statuses {
			statuses[i] = strings.TrimSpace(statuses[i])
		}
		config.AllowedStatuses = statuses
		return nil
	})
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
	if config.DebugMode {
		sugaredLogger.Info("Debug mode is enabled")
	}
	logFlags(sugaredLogger)
	logger.Info("Starting AVM module sync")
	ctx := context.Background()
	clients := ado.NewAdoClients(logger, ctx)
	avmmodules.CleanUpTempDirs(logger)

	var repoId uuid.UUID
	if config.AdoRepoId != "" {
		repoId = uuid.MustParse(config.AdoRepoId)
	}

	processor := avmmodules.ModuleProcessor{
		Logger:        logger,
		SugaredLogger: sugaredLogger,
		Clients:       clients,
		Context:       ctx,
		Project:       config.AdoProject,
		RepoId:        &repoId,
	}

	if config.ProcessResourceModules {
		sugaredLogger.Infow("Processing resource modules")
		err := processor.ProcessResourceModules(func(module avmmodules.ResourceModulesStruct) {
			sugaredLogger.Infow(
				"Processed resource module",
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
				"Processed pattern module",
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
				"Processed utility module",
				"module", module.ModuleName,
				"status", module.ModuleStatus,
				"firstPublishedIn", module.FirstPublishedIn,
			)
		})
		if err != nil {
			sugaredLogger.Error("error processing utility modules:", zap.Error(err))
		}
	}

	logger.Info("AVM module sync complete")
}
