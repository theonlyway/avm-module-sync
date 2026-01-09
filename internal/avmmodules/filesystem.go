package avmmodules

import (
	"os"

	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

// removeGitFolder removes the .git directory from a cloned repository.
func removeGitFolder(p *ModuleProcessor, path string, moduleName string) {
	p.Logger.Info("Removing .git folder from", zap.String("module", moduleName), zap.String("path", path))
	gitPath := path + "/.git"
	os.RemoveAll(gitPath)
}

// renameFolders renames a folder from oldPath to newPath, removing the newPath if it already exists.
func renameFolders(p *ModuleProcessor, oldPath string, newPath string, moduleName string) {
	if oldPath == newPath {
		return
	}
	if _, err := os.Stat(newPath); err == nil {
		p.Logger.Warn("New path already exists, removing", zap.String("module", moduleName), zap.String("path", newPath))
		os.RemoveAll(newPath)
	}
	p.Logger.Info("Renaming folder", zap.String("module", moduleName), zap.String("old", oldPath), zap.String("new", newPath))
	err := os.Rename(oldPath, newPath)
	if err != nil {
		p.Logger.Error("Error renaming folder", zap.String("module", moduleName), zap.String("old", oldPath), zap.String("new", newPath), zap.Error(err))
	}
}

// CleanUpTempDirs removes temporary directories used during module processing if cleanup is enabled.
func CleanUpTempDirs(logger *zap.Logger) {
	if !config.CleanTempDirs {
		return
	}
	logger.Info("Cleaning up temporary directories")
	os.RemoveAll(config.TempAvmModuleRepoPath)
	os.RemoveAll(config.SourceRepoPath)
}
