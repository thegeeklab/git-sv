package app

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
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

const (
	logSeparator = ">###"
	endLine      = ">~~~"
)

var (
	errUnknownGitError  = errors.New("git command failed")
	errInvalidCommitLog = errors.New("invalid commit log format")
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
	// Open the repository
	repo, err := git.PlainOpen(".")
	if err != nil {
		return ""
	}

	// Get all tag references
	tagRefs, err := repo.Tags()
	if err != nil {
		return ""
	}

	var tags []struct {
		Name string
		Time time.Time
	}

	// Collect all tags with their creation time
	err = tagRefs.ForEach(func(ref *plumbing.Reference) error {
		// Skip tags that don't match the filter
		if *g.Config.Tag.Filter != "" {
			matched, err := filepath.Match(*g.Config.Tag.Filter, ref.Name().Short())
			if err != nil || !matched {
				return nil
			}
		}

		// Try to get the tag object (for annotated tags)
		tagObj, err := repo.TagObject(ref.Hash())
		if err == nil {
			// For annotated tags, use the tagger date
			tags = append(tags, struct {
				Name string
				Time time.Time
			}{
				Name: ref.Name().Short(),
				Time: tagObj.Tagger.When,
			})
			return nil
		}

		// For lightweight tags, try to get the commit
		commit, err := repo.CommitObject(ref.Hash())
		if err != nil {
			// If we can't get the commit, just use the tag name without date info
			tags = append(tags, struct {
				Name string
				Time time.Time
			}{
				Name: ref.Name().Short(),
				Time: time.Time{}, // Zero time
			})
			return nil
		}

		// Use the commit date for lightweight tags
		tags = append(tags, struct {
			Name string
			Time time.Time
		}{
			Name: ref.Name().Short(),
			Time: commit.Committer.When,
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
		return tags[i].Time.Before(tags[j].Time)
	})

	// Return the last tag (highest version or newest)
	return tags[len(tags)-1].Name
}

// Log return git log.
func (g GitSV) Log(lr LogRange) ([]sv.CommitLog, error) {
	// Open the repository
	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}

	// Prepare log options
	logOptions := &git.LogOptions{}

	// Handle different range types
	if lr.rangeType == DateRange {
		// Convert date strings to time.Time
		if lr.start != "" {
			startTime, err := time.Parse("2006-01-02", lr.start)
			if err == nil {
				logOptions.Since = &startTime
			}
		}
		if lr.end != "" {
			endTime, err := time.Parse("2006-01-02", lr.end)
			if err == nil {
				// Add a day to make it inclusive, matching the original behavior
				endTime = endTime.AddDate(0, 0, 1)
				logOptions.Until = &endTime
			}
		}
	} else {
		// For hash/tag ranges, we need to resolve the end revision
		endRevision := plumbing.Revision(str(lr.end, "HEAD"))
		endRef, err := repo.ResolveRevision(endRevision)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve end revision: %w", err)
		}
		logOptions.From = *endRef
	}

	// Get the commit iterator
	iter, err := repo.Log(logOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit log: %w", err)
	}

	// If we have a start revision for hash/tag range, we need to exclude commits reachable from it
	var excludeHashes map[plumbing.Hash]bool
	if lr.start != "" && lr.rangeType != DateRange {
		excludeHashes, err = getCommitHashesFrom(repo, lr.start)
		if err != nil {
			return nil, fmt.Errorf("failed to get excluded commits: %w", err)
		}
	}

	var logs []sv.CommitLog
	err = iter.ForEach(func(c *object.Commit) error {
		// Skip commits that are reachable from the start revision
		if excludeHashes != nil && excludeHashes[c.Hash] {
			return nil
		}

		// Parse the commit message
		message, err := g.MessageProcessor.Parse(c.Message, "")
		if err != nil {
			return nil // Skip commits with parsing errors
		}

		// Format the date as YYYY-MM-DD
		date := c.Author.When.Format("2006-01-02")

		// Create the commit log
		log := sv.CommitLog{
			Date:       date,
			Timestamp:  int(c.Author.When.Unix()),
			AuthorName: c.Author.Name,
			Hash:       c.Hash.String()[:7], // Short hash
			Message:    message,
		}

		logs = append(logs, log)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate through commits: %w", err)
	}

	return logs, nil
}

// isAncestor checks if commit is an ancestor of potentialAncestor
func isAncestor(repo *git.Repository, commit, potentialAncestor plumbing.Hash) (bool, error) {
	// Get the commit object for the potential ancestor
	ancestorCommit, err := repo.CommitObject(potentialAncestor)
	if err != nil {
		return false, err
	}

	// Get the commit object for the commit we're checking
	targetCommit, err := repo.CommitObject(commit)
	if err != nil {
		return false, err
	}

	// Check if the potential ancestor is reachable from the commit
	return ancestorCommit.IsAncestor(targetCommit)
}

// Commit runs git sv.
func (g GitSV) Commit(header, body, footer string) error {
	// Check if all parts are empty
	if header == "" && body == "" && footer == "" {
		return errors.New("commit message cannot be empty")
	}

	// Open the repository
	repo, err := git.PlainOpen(".")
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Construct the full commit message with proper spacing
	commitMsg := header
	if body != "" {
		commitMsg += "\n\n" + body
	}
	if footer != "" {
		commitMsg += "\n\n" + footer
	}

	// Commit the changes
	_, err = worktree.Commit(commitMsg, &git.CommitOptions{
		All: true, // Stage all modified files
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

	// Open the repository
	repo, err := git.PlainOpen(".")
	if err != nil {
		return tag, fmt.Errorf("failed to open git repository: %w", err)
	}

	// Get the HEAD reference
	head, err := repo.Head()
	if err != nil {
		return tag, fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	// Create the tag
	var tagOpts *git.CreateTagOptions
	if annotate {
		// Create an annotated tag with a message
		tagOpts = &git.CreateTagOptions{
			Message: tagMsg,
		}
	}

	// Create the tag pointing to the current HEAD
	_, err = repo.CreateTag(tag, head.Hash(), tagOpts)
	if err != nil {
		return tag, fmt.Errorf("failed to create tag: %w", err)
	}

	// If local is true, don't push the tag
	if local {
		return tag, nil
	}

	// Push the tag to the remote
	remote, err := repo.Remote("origin")
	if err != nil {
		return tag, fmt.Errorf("failed to get remote: %w", err)
	}

	// Create the refspec for the tag
	refspec := fmt.Sprintf("refs/tags/%s:refs/tags/%s", tag, tag)

	// Push the tag
	err = remote.Push(&git.PushOptions{
		RefSpecs: []config.RefSpec{config.RefSpec(refspec)},
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return tag, fmt.Errorf("failed to push tag: %w", err)
	}

	return tag, nil
}

// Tags list repository tags.
func (g GitSV) Tags() ([]Tag, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, err
	}

	// Get all references
	refs, err := repo.References()
	if err != nil {
		return nil, err
	}

	var tags []Tag
	// Filter for tag references and apply the filter pattern
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		// Check if it's a tag reference
		if !ref.Name().IsTag() {
			return nil
		}

		tagName := ref.Name().Short()

		// Apply the filter pattern if specified
		if *g.Config.Tag.Filter != "" {
			matched, err := filepath.Match(*g.Config.Tag.Filter, tagName)
			if err != nil || !matched {
				return nil
			}
		}

		// Get the tag object or commit to extract the date
		var tagDate time.Time
		tagObj, err := repo.TagObject(ref.Hash())
		if err == nil {
			// Annotated tag
			tagDate = tagObj.Tagger.When
		} else {
			// Lightweight tag - get the commit
			commit, err := repo.CommitObject(ref.Hash())
			if err != nil {
				// Skip if we can't get date information
				return nil
			}
			tagDate = commit.Committer.When
		}

		tags = append(tags, Tag{
			Name: tagName,
			Date: tagDate,
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort tags by date (oldest first) to match the original behavior
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

func parseTagsOutput(input string) ([]Tag, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))

	var result []Tag

	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			values := strings.Split(line, "#")
			date, _ := time.Parse("2006-01-02 15:04:05 -0700", values[0]) // ignore invalid dates
			result = append(result, Tag{Name: values[1], Date: date})
		}
	}

	return result, nil
}

func parseLogOutput(messageProcessor sv.MessageProcessor, log string) ([]sv.CommitLog, error) {
	scanner := bufio.NewScanner(strings.NewReader(log))
	scanner.Split(splitAt([]byte(endLine)))

	var logs []sv.CommitLog

	for scanner.Scan() {
		if text := strings.TrimSpace(strings.Trim(scanner.Text(), "\"")); text != "" {
			log, err := parseCommitLog(messageProcessor, text)
			if err != nil {
				return nil, err
			}

			logs = append(logs, log)
		}
	}

	return logs, nil
}

func parseCommitLog(messageProcessor sv.MessageProcessor, c string) (sv.CommitLog, error) {
	logFieldCount := 6
	content := strings.Split(strings.Trim(c, "\""), logSeparator)

	if len(content) < logFieldCount {
		return sv.CommitLog{}, fmt.Errorf("%w: missing required fields", errInvalidCommitLog)
	}

	timestamp, _ := strconv.Atoi(content[1])

	message, err := messageProcessor.Parse(content[4], content[5])
	if err != nil {
		return sv.CommitLog{}, err
	}

	return sv.CommitLog{
		Date:       content[0],
		Timestamp:  timestamp,
		AuthorName: content[2],
		Hash:       content[3],
		Message:    message,
	}, nil
}

func splitAt(b []byte) func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) { //nolint:nonamedreturns
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		if i := bytes.Index(data, b); i >= 0 {
			return i + len(b), data[0:i], nil
		}

		if atEOF {
			return len(data), data, nil
		}

		return 0, nil, nil
	}
}

func addDay(value string) string {
	if value == "" {
		return value
	}

	t, err := time.Parse("2006-01-02", value)
	if err != nil { // keep original value if is not date format
		return value
	}

	return t.AddDate(0, 0, 1).Format("2006-01-02")
}

func str(value, defaultValue string) string {
	if value != "" {
		return value
	}

	return defaultValue
}

// getCommitHashesFrom returns a map of commit hashes reachable from the given revision
func getCommitHashesFrom(repo *git.Repository, revision string) (map[plumbing.Hash]bool, error) {
	startRef, err := repo.ResolveRevision(plumbing.Revision(revision))
	if err != nil {
		return nil, err
	}

	// Get all commits reachable from the start revision
	iter, err := repo.Log(&git.LogOptions{From: *startRef})
	if err != nil {
		return nil, err
	}

	hashes := make(map[plumbing.Hash]bool)
	err = iter.ForEach(func(c *object.Commit) error {
		hashes[c.Hash] = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	return hashes, nil
}
