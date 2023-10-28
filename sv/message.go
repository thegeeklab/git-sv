package sv

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	BreakingChangeFooterKey   = "BREAKING CHANGE"
	BreakingChangeMetadataKey = "breaking-change"
	IssueMetadataKey          = "issue"
	MessageRegexGroupName     = "header"
)

var (
	errInvalidCommitMessage = errors.New("commit message not valid")
	errIssueIDNotFound      = errors.New("could not find issue id using configured regex")
	errInvalidIssueRegex    = errors.New("could not compile issue regex")
	errInvalidHeaderRegex   = errors.New("invalid regex on header-selector")
)

// CommitMessage is a message using conventional commits.
type CommitMessage struct {
	Type             string            `json:"type,omitempty"`
	Scope            string            `json:"scope,omitempty"`
	Description      string            `json:"description,omitempty"`
	Body             string            `json:"body,omitempty"`
	IsBreakingChange bool              `json:"isBreakingChange,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type CommitMessageConfig struct {
	Types          []string                             `yaml:"types,flow"`
	HeaderSelector string                               `yaml:"header-selector"`
	Scope          CommitMessageScopeConfig             `yaml:"scope"`
	Footer         map[string]CommitMessageFooterConfig `yaml:"footer"`
	Issue          CommitMessageIssueConfig             `yaml:"issue"`
}

// IssueFooterConfig config for issue.
func (c CommitMessageConfig) IssueFooterConfig() CommitMessageFooterConfig {
	if v, exists := c.Footer[IssueMetadataKey]; exists {
		return v
	}

	return CommitMessageFooterConfig{}
}

// CommitMessageScopeConfig config scope preferences.
type CommitMessageScopeConfig struct {
	Values []string `yaml:"values"`
}

// CommitMessageFooterConfig config footer metadata.
type CommitMessageFooterConfig struct {
	Key            string   `yaml:"key"`
	KeySynonyms    []string `yaml:"key-synonyms,flow"`
	UseHash        bool     `yaml:"use-hash"`
	AddValuePrefix string   `yaml:"add-value-prefix"`
}

// CommitMessageIssueConfig issue preferences.
type CommitMessageIssueConfig struct {
	Regex string `yaml:"regex"`
}

// BranchesConfig branches preferences.
type BranchesConfig struct {
	Prefix       string   `yaml:"prefix"`
	Suffix       string   `yaml:"suffix"`
	DisableIssue bool     `yaml:"disable-issue"`
	Skip         []string `yaml:"skip,flow"`
	SkipDetached *bool    `yaml:"skip-detached"`
}

// NewCommitMessage commit message constructor.
func NewCommitMessage(ctype, scope, description, body, issue, breakingChanges string) CommitMessage {
	metadata := make(map[string]string)
	if issue != "" {
		metadata[IssueMetadataKey] = issue
	}

	if breakingChanges != "" {
		metadata[BreakingChangeMetadataKey] = breakingChanges
	}

	return CommitMessage{
		Type:             ctype,
		Scope:            scope,
		Description:      description,
		Body:             body,
		IsBreakingChange: breakingChanges != "",
		Metadata:         metadata,
	}
}

// Issue return issue from metadata.
func (m CommitMessage) Issue() string {
	return m.Metadata[IssueMetadataKey]
}

// BreakingMessage return breaking change message from metadata.
func (m CommitMessage) BreakingMessage() string {
	return m.Metadata[BreakingChangeMetadataKey]
}

// MessageProcessor interface.
type MessageProcessor interface {
	SkipBranch(branch string, detached bool) bool
	Validate(message string) error
	ValidateType(ctype string) error
	ValidateScope(scope string) error
	ValidateDescription(description string) error
	Enhance(branch, message string) (string, error)
	IssueID(branch string) (string, error)
	Format(msg CommitMessage) (string, string, string)
	Parse(subject, body string) (CommitMessage, error)
}

// NewMessageProcessor BaseMessageProcessor constructor.
func NewMessageProcessor(mcfg CommitMessageConfig, bcfg BranchesConfig) *BaseMessageProcessor {
	return &BaseMessageProcessor{
		messageCfg:  mcfg,
		branchesCfg: bcfg,
	}
}

// BaseMessageProcessor process validate message hook.
type BaseMessageProcessor struct {
	messageCfg  CommitMessageConfig
	branchesCfg BranchesConfig
}

// SkipBranch check if branch should be ignored.
func (p BaseMessageProcessor) SkipBranch(branch string, detached bool) bool {
	return contains(branch, p.branchesCfg.Skip) ||
		(p.branchesCfg.SkipDetached != nil && *p.branchesCfg.SkipDetached && detached)
}

// Validate commit message.
func (p BaseMessageProcessor) Validate(message string) error {
	subject, body := splitCommitMessageContent(message)
	msg, parseErr := p.Parse(subject, body)

	if parseErr != nil {
		return parseErr
	}

	if !regexp.MustCompile(`^[a-z+]+(\(.+\))?!?: .+$`).MatchString(subject) {
		return fmt.Errorf("%w: subject [%s] not valid", errInvalidCommitMessage, subject)
	}

	if err := p.ValidateType(msg.Type); err != nil {
		return err
	}

	if err := p.ValidateScope(msg.Scope); err != nil {
		return err
	}

	return p.ValidateDescription(msg.Description)
}

// ValidateType check if commit type is valid.
func (p BaseMessageProcessor) ValidateType(ctype string) error {
	if ctype == "" || !contains(ctype, p.messageCfg.Types) {
		return fmt.Errorf(
			"%w: type must be one of [%s]",
			errInvalidCommitMessage,
			strings.Join(p.messageCfg.Types, ", "),
		)
	}

	return nil
}

// ValidateScope check if commit scope is valid.
func (p BaseMessageProcessor) ValidateScope(scope string) error {
	if len(p.messageCfg.Scope.Values) > 0 && !contains(scope, p.messageCfg.Scope.Values) {
		return fmt.Errorf(
			"%w: scope must one of [%s]",
			errInvalidCommitMessage,
			strings.Join(p.messageCfg.Scope.Values, ", "),
		)
	}

	return nil
}

// ValidateDescription check if commit description is valid.
func (p BaseMessageProcessor) ValidateDescription(description string) error {
	if !regexp.MustCompile("^[a-z]+.*$").MatchString(description) {
		return fmt.Errorf("%w: description [%s] must start with lowercase", errInvalidCommitMessage, description)
	}

	return nil
}

// Enhance add metadata on commit message.
func (p BaseMessageProcessor) Enhance(branch, message string) (string, error) {
	if p.branchesCfg.DisableIssue || p.messageCfg.IssueFooterConfig().Key == "" ||
		hasIssueID(message, p.messageCfg.IssueFooterConfig()) {
		return "", nil // enhance disabled
	}

	issue, err := p.IssueID(branch)
	if err != nil {
		return "", err
	}

	if issue == "" {
		return "", errIssueIDNotFound
	}

	footer := formatIssueFooter(p.messageCfg.IssueFooterConfig(), issue)
	if !hasFooter(message) {
		return "\n" + footer, nil
	}

	return footer, nil
}

func formatIssueFooter(cfg CommitMessageFooterConfig, issue string) string {
	if !strings.HasPrefix(issue, cfg.AddValuePrefix) {
		issue = cfg.AddValuePrefix + issue
	}

	if cfg.UseHash {
		return fmt.Sprintf("%s #%s", cfg.Key, strings.TrimPrefix(issue, "#"))
	}

	return fmt.Sprintf("%s: %s", cfg.Key, issue)
}

// IssueID try to extract issue id from branch, return empty if not found.
func (p BaseMessageProcessor) IssueID(branch string) (string, error) {
	if p.branchesCfg.DisableIssue || p.messageCfg.Issue.Regex == "" {
		return "", nil
	}

	rstr := fmt.Sprintf("^%s(%s)%s$", p.branchesCfg.Prefix, p.messageCfg.Issue.Regex, p.branchesCfg.Suffix)

	r, err := regexp.Compile(rstr)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %v", errInvalidIssueRegex, rstr, err.Error())
	}

	groups := r.FindStringSubmatch(branch)
	if len(groups) != 4 { //nolint:gomnd
		return "", nil
	}

	return groups[2], nil
}

// Format a commit message returning header, body and footer.
func (p BaseMessageProcessor) Format(msg CommitMessage) (string, string, string) {
	var header strings.Builder

	header.WriteString(msg.Type)

	if msg.Scope != "" {
		header.WriteString("(" + msg.Scope + ")")
	}

	header.WriteString(": ")
	header.WriteString(msg.Description)

	var footer strings.Builder
	if msg.BreakingMessage() != "" {
		footer.WriteString(fmt.Sprintf("%s: %s", BreakingChangeFooterKey, msg.BreakingMessage()))
	}

	if issue, exists := msg.Metadata[IssueMetadataKey]; exists && p.messageCfg.IssueFooterConfig().Key != "" {
		if footer.Len() > 0 {
			footer.WriteString("\n")
		}

		footer.WriteString(formatIssueFooter(p.messageCfg.IssueFooterConfig(), issue))
	}

	return header.String(), msg.Body, footer.String()
}

func removeCarriage(commit string) string {
	return regexp.MustCompile(`\r`).ReplaceAllString(commit, "")
}

// Parse a commit message.
func (p BaseMessageProcessor) Parse(subject, body string) (CommitMessage, error) {
	preparedSubject, err := p.prepareHeader(subject)
	m := CommitMessage{}

	if err != nil {
		return m, err
	}

	m.Metadata = make(map[string]string)
	m.Body = removeCarriage(body)
	m.Type, m.Scope, m.Description, m.IsBreakingChange = parseSubjectMessage(preparedSubject)

	for key, mdCfg := range p.messageCfg.Footer {
		if mdCfg.Key != "" {
			prefixes := append([]string{mdCfg.Key}, mdCfg.KeySynonyms...)
			for _, prefix := range prefixes {
				if tagValue := extractFooterMetadata(prefix, m.Body, mdCfg.UseHash); tagValue != "" {
					m.Metadata[key] = tagValue

					break
				}
			}
		}
	}

	if m.IsBreakingChange {
		m.Metadata[BreakingChangeMetadataKey] = m.Description
	}

	if tagValue := extractFooterMetadata(BreakingChangeFooterKey, m.Body, false); tagValue != "" {
		m.IsBreakingChange = true
		m.Metadata[BreakingChangeMetadataKey] = tagValue
	}

	return m, nil
}

func (p BaseMessageProcessor) prepareHeader(header string) (string, error) {
	if p.messageCfg.HeaderSelector == "" {
		return header, nil
	}

	regex, err := regexp.Compile(p.messageCfg.HeaderSelector)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %s", errInvalidHeaderRegex, p.messageCfg.HeaderSelector, err.Error())
	}

	index := regex.SubexpIndex(MessageRegexGroupName)
	if index < 0 {
		return "", fmt.Errorf("%w: could not find group %s", errInvalidHeaderRegex, MessageRegexGroupName)
	}

	match := regex.FindStringSubmatch(header)

	if match == nil || len(match) < index {
		return "", fmt.Errorf(
			"%w: could not find group %s in match result for '%s'",
			errInvalidHeaderRegex,
			MessageRegexGroupName,
			header,
		)
	}

	return match[index], nil
}

func parseSubjectMessage(message string) (string, string, string, bool) {
	regex := regexp.MustCompile(`([a-z]+)(\((.*)\))?(!)?: (.*)`)

	result := regex.FindStringSubmatch(message)
	if len(result) != 6 { //nolint:gomnd
		return "", "", message, false
	}

	return result[1], result[3], strings.TrimSpace(result[5]), result[4] == "!"
}

func extractFooterMetadata(key, text string, useHash bool) string {
	regex := regexp.MustCompile(key + ": (.*)")

	if useHash {
		regex = regexp.MustCompile(key + " (#.*)")
	}

	result := regex.FindStringSubmatch(text)
	if len(result) < 2 { //nolint:gomnd
		return ""
	}

	return result[1]
}

func hasFooter(message string) bool {
	r := regexp.MustCompile("^[a-zA-Z-]+: .*|^[a-zA-Z-]+ #.*|^" + BreakingChangeFooterKey + ": .*")

	scanner := bufio.NewScanner(strings.NewReader(message))
	lines := 0

	for scanner.Scan() {
		if lines > 0 && r.MatchString(scanner.Text()) {
			return true
		}
		lines++
	}

	return false
}

func hasIssueID(message string, issueConfig CommitMessageFooterConfig) bool {
	var r *regexp.Regexp
	if issueConfig.UseHash {
		r = regexp.MustCompile(fmt.Sprintf("(?m)^%s #.+$", issueConfig.Key))
	} else {
		r = regexp.MustCompile(fmt.Sprintf("(?m)^%s: .+$", issueConfig.Key))
	}

	return r.MatchString(message)
}

func contains(value string, content []string) bool {
	for _, v := range content {
		if value == v {
			return true
		}
	}

	return false
}

func splitCommitMessageContent(content string) (string, string) {
	scanner := bufio.NewScanner(strings.NewReader(content))

	scanner.Scan()
	subject := scanner.Text()

	var body strings.Builder

	first := true
	for scanner.Scan() {
		if !first {
			body.WriteString("\n")
		}

		body.WriteString(scanner.Text())

		first = false
	}

	return subject, body.String()
}
