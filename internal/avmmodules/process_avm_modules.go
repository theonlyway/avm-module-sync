package avmmodules

import (
	"github.com/theonlyway/avm-module-sync/internal/config"
)

func (p *ModuleProcessor) ProcessResourceModules(processFunc func(ResourceModulesStruct)) error {
	modules, err := getModules()
	if err != nil {
		return err
	}
	batches := batchSlice(modules.ResourceModules, config.BatchSize)
	for _, batch := range batches {
		CloneModulesInBatches(batch, config.TempAvmModuleRepoPath, p.Logger, p, resourceNameTransformer)
	}
	for _, module := range modules.ResourceModules {
		CommitAndPushModulesToGit(module, config.TempSourceRepoPath, resourceNameTransformer, p.Logger)
		processFunc(module)
	}
	return nil
}

func (p *ModuleProcessor) ProcessPatternModules(processFunc func(PatternModulesStruct)) error {
	modules, err := getModules()
	if err != nil {
		return err
	}
	batches := batchSlice(modules.PatternModules, config.BatchSize)
	for _, batch := range batches {
		CloneModulesInBatches(batch, config.TempAvmModuleRepoPath, p.Logger, p, patternNameTransformer)
	}
	for _, module := range modules.PatternModules {
		CommitAndPushModulesToGit(module, config.TempSourceRepoPath, resourceNameTransformer, p.Logger)
		processFunc(module)
	}
	return nil
}

func (p *ModuleProcessor) ProcessUtilityModules(processFunc func(UtilityModulesStruct)) error {
	modules, err := getModules()
	if err != nil {
		return err
	}
	batches := batchSlice(modules.UtilityModules, config.BatchSize)
	for _, batch := range batches {
		CloneModulesInBatches(batch, config.TempAvmModuleRepoPath, p.Logger, p, utilityNameTransformer)
	}
	for _, module := range modules.UtilityModules {
		CommitAndPushModulesToGit(module, config.TempSourceRepoPath, utilityNameTransformer, p.Logger)
		processFunc(module)
	}

	return nil
}
