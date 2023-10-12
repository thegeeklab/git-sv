package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/thegeeklab/git-sv/v2/sv"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

const laxFilePerm = 0o644

var (
	errCanNotCreateTagFlag = errors.New("cannot define tag flag with range, start or end flags")
	errUnknownTag          = errors.New("unknown tag")
	errReadCommitMessage   = errors.New("failed to read commit message")
	errAppendFooter        = errors.New("failed to append meta-informations on footer")
	errInvalidRange        = errors.New("invalid log range")
)

func configDefaultHandler() func(c *cli.Context) error {
	cfg := defaultConfig()

	return func(c *cli.Context) error {
		content, err := yaml.Marshal(&cfg)
		if err != nil {
			return err
		}

		fmt.Println(string(content))

		return nil
	}
}

func configShowHandler(cfg Config) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		content, err := yaml.Marshal(&cfg)
		if err != nil {
			return err
		}

		fmt.Println(string(content))

		return nil
	}
}

func currentVersionHandler(git sv.Git) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		lastTag := git.LastTag()

		currentVer, err := sv.ToVersion(lastTag)
		if err != nil {
			return fmt.Errorf("error parsing version: %s from git tag, message: %w", lastTag, err)
		}

		fmt.Printf("%d.%d.%d\n", currentVer.Major(), currentVer.Minor(), currentVer.Patch())

		return nil
	}
}

func nextVersionHandler(git sv.Git, semverProcessor sv.SemVerCommitsProcessor) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		lastTag := git.LastTag()

		currentVer, err := sv.ToVersion(lastTag)
		if err != nil {
			return fmt.Errorf("error parsing version: %s from git tag, message: %w", lastTag, err)
		}

		commits, err := git.Log(sv.NewLogRange(sv.TagRange, lastTag, ""))
		if err != nil {
			return fmt.Errorf("error getting git log, message: %w", err)
		}

		nextVer, _ := semverProcessor.NextVersion(currentVer, commits)

		fmt.Printf("%d.%d.%d\n", nextVer.Major(), nextVer.Minor(), nextVer.Patch())

		return nil
	}
}

func commitLogFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "t",
			Aliases: []string{"tag"},
			Usage:   "get commit log from a specific tag",
		},
		&cli.StringFlag{
			Name:    "r",
			Aliases: []string{"range"},
			Usage:   "type of range of commits, use: tag, date or hash",
			Value:   string(sv.TagRange),
		},
		&cli.StringFlag{
			Name:    "s",
			Aliases: []string{"start"},
			Usage:   "start range of git log revision range, if date, the value is used on since flag instead",
		},
		&cli.StringFlag{
			Name:    "e",
			Aliases: []string{"end"},
			Usage:   "end range of git log revision range, if date, the value is used on until flag instead",
		},
	}
}

func commitLogHandler(git sv.Git) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		var (
			commits []sv.GitCommitLog
			err     error
		)

		tagFlag := c.String("t")
		rangeFlag := c.String("r")
		startFlag := c.String("s")
		endFlag := c.String("e")

		if tagFlag != "" && (rangeFlag != string(sv.TagRange) || startFlag != "" || endFlag != "") {
			return errCanNotCreateTagFlag
		}

		if tagFlag != "" {
			commits, err = getTagCommits(git, tagFlag)
		} else {
			r, rerr := logRange(git, rangeFlag, startFlag, endFlag)
			if rerr != nil {
				return rerr
			}
			commits, err = git.Log(r)
		}

		if err != nil {
			return fmt.Errorf("error getting git log, message: %w", err)
		}

		for _, commit := range commits {
			content, err := json.Marshal(commit)
			if err != nil {
				return err
			}

			fmt.Println(string(content))
		}

		return nil
	}
}

func getTagCommits(git sv.Git, tag string) ([]sv.GitCommitLog, error) {
	prev, _, err := getTags(git, tag)
	if err != nil {
		return nil, err
	}

	return git.Log(sv.NewLogRange(sv.TagRange, prev, tag))
}

