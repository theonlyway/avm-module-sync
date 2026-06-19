package avmmodules

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/client"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/google/uuid"
	adogit "github.com/microsoft/azure-devops-go-api/azuredevops/git"
	cp "github.com/otiai10/copy"
	"github.com/theonlyway/avm-module-sync/internal/ado"
	"github.com/theonlyway/avm-module-sync/internal/config"
	"go.uber.org/zap"
	"golang.org/x/mod/semver"
)

// copyModuleToBranch copies a module from the temporary clone location to the target repository branch.
// Backs up the patches directory, does a clean copy, then restores patches to ensure files deleted from source are removed.
func copyModuleToBranch[T Module](module T, localRepoPath string, nameTransformer ModuleNameTransformer, logger *zap.Logger) {
	var sourcePath string
	moduleName := nameTransformer(module.GetModuleName())
	modulePath := config.TempAvmModuleRepoPath + "/" + moduleName
	if config.ModuleSyncSourceRepoChildPath != "" {
		sourcePath = localRepoPath + "/" + config.ModuleSyncSourceRepoChildPath + "/" + moduleName
	} else {
		sourcePath = localRepoPath + "/" + moduleName
	}

	// Backup patches directory if it exists
	patchesPath := filepath.Join(sourcePath, config.PatchesFolderName)
	tempPatchesPath := filepath.Join(config.TempAvmModuleRepoPath, ".backup-patches-"+moduleName)
	hasPatchesBackup := false

	if _, err := os.Stat(patchesPath); err == nil {
		logger.Info("Backing up patches directory", zap.String("module", moduleName), zap.String("from", patchesPath), zap.String("to", tempPatchesPath))
		err = os.Rename(patchesPath, tempPatchesPath)
		if err != nil {
			logger.Error("Error backing up patches directory", zap.String("module", moduleName), zap.Error(err))
		} else {
			hasPatchesBackup = true
		}
	}

	// Remove destination directory for clean sync
	if _, err := os.Stat(sourcePath); err == nil {
		logger.Info("Removing existing module directory for clean sync", zap.String("module", moduleName), zap.String("path", sourcePath))
		err = os.RemoveAll(sourcePath)
		if err != nil {
			logger.Error("Error removing module directory", zap.String("module", moduleName), zap.String("path", sourcePath), zap.Error(err))
		}
	}

	// Copy fresh module content
	opt := cp.Options{
		NumOfWorkers: int64(config.BatchSize),
	}
	logger.Info("Copying module to branch", zap.String("module", moduleName), zap.String("source", modulePath), zap.String("dest", sourcePath))
	err := cp.Copy(modulePath, sourcePath, opt)
	if err != nil {
		logger.Error("Error copying module to branch", zap.String("module", moduleName), zap.String("modulePath", modulePath), zap.String("sourcePath", sourcePath), zap.Error(err))
	}

	// Restore patches directory if we backed it up
	if hasPatchesBackup {
		logger.Info("Restoring patches directory", zap.String("module", moduleName), zap.String("from", tempPatchesPath), zap.String("to", patchesPath))
		err = os.Rename(tempPatchesPath, patchesPath)
		if err != nil {
			logger.Error("Error restoring patches directory", zap.String("module", moduleName), zap.Error(err))
		}
	}
}

