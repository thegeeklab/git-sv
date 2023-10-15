package commands

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/thegeeklab/git-sv/v2/app"
	"github.com/thegeeklab/git-sv/v2/sv"
)

func getTagCommits(gsv app.GitSV, tag string) ([]sv.CommitLog, error) {
	prev, _, err := getTags(gsv, tag)
	if err != nil {
		return nil, err
	}

	return gsv.Log(app.NewLogRange(app.TagRange, prev, tag))
}

func getTags(gsv app.GitSV, tag string) (string, app.Tag, error) {
	tags, err := gsv.Tags()
	if err != nil {
		return "", app.Tag{}, err
	}

	index := find(tag, tags)
	if index < 0 {
		return "", app.Tag{}, fmt.Errorf("%w: %s not found, check tag filter", errUnknownTag, tag)
	}

	previousTag := ""
	if index > 0 {
		previousTag = tags[index-1].Name
	}

	return previousTag, tags[index], nil
}

func find(tag string, tags []app.Tag) int {
	for i := 0; i < len(tags); i++ {
		if tag == tags[i].Name {
			return i
		}
	}

	return -1
}

func logRange(gsv app.GitSV, rangeFlag, startFlag, endFlag string) (app.LogRange, error) {
	switch rangeFlag {
	case string(app.TagRange):
		return app.NewLogRange(app.TagRange, str(startFlag, gsv.LastTag()), endFlag), nil
	case string(app.DateRange):
		return app.NewLogRange(app.DateRange, startFlag, endFlag), nil
	case string(app.HashRange):
		return app.NewLogRange(app.HashRange, startFlag, endFlag), nil
	default:
		return app.LogRange{}, fmt.Errorf(
			"%w: %s, expected: %s, %s or %s",
			errInvalidRange,
			rangeFlag,
			app.TagRange,
			app.DateRange,
			app.HashRange,
		)
	}
}

func str(value, defaultValue string) string {
	if value != "" {
		return value
	}

	return defaultValue
}

func getTagVersionInfo(gsv app.GitSV, tag string) (*semver.Version, time.Time, []sv.CommitLog, error) {
	tagVersion, _ := sv.ToVersion(tag)

	previousTag, currentTag, err := getTags(gsv, tag)
	if err != nil {
		return nil, time.Time{}, nil, fmt.Errorf("error listing tags: %w", err)
	}

	commits, err := gsv.Log(app.NewLogRange(app.TagRange, previousTag, tag))
	if err != nil {
		return nil, time.Time{}, nil, fmt.Errorf("error getting git log from tag: %s: %w", tag, err)
	}

	return tagVersion, currentTag.Date, commits, nil
}

func getNextVersionInfo(
	gsv app.GitSV, semverProcessor sv.CommitProcessor,
) (*semver.Version, bool, time.Time, []sv.CommitLog, error) {
	lastTag := gsv.LastTag()

	commits, err := gsv.Log(app.NewLogRange(app.TagRange, lastTag, ""))
	if err != nil {
		return nil, false, time.Time{}, nil, fmt.Errorf("error getting git log: %w", err)
	}

	currentVer, _ := sv.ToVersion(lastTag)
	version, updated := semverProcessor.NextVersion(currentVer, commits)

	return version, updated, time.Now(), commits, nil
}

func getCommitType(cfg *app.Config, p sv.MessageProcessor, input string) (string, error) {
	if input == "" {
		t, err := promptType(cfg.CommitMessage.Types)

		return t.Type, err
	}

	return input, p.ValidateType(input)
}

func getCommitScope(cfg *app.Config, p sv.MessageProcessor, input string, noScope bool) (string, error) {
	if input == "" && !noScope {
		return promptScope(cfg.CommitMessage.Scope.Values)
	}

	return input, p.ValidateScope(input)
}

func getCommitDescription(p sv.MessageProcessor, input string) (string, error) {
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

func getCommitIssue(cfg *app.Config, p sv.MessageProcessor, branch string, noIssue bool) (string, error) {
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
