package avmmodules

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

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

// avmRegistrySourceRe matches a Terraform/OpenTofu `source` argument that references a module on
// the public AVM registry, capturing the AVM module name (e.g. avm-utl-interfaces) and any optional
// submodule subpath (e.g. //modules/subnet). The optional host prefix covers both the Terraform
// (registry.terraform.io) and OpenTofu (registry.opentofu.org) default registries; the bare
// namespace/name/provider form is the common case. The provider segment is matched generically
// (e.g. azurerm, azure) since the rewritten Artifactory source supplies its own provider.
var avmRegistrySourceRe = regexp.MustCompile(`(source\s*=\s*")(?:registry\.(?:terraform\.io|opentofu\.org)/)?Azure/(avm-[a-z0-9-]+)/[a-z0-9]+(//[^"]*)?(")`)

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

// buildCommitMessage constructs the conventional commit message used for a module sync.
// The innersource version is kept in lock-step with the upstream tag rather than being
// derived from the commit type, so a fixed "chore" type is used; it only needs to satisfy
// the pipeline's conventional-commit validation.
func buildCommitMessage(moduleName string) string {
	return "chore(module): Synced AVM module " + moduleName
}

// CommitAndPushModulesToGit handles the complete Git workflow for syncing a module.
// It creates a feature branch, copies the module, applies patches, commits changes,
// pushes to remote, and creates a pull request in Azure DevOps.
// latestAvmTag is the most recent tag from the upstream AVM repo and latestAvmCommit is the
// commit hash that tag points to; both are written to .avm-version inside the module folder
// so the next run knows where to start from and a downstream pipeline can package the module.
func CommitAndPushModulesToGit[T Module](clients *ado.AdoClients, ctx context.Context, project string, repoId *uuid.UUID, module T, localRepoPath string, nameTransformer ModuleNameTransformer, latestAvmTag string, latestAvmCommit string, logger *zap.Logger) error {
	branchName := "feat/avm-module-sync/" + nameTransformer(module.GetModuleName())
	authorName := config.ModuleSyncAuthorName
	authorEmail := config.ModuleSyncAuthorEmail
	moduleName := nameTransformer(module.GetModuleName())

	// Skip if the upstream tag hasn't advanced since the last sync, unless this module is
	// force-updated via the force-update-all or force-update-modules flags. When the tag name
	// is unchanged but the commit it points to has moved, the module is re-synced.
	lastSyncedTag, lastSyncedCommit := readAvmVersionFile(moduleName, logger)
	if isModuleForced(module.GetModuleName()) {
		logger.Info("Force-updating module, bypassing tag advancement check",
			zap.String("module", moduleName),
			zap.String("lastSyncedTag", lastSyncedTag),
			zap.String("latestAvmTag", latestAvmTag))
	} else if latestAvmTag != "" && lastSyncedTag != "" {
		latest := ensureSemverPrefix(latestAvmTag)
		synced := ensureSemverPrefix(lastSyncedTag)
		if semver.IsValid(latest) && semver.IsValid(synced) {
			cmp := semver.Compare(latest, synced)
			switch {
			case cmp < 0:
				logger.Info("Upstream tag is older than last sync, skipping",
					zap.String("module", moduleName),
					zap.String("lastSyncedTag", lastSyncedTag),
					zap.String("latestAvmTag", latestAvmTag))
				return nil
			case cmp == 0:
				// Same tag name: skip only when the commit also matches (or either commit is
				// unknown, e.g. an older version file without a recorded commit).
				if lastSyncedCommit == "" || latestAvmCommit == "" || latestAvmCommit == lastSyncedCommit {
					logger.Info("Upstream tag has not advanced since last sync, skipping",
						zap.String("module", moduleName),
						zap.String("lastSyncedTag", lastSyncedTag),
						zap.String("latestAvmTag", latestAvmTag))
					return nil
				}
				logger.Info("Upstream tag unchanged but commit moved, re-syncing",
					zap.String("module", moduleName),
					zap.String("tag", latestAvmTag),
					zap.String("lastSyncedCommit", lastSyncedCommit),
					zap.String("latestAvmCommit", latestAvmCommit))
			}
		}
	}
	commitMsg := buildCommitMessage(moduleName)
	defaultBranch := config.DefaultBranchName
	baseRef := "origin/" + defaultBranch
	logger.Info("Starting git operations", zap.String("module", moduleName), zap.String("path", localRepoPath))

	// Configure the commit identity (CI checkouts often have none set).
	_, _ = runGit(localRepoPath, logger, moduleName, "config", "user.name", authorName)
	_, _ = runGit(localRepoPath, logger, moduleName, "config", "user.email", authorEmail)

	// Create (or reset) the module branch fresh from the default branch using a single raw-git
	// checkout -B. This guarantees the branch contains only the default branch plus this one
	// module, eliminating the index desync that previously let other modules' files leak into
	// the commit. -f discards any local modifications left from the prior module.
	logger.Info("Creating module branch from default branch", zap.String("module", moduleName), zap.String("branch", branchName), zap.String("base", baseRef))
	if out, err := runGit(localRepoPath, logger, moduleName, "checkout", "-f", "-B", branchName, baseRef); err != nil {
		logger.Error("Failed to create module branch", zap.String("module", moduleName), zap.String("branch", branchName), zap.String("output", out), zap.Error(err))
		return err
	}

	// Remove any untracked files/directories left over from a previously synced module so they
	// are not swept into this module's commit by `git add -A`.
	if out, err := runGit(localRepoPath, logger, moduleName, "clean", "-ffd"); err != nil {
		logger.Error("Failed to clean working tree", zap.String("module", moduleName), zap.String("output", out), zap.Error(err))
		return err
	}

	copyModuleToBranch(module, localRepoPath, nameTransformer, logger)

	// Write the version file so the next sync knows which AVM tag was last applied
	writeAvmVersionFile(moduleName, localRepoPath, latestAvmTag, latestAvmCommit, logger)

	// Apply patches if they exist
	if err := applyPatchesIfExist(moduleName, localRepoPath, logger); err != nil {
		logger.Warn("Errors occurred while applying patches, but continuing with commit", zap.String("module", moduleName), zap.Error(err))
	}

	// Rewrite public AVM registry module sources to Artifactory if a template is configured
	if err := rewriteRegistrySourcesToArtifactory(moduleName, localRepoPath, logger); err != nil {
		logger.Warn("Errors occurred while rewriting registry sources, but continuing with commit", zap.String("module", moduleName), zap.Error(err))
	}

	// Stage all module files (respecting .gitattributes/line endings) including deletions.
	logger.Info("Staging changes", zap.String("module", moduleName))
	if out, err := runGit(localRepoPath, logger, moduleName, "add", "-A", "."); err != nil {
		logger.Error("Failed to stage changes", zap.String("module", moduleName), zap.String("output", out), zap.Error(err))
		return err
	}

	// Skip the commit/PR when nothing actually changed for this module.
	statusOut, err := runGit(localRepoPath, logger, moduleName, "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(statusOut) == "" {
		logger.Info("No staged changes to commit", zap.String("module", moduleName))
		return nil
	}

	logger.Info("Committing changes", zap.String("module", moduleName), zap.String("commit_msg", commitMsg))
	if out, err := runGit(localRepoPath, logger, moduleName, "commit", "-m", commitMsg); err != nil {
		logger.Error("Failed to commit changes", zap.String("module", moduleName), zap.String("output", out), zap.Error(err))
		return err
	}

	// Force-push so the remote branch always reflects exactly this module's state, healing any
	// previously polluted branch. In an ADO pipeline persistCredentials handles auth; for local
	// runs a PAT is injected via an http extra header.
	logger.Info("Pushing changes to origin", zap.String("module", moduleName), zap.String("branch", branchName))
	pushArgs := []string{}
	if config.AdoPat != "" {
		authHeader := "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte(":"+config.AdoPat))
		pushArgs = append(pushArgs, "-c", "http.extraheader="+authHeader)
	}
	pushArgs = append(pushArgs, "push", "-f", "origin", "HEAD:refs/heads/"+branchName)
	if out, err := runGit(localRepoPath, logger, moduleName, pushArgs...); err != nil {
		logger.Error("Failed to push changes to origin", zap.String("module", moduleName), zap.String("output", out), zap.Error(err))
		return err
	}
	// Create pull request
	title := buildCommitMessage(moduleName)
	description := "This is an automated pull request to sync the " + moduleName + " module from the source AVM repository " + module.GetRepoURL()
	sourceRef := "refs/heads/" + branchName
	targetRef := "refs/heads/" + config.DefaultBranchName
	pr, err := createPullRequest(clients.GitClient, ctx, repoId, project, sourceRef, targetRef, title, description)
	if err != nil {
		// An active PR for this branch already exists (e.g. on a re-run); the force-push above
		// already updated it, so treat this as success rather than failing the module.
		if strings.Contains(strings.ToLower(err.Error()), "active pull request") {
			logger.Info("Pull request already exists for branch, skipping creation", zap.String("module", moduleName), zap.String("branch", branchName))
			return nil
		}
		logger.Error("Failed to create pull request", zap.String("module", moduleName), zap.Error(err))
		return err
	}
	logger.Info("Created pull request", zap.String("module", moduleName), zap.Int("prId", *pr.PullRequestId))
	return nil
}

// runGit runs a git subcommand in dir, logging the command and combined output on failure.
func runGit(dir string, logger *zap.Logger, moduleName string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("git command failed",
			zap.String("module", moduleName),
			zap.Strings("args", args),
			zap.String("output", string(out)),
			zap.Error(err))
	}
	return string(out), err
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
