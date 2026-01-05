package avmmodules

import (
	"io"
	"os"
	"sync"

	"github.com/go-git/go-git/v6"
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

func GetModules() (*ModulesStruct, error) {
	resourceModules, err := getResourceModules()
	if err != nil {
		return nil, err
	}

	patternModules, err := getPatternModules()
	if err != nil {
		return nil, err
	}

	utilityModules, err := getUtilityModules()
	if err != nil {
		return nil, err
	}

	return &ModulesStruct{
		ResourceModules: resourceModules,
		PatternModules:  patternModules,
		UtilityModules:  utilityModules,
	}, nil
}

func batchSlice[T any](items []T, batchSize int) [][]T {
	var batches [][]T
	for batchSize < len(items) {
		items, batches = items[batchSize:], append(batches, items[0:batchSize:batchSize])
	}
	batches = append(batches, items)
	return batches
}

func CloneRepo(repoURL string, destPath string) error {
	var progressWriter io.Writer
	if config.DebugMode {
		progressWriter = os.Stdout
	} else {
		progressWriter = nil
	}
	_, err := git.PlainClone(destPath, &git.CloneOptions{
		URL:      repoURL,
		Progress: progressWriter,
	})

	if err != nil {
		return err
	}

	return nil
}

func CloneModulesInBatches[T Module](modules []T, destDir string, logger *zap.Logger, processor *ModuleProcessor, nameTransformer ModuleNameTransformer) {
	var wg sync.WaitGroup
	jobs := make(chan T)

	// Start goroutines to process modules in batches
	for range config.BatchSize {
		// Increment wait group counter
		wg.Add(1)
		go func() {
			defer wg.Done()
			for module := range jobs {
				tempPath := destDir + "/" + module.GetModuleName()
				newModuleName := nameTransformer(module.GetModuleName())
				newPath := destDir + "/" + newModuleName
				logger.Info("Transformed module name", zap.String("old", module.GetModuleName()), zap.String("new", newModuleName))

				if _, err := os.Stat(tempPath); err == nil {
					logger.Warn("Temporary repository path exists", zap.String("path", tempPath))
					removeGitFolder(processor, tempPath)
					renameFolders(processor, tempPath, newPath)
				} else if os.IsNotExist(err) {
					CloneRepo(module.GetRepoURL(), tempPath)
					removeGitFolder(processor, tempPath)
					renameFolders(processor, tempPath, newPath)
				} else {
					logger.Error("Error checking temporary repository path", zap.String("path", tempPath), zap.Error(err))
				}
			}
		}()
	}

	// Send modules to jobs channel
	for _, module := range modules {
		jobs <- module
	}

	// Close jobs channel and wait for goroutines to finish
	close(jobs)
	// Wait for goroutines to finish
	wg.Wait()
}
