package avmmodules

import (
	"io"
	"os"

	"github.com/go-git/go-git/v6"
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

const ()

type ModuleProcessor struct {
	Logger        *zap.Logger
	SugaredLogger *zap.SugaredLogger
}

func (p *ModuleProcessor) ProcessResourceModules(processFunc func(ResourceModulesStruct)) error {
	cleanupTempRepos()
	modules, err := getResourceModules()
	if err != nil {
		return err
	}
	for _, module := range modules {
		cloneRepo(module.RepoURL, config.TempRepoPath+"/resources/"+module.ModuleName)
		processFunc(module)
	}
	return nil
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

func cleanupTempRepos() {
	os.RemoveAll(config.TempRepoPath)
}
