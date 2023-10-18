package app

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/thegeeklab/git-sv/sv"
	"github.com/thegeeklab/git-sv/sv/formatter"
	"github.com/thegeeklab/git-sv/templates"
)

const (
	logSeparator = "###"
	endLine      = "~~~"

	configFilename = "config.yml"
	configDir      = ".gitsv"
)

var errUnknownGitError = errors.New("git command failed")

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
	g := GitSV{
		Settings: &Settings{},
		Config:   NewConfig(configDir, configFilename),
	}

	g.MessageProcessor = sv.NewMessageProcessor(g.Config.CommitMessage, g.Config.Branches)
	g.CommitProcessor = sv.NewSemVerCommitProcessor(g.Config.Versioning, g.Config.CommitMessage)
	g.ReleasenotesProcessor = sv.NewReleaseNoteProcessor(g.Config.ReleaseNotes)
	g.OutputFormatter = formatter.NewOutputFormatter(templates.New(configDir))

	return g
}

// LastTag get last tag, if no tag found, return empty.
func (g GitSV) LastTag() string {
	//nolint:gosec
	cmd := exec.Command(
		"git",
		"for-each-ref",
		fmt.Sprintf("refs/tags/%s", *g.Config.Tag.Filter),
		"--sort",
		"-creatordate",
		"--format",
		"%(refname:short)",
		"--count",
		"1",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(strings.Trim(string(out), "\n"))
}

// Log return git log.
func (g GitSV) Log(lr LogRange) ([]sv.CommitLog, error) {
	format := "--pretty=format:\"%ad" + logSeparator +
		"%at" + logSeparator +
		"%cN" + logSeparator +
		"%h" + logSeparator +
		"%s" + logSeparator +
		"%b" + endLine + "\""
	params := []string{"log", "--date=short", format}

	if lr.start != "" || lr.end != "" {
		switch lr.rangeType {
		case DateRange:
			params = append(params, "--since", lr.start, "--until", addDay(lr.end))
		default:
			if lr.start == "" {
				params = append(params, lr.end)
			} else {
				params = append(params, lr.start+".."+str(lr.end, "HEAD"))
			}
		}
	}

	cmd := exec.Command("git", params...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, combinedOutputErr(err, out)
	}

	logs, parseErr := parseLogOutput(g.MessageProcessor, string(out))
	if parseErr != nil {
		return nil, parseErr
	}

	return logs, nil
}

// Commit runs git sv.
func (g GitSV) Commit(header, body, footer string) error {
	cmd := exec.Command("git", "commit", "-m", header, "-m", "", "-m", body, "-m", "", "-m", footer)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Tag create a git tag.
func (g GitSV) Tag(version semver.Version, annotate bool) (string, error) {
	tag := fmt.Sprintf(*g.Config.Tag.Pattern, version.Major(), version.Minor(), version.Patch())
	tagMsg := fmt.Sprintf("Version %d.%d.%d", version.Major(), version.Minor(), version.Patch())

	tagCommand := exec.Command("git", "tag", tag)
	if annotate {
		tagCommand = exec.Command("git", "tag", "-a", tag, "-m", tagMsg)
	}

	if out, err := tagCommand.CombinedOutput(); err != nil {
		return tag, combinedOutputErr(err, out)
	}

	pushCommand := exec.Command("git", "push", "origin", tag)
	if out, err := pushCommand.CombinedOutput(); err != nil {
		return tag, combinedOutputErr(err, out)
	}

	return tag, nil
}

// Tags list repository tags.
func (g GitSV) Tags() ([]Tag, error) {
	//nolint:gosec
	cmd := exec.Command(
		"git",
		"for-each-ref",
		"--sort",
		"creatordate",
		"--format",
		"%(creatordate:iso8601)#%(refname:short)",
		fmt.Sprintf("refs/tags/%s", *g.Config.Tag.Filter),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, combinedOutputErr(err, out)
	}

	return parseTagsOutput(string(out))
}

// Branch get git branch.
func (g GitSV) Branch() string {
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(strings.Trim(string(out), "\n"))
}

// IsDetached check if is detached.
func (g GitSV) IsDetached() (bool, error) {
	cmd := exec.Command("git", "symbolic-ref", "-q", "HEAD")

	out, err := cmd.CombinedOutput()
	// -q: do not issue an error message if the <name> is not a symbolic ref, but a detached HEAD;
	// instead exit with non-zero status silently.
	if output := string(out); err != nil {
		if output == "" {
			return true, nil
		}

		return false, fmt.Errorf("%w: %s", errUnknownGitError, output)
	}

	return false, nil
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
	content := strings.Split(strings.Trim(c, "\""), logSeparator)
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

func combinedOutputErr(err error, out []byte) error {
	msg := strings.Split(string(out), "\n")

	return fmt.Errorf("%w - %s", err, msg[0])
}
