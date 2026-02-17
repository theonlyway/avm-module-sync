package avmmodules

import (
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

// isStatusAllowed checks if the given status is in the list of allowed statuses.
// It returns true if the status is allowed, false otherwise.
func isStatusAllowed(status string) bool {
	for _, allowed := range config.AllowedStatuses {
		if status == allowed {
			return true
		}
	}
	return false
}

// isModuleOverride checks if the given module name is in the override list.
// Modules in the override list will be processed regardless of their status.
func isModuleOverride(moduleName string) bool {
	for _, override := range config.AllowedModuleNames {
		if moduleName == override {
			return true
		}
	}
	return false
}

// isModuleExcluded checks if the given module name is in the exclusion list.
// Modules in the exclusion list will not be processed.
func isModuleExcluded(moduleName string) bool {
	for _, excluded := range config.ExcludedModuleNames {
		if moduleName == excluded {
			return true
		}
	}
	return false
}

// ProcessResourceModules filters, clones, and processes resource modules based on their status.
// It applies the given processFunc to each filtered module after cloning and pushing to Git.
// Modules are filtered by allowed statuses or included via the override list.
func (p *ModuleProcessor) ProcessResourceModules(processFunc func(ResourceModulesStruct)) error {
	// Filter modules by allowed statuses or override list
	filteredModules := []ResourceModulesStruct{}
	for _, module := range p.Modules.ResourceModules {
		if isModuleExcluded(module.ModuleName) {
			p.Logger.Info("Module excluded via exclusion list",
				zap.String("module", module.ModuleName),
				zap.String("status", module.ModuleStatus))
			continue
		}
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

// ProcessPatternModules filters, clones, and processes pattern modules based on their status.
// It applies the given processFunc to each filtered module after cloning and pushing to Git.
// Modules are filtered by allowed statuses or included via the override list.
func (p *ModuleProcessor) ProcessPatternModules(processFunc func(PatternModulesStruct)) error {
	// Filter modules by allowed statuses or override list
	filteredModules := []PatternModulesStruct{}
	for _, module := range p.Modules.PatternModules {
		if isModuleExcluded(module.ModuleName) {
			p.Logger.Info("Module excluded via exclusion list",
				zap.String("module", module.ModuleName),
				zap.String("status", module.ModuleStatus))
			continue
		}
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

// ProcessUtilityModules filters, clones, and processes utility modules based on their status.
// It applies the given processFunc to each filtered module after cloning and pushing to Git.
// Modules are filtered by allowed statuses or included via the override list.
func (p *ModuleProcessor) ProcessUtilityModules(processFunc func(UtilityModulesStruct)) error {
	// Filter modules by allowed statuses or override list
	filteredModules := []UtilityModulesStruct{}
	for _, module := range p.Modules.UtilityModules {
		if isModuleExcluded(module.ModuleName) {
			p.Logger.Info("Module excluded via exclusion list",
				zap.String("module", module.ModuleName),
				zap.String("status", module.ModuleStatus))
			continue
		}
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
