package avmmodules

import (
	"os"
	"os/exec"
	"sync"

	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

// GetModules loads and returns all module types (resource, pattern, and utility) from their respective sources.
func GetModules(logger *zap.Logger) (*ModulesStruct, error) {
	logger.Info("Starting to load all module types")
	resourceModules, err := getResourceModules(logger)
	if err != nil {
		logger.Error("Failed to get resource modules", zap.Error(err))
		return nil, err
	}

	patternModules, err := getPatternModules(logger)
	if err != nil {
		logger.Error("Failed to get pattern modules", zap.Error(err))
		return nil, err
	}

	utilityModules, err := getUtilityModules(logger)
	if err != nil {
		logger.Error("Failed to get utility modules", zap.Error(err))
		return nil, err
	}

	logger.Info("Successfully loaded all module types",
		zap.Int("resource_modules", len(resourceModules)),
		zap.Int("pattern_modules", len(patternModules)),
		zap.Int("utility_modules", len(utilityModules)),
		zap.Int("total_modules", len(resourceModules)+len(patternModules)+len(utilityModules)))

	return &ModulesStruct{
		ResourceModules: resourceModules,
		PatternModules:  patternModules,
		UtilityModules:  utilityModules,
	}, nil
}

// batchSlice divides a slice into smaller batches of the specified size.
func batchSlice[T any](items []T, batchSize int) [][]T {
	var batches [][]T
	for batchSize < len(items) {
		items, batches = items[batchSize:], append(batches, items[0:batchSize:batchSize])
	}
	batches = append(batches, items)
	return batches
}

// CloneRepo clones a Git repository from the specified URL to the destination path.
func CloneRepo(repoURL string, destPath string) error {
	args := []string{"clone"}
	if !config.DebugMode {
		args = append(args, "--quiet")
	}
	args = append(args, repoURL, destPath)
	cmd := exec.Command("git", args...)
	if config.DebugMode {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

// checkoutCommit checks out the given commit hash in the cloned repo so the copied module
// content matches the tag recorded in .avm-version. When the hash is empty (no tags were
// found) the default-branch HEAD is left in place.
func checkoutCommit(repoPath string, commitHash string, moduleName string, logger *zap.Logger) {
	if commitHash == "" {
		logger.Warn("No tag commit to checkout, using default branch HEAD", zap.String("module", moduleName), zap.String("path", repoPath))
		return
	}
	if out, err := runGit(repoPath, logger, moduleName, "checkout", "-f", commitHash); err != nil {
		logger.Error("Failed to checkout tag commit, using default branch HEAD", zap.String("module", moduleName), zap.String("commit", commitHash), zap.String("output", out), zap.Error(err))
		return
	}
	logger.Info("Checked out tag commit for module", zap.String("module", moduleName), zap.String("commit", commitHash))
}

// CloneModulesInBatches clones multiple modules in parallel using a worker pool pattern.
// It applies the specified name transformer to each module and removes the .git folder after cloning.
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
				logger.Info("Transformed module name", zap.String("module", newModuleName), zap.String("old", module.GetModuleName()), zap.String("new", newModuleName))

				if _, err := os.Stat(tempPath); err == nil {
					logger.Warn("Temporary repository path exists", zap.String("module", newModuleName), zap.String("path", tempPath))
					// .git already removed in a previous run; latest tag cannot be determined
					processor.LatestAvmTagMap.Store(newModuleName, "")
					processor.LatestAvmCommitMap.Store(newModuleName, "")
					renameFolders(processor, tempPath, newPath, newModuleName)
				} else if os.IsNotExist(err) {
					CloneRepo(module.GetRepoURL(), tempPath)
					latestTag, latestCommit := findLatestAvmTag(tempPath, logger)
					checkoutCommit(tempPath, latestCommit, newModuleName, logger)
					processor.LatestAvmTagMap.Store(newModuleName, latestTag)
					processor.LatestAvmCommitMap.Store(newModuleName, latestCommit)
					removeGitFolder(processor, tempPath, newModuleName)
					renameFolders(processor, tempPath, newPath, newModuleName)
				} else {
					logger.Error("Error checking temporary repository path", zap.String("module", newModuleName), zap.String("path", tempPath), zap.Error(err))
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
