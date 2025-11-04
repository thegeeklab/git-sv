package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/thegeeklab/git-sv/sv"
	"github.com/thegeeklab/git-sv/sv/formatter"
	"github.com/thegeeklab/git-sv/templates"
)

var (
	ErrNoDataFound        = errors.New("no data found in repository")
	ErrCommitMessageEmpty = errors.New("commit message is empty")
)

// Tag git tag info.
type Tag struct {
	Name string
	Date time.Time
}

// LogRangeType type of log range.
type LogRangeType string

// constants for log range type.
const (
	TagRange  LogRangeType = "tag"
	DateRange LogRangeType = "date"
	HashRange LogRangeType = "hash"
)

// LogRange git log range.
type LogRange struct {
	rangeType LogRangeType
	start     string
	end       string
}

// NewLogRange LogRange constructor.
func NewLogRange(t LogRangeType, start, end string) LogRange {
	return LogRange{rangeType: t, start: start, end: end}
}

// Impl git command implementation.
type GitSV struct {
	Settings *Settings
	Config   *Config

	MessageProcessor      sv.MessageProcessor
	CommitProcessor       sv.CommitProcessor
	ReleasenotesProcessor sv.ReleaseNoteProcessor
	OutputFormatter       formatter.OutputFormatter
}

// New constructor.
func New() GitSV {
	configDir := ".gitsv"
	configFilenames := []string{"config.yaml", "config.yml"}

	g := GitSV{
		Settings: &Settings{},
		Config:   NewConfig(configDir, configFilenames),
	}

	g.MessageProcessor = sv.NewMessageProcessor(g.Config.CommitMessage, g.Config.Branches)
	g.CommitProcessor = sv.NewSemVerCommitProcessor(g.Config.Versioning, g.Config.CommitMessage)
	g.ReleasenotesProcessor = sv.NewReleaseNoteProcessor(g.Config.ReleaseNotes)
	g.OutputFormatter = formatter.NewOutputFormatter(templates.New(configDir))

	return g
}

// LastTag get last tag, if no tag found, return empty.
func (g GitSV) LastTag() string {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return ""
	}

	tagRefs, err := repo.Tags()
	if err != nil {
		return ""
	}

	var tags []Tag

	// Collect all tags with their creation time
	err = tagRefs.ForEach(func(ref *plumbing.Reference) error {
		tagName := ref.Name().Short()

		// Skip tags that don't match the filter
		if *g.Config.Tag.Filter != "" {
			matched, err := filepath.Match(*g.Config.Tag.Filter, ref.Name().Short())
			if err != nil {
				return err
			}

			if !matched {
				return nil
			}
		}

		// Get tag date (try annotated tag first, then commit)
		var tagDate time.Time

		// Try to get the tag object (for annotated tags)
		tagObj, err := repo.TagObject(ref.Hash())
		if err == nil {
			tagDate = tagObj.Tagger.When
		} else {
			// For lightweight tags, try to get the commit
			// If we can't get a date, it's ok - we'll use zero time
			commit, err := repo.CommitObject(ref.Hash())
			if err == nil {
				tagDate = commit.Committer.When
			}
		}

		tags = append(tags, Tag{
			Name: tagName,
			Date: tagDate,
		})

		return nil
	})
	if err != nil || len(tags) == 0 {
		return ""
	}

	// Sort tags by version (if they are semver) and creation date
	sort.Slice(tags, func(i, j int) bool {
		// Try to parse as semver first
		vi, errI := semver.NewVersion(tags[i].Name)
		vj, errJ := semver.NewVersion(tags[j].Name)

		// If both are valid semver, compare by version
		if errI == nil && errJ == nil {
			return vi.LessThan(vj)
		}

		// Otherwise, compare by date (newer first)
		return tags[i].Date.Before(tags[j].Date)
	})

	// Return the last tag (highest version or newest)
	return tags[len(tags)-1].Name
}

