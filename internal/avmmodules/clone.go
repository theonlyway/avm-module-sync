package avmmodules

import (
	"os"
	"os/exec"
	"strings"
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

// findTagCommit returns the commit hash that a specific upstream tag points to by running
// git rev-list, which correctly dereferences annotated tags to their target commit.
func findTagCommit(repoPath string, tag string, moduleName string, logger *zap.Logger) string {
	out, err := runGit(repoPath, logger, moduleName, "rev-list", "-n", "1", tag)
	if err != nil || strings.TrimSpace(out) == "" {
		logger.Warn("Could not resolve commit for tag", zap.String("module", moduleName), zap.String("tag", tag))
		return ""
	}
	return strings.TrimSpace(out)
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
					// Check before cloning whether this module is flagged for backfill so we
					// can target the stored tag instead of the latest upstream tag.
					storedTag, _, backfill := readAvmVersionFile(newModuleName, logger)
					CloneRepo(module.GetRepoURL(), tempPath)
					var latestTag, latestCommit string
					if backfill && storedTag != "" {
						logger.Info("Backfill mode: cloning upstream then checking out stored tag",
							zap.String("module", newModuleName),
							zap.String("tag", storedTag),
							zap.String("repoURL", module.GetRepoURL()))
						latestTag = storedTag
						latestCommit = findTagCommit(tempPath, storedTag, newModuleName, logger)
						logger.Info("Backfill mode: resolved tag commit",
							zap.String("module", newModuleName),
							zap.String("tag", storedTag),
							zap.String("commit", latestCommit))
					} else {
						latestTag, latestCommit = findLatestAvmTag(tempPath, logger)
					}
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
