package commands

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/thegeeklab/git-sv/v2/pkg/config"
	"github.com/thegeeklab/git-sv/v2/pkg/git"
)

func getTagCommits(gsv git.SV, tag string) ([]git.CommitLog, error) {
	prev, _, err := getTags(gsv, tag)
	if err != nil {
		return nil, err
	}

	return gsv.Log(git.NewLogRange(git.TagRange, prev, tag))
}

func getTags(gsv git.SV, tag string) (string, git.Tag, error) {
	tags, err := gsv.Tags()
	if err != nil {
		return "", git.Tag{}, err
	}

	index := find(tag, tags)
	if index < 0 {
		return "", git.Tag{}, fmt.Errorf("%w: %s not found, check tag filter", errUnknownTag, tag)
	}

	previousTag := ""
	if index > 0 {
		previousTag = tags[index-1].Name
	}

	return previousTag, tags[index], nil
}

func find(tag string, tags []git.Tag) int {
	for i := 0; i < len(tags); i++ {
		if tag == tags[i].Name {
			return i
		}
	}

	return -1
}

func logRange(gsv git.SV, rangeFlag, startFlag, endFlag string) (git.LogRange, error) {
	switch rangeFlag {
	case string(git.TagRange):
		return git.NewLogRange(git.TagRange, str(startFlag, gsv.LastTag()), endFlag), nil
	case string(git.DateRange):
		return git.NewLogRange(git.DateRange, startFlag, endFlag), nil
	case string(git.HashRange):
		return git.NewLogRange(git.HashRange, startFlag, endFlag), nil
	default:
		return git.LogRange{}, fmt.Errorf(
			"%w: %s, expected: %s, %s or %s",
			errInvalidRange,
			rangeFlag,
			git.TagRange,
			git.DateRange,
			git.HashRange,
		)
	}
}

func str(value, defaultValue string) string {
	if value != "" {
		return value
	}

	return defaultValue
}

func getTagVersionInfo(gsv git.SV, tag string) (*semver.Version, time.Time, []git.CommitLog, error) {
	tagVersion, _ := git.ToVersion(tag)

	previousTag, currentTag, err := getTags(gsv, tag)
	if err != nil {
		return nil, time.Time{}, nil, fmt.Errorf("error listing tags, message: %w", err)
	}

	commits, err := gsv.Log(git.NewLogRange(git.TagRange, previousTag, tag))
	if err != nil {
		return nil, time.Time{}, nil, fmt.Errorf("error getting git log from tag: %s, message: %w", tag, err)
	}

	return tagVersion, currentTag.Date, commits, nil
}

func getNextVersionInfo(
	gsv git.SV, semverProcessor git.CommitsProcessor,
) (*semver.Version, bool, time.Time, []git.CommitLog, error) {
	lastTag := gsv.LastTag()

	commits, err := gsv.Log(git.NewLogRange(git.TagRange, lastTag, ""))
	if err != nil {
		return nil, false, time.Time{}, nil, fmt.Errorf("error getting git log, message: %w", err)
	}

	currentVer, _ := git.ToVersion(lastTag)
	version, updated := semverProcessor.NextVersion(currentVer, commits)

	return version, updated, time.Now(), commits, nil
}

func getCommitType(cfg *config.Config, p git.MessageProcessor, input string) (string, error) {
	if input == "" {
		t, err := promptType(cfg.CommitMessage.Types)

		return t.Type, err
	}

	return input, p.ValidateType(input)
}

func getCommitScope(cfg *config.Config, p git.MessageProcessor, input string, noScope bool) (string, error) {
	if input == "" && !noScope {
		return promptScope(cfg.CommitMessage.Scope.Values)
	}

	return input, p.ValidateScope(input)
}

func getCommitDescription(p git.MessageProcessor, input string) (string, error) {
	if input == "" {
		return promptSubject()
	}

	return input, p.ValidateDescription(input)
}

func getCommitBody(noBody bool) (string, error) {
	if noBody {
		return "", nil
	}

	var fullBody strings.Builder

	for body, err := promptBody(); body != "" || err != nil; body, err = promptBody() {
		if err != nil {
			return "", err
		}

		if fullBody.Len() > 0 {
			fullBody.WriteString("\n")
		}

		if body != "" {
			fullBody.WriteString(body)
		}
	}

	return fullBody.String(), nil
}

func getCommitIssue(cfg *config.Config, p git.MessageProcessor, branch string, noIssue bool) (string, error) {
	branchIssue, err := p.IssueID(branch)
	if err != nil {
		return "", err
	}

	if cfg.CommitMessage.IssueFooterConfig().Key == "" || cfg.CommitMessage.Issue.Regex == "" {
		return "", nil
	}

	if noIssue {
		return branchIssue, nil
	}

	return promptIssueID("issue id", cfg.CommitMessage.Issue.Regex, branchIssue)
}

func getCommitBreakingChange(noBreaking bool, input string) (string, error) {
	if noBreaking {
		return "", nil
	}

	if strings.TrimSpace(input) != "" {
		return input, nil
	}

	hasBreakingChanges, err := promptConfirm("has breaking change?")
	if err != nil {
		return "", err
	}

	if !hasBreakingChanges {
		return "", nil
	}

	return promptBreakingChanges()
}

func readFile(filepath string) (string, error) {
	f, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	return string(f), nil
}

func appendOnFile(message, filepath string, permissions fs.FileMode) error {
	f, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, permissions)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(message)

	return err
}
