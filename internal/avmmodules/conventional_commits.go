package avmmodules

import (
	"sort"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"go.uber.org/zap"
)

// ensureSemverPrefix returns v with a leading "v" if it doesn't already have one,
// so that tags like "1.2.3" and "v1.2.3" are both accepted by golang.org/x/mod/semver.
func ensureSemverPrefix(v string) string {
	if strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

// findLatestAvmTag opens the cloned AVM repo and returns the name of the most recent tag
// (ordered by semantic version, falling back to the tag's timestamp for non-semver tags)
// along with the commit hash that tag points to.  Returns ("", "") when the repository has
// no tags or cannot be read.  The published innersource version is kept in lock-step with
// this upstream tag, so no commit-message analysis is performed.
func findLatestAvmTag(repoPath string, logger *zap.Logger) (latestTag string, latestTagCommit string) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		logger.Warn("Could not open cloned repo to find latest tag", zap.String("path", repoPath), zap.Error(err))
		return "", ""
	}

	type tagInfo struct {
		name       string
		commitHash plumbing.Hash
		when       int64 // unix seconds for sorting non-semver tags
	}

	var tags []tagInfo

	tagIter, err := repo.Tags()
	if err != nil {
		logger.Warn("Could not list tags", zap.String("path", repoPath), zap.Error(err))
		return "", ""
	}
	_ = tagIter.ForEach(func(ref *plumbing.Reference) error {
		// Annotated tag
		if tagObj, err := repo.TagObject(ref.Hash()); err == nil {
			tags = append(tags, tagInfo{name: ref.Name().Short(), commitHash: tagObj.Target, when: tagObj.Tagger.When.Unix()})
			return nil
		}
		// Lightweight tag
		if c, err := repo.CommitObject(ref.Hash()); err == nil {
			tags = append(tags, tagInfo{name: ref.Name().Short(), commitHash: ref.Hash(), when: c.Author.When.Unix()})
		}
		return nil
	})

	if len(tags) == 0 {
		logger.Info("No tags found in AVM repo", zap.String("path", repoPath))
		return "", ""
	}

	// Sort tags newest-first by semantic version, falling back to timestamp for non-semver tags.
	sort.Slice(tags, func(i, j int) bool {
		ni, nj := ensureSemverPrefix(tags[i].name), ensureSemverPrefix(tags[j].name)
		if semver.IsValid(ni) && semver.IsValid(nj) {
			return semver.Compare(ni, nj) > 0
		}
		return tags[i].when > tags[j].when
	})

	logger.Info("Found latest AVM tag",
		zap.String("path", repoPath),
		zap.String("tag", tags[0].name),
		zap.String("commit", tags[0].commitHash.String()))
	return tags[0].name, tags[0].commitHash.String()
}
