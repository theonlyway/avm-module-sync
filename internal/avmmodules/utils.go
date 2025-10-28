package avmmodules

import (
	"context"
	"io"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/google/uuid"
	adogit "github.com/microsoft/azure-devops-go-api/azuredevops/git"
	cp "github.com/otiai10/copy"
	"github.com/theonlyway/avm-module-sync/internal/ado"
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

type ModuleProcessor struct {
	Logger        *zap.Logger
	SugaredLogger *zap.SugaredLogger
	Clients       *ado.AdoClients
	Context       context.Context
	Project       string
	RepoId        *uuid.UUID
}

type ModulesStruct struct {
	ResourceModules []ResourceModulesStruct
	PatternModules  []PatternModulesStruct
	UtilityModules  []UtilityModulesStruct
}

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

// ModuleNameTransformer allows custom name transformation per module type
type ModuleNameTransformer func(string) string

func resourceNameTransformer(name string) string {
	var resourceRegex = regexp.MustCompile(`^(avm)-(res-)(.+)$`)
	if matches := resourceRegex.FindStringSubmatch(name); len(matches) == 4 {
		return "rvm-" + matches[2] + "azurerm-" + matches[3]
	}
	return name
}

func patternNameTransformer(name string) string {
	var patternRegex = regexp.MustCompile(`^avm-(ptn)-(.*)$`)
	if matches := patternRegex.FindStringSubmatch(name); len(matches) == 3 {
		return "rvm-pat-azurerm-" + matches[2]
	}
	return name
}

func utilityNameTransformer(name string) string {
	var utilityRegex = regexp.MustCompile(`^avm-(utl)-(.*)$`)
	if matches := utilityRegex.FindStringSubmatch(name); len(matches) == 3 {
		return "rvm-" + matches[1] + "-azurerm-" + matches[2]
	}
	return name
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

func removeGitFolder(p *ModuleProcessor, path string) {
	p.Logger.Info("Removing .git folder from", zap.String("path", path))
	gitPath := path + "/.git"
	os.RemoveAll(gitPath)
}

func batchSlice[T any](items []T, batchSize int) [][]T {
	var batches [][]T
	for batchSize < len(items) {
		items, batches = items[batchSize:], append(batches, items[0:batchSize:batchSize])
	}
	batches = append(batches, items)
	return batches
}

func renameFolders(p *ModuleProcessor, oldPath string, newPath string) {
	if oldPath == newPath {
		return
	}
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

func CleanUpTempDirs(logger *zap.Logger) {
	if !config.CleanTempDirs {
		return
	}
	logger.Info("Cleaning up temporary directories")
	os.RemoveAll(config.TempAvmModuleRepoPath)
	os.RemoveAll(config.TempSourceRepoPath)
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

	for range config.BatchSize {
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

	for _, module := range modules {
		jobs <- module
	}
	close(jobs)
	wg.Wait()
}

func copyModuleToBranch[T Module](module T, localRepoPath string, nameTransformer ModuleNameTransformer, logger *zap.Logger) {
	var sourcePath string
	moduleName := nameTransformer(module.GetModuleName())
	modulePath := config.TempAvmModuleRepoPath + "/" + moduleName
	if config.ModuleSyncSourceRepoChildPath != "" {
		sourcePath = localRepoPath + "/" + config.ModuleSyncSourceRepoChildPath + "/" + moduleName
	} else {
		sourcePath = localRepoPath + "/" + moduleName
	}
	opt := cp.Options{
		NumOfWorkers: int64(config.BatchSize),
	}
	logger.Info("Copying module to branch", zap.String("source", modulePath), zap.String("dest", sourcePath))
	err := cp.Copy(modulePath, sourcePath, opt)
	if err != nil {
		logger.Error("Error copying module to branch", zap.String("modulePath", modulePath), zap.String("sourcePath", sourcePath), zap.Error(err))
	}
}

func CommitAndPushModulesToGit[T Module](clients *ado.AdoClients, ctx context.Context, project string, repoId *uuid.UUID, module T, localRepoPath string, nameTransformer ModuleNameTransformer, logger *zap.Logger) error {
	branchName := "feat/avm-module-sync/" + nameTransformer(module.GetModuleName())
	authorName := config.ModuleSyncAuthorName
	authorEmail := config.ModuleSyncAuthorEmail
	moduleName := nameTransformer(module.GetModuleName())
	commitMsg := "feat(module): Syncing AVM module " + moduleName + " from source repository"
	sourcePath := config.TempSourceRepoPath
	defaultBranchName := plumbing.ReferenceName("refs/heads/" + config.DefaultBranchName)
	remoteRef := plumbing.ReferenceName("refs/remotes/origin/" + branchName)
	repo, err := git.PlainOpen(sourcePath)
	logger.Info("Commiting git changes", zap.String("module", moduleName), zap.String("path", localRepoPath))
	if err != nil {
		logger.Error("Failed to open repo", zap.String("module", moduleName), zap.String("path", localRepoPath), zap.Error(err))
		return err
	}

	w, err := repo.Worktree()
	if err != nil {
		logger.Error("Failed to get worktree", zap.String("module", moduleName), zap.String("path", localRepoPath), zap.Error(err))
		return err
	}

	logger.Info("Checking out default branch", zap.String("module", moduleName), zap.String("branch", defaultBranchName.String()))
	err = w.Checkout(&git.CheckoutOptions{
		Branch: defaultBranchName,
		Force:  true,
	})
	if err != nil {
		logger.Error("Failed to create branch", zap.String("branch", branchName), zap.String("path", localRepoPath), zap.Error(err))
		return err
	} else {
		logger.Info("Created branch for module", zap.String("branch", branchName), zap.String("path", localRepoPath))
	}

	// Check if branch already exists remotely
	exists, err := remoteBranchExists(repo, remoteRef, logger)
	if err != nil {
		logger.Error("Failed to check if branch exists remotely", zap.String("branch", remoteRef.String()), zap.String("path", localRepoPath), zap.Error(err))
		return err
	}
	if exists {
		logger.Info("Branch already exists remotely", zap.String("branch", branchName), zap.String("path", localRepoPath))
		err = w.Checkout(&git.CheckoutOptions{
			Branch: remoteRef,
			Force:  true,
		})
		if err != nil {
			logger.Error("Failed to create branch", zap.String("branch", branchName), zap.String("path", localRepoPath), zap.Error(err))
			return err
		} else {
			logger.Info("Created branch for module", zap.String("branch", branchName), zap.String("path", localRepoPath))
		}
	} else {
		logger.Info("Creating and checking out module branch", zap.String("module", moduleName), zap.String("branch", branchName))
		err = w.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(branchName),
			Create: true,
		})
		if err != nil {
			logger.Error("Failed to create branch", zap.String("branch", branchName), zap.String("path", localRepoPath), zap.Error(err))
			return err
		} else {
			logger.Info("Created branch for module", zap.String("branch", branchName), zap.String("path", localRepoPath))
		}
	}

	copyModuleToBranch(module, localRepoPath, nameTransformer, logger)

	logger.Info("Checking git status", zap.String("module", moduleName))
	status, err := w.Status()
	if err != nil {
		logger.Error("Failed to get git status", zap.String("module", moduleName), zap.Error(err))
		return err
	}
	if config.DebugMode {
		// Log all changed files and their status
		for file, fileStatus := range status {
			logger.Info("Git file status", zap.String("module", moduleName), zap.String("file", file), zap.String("worktree", string(fileStatus.Worktree)), zap.String("staging", string(fileStatus.Staging)))
		}
	}

	if status.IsClean() {
		logger.Info("No changes to commit", zap.String("module", moduleName))
		return nil
	}

	// Add all changes
	logger.Info("Adding changes", zap.String("module", moduleName))
	err = w.AddWithOptions(&git.AddOptions{All: true})
	if err != nil {
		logger.Error("Failed to add changes", zap.String("module", moduleName), zap.Error(err))
	}

	// Commit
	logger.Info("Commiting changes", zap.String("module", moduleName), zap.String("commit_msg", commitMsg))
	_, err = w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		logger.Error("Failed to commit changes", zap.String("module", moduleName), zap.Error(err))
		return err
	}

	// Push
	logger.Info("Pushing changes to origin", zap.String("module", moduleName))
	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       &http.BasicAuth{Username: "anything", Password: config.AdoPat},
	})
	if err != nil {
		logger.Error("Failed to push changes to origin", zap.String("module", moduleName), zap.Error(err))
		return err
	}
	// Create pull request
	title := "feat(module): Syncing AVM module " + moduleName + " from source repository"
	description := "This is an automated pull request to sync the " + moduleName + " module from the source AVM repository " + module.GetRepoURL()
	sourceRef := "refs/heads/" + branchName
	targetRef := "refs/heads/" + config.DefaultBranchName
	pr, err := createPullRequest(clients.GitClient, ctx, repoId, project, sourceRef, targetRef, title, description)
	if err != nil {
		logger.Error("Failed to create pull request", zap.String("module", moduleName), zap.Error(err))
		return err
	}
	logger.Info("Created pull request", zap.String("module", moduleName), zap.Int("prId", *pr.PullRequestId))
	return nil
}

func remoteBranchExists(repo *git.Repository, remoteRef plumbing.ReferenceName, logger *zap.Logger) (bool, error) {
	logger.Info("Checking if the branch exists on the origin", zap.String("remoteRef", remoteRef.String()))
	_, err := repo.Reference(remoteRef, true)
	if err == plumbing.ErrReferenceNotFound {
		return false, nil
	}
	return err == nil, err
}

func createPullRequest(client adogit.Client, ctx context.Context, repoId *uuid.UUID, project string, sourceBranch, targetBranch, title, description string) (*adogit.GitPullRequest, error) {
	repoIdStr := repoId.String()
	pr := adogit.GitPullRequest{
		Title:         &title,
		Description:   &description,
		SourceRefName: &sourceBranch,
		TargetRefName: &targetBranch,
	}
	args := adogit.CreatePullRequestArgs{
		GitPullRequestToCreate: &pr,
		RepositoryId:           &repoIdStr,
		Project:                &project,
	}
	return client.CreatePullRequest(ctx, args)
}