// applyPatchesIfExist searches for and applies any .patch files found in the module's patches directory.
// Patches are applied recursively from subdirectories using git apply.
func applyPatchesIfExist(moduleName string, localRepoPath string, logger *zap.Logger) error {
	// Construct the patch folder path
	var patchFolderPath string
	if config.ModuleSyncSourceRepoChildPath != "" {
		patchFolderPath = filepath.Join(localRepoPath, config.ModuleSyncSourceRepoChildPath, moduleName, config.PatchesFolderName)
	} else {
		patchFolderPath = filepath.Join(localRepoPath, moduleName, config.PatchesFolderName)
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
	failedPatches := 0
	for _, patchFile := range patchFiles {
		logger.Info("Applying patch file", zap.String("module", moduleName), zap.String("patchFile", patchFile))

		// Use git apply to apply the patch
		cmd := exec.Command("git", "apply", patchFile)
		cmd.Dir = localRepoPath
		output, err := cmd.CombinedOutput()

		if err != nil {
			logger.Error("Failed to apply patch, continuing with remaining patches", zap.String("module", moduleName), zap.String("patchFile", patchFile), zap.String("output", string(output)), zap.Error(err))
			failedPatches++
			continue
		}

		logger.Info("Successfully applied patch", zap.String("module", moduleName), zap.String("patchFile", patchFile))
	}

	if failedPatches > 0 {
		logger.Warn("Some patches failed to apply, but continuing with commit", zap.String("module", moduleName), zap.Int("failedCount", failedPatches), zap.Int("totalCount", len(patchFiles)))
	}

	return nil
}

// avmRegistrySourceRe matches a Terraform `source` argument that references a module on the
// public AVM Terraform registry, capturing the AVM module name (e.g. avm-utl-regions) and any
// optional submodule subpath (e.g. //modules/subnet).
var avmRegistrySourceRe = regexp.MustCompile(`(source\s*=\s*")(?:registry\.terraform\.io/)?Azure/(avm-[a-z0-9-]+)/azurerm(//[^"]*)?(")`)

// rewriteRegistrySourcesToArtifactory rewrites Terraform module `source` arguments that point at
// the public AVM Terraform registry so they instead point at the configured Artifactory path.
// Every .tf file under the module directory is processed except those inside an examples folder.
// The Artifactory source is produced by executing config.ArtifactorySourceTemplate with the
// transformed (RVM) module name available as {{ .ModuleName }}. The version argument is left
// untouched. When no template is configured the function is a no-op.
func rewriteRegistrySourcesToArtifactory(moduleName string, localRepoPath string, logger *zap.Logger) error {
	if config.ArtifactorySourceTemplate == "" {
		return nil
	}

	tmpl, err := template.New("artifactory-source").Parse(config.ArtifactorySourceTemplate)
	if err != nil {
		logger.Error("Failed to parse Artifactory source template", zap.String("template", config.ArtifactorySourceTemplate), zap.Error(err))
		return err
	}

	var moduleDir string
	if config.ModuleSyncSourceRepoChildPath != "" {
		moduleDir = filepath.Join(localRepoPath, config.ModuleSyncSourceRepoChildPath, moduleName)
	} else {
		moduleDir = filepath.Join(localRepoPath, moduleName)
	}

	logger.Info("Rewriting public registry module sources to Artifactory", zap.String("module", moduleName), zap.String("moduleDir", moduleDir))

	return filepath.Walk(moduleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip examples directories entirely so example usage keeps pointing at the public registry
			if info.Name() == config.ExamplesFolderName {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".tf" {
			return nil
		}
		return rewriteTfFileSources(path, tmpl, logger)
	})
}

// rewriteTfFileSources rewrites public AVM registry `source` references in a single .tf file
// to the Artifactory equivalent, writing the file back only when a change is made.
func rewriteTfFileSources(path string, tmpl *template.Template, logger *zap.Logger) error {
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error("Failed to read .tf file for source rewrite", zap.String("file", path), zap.Error(err))
		return err
	}

	changed := false
	out := avmRegistrySourceRe.ReplaceAllFunc(data, func(match []byte) []byte {
		groups := avmRegistrySourceRe.FindSubmatch(match)
		prefix := string(groups[1])
		avmName := string(groups[2])
		subpath := string(groups[3])
		suffix := string(groups[4])

		var sb strings.Builder
		if execErr := tmpl.Execute(&sb, struct{ ModuleName string }{ModuleName: transformAvmModuleName(avmName)}); execErr != nil {
			logger.Error("Failed to render Artifactory source template", zap.String("file", path), zap.String("module", avmName), zap.Error(execErr))
			return match
		}

		changed = true
		logger.Info("Rewriting registry source to Artifactory", zap.String("file", path), zap.String("from", avmName), zap.String("to", sb.String()))
		return []byte(prefix + sb.String() + subpath + suffix)
	})

	if !changed {
		return nil
	}

	if err := os.WriteFile(path, out, 0644); err != nil {
		logger.Error("Failed to write rewritten .tf file", zap.String("file", path), zap.Error(err))
		return err
	}
	return nil
}