func logRange(git sv.Git, rangeFlag, startFlag, endFlag string) (sv.LogRange, error) {
	switch rangeFlag {
	case string(sv.TagRange):
		return sv.NewLogRange(sv.TagRange, str(startFlag, git.LastTag()), endFlag), nil
	case string(sv.DateRange):
		return sv.NewLogRange(sv.DateRange, startFlag, endFlag), nil
	case string(sv.HashRange):
		return sv.NewLogRange(sv.HashRange, startFlag, endFlag), nil
	default:
		return sv.LogRange{}, fmt.Errorf(
			"%w: %s, expected: %s, %s or %s",
			errInvalidRange,
			rangeFlag,
			sv.TagRange,
			sv.DateRange,
			sv.HashRange,
		)
	}
}

func commitNotesFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name: "r", Aliases: []string{"range"},
			Usage:    "type of range of commits, use: tag, date or hash",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "s",
			Aliases: []string{"start"},
			Usage:   "start range of git log revision range, if date, the value is used on since flag instead",
		},
		&cli.StringFlag{
			Name:    "e",
			Aliases: []string{"end"},
			Usage:   "end range of git log revision range, if date, the value is used on until flag instead",
		},
	}
}

func commitNotesHandler(
	git sv.Git, rnProcessor sv.ReleaseNoteProcessor, outputFormatter sv.OutputFormatter,
) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		var date time.Time

		rangeFlag := c.String("r")

		lr, err := logRange(git, rangeFlag, c.String("s"), c.String("e"))
		if err != nil {
			return err
		}

		commits, err := git.Log(lr)
		if err != nil {
			return fmt.Errorf("error getting git log from range: %s, message: %w", rangeFlag, err)
		}

		if len(commits) > 0 {
			date, _ = time.Parse("2006-01-02", commits[0].Date)
		}

		output, err := outputFormatter.FormatReleaseNote(rnProcessor.Create(nil, "", date, commits))
		if err != nil {
			return fmt.Errorf("could not format release notes, message: %w", err)
		}

		fmt.Println(output)

		return nil
	}
}

func releaseNotesFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "t",
			Aliases: []string{"tag"},
			Usage:   "get release note from tag",
		},
	}
}

func releaseNotesHandler(
	git sv.Git,
	semverProcessor sv.SemVerCommitsProcessor,
	rnProcessor sv.ReleaseNoteProcessor,
	outputFormatter sv.OutputFormatter,
) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		var (
			commits   []sv.GitCommitLog
			rnVersion *semver.Version
			tag       string
			date      time.Time
			err       error
		)

		if tag = c.String("t"); tag != "" {
			rnVersion, date, commits, err = getTagVersionInfo(git, tag)
		} else {
			// TODO: should generate release notes if version was not updated?
			rnVersion, _, date, commits, err = getNextVersionInfo(git, semverProcessor)
		}

		if err != nil {
			return err
		}

		releasenote := rnProcessor.Create(rnVersion, tag, date, commits)

		output, err := outputFormatter.FormatReleaseNote(releasenote)
		if err != nil {
			return fmt.Errorf("could not format release notes, message: %w", err)
		}

		fmt.Println(output)

		return nil
	}
}

func getTagVersionInfo(git sv.Git, tag string) (*semver.Version, time.Time, []sv.GitCommitLog, error) {
	tagVersion, _ := sv.ToVersion(tag)

	previousTag, currentTag, err := getTags(git, tag)
	if err != nil {
		return nil, time.Time{}, nil, fmt.Errorf("error listing tags, message: %w", err)
	}

	commits, err := git.Log(sv.NewLogRange(sv.TagRange, previousTag, tag))
	if err != nil {
		return nil, time.Time{}, nil, fmt.Errorf("error getting git log from tag: %s, message: %w", tag, err)
	}

	return tagVersion, currentTag.Date, commits, nil
}

func getTags(git sv.Git, tag string) (string, sv.GitTag, error) {
	tags, err := git.Tags()
	if err != nil {
		return "", sv.GitTag{}, err
	}

	index := find(tag, tags)
	if index < 0 {
		return "", sv.GitTag{}, fmt.Errorf("%w: %s not found, check tag filter", errUnknownTag, tag)
	}

	previousTag := ""
	if index > 0 {
		previousTag = tags[index-1].Name
	}

	return previousTag, tags[index], nil
}

func find(tag string, tags []sv.GitTag) int {
	for i := 0; i < len(tags); i++ {
		if tag == tags[i].Name {
			return i
		}
	}

	return -1
}

