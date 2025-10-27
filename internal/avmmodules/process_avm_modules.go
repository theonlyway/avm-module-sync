package avmmodules

import (
	"github.com/theonlyway/avm-module-sync/internal/config"
)

func (p *ModuleProcessor) ProcessResourceModules(processFunc func(ResourceModulesStruct)) error {
	modules, err := getModules()
	if err != nil {
		return err
	}
	batches := batchSlice(modules.ResourceModules, config.CloneBatchSize)
	for _, batch := range batches {
		CloneModulesInBatches(batch, config.TempAvmModuleRepoPath, p.Logger, processFunc, p, resourceNameTransformer)
	}
	return nil
}

func (p *ModuleProcessor) ProcessPatternModules(processFunc func(PatternModulesStruct)) error {
	modules, err := getModules()
	if err != nil {
		return err
	}
	batches := batchSlice(modules.PatternModules, config.CloneBatchSize)
	for _, batch := range batches {
		CloneModulesInBatches(batch, config.TempAvmModuleRepoPath, p.Logger, processFunc, p, patternNameTransformer)
	}
	return nil
}

func (p *ModuleProcessor) ProcessUtilityModules(processFunc func(UtilityModulesStruct)) error {
	modules, err := getModules()
	if err != nil {
		return err
	}
	batches := batchSlice(modules.UtilityModules, config.CloneBatchSize)
	for _, batch := range batches {
		CloneModulesInBatches(batch, config.TempAvmModuleRepoPath, p.Logger, processFunc, p, utilityNameTransformer)
	}
	return nil
}