// buildConventionalCommitMessage constructs a conventional commit message for the given
// commit type and module name.  Recognised types: "breaking", "feat", "fix", "chore".
func buildConventionalCommitMessage(commitType, moduleName string) string {
	switch commitType {
	case "breaking":
		return "feat(module)!: Synced AVM module " + moduleName
	case "fix":
		return "fix(module): Synced AVM module " + moduleName
	case "chore":
		return "chore(module): Synced AVM module " + moduleName
	default: // "feat" and any unknown value
		return "feat(module): Synced AVM module " + moduleName
	}
}

// CommitAndPushModulesToGit handles the complete Git workflow for syncing a module.
// It creates a feature branch, copies the module, applies patches, commits changes,
// pushes to remote, and creates a pull request in Azure DevOps.
// commitType should be one of "breaking", "feat", "fix", or "chore" as determined by
// analysing the upstream AVM repo's conventional commit history.
// latestAvmTag is the most recent tag from the upstream AVM repo and latestAvmCommit is the
// commit hash that tag points to; both are written to .avm-version inside the module folder
// so the next run knows where to start from and a downstream pipeline can package the module.
func CommitAndPushModulesToGit[T Module](clients *ado.AdoClients, ctx context.Context, project string, repoId *uuid.UUID, module T, localRepoPath string, nameTransformer ModuleNameTransformer, commitType string, latestAvmTag string, latestAvmCommit string, logger *zap.Logger) error {
	branchName := "feat/avm-module-sync/" + nameTransformer(module.GetModuleName())
	authorName := config.ModuleSyncAuthorName
	authorEmail := config.ModuleSyncAuthorEmail
	moduleName := nameTransformer(module.GetModuleName())

	// Skip if the upstream tag hasn't advanced since the last sync, unless this module is
	// force-updated via the force-update-all or force-update-modules flags.
	lastSyncedTag := readAvmVersionFile(moduleName, logger)
	if isModuleForced(module.GetModuleName()) {
		logger.Info("Force-updating module, bypassing tag advancement check",
			zap.String("module", moduleName),
			zap.String("lastSyncedTag", lastSyncedTag),
			zap.String("latestAvmTag", latestAvmTag))
	} else if latestAvmTag != "" && lastSyncedTag != "" {
		latest := ensureSemverPrefix(latestAvmTag)
		synced := ensureSemverPrefix(lastSyncedTag)
		if semver.IsValid(latest) && semver.IsValid(synced) {
			if semver.Compare(latest, synced) <= 0 {
				logger.Info("Upstream tag has not advanced since last sync, skipping",
					zap.String("module", moduleName),
					zap.String("lastSyncedTag", lastSyncedTag),
					zap.String("latestAvmTag", latestAvmTag))
				return nil
			}
		}
	}
	commitMsg := buildConventionalCommitMessage(commitType, moduleName)
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
			logger.Error("Failed to get remote default branch reference", zap.String("module", moduleName), zap.String("branch", remoteDefaultRef.String()), zap.Error(err))
			return err
		}
		// Create local branch pointing to same commit as remote
		headRef := plumbing.NewHashReference(defaultBranchName, remoteRef.Hash())
		err = repo.Storer.SetReference(headRef)
		if err != nil {
			logger.Error("Failed to create local default branch", zap.String("module", moduleName), zap.String("branch", defaultBranchName.String()), zap.Error(err))
			return err
		}
		logger.Info("Created local default branch", zap.String("module", moduleName), zap.String("branch", defaultBranchName.String()), zap.String("commit", remoteRef.Hash().String()))
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: defaultBranchName,
		Force:  true,
	})
	if err != nil {
		logger.Error("Failed to checkout default branch", zap.String("module", moduleName), zap.String("branch", defaultBranchName.String()), zap.String("path", localRepoPath), zap.Error(err))
		return err
	}
	logger.Info("Checked out default branch", zap.String("module", moduleName), zap.String("branch", defaultBranchName.String()), zap.String("path", localRepoPath))

	// Check if branch already exists remotely
	exists, err := remoteBranchExists(repo, remoteRef, logger, moduleName)
	if err != nil {
		logger.Error("Failed to check if branch exists remotely", zap.String("module", moduleName), zap.String("branch", remoteRef.String()), zap.String("path", localRepoPath), zap.Error(err))
		return err
	}
	if exists {
		logger.Info("Branch already exists remotely", zap.String("module", moduleName), zap.String("branch", branchName), zap.String("path", localRepoPath))
		err = w.Checkout(&git.CheckoutOptions{
			Branch: remoteRef,
			Force:  true,
		})
		if err != nil {
			logger.Error("Failed to create branch", zap.String("module", moduleName), zap.String("branch", branchName), zap.String("path", localRepoPath), zap.Error(err))
			return err
		} else {
			logger.Info("Created branch for module", zap.String("module", moduleName), zap.String("branch", branchName), zap.String("path", localRepoPath))
		}
	} else {
		logger.Info("Creating and checking out module branch", zap.String("module", moduleName), zap.String("branch", branchName))
		err = w.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(branchName),
			Create: true,
		})
		if err != nil {
			logger.Error("Failed to create branch", zap.String("module", moduleName), zap.String("branch", branchName), zap.String("path", localRepoPath), zap.Error(err))
			return err
		} else {
			logger.Info("Created branch for module", zap.String("module", moduleName), zap.String("branch", branchName), zap.String("path", localRepoPath))
		}
	}

	copyModuleToBranch(module, localRepoPath, nameTransformer, logger)

	// Write the version file so the next sync knows which AVM tag was last applied
	writeAvmVersionFile(moduleName, localRepoPath, latestAvmTag, latestAvmCommit, logger)

	// Apply patches if they exist
	err = applyPatchesIfExist(moduleName, localRepoPath, logger)
	if err != nil {
		logger.Warn("Errors occurred while applying patches, but continuing with commit", zap.String("module", moduleName), zap.Error(err))
	}

	// Rewrite public AVM registry module sources to Artifactory if a template is configured
	err = rewriteRegistrySourcesToArtifactory(moduleName, localRepoPath, logger)
	if err != nil {
		logger.Warn("Errors occurred while rewriting registry sources, but continuing with commit", zap.String("module", moduleName), zap.Error(err))
	}

	// Add all module files to staging using system git to respect .gitattributes and line endings since it seems go-git doesn't
	// Use -A to also stage deletions of files that exist in repo but not in source
	logger.Info("Adding all files to staging using system git", zap.String("module", moduleName))
	cmdAdd := exec.Command("git", "add", "-A", ".")
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
			pushOpts.ClientOptions = []client.Option{client.WithHTTPAuth(&http.BasicAuth{Username: "anything", Password: config.AdoPat})}
		}
		logger.Info("Pushing using go-git", zap.String("module", moduleName))
		err = repo.Push(pushOpts)
		if err != nil {
			logger.Error("Failed to push changes to origin", zap.String("module", moduleName), zap.Error(err))
			return err
		}
	} else {
		logger.Info("Pushing using system git", zap.String("module", moduleName))
		cmd := exec.Command("git", "push", "origin", "HEAD:"+branchName)
		cmd.Dir = localRepoPath
		output, err := cmd.CombinedOutput()
		logger.Info("git push output", zap.String("module", moduleName), zap.String("output", string(output)))
		if err != nil {
			logger.Error("git push failed", zap.Error(err))
			return err
		}
	}
	// Create pull request
	title := buildConventionalCommitMessage(commitType, moduleName)
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

// remoteBranchExists checks if a branch exists on the remote repository.
func remoteBranchExists(repo *git.Repository, remoteRef plumbing.ReferenceName, logger *zap.Logger, moduleName string) (bool, error) {
	logger.Info("Checking if the branch exists on the origin", zap.String("module", moduleName), zap.String("remoteRef", remoteRef.String()))
	_, err := repo.Reference(remoteRef, true)
	if err == plumbing.ErrReferenceNotFound {
		return false, nil
	}
	return err == nil, err
}

// createPullRequest creates a new pull request in Azure DevOps using the provided parameters.
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
