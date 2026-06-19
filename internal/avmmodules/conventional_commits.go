package avmmodules

import (
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"

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

// findLatestAvmTag runs git in the cloned AVM repo and returns the name of the most recent tag
// (ordered by semantic version, falling back to the tag's timestamp for non-semver tags) along
// with the commit hash that tag points to.  Returns ("", "") when the repository has no tags or
// git fails.  The published innersource version is kept in lock-step with this upstream tag, so
// no commit-message analysis is performed.
func findLatestAvmTag(repoPath string, logger *zap.Logger) (latestTag string, latestTagCommit string) {
	// %(*objectname) is the dereferenced commit for annotated tags (empty for lightweight tags).
	cmd := exec.Command("git", "for-each-ref",
		"--format=%(refname:short)%09%(objectname)%09%(*objectname)%09%(creatordate:unix)",
		"refs/tags")
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warn("Could not list tags to find latest", zap.String("path", repoPath), zap.String("output", string(out)), zap.Error(err))
		return "", ""
	}

	type tagInfo struct {
		name   string
		commit string
		when   int64 // unix seconds for sorting non-semver tags
	}

	var tags []tagInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 4 {
			continue
		}
		commit := fields[2] // dereferenced commit for annotated tags
		if commit == "" {
			commit = fields[1] // lightweight tag points straight at the commit
		}
		when, _ := strconv.ParseInt(fields[3], 10, 64)
		tags = append(tags, tagInfo{name: fields[0], commit: commit, when: when})
	}

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
		zap.String("commit", tags[0].commit))
	return tags[0].name, tags[0].commit
}
