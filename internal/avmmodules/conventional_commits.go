package avmmodules

import (
	"regexp"
	"sort"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/plumbing/storer"
	"go.uber.org/zap"
)

// conventionalCommitPriority returns a numeric priority for a conventional commit type.
// Higher numbers indicate a more significant change.
func conventionalCommitPriority(t string) int {
	switch t {
	case "breaking":
		return 3
	case "feat":
		return 2
	case "fix":
		return 1
	default:
		return 0
	}
}

// higherConventionalCommitType returns the more significant of two conventional commit types.
func higherConventionalCommitType(a, b string) string {
	if conventionalCommitPriority(b) > conventionalCommitPriority(a) {
		return b
	}
	return a
}

// ensureSemverPrefix returns v with a leading "v" if it doesn't already have one,
// so that tags like "1.2.3" and "v1.2.3" are both accepted by golang.org/x/mod/semver.
func ensureSemverPrefix(v string) string {
	if strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

// footerBreakingRe matches a BREAKING CHANGE or BREAKING-CHANGE footer token with either
// the ": " or " #" separator defined by the Conventional Commits spec.
var footerBreakingRe = regexp.MustCompile(`(?m)^BREAKING[\- ]CHANGE(: | #)`)

// subjectBreakingRe matches a subject line that carries a ! immediately before the colon.
var subjectBreakingRe = regexp.MustCompile(`^[a-zA-Z]+(\([^)]*\))?!:`)

// subjectTypeRe captures the type token from a conventional commit subject line.
var subjectTypeRe = regexp.MustCompile(`^([a-zA-Z]+)(\([^)]*\))?!?:`)

// parseConventionalCommitType parses a single commit message and returns its conventional
// commit type as one of: "breaking", "feat", "fix", or "chore" (catch-all).
//
// Breaking-change detection follows the spec:
//   - footer token BREAKING CHANGE or BREAKING-CHANGE with ": " or " #" separator
//   - "!" immediately before the ":" in the subject line
//
// The footer section is defined as everything after the last blank line in the message,
// so a "BREAKING CHANGE:" string that happens to appear in the commit body is ignored.
func parseConventionalCommitType(msg string) string {
	// Split into subject / body+footer on the first blank line
	parts := strings.SplitN(msg, "\n\n", 2)
	subject := strings.TrimSpace(parts[0])
	footer := ""
	if len(parts) == 2 {
		// Footer is everything after the LAST blank line
		sections := strings.Split(parts[1], "\n\n")
		footer = sections[len(sections)-1]
	}

	// Check footer for BREAKING CHANGE / BREAKING-CHANGE with either separator
	if footerBreakingRe.MatchString(footer) {
		return "breaking"
	}

	// Check subject for ! breaking indicator
	if subjectBreakingRe.MatchString(subject) {
		return "breaking"
	}

	matches := subjectTypeRe.FindStringSubmatch(subject)
	if len(matches) < 2 {
		return "chore"
	}
	switch strings.ToLower(matches[1]) {
	case "feat", "feature":
		return "feat"
	case "fix", "bugfix":
		return "fix"
	default:
		return "chore"
	}
}

// analyzeConventionalCommits opens the cloned AVM repo and walks the commits between the
// last synced tag (read from the module's .avm-version file) and the latest tag.  When no
// version file exists it falls back to the range between the two most recent tags.  Returns
// the highest conventional commit type found ("breaking", "feat", "fix", or "chore") and the
// name of the latest tag so the caller can persist it.  Falls back to ("feat", "") on any
// unrecoverable error.
func analyzeConventionalCommits(repoPath string, lastSyncedTag string, logger *zap.Logger) (commitType string, latestTag string) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		logger.Warn("Could not open cloned repo for commit analysis, defaulting to feat",
			zap.String("path", repoPath), zap.Error(err))
		return "feat", ""
	}

	type tagInfo struct {
		name       string
		commitHash plumbing.Hash
		when       int64 // unix seconds for sorting
	}

	var tags []tagInfo
	tagByName := make(map[string]tagInfo)

	tagIter, err := repo.Tags()
	if err != nil {
		logger.Warn("Could not list tags, defaulting to feat", zap.String("path", repoPath), zap.Error(err))
		return "feat", ""
	}
	_ = tagIter.ForEach(func(ref *plumbing.Reference) error {
		// Annotated tag
		if tagObj, err := repo.TagObject(ref.Hash()); err == nil {
			ti := tagInfo{name: ref.Name().Short(), commitHash: tagObj.Target, when: tagObj.Tagger.When.Unix()}
			tags = append(tags, ti)
			tagByName[ti.name] = ti
			return nil
		}
		// Lightweight tag
		if c, err := repo.CommitObject(ref.Hash()); err == nil {
			ti := tagInfo{name: ref.Name().Short(), commitHash: ref.Hash(), when: c.Author.When.Unix()}
			tags = append(tags, ti)
			tagByName[ti.name] = ti
		}
		return nil
	})

	// Sort tags newest-first by semantic version, falling back to timestamp for non-semver tags.
	sort.Slice(tags, func(i, j int) bool {
		ni, nj := ensureSemverPrefix(tags[i].name), ensureSemverPrefix(tags[j].name)
		if semver.IsValid(ni) && semver.IsValid(nj) {
			return semver.Compare(ni, nj) > 0
		}
		return tags[i].when > tags[j].when
	})

	if len(tags) == 0 {
		head, err := repo.Head()
		if err != nil {
			logger.Warn("No tags and no HEAD, defaulting to feat", zap.String("path", repoPath))
			return "feat", ""
		}
		logger.Info("No tags found, analysing all commits from HEAD", zap.String("path", repoPath))
		return scanCommits(repo, head.Hash(), plumbing.ZeroHash, false, repoPath, logger), ""
	}

	latestTag = tags[0].name
	startHash := tags[0].commitHash
	var stopHash plumbing.Hash
	hasStop := false

	// Prefer the last synced tag as the lower boundary
	if lastSyncedTag != "" {
		if syncedInfo, ok := tagByName[lastSyncedTag]; ok {
			stopHash = syncedInfo.commitHash
			hasStop = true
			logger.Info("Analysing commits since last synced tag",
				zap.String("path", repoPath),
				zap.String("from", lastSyncedTag),
				zap.String("to", latestTag))
		} else {
			logger.Warn("Last synced tag not found in AVM repo, falling back to second-most-recent tag",
				zap.String("path", repoPath),
				zap.String("lastSyncedTag", lastSyncedTag))
		}
	}

	// Fall back to second-most-recent tag when no version file exists
	if !hasStop && len(tags) >= 2 {
		stopHash = tags[1].commitHash
		hasStop = true
		logger.Info("Analysing commits between two most recent tags",
			zap.String("path", repoPath),
			zap.String("newest", tags[0].name),
			zap.String("previous", tags[1].name))
	}

	if !hasStop {
		logger.Info("Only one tag and no version file, analysing all commits up to it",
			zap.String("path", repoPath), zap.String("tag", latestTag))
	}

	return scanCommits(repo, startHash, stopHash, hasStop, repoPath, logger), latestTag
}

// scanCommits walks the commit graph from startHash, stopping at stopHash (when hasStop is true),
// and returns the highest conventional commit type encountered.
func scanCommits(repo *git.Repository, startHash plumbing.Hash, stopHash plumbing.Hash, hasStop bool, repoPath string, logger *zap.Logger) string {
	commitIter, err := repo.Log(&git.LogOptions{From: startHash})
	if err != nil {
		logger.Warn("Could not get commit log, defaulting to feat", zap.String("path", repoPath), zap.Error(err))
		return "feat"
	}

	highestType := "chore"
	commitCount := 0
	_ = commitIter.ForEach(func(c *object.Commit) error {
		if hasStop && c.Hash == stopHash {
			return storer.ErrStop
		}
		commitCount++
		highestType = higherConventionalCommitType(highestType, parseConventionalCommitType(c.Message))
		return nil
	})

	logger.Info("Conventional commit analysis complete",
		zap.String("path", repoPath),
		zap.String("highestType", highestType),
		zap.Int("commitsScanned", commitCount))
	return highestType
}