func getNextVersionInfo(
	git sv.Git, semverProcessor sv.SemVerCommitsProcessor,
) (*semver.Version, bool, time.Time, []sv.GitCommitLog, error) {
	lastTag := git.LastTag()

	commits, err := git.Log(sv.NewLogRange(sv.TagRange, lastTag, ""))
	if err != nil {
		return nil, false, time.Time{}, nil, fmt.Errorf("error getting git log, message: %w", err)
	}

	currentVer, _ := sv.ToVersion(lastTag)
	version, updated := semverProcessor.NextVersion(currentVer, commits)

	return version, updated, time.Now(), commits, nil
}

func tagHandler(git sv.Git, semverProcessor sv.SemVerCommitsProcessor) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		lastTag := git.LastTag()

		currentVer, err := sv.ToVersion(lastTag)
		if err != nil {
			return fmt.Errorf("error parsing version: %s from git tag, message: %w", lastTag, err)
		}

		commits, err := git.Log(sv.NewLogRange(sv.TagRange, lastTag, ""))
		if err != nil {
			return fmt.Errorf("error getting git log, message: %w", err)
		}

		nextVer, _ := semverProcessor.NextVersion(currentVer, commits)
		tagname, err := git.Tag(*nextVer)

		fmt.Println(tagname)

		if err != nil {
			return fmt.Errorf("error generating tag version: %s, message: %w", nextVer.String(), err)
		}

		return nil
	}
}

func getCommitType(cfg Config, p sv.MessageProcessor, input string) (string, error) {
	if input == "" {
		t, err := promptType(cfg.CommitMessage.Types)

		return t.Type, err
	}

	return input, p.ValidateType(input)
}

func getCommitScope(cfg Config, p sv.MessageProcessor, input string, noScope bool) (string, error) {
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

func getCommitIssue(cfg Config, p sv.MessageProcessor, branch string, noIssue bool) (string, error) {
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

func commitFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "no-scope",
			Aliases: []string{"nsc"},
			Usage:   "do not prompt for commit scope",
		},
		&cli.BoolFlag{
			Name:    "no-body",
			Aliases: []string{"nbd"},
			Usage:   "do not prompt for commit body",
		},
		&cli.BoolFlag{
			Name:    "no-issue",
			Aliases: []string{"nis"},
			Usage:   "do not prompt for commit issue, will try to recover from branch if enabled",
		},
		&cli.BoolFlag{
			Name:    "no-breaking",
			Aliases: []string{"nbc"},
			Usage:   "do not prompt for breaking changes",
		},
		&cli.StringFlag{
			Name:    "type",
			Aliases: []string{"t"},
			Usage:   "define commit type",
		},
		&cli.StringFlag{
			Name:    "scope",
			Aliases: []string{"s"},
			Usage:   "define commit scope",
		},
		&cli.StringFlag{
			Name:    "description",
			Aliases: []string{"d"},
			Usage:   "define commit description",
		},
		&cli.StringFlag{
			Name:    "breaking-change",
			Aliases: []string{"b"},
			Usage:   "define commit breaking change message",
		},
	}
}

func commitHandler(cfg Config, git sv.Git, messageProcessor sv.MessageProcessor) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		noBreaking := c.Bool("no-breaking")
		noBody := c.Bool("no-body")
		noIssue := c.Bool("no-issue")
		noScope := c.Bool("no-scope")
		inputType := c.String("type")
		inputScope := c.String("scope")
		inputDescription := c.String("description")
		inputBreakingChange := c.String("breaking-change")

		ctype, err := getCommitType(cfg, messageProcessor, inputType)
		if err != nil {
			return err
		}

		scope, err := getCommitScope(cfg, messageProcessor, inputScope, noScope)
		if err != nil {
			return err
		}

		subject, err := getCommitDescription(messageProcessor, inputDescription)
		if err != nil {
			return err
		}

		fullBody, err := getCommitBody(noBody)
		if err != nil {
			return err
		}

		issue, err := getCommitIssue(cfg, messageProcessor, git.Branch(), noIssue)
		if err != nil {
			return err
		}

		breakingChange, err := getCommitBreakingChange(noBreaking, inputBreakingChange)
		if err != nil {
			return err
		}

		header, body, footer := messageProcessor.Format(
			sv.NewCommitMessage(ctype, scope, subject, fullBody, issue, breakingChange),
		)

		err = git.Commit(header, body, footer)
		if err != nil {
			return fmt.Errorf("error executing git commit, message: %w", err)
		}

		return nil
	}
}

func changelogFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:    "size",
			Value:   10, //nolint:gomnd
			Aliases: []string{"n"},
			Usage:   "get changelog from last 'n' tags",
		},
		&cli.BoolFlag{
			Name:  "all",
			Usage: "ignore size parameter, get changelog for every tag",
		},
		&cli.BoolFlag{
			Name:  "add-next-version",
			Usage: "add next version on change log (commits since last tag, but only if there is a new version to release)",
		},
		&cli.BoolFlag{
			Name:  "semantic-version-only",
			Usage: "only show tags 'SemVer-ish'",
		},
	}
}

func changelogHandler(
	git sv.Git,
	semverProcessor sv.SemVerCommitsProcessor,
	rnProcessor sv.ReleaseNoteProcessor,
	formatter sv.OutputFormatter,
) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		tags, err := git.Tags()
		if err != nil {
			return err
		}

		sort.Slice(tags, func(i, j int) bool {
			return tags[i].Date.After(tags[j].Date)
		})

		var releaseNotes []sv.ReleaseNote

		size := c.Int("size")
		all := c.Bool("all")
		addNextVersion := c.Bool("add-next-version")
		semanticVersionOnly := c.Bool("semantic-version-only")

		if addNextVersion {
			rnVersion, updated, date, commits, uerr := getNextVersionInfo(git, semverProcessor)
			if uerr != nil {
				return uerr
			}

			if updated {
				releaseNotes = append(releaseNotes, rnProcessor.Create(rnVersion, "", date, commits))
			}
		}

		for i, tag := range tags {
			if !all && i >= size {
				break
			}

			previousTag := ""
			if i+1 < len(tags) {
				previousTag = tags[i+1].Name
			}

			if semanticVersionOnly && !sv.IsValidVersion(tag.Name) {
				continue
			}

			commits, err := git.Log(sv.NewLogRange(sv.TagRange, previousTag, tag.Name))
			if err != nil {
				return fmt.Errorf("error getting git log from tag: %s, message: %w", tag.Name, err)
			}

			currentVer, _ := sv.ToVersion(tag.Name)
			releaseNotes = append(releaseNotes, rnProcessor.Create(currentVer, tag.Name, tag.Date, commits))
		}

		output, err := formatter.FormatChangelog(releaseNotes)
		if err != nil {
			return fmt.Errorf("could not format changelog, message: %w", err)
		}

		fmt.Println(output)

		return nil
	}
}

func validateCommitMessageFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "path",
			Required: true,
			Usage:    "git working directory",
		},
		&cli.StringFlag{
			Name:     "file",
			Required: true,
			Usage:    "name of the file that contains the commit log message",
		},
		&cli.StringFlag{
			Name:     "source",
			Required: true,
			Usage:    "source of the commit message",
		},
	}
}

func validateCommitMessageHandler(git sv.Git, messageProcessor sv.MessageProcessor) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		branch := git.Branch()
		detached, derr := git.IsDetached()

		if messageProcessor.SkipBranch(branch, derr == nil && detached) {
			warnf("commit message validation skipped, branch in ignore list or detached...")

			return nil
		}

		if source := c.String("source"); source == "merge" {
			warnf("commit message validation skipped, ignoring source: %s...", source)

			return nil
		}

		filepath := filepath.Join(c.String("path"), c.String("file"))

		commitMessage, err := readFile(filepath)
		if err != nil {
			return fmt.Errorf("%w: %s", errReadCommitMessage, err.Error())
		}

		if err := messageProcessor.Validate(commitMessage); err != nil {
			return fmt.Errorf("%w: %s", errReadCommitMessage, err.Error())
		}

		msg, err := messageProcessor.Enhance(branch, commitMessage)
		if err != nil {
			warnf("could not enhance commit message, %s", err.Error())

			return nil
		}

		if msg == "" {
			return nil
		}

		if err := appendOnFile(msg, filepath); err != nil {
			return fmt.Errorf("%w: %s", errAppendFooter, err.Error())
		}

		return nil
	}
}

func readFile(filepath string) (string, error) {
	f, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	return string(f), nil
}

func appendOnFile(message, filepath string) error {
	f, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, laxFilePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(message)

	return err
}

func str(value, defaultValue string) string {
	if value != "" {
		return value
	}

	return defaultValue
}
