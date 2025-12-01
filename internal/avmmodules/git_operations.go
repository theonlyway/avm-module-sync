package avmmodules

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/google/uuid"
	adogit "github.com/microsoft/azure-devops-go-api/azuredevops/git"
	cp "github.com/otiai10/copy"
	"github.com/theonlyway/avm-module-sync/internal/ado"
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
)

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

func applyPatchesIfExist(moduleName string, localRepoPath string, logger *zap.Logger) error {
	// Construct the patch folder path
	var patchFolderPath string
	if config.ModuleSyncSourceRepoChildPath != "" {
		patchFolderPath = filepath.Join(localRepoPath, config.ModuleSyncSourceRepoChildPath, moduleName, "patches")
	} else {
		patchFolderPath = filepath.Join(localRepoPath, moduleName, "patches")
	}

	// Check if the patch folder exists
	if _, err := os.Stat(patchFolderPath); os.IsNotExist(err) {
		logger.Info("No patches folder found for module", zap.String("module", moduleName), zap.String("patchFolder", patchFolderPath))
		return nil
	}

	logger.Info("Found patches folder, searching for patch files", zap.String("module", moduleName), zap.String("patchFolder", patchFolderPath))

	// Walk through the patch folder and subdirectories to find all .patch files
	var patchFiles []string
	err := filepath.Walk(patchFolderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".patch" {
			patchFiles = append(patchFiles, path)
		}
		return nil
	})

	if err != nil {
		logger.Error("Error walking patch folder", zap.String("module", moduleName), zap.String("patchFolder", patchFolderPath), zap.Error(err))
		return err
	}

	if len(patchFiles) == 0 {
		logger.Info("No patch files found in patches folder", zap.String("module", moduleName), zap.String("patchFolder", patchFolderPath))
		return nil
	}

	logger.Info("Found patch files to apply", zap.String("module", moduleName), zap.Int("count", len(patchFiles)))

	// Apply each patch file
	for _, patchFile := range patchFiles {
		logger.Info("Applying patch file", zap.String("module", moduleName), zap.String("patchFile", patchFile))

		// Use git apply to apply the patch
		cmd := exec.Command("git", "apply", patchFile)
		cmd.Dir = localRepoPath
		output, err := cmd.CombinedOutput()

		if err != nil {
			logger.Error("Failed to apply patch", zap.String("module", moduleName), zap.String("patchFile", patchFile), zap.String("output", string(output)), zap.Error(err))
			return err
		}

		logger.Info("Successfully applied patch", zap.String("module", moduleName), zap.String("patchFile", patchFile))
	}

	return nil
}

