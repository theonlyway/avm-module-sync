package avmmodules

import (
	"os"
	"path/filepath"
	"strings"

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

// moduleVersionFilePath returns the absolute path of the .avm-version file for a module
// inside the ADO source repository.
func moduleVersionFilePath(moduleName string) string {
	if config.ModuleSyncSourceRepoChildPath != "" {
		return filepath.Join(config.SourceRepoPath, config.ModuleSyncSourceRepoChildPath, moduleName, config.AvmVersionFileName)
	}
	return filepath.Join(config.SourceRepoPath, moduleName, config.AvmVersionFileName)
}

// parseAvmVersionTag extracts the tag value from the contents of a .avm-version file.
// It supports the current "tag=...\ncommit=..." format and falls back to treating the
// entire trimmed content as the tag for older files that stored only the bare tag.
func parseAvmVersionTag(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if tag, ok := strings.CutPrefix(line, "tag="); ok {
			return strings.TrimSpace(tag)
		}
	}
	return strings.TrimSpace(content)
}

// readAvmVersionFile reads the last-synced AVM tag from the module's version file.
// Returns an empty string if the file does not exist or cannot be read.
func readAvmVersionFile(moduleName string, logger *zap.Logger) string {
	path := moduleVersionFilePath(moduleName)
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warn("Could not read AVM version file", zap.String("module", moduleName), zap.String("path", path), zap.Error(err))
		}
		return ""
	}
	tag := parseAvmVersionTag(string(data))
	logger.Info("Read last synced AVM tag from version file", zap.String("module", moduleName), zap.String("tag", tag))
	return tag
}

// writeAvmVersionFile writes the latest AVM tag and the commit it points to to the module's
// version file so subsequent runs know which tag was last synced and a downstream pipeline
// can package the module from that exact commit.
func writeAvmVersionFile(moduleName string, localRepoPath string, latestAvmTag string, latestAvmCommit string, logger *zap.Logger) {
	if latestAvmTag == "" {
		logger.Warn("No AVM tag available to write to version file, skipping", zap.String("module", moduleName))
		return
	}
	var versionFilePath string
	if config.ModuleSyncSourceRepoChildPath != "" {
		versionFilePath = filepath.Join(localRepoPath, config.ModuleSyncSourceRepoChildPath, moduleName, config.AvmVersionFileName)
	} else {
		versionFilePath = filepath.Join(localRepoPath, moduleName, config.AvmVersionFileName)
	}
	content := "tag=" + latestAvmTag + "\ncommit=" + latestAvmCommit + "\n"
	err := os.WriteFile(versionFilePath, []byte(content), 0644)
	if err != nil {
		logger.Error("Failed to write AVM version file", zap.String("module", moduleName), zap.String("path", versionFilePath), zap.Error(err))
		return
	}
	logger.Info("Wrote AVM version file", zap.String("module", moduleName), zap.String("tag", latestAvmTag), zap.String("commit", latestAvmCommit), zap.String("path", versionFilePath))
}
