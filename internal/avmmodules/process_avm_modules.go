package avmmodules

import (
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

func isStatusAllowed(status string) bool {
	for _, allowed := range config.AllowedStatuses {
		if status == allowed {
			return true
		}
	}
	return false
}

func isModuleOverride(moduleName string) bool {
	for _, override := range config.OverrideModuleNames {
		if moduleName == override {
			return true
		}
	}
	return false
}

func (p *ModuleProcessor) ProcessResourceModules(processFunc func(ResourceModulesStruct)) error {
	// Filter modules by allowed statuses or override list
	filteredModules := []ResourceModulesStruct{}
	for _, module := range p.Modules.ResourceModules {
		if isStatusAllowed(module.ModuleStatus) {
			filteredModules = append(filteredModules, module)
		} else if isModuleOverride(module.ModuleName) {
			p.Logger.Info("Module included via override list",
				zap.String("module", module.ModuleName),
				zap.String("status", module.ModuleStatus))
			filteredModules = append(filteredModules, module)
		} else {
			p.Logger.Info("Module filtered out due to status",
				zap.String("module", module.ModuleName),
				zap.String("status", module.ModuleStatus))
		}
	}

	batches := batchSlice(filteredModules, config.BatchSize)
	for _, batch := range batches {
		CloneModulesInBatches(batch, config.TempAvmModuleRepoPath, p.Logger, p, resourceNameTransformer)
	}
	for _, module := range filteredModules {
		CommitAndPushModulesToGit(p.Clients, p.Context, p.Project, p.RepoId, module, config.SourceRepoPath, resourceNameTransformer, p.Logger)
		processFunc(module)
	}
	return nil
}

func (p *ModuleProcessor) ProcessPatternModules(processFunc func(PatternModulesStruct)) error {
	// Filter modules by allowed statuses or override list
	filteredModules := []PatternModulesStruct{}
	for _, module := range p.Modules.PatternModules {
		if isStatusAllowed(module.ModuleStatus) {
			filteredModules = append(filteredModules, module)
		} else if isModuleOverride(module.ModuleName) {
			p.Logger.Info("Module included via override list",
				zap.String("module", module.ModuleName),
				zap.String("status", module.ModuleStatus))
			filteredModules = append(filteredModules, module)
		} else {
			p.Logger.Info("Module filtered out due to status",
				zap.String("module", module.ModuleName),
				zap.String("status", module.ModuleStatus))
		}
	}

	batches := batchSlice(filteredModules, config.BatchSize)
	for _, batch := range batches {
		CloneModulesInBatches(batch, config.TempAvmModuleRepoPath, p.Logger, p, patternNameTransformer)
	}
	for _, module := range filteredModules {
		CommitAndPushModulesToGit(p.Clients, p.Context, p.Project, p.RepoId, module, config.SourceRepoPath, patternNameTransformer, p.Logger)
		processFunc(module)
	}
	return nil
}

func (p *ModuleProcessor) ProcessUtilityModules(processFunc func(UtilityModulesStruct)) error {
	// Filter modules by allowed statuses or override list
	filteredModules := []UtilityModulesStruct{}
	for _, module := range p.Modules.UtilityModules {
		if isStatusAllowed(module.ModuleStatus) {
			filteredModules = append(filteredModules, module)
		} else if isModuleOverride(module.ModuleName) {
			p.Logger.Info("Module included via override list",
				zap.String("module", module.ModuleName),
				zap.String("status", module.ModuleStatus))
			filteredModules = append(filteredModules, module)
		} else {
			p.Logger.Info("Module filtered out due to status",
				zap.String("module", module.ModuleName),
				zap.String("status", module.ModuleStatus))
		}
	}

	batches := batchSlice(filteredModules, config.BatchSize)
	for _, batch := range batches {
		CloneModulesInBatches(batch, config.TempAvmModuleRepoPath, p.Logger, p, utilityNameTransformer)
	}
	for _, module := range filteredModules {
		CommitAndPushModulesToGit(p.Clients, p.Context, p.Project, p.RepoId, module, config.SourceRepoPath, utilityNameTransformer, p.Logger)
		processFunc(module)
	}

	return nil
}