func CommitAndPushModulesToGit[T Module](clients *ado.AdoClients, ctx context.Context, project string, repoId *uuid.UUID, module T, localRepoPath string, nameTransformer ModuleNameTransformer, logger *zap.Logger) error {
	branchName := "feat/avm-module-sync/" + nameTransformer(module.GetModuleName())
	authorName := config.ModuleSyncAuthorName
	authorEmail := config.ModuleSyncAuthorEmail
	moduleName := nameTransformer(module.GetModuleName())
	commitMsg := "feat(module): Synced AVM module " + moduleName
	sourcePath := config.SourceRepoPath
	defaultBranchName := plumbing.ReferenceName("refs/heads/" + config.DefaultBranchName)
	remoteRef := plumbing.ReferenceName("refs/remotes/origin/" + branchName)
	repo, err := git.PlainOpen(sourcePath)
	logger.Info("Starting git operations", zap.String("module", moduleName), zap.String("path", localRepoPath))
	if err != nil {
		logger.Error("Failed to open repo", zap.String("module", moduleName), zap.String("path", localRepoPath), zap.Error(err))
		return err
	}

	w, err := repo.Worktree()
	if err != nil {
		logger.Error("Failed to get worktree", zap.String("module", moduleName), zap.String("path", localRepoPath), zap.Error(err))
		return err
	}

	// Ensure local branch exists before checking out
	// In ADO pipelines, repo may be in detached HEAD state
	logger.Info("Checking out default branch", zap.String("module", moduleName), zap.String("branch", defaultBranchName.String()))
	_, err = repo.Reference(defaultBranchName, false)
	if err == plumbing.ErrReferenceNotFound {
		// Local branch doesn't exist, create it from remote
		logger.Info("Local default branch not found, creating from remote", zap.String("module", moduleName))
		remoteDefaultRef := plumbing.ReferenceName("refs/remotes/origin/" + config.DefaultBranchName)
		remoteRef, err := repo.Reference(remoteDefaultRef, true)
		if err != nil {
			logger.Error("Failed to get remote default branch reference", zap.String("branch", remoteDefaultRef.String()), zap.Error(err))
			return err
		}
		// Create local branch pointing to same commit as remote
		headRef := plumbing.NewHashReference(defaultBranchName, remoteRef.Hash())
		err = repo.Storer.SetReference(headRef)
		if err != nil {
			logger.Error("Failed to create local default branch", zap.String("branch", defaultBranchName.String()), zap.Error(err))
			return err
		}
		logger.Info("Created local default branch", zap.String("branch", defaultBranchName.String()), zap.String("commit", remoteRef.Hash().String()))
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: defaultBranchName,
		Force:  true,
	})
	if err != nil {
		logger.Error("Failed to checkout default branch", zap.String("branch", defaultBranchName.String()), zap.String("path", localRepoPath), zap.Error(err))
		return err
	}
	logger.Info("Checked out default branch", zap.String("branch", defaultBranchName.String()), zap.String("path", localRepoPath))

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

	// Apply patches if they exist
	err = applyPatchesIfExist(moduleName, localRepoPath, logger)
	if err != nil {
		logger.Error("Failed to apply patches", zap.String("module", moduleName), zap.Error(err))
		return err
	}

	// Add all module files to staging using system git to respect .gitattributes and line endings since it seems go-git doesn't
	logger.Info("Adding all files to staging using system git", zap.String("module", moduleName))
	cmdAdd := exec.Command("git", "add", ".")
	cmdAdd.Dir = localRepoPath
	outputAdd, err := cmdAdd.CombinedOutput()
	if err != nil {
		logger.Error("Failed to add changes with system git", zap.String("module", moduleName), zap.String("output", string(outputAdd)), zap.Error(err))
	}

	// Check if there are any staged changes to commit
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

	hasStagedChanges := false
	for _, fileStatus := range status {
		if fileStatus.Staging != git.Unmodified {
			hasStagedChanges = true
			break
		}
	}

	if !hasStagedChanges {
		logger.Info("No staged changes to commit", zap.String("module", moduleName))
		return nil
	}

	// Configure git user for this repository (needed for system git commit)
	cmdConfigName := exec.Command("git", "config", "user.name", authorName)
	cmdConfigName.Dir = localRepoPath
	_, _ = cmdConfigName.CombinedOutput() // Ignore errors, config might already be set

	cmdConfigEmail := exec.Command("git", "config", "user.email", authorEmail)
	cmdConfigEmail.Dir = localRepoPath
	_, _ = cmdConfigEmail.CombinedOutput() // Ignore errors, config might already be set

	// Commit using system git (since we're using system git for add)
	logger.Info("Commiting changes", zap.String("module", moduleName), zap.String("commit_msg", commitMsg))
	cmdCommit := exec.Command("git", "commit", "-m", commitMsg)
	cmdCommit.Dir = localRepoPath
	outputCommit, err := cmdCommit.CombinedOutput()
	if err != nil {
		logger.Error("Failed to commit changes", zap.String("module", moduleName), zap.String("output", string(outputCommit)), zap.Error(err))
		return err
	}

	// Push
	logger.Info("Pushing changes to origin", zap.String("module", moduleName))
	if config.AdoPat != "" {
		pushOpts := &git.PushOptions{}
		// Only add auth if PAT is configured (not needed in ADO pipeline with persistCredentials: true)
		if config.AdoPat != "" {
			pushOpts.Auth = &http.BasicAuth{Username: "anything", Password: config.AdoPat}
		}
		logger.Info("Pushing using go-git")
		err = repo.Push(pushOpts)
		if err != nil {
			logger.Error("Failed to push changes to origin", zap.String("module", moduleName), zap.Error(err))
			return err
		}
	} else {
		logger.Info("Pushing using system git")
		cmd := exec.Command("git", "push", "origin", "HEAD:"+branchName)
		cmd.Dir = localRepoPath
		output, err := cmd.CombinedOutput()
		logger.Info("git push output", zap.String("output", string(output)))
		if err != nil {
			logger.Error("git push failed", zap.Error(err))
			return err
		}
	}
	// Create pull request
	title := "feat(module): Synced AVM module " + moduleName
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