// Log return git log.
func (g GitSV) Log(lr LogRange) ([]sv.CommitLog, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}

	logOptions := &git.LogOptions{}

	if err := configureLogOptions(repo, lr, logOptions); err != nil {
		return nil, err
	}

	iter, err := repo.Log(logOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit log: %w", err)
	}

	excludeHashes, err := getExcludeHashes(repo, lr)
	if err != nil && !errors.Is(err, ErrNoDataFound) {
		return nil, err
	}

	var logs []sv.CommitLog

	err = iter.ForEach(func(c *object.Commit) error {
		// Skip excluded commits
		if excludeHashes != nil && excludeHashes[c.Hash] {
			return nil
		}

		// Split the commit message into subject and body
		subject, body := splitCommitMessage(c.Message)

		// Parse the commit message
		message, err := g.MessageProcessor.Parse(subject, body)
		if err != nil {
			return err
		}

		// Add commit to logs
		logs = append(logs, sv.CommitLog{
			Date:       c.Author.When.Format("2006-01-02"),
			Timestamp:  int(c.Author.When.Unix()),
			AuthorName: c.Author.Name,
			Hash:       c.Hash.String()[:7],
			Message:    message,
		})

		return nil
	})
	if err != nil {
		if isShallowClone(repo) && isObjectNotFoundError(err) {
			return logs, nil
		}

		return nil, fmt.Errorf("failed to iterate through commits: %w", err)
	}

	return logs, nil
}

