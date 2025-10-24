package avmmodules

import (
	"io"
	"os"
	"regexp"

	"github.com/go-git/go-git/v6"
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

const ()

type ModuleProcessor struct {
	Logger        *zap.Logger
	SugaredLogger *zap.SugaredLogger
}

type ModulesStruct struct {
	ResourceModules []ResourceModulesStruct
	PatternModules  []PatternModulesStruct
	UtilityModules  []UtilityModulesStruct
}

var moduleNameRegex = regexp.MustCompile(`^(avm)-(res-)(.+)$`)

func (p *ModuleProcessor) ProcessResourceModules(processFunc func(ResourceModulesStruct)) error {
	cleanupTempRepos(p)
	modules, err := getModules()
	if err != nil {
		return err
	}

	for _, module := range modules.ResourceModules {
		tempPath := config.TempRepoPath + "/resources/" + module.ModuleName
		newModuleName := transformModuleName(module.ModuleName)
		newPath := config.TempRepoPath + "/resources/" + newModuleName
		p.Logger.Info("Transformed module name", zap.String("old", module.ModuleName), zap.String("new", newModuleName))

		if _, err := os.Stat(tempPath); err == nil {
			p.Logger.Warn("Temporary repository path exists", zap.String("path", tempPath))
			removeGitFolder(p, tempPath)
			renameFolders(p, tempPath, newPath)
		} else if os.IsNotExist(err) {
			cloneRepo(module.RepoURL, tempPath)
			removeGitFolder(p, tempPath)
			renameFolders(p, tempPath, newPath)
			processFunc(module)
		} else {
			p.Logger.Error("Error checking temporary repository path", zap.String("path", tempPath), zap.Error(err))
		}

	}
	return nil
}

func getModules() (*ModulesStruct, error) {
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


func ProcessPatternModules(processFunc func(PatternModulesStruct)) error {
	modules, err := getPatternModules()
	if err != nil {
		return err
	}
	for _, module := range modules {
		processFunc(module)
	}
	return nil
}

func ProcessUtilityModules(processFunc func(UtilityModulesStruct)) error {
	modules, err := getUtilityModules()
	if err != nil {
		return err
	}
	for _, module := range modules {
		processFunc(module)
	}
	return nil
}

func cloneRepo(repoURL string, destPath string) error {
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

func cleanupTempRepos(p *ModuleProcessor) {
	if !config.CleanTempModulesDir {
		return
	}
	p.Logger.Info("Cleaning up temporary repositories")
	os.RemoveAll(config.TempRepoPath)
}

func removeGitFolder(p *ModuleProcessor, path string) {
	p.Logger.Info("Removing .git folder from", zap.String("path", path))
	gitPath := path + "/.git"
	os.RemoveAll(gitPath)
}

func transformModuleName(name string) string {
	// Replace 'avm' with 'rvm' at the start, and insert 'azurerm' after 'res-'
	if matches := moduleNameRegex.FindStringSubmatch(name); len(matches) == 4 {
		return "rvm-" + matches[2] + "azurerm-" + matches[3]
	}
	return name
}

func renameFolders(p *ModuleProcessor, oldPath string, newPath string) {

	if _, err := os.Stat(newPath); err == nil {
		p.Logger.Warn("New path already exists, removing", zap.String("path", newPath))
		os.RemoveAll(newPath)
	}
	p.Logger.Info("Renaming folder", zap.String("old", oldPath), zap.String("new", newPath))
	err := os.Rename(oldPath, newPath)
	if err != nil {
		p.Logger.Error("Error renaming folder", zap.String("old", oldPath), zap.String("new", newPath), zap.Error(err))
	}
}
