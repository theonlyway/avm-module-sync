package avmmodules

import (
	"os"

	"github.com/go-git/go-git/v6"
)

// ProcessResourceModules processes the resource modules returned by getResourceModules
func ProcessResourceModules(processFunc func(ResourceModulesStruct)) error {
	modules, err := getResourceModules()
	if err != nil {
		return err
	}
	for _, module := range modules {
		_, err := git.PlainClone("./modules/resources/"+module.ModuleName, &git.CloneOptions{
			URL:      module.RepoURL,
			Progress: os.Stdout,
		})
		if err != nil {
			return err
		}
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