// Commit runs git sv.
func (g GitSV) Commit(header, body, footer string) error {
	if header == "" && body == "" && footer == "" {
		return ErrCommitMessageEmpty
	}

	repo, err := git.PlainOpen(".")
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	commitMsg := header
	if body != "" {
		commitMsg += "\n\n" + body
	}

	if footer != "" {
		commitMsg += "\n\n" + footer
	}

	_, err = worktree.Commit(commitMsg, &git.CommitOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// Tag create a git tag.
func (g GitSV) Tag(version semver.Version, annotate, local bool) (string, error) {
	tag := fmt.Sprintf(*g.Config.Tag.Pattern, version.Major(), version.Minor(), version.Patch())
	tagMsg := fmt.Sprintf("Version %d.%d.%d", version.Major(), version.Minor(), version.Patch())

	repo, err := git.PlainOpen(".")
	if err != nil {
		return tag, fmt.Errorf("failed to open git repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return tag, fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	var tagOpts *git.CreateTagOptions
	if annotate {
		// Create an annotated tag with a message
		tagOpts = &git.CreateTagOptions{
			Message: tagMsg,
		}
	}

	_, err = repo.CreateTag(tag, head.Hash(), tagOpts)
	if err != nil {
		return tag, fmt.Errorf("failed to create tag: %w", err)
	}

	if local {
		return tag, nil
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		return tag, fmt.Errorf("failed to get remote: %w", err)
	}

	refSpec := fmt.Sprintf("refs/tags/%s:refs/tags/%s", tag, tag)

	err = remote.Push(&git.PushOptions{
		RefSpecs: []config.RefSpec{config.RefSpec(refSpec)},
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return tag, fmt.Errorf("failed to push tag: %w", err)
	}

	return tag, nil
}

// Tags list repository tags.
func (g GitSV) Tags() ([]Tag, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}

	tagIter, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository tags: %w", err)
	}

	var tags []Tag

	// Process tag references directly instead of getting all references first
	err = tagIter.ForEach(func(ref *plumbing.Reference) error {
		tagName := ref.Name().Short()

		// Apply the filter pattern if specified
		if *g.Config.Tag.Filter != "" {
			matched, err := filepath.Match(*g.Config.Tag.Filter, tagName)
			if err != nil {
				return fmt.Errorf("invalid tag filter pattern: %w", err)
			}

			if !matched {
				return nil
			}
		}

		var tagDate time.Time

		// Try to get annotated tag first
		tagObj, err := repo.TagObject(ref.Hash())
		if err == nil {
			tagDate = tagObj.Tagger.When
		} else {
			// For lightweight tags, get the commit
			commit, err := repo.CommitObject(ref.Hash())
			// If we can't get a date, use zero time but don't fail
			if err == nil {
				tagDate = commit.Committer.When
			}
		}

		tags = append(tags, Tag{
			Name: tagName,
			Date: tagDate,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error processing tags: %w", err)
	}

	// Sort tags by date (oldest first)
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Date.Before(tags[j].Date)
	})

	return tags, nil
}

// Branch get git branch.
func (g GitSV) Branch() string {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return ""
	}

	head, err := repo.Head()
	if err != nil {
		return ""
	}

	if !head.Name().IsBranch() {
		return ""
	}

	return head.Name().Short()
}

// IsDetached check if is detached.
func (g GitSV) IsDetached() (bool, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return false, err
	}

	head, err := repo.Head()
	if err != nil {
		return false, err
	}

	if head.Name().IsBranch() {
		return false, nil
	}

	return true, nil
}

func str(value, defaultValue string) string {
	if value != "" {
		return value
	}

	return defaultValue
}

// getCommitHashesFrom returns a map of commit hashes reachable from the given revision.
func getCommitHashesFrom(repo *git.Repository, revision string) (map[plumbing.Hash]bool, error) {
	hashes := make(map[plumbing.Hash]bool)

	startRef, err := repo.ResolveRevision(plumbing.Revision(revision))
	if err != nil {
		if isShallowClone(repo) && isObjectNotFoundError(err) {
			return hashes, nil
		}

		return nil, err
	}

	// Get all commits reachable from the start revision
	iter, err := repo.Log(&git.LogOptions{From: *startRef})
	if err != nil {
		// For shallow clones with missing objects, return empty map
		if isShallowClone(repo) && isObjectNotFoundError(err) {
			return hashes, nil
		}

		return nil, err
	}

	err = iter.ForEach(func(c *object.Commit) error {
		hashes[c.Hash] = true

		return nil
	})
	if err != nil {
		if isShallowClone(repo) && isObjectNotFoundError(err) {
			return hashes, nil
		}

		return nil, err
	}

	return hashes, nil
}

// isObjectNotFoundError checks if the error is due to an object not found.
func isObjectNotFoundError(err error) bool {
	return err != nil && (errors.Is(err, plumbing.ErrObjectNotFound) ||
		errors.Is(err, plumbing.ErrReferenceNotFound))
}

// isShallowClone checks if the repository is a shallow clone.
func isShallowClone(repo *git.Repository) bool {
	_, err := repo.Storer.Reference(plumbing.ReferenceName("shallow"))

	return err == nil
}

// configureLogOptions sets up the git.LogOptions based on the LogRange.
func configureLogOptions(repo *git.Repository, lr LogRange, options *git.LogOptions) error {
	if lr.rangeType == DateRange {
		configureDateRangeOptions(lr, options)

		return nil
	}

	// For hash/tag ranges, resolve the end revision
	endRevision := plumbing.Revision(str(lr.end, "HEAD"))

	endRef, err := repo.ResolveRevision(endRevision)
	if err != nil {
		return fmt.Errorf("failed to resolve end revision: %w", err)
	}

	options.From = *endRef

	return nil
}

// configureDateRangeOptions configures git.LogOptions for date-based ranges.
func configureDateRangeOptions(lr LogRange, options *git.LogOptions) {
	if lr.start != "" {
		startTime, err := time.Parse("2006-01-02", lr.start)
		if err == nil {
			options.Since = &startTime
		}
	}

	if lr.end != "" {
		endTime, err := time.Parse("2006-01-02", lr.end)
		if err == nil {
			// Add a day to make it inclusive
			endTime = endTime.AddDate(0, 0, 1)
			options.Until = &endTime
		}
	}
}

// getExcludeHashes returns commits to exclude based on the start revision.
func getExcludeHashes(repo *git.Repository, lr LogRange) (map[plumbing.Hash]bool, error) {
	if lr.start == "" || lr.rangeType == DateRange {
		return nil, ErrNoDataFound
	}

	excludeHashes, err := getCommitHashesFrom(repo, lr.start)
	if err != nil {
		return nil, fmt.Errorf("failed to get excluded commits: %w", err)
	}

	return excludeHashes, nil
}

// splitCommitMessage separates a commit message into subject and body.
// It returns the first line as subject and the rest (if any) as body.
func splitCommitMessage(message string) (string, string) {
	message = strings.TrimRight(message, "\n")
	subject, body, _ := strings.Cut(message, "\n")

	return strings.TrimSpace(subject), strings.TrimSpace(body)
}
