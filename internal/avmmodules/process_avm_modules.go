package avmmodules

import (
	"io"
	"os"
	"regexp"
	"sync"

	"github.com/go-git/go-git/v6"
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

type Module interface {
    GetRepoURL() string
    GetModuleName() string
}

func (m ResourceModulesStruct) GetRepoURL() string    { return m.RepoURL }
func (m PatternModulesStruct) GetRepoURL() string     { return m.RepoURL }
func (m UtilityModulesStruct) GetRepoURL() string     { return m.RepoURL }
func (m ResourceModulesStruct) GetModuleName() string { return m.ModuleName }
func (m PatternModulesStruct) GetModuleName() string  { return m.ModuleName }
func (m UtilityModulesStruct) GetModuleName() string  { return m.ModuleName }

type ModuleNameTransformer func(string) string

func resourceNameTransformer(name string) string {
	var resourceRegex = regexp.MustCompile(`^(avm)-(res-)(.+)$`)
	if matches := resourceRegex.FindStringSubmatch(name); len(matches) == 4 {
		return "rvm-" + matches[2] + "azurerm-" + matches[3]
	}
	return name
}

func patternNameTransformer(name string) string {
	var resourceRegex = regexp.MustCompile(`^(avm)-(ptn-)(.+)$`)
	if matches := resourceRegex.FindStringSubmatch(name); len(matches) == 4 {
		re := regexp.MustCompile(`^avm-(ptn)-(.*)$`)
		if matches := re.FindStringSubmatch(name); len(matches) == 3 {
			return "rvm-pat-azurerm-" + matches[2]
		}
	}
	return name
}

func utilityNameTransformer(name string) string {
	var resourceRegex = regexp.MustCompile(`^(avm)-(utl-)(.+)$`)
	if matches := resourceRegex.FindStringSubmatch(name); len(matches) == 4 {
		re := regexp.MustCompile(`^avm-(utl)-(.*)$`)
		if matches := re.FindStringSubmatch(name); len(matches) == 3 {
			return "rvm-" + matches[1] + "-azurerm-" + matches[2]
		}
	}
	return name
}

type ModuleProcessor struct {
	Logger        *zap.Logger
	SugaredLogger *zap.SugaredLogger
}

type ModulesStruct struct {
	ResourceModules []ResourceModulesStruct
	PatternModules  []PatternModulesStruct
	UtilityModules  []UtilityModulesStruct
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

func batchSlice[T any](items []T, batchSize int) [][]T {
    var batches [][]T
    for batchSize < len(items) {
        items, batches = items[batchSize:], append(batches, items[0:batchSize:batchSize])
    }
    batches = append(batches, items)
    return batches
}

func (p *ModuleProcessor) ProcessResourceModules(processFunc func(ResourceModulesStruct)) error {
	modules, err := getModules()
	if err != nil {
		return err
	}
	batches := batchSlice(modules.ResourceModules, config.CloneBatchSize)
	for _, batch := range batches {
		CloneModulesInBatches(batch, config.TempRepoPath, p.Logger, processFunc, p, resourceNameTransformer)
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
		CloneModulesInBatches(batch, config.TempRepoPath, p.Logger, processFunc, p, patternNameTransformer)
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
		CloneModulesInBatches(batch, config.TempRepoPath, p.Logger, processFunc, p, utilityNameTransformer)
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

func CloneModulesInBatches[T Module](modules []T, destDir string, logger *zap.Logger, processFunc func(T), processor *ModuleProcessor, nameTransformer ModuleNameTransformer) {
	var wg sync.WaitGroup
	jobs := make(chan T)

	for i := 0; i < config.CloneBatchSize; i++ {
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
					cloneRepo(module.GetRepoURL(), tempPath)
					removeGitFolder(processor, tempPath)
					renameFolders(processor, tempPath, newPath)
					processFunc(module)
				} else {
					logger.Error("Error checking temporary repository path", zap.String("path", tempPath), zap.Error(err))
				}
			}
		}()
	}

	for _, module := range modules {
		jobs <- module
	}
	close(jobs)
	wg.Wait()
}

func (p *ModuleProcessor) CleanupTempRepos() {
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
