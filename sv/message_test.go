package sv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var ccfg = CommitMessageConfig{
	Types: []string{"feat", "fix"},
	Scope: CommitMessageScopeConfig{},
	Footer: map[string]CommitMessageFooterConfig{
		"issue": {Key: "jira", KeySynonyms: []string{"Jira"}},
		"refs":  {Key: "Refs", UseHash: true},
	},
	Issue: CommitMessageIssueConfig{Regex: "[A-Z]+-[0-9]+"},
}

var ccfgHash = CommitMessageConfig{
	Types: []string{"feat", "fix"},
	Scope: CommitMessageScopeConfig{},
	Footer: map[string]CommitMessageFooterConfig{
		"issue": {Key: "jira", KeySynonyms: []string{"Jira"}, UseHash: true},
		"refs":  {Key: "Refs", UseHash: true},
	},
	Issue: CommitMessageIssueConfig{Regex: "[A-Z]+-[0-9]+"},
}

var ccfgGitIssue = CommitMessageConfig{
	Types: []string{"feat", "fix"},
	Scope: CommitMessageScopeConfig{},
	Footer: map[string]CommitMessageFooterConfig{
		"issue": {Key: "issue", KeySynonyms: []string{"Issue"}, UseHash: false, AddValuePrefix: "#"},
	},
	Issue: CommitMessageIssueConfig{Regex: "#?[0-9]+"},
}

var ccfgEmptyIssue = CommitMessageConfig{
	Types: []string{"feat", "fix"},
	Scope: CommitMessageScopeConfig{},
	Footer: map[string]CommitMessageFooterConfig{
		"issue": {},
	},
	Issue: CommitMessageIssueConfig{Regex: "[A-Z]+-[0-9]+"},
}

var ccfgWithScope = CommitMessageConfig{
	Types: []string{"feat", "fix"},
	Scope: CommitMessageScopeConfig{Values: []string{"", "scope"}},
	Footer: map[string]CommitMessageFooterConfig{
		"issue": {Key: "jira", KeySynonyms: []string{"Jira"}},
		"refs":  {Key: "Refs", UseHash: true},
	},
	Issue: CommitMessageIssueConfig{Regex: "[A-Z]+-[0-9]+"},
}

func newBranchCfg(skipDetached bool) BranchesConfig {
	return BranchesConfig{
		Prefix:       "([a-z]+\\/)?",
		Suffix:       "(-.*)?",
		Skip:         []string{"develop", "master"},
		SkipDetached: &skipDetached,
	}
}

func newCommitMessageCfg(headerSelector string) CommitMessageConfig {
	return CommitMessageConfig{
		Types: []string{"feat", "fix"},
		Scope: CommitMessageScopeConfig{Values: []string{"", "scope"}},
		Footer: map[string]CommitMessageFooterConfig{
			"issue": {Key: "jira", KeySynonyms: []string{"Jira"}},
			"refs":  {Key: "Refs", UseHash: true},
		},
		Issue:          CommitMessageIssueConfig{Regex: "[A-Z]+-[0-9]+"},
		HeaderSelector: headerSelector,
	}
}

// messages samples start.
var fullMessage = `fix: correct minor typos in code

see the issue for details

on typos fixed.

Reviewed-by: Z
Refs #133`

var fullMessageWithJira = `fix: correct minor typos in code

see the issue for details

on typos fixed.

Reviewed-by: Z
Refs #133
jira: JIRA-456`

var fullMessageRefs = `fix: correct minor typos in code

see the issue for details

on typos fixed.

Refs #133`

var subjectAndBodyMessage = `fix: correct minor typos in code

see the issue for details

on typos fixed.`

var subjectAndFooterMessage = `refactor!: drop support for Node 6

BREAKING CHANGE: refactor to use JavaScript features not available in Node 6.`

// multiline samples end

func TestBaseMessageProcessor_SkipBranch(t *testing.T) {
	tests := []struct {
		name     string
		bcfg     BranchesConfig
		branch   string
		detached bool
		want     bool
	}{
		{
			name:     "normal branch",
			bcfg:     newBranchCfg(false),
			branch:   "JIRA-123",
			detached: false,
			want:     false,
		},
		{
			name:     "dont ignore detached branch",
			bcfg:     newBranchCfg(false),
			branch:   "JIRA-123",
			detached: true,
			want:     false,
		},
		{
			name:     "ignore branch on skip list",
			bcfg:     newBranchCfg(false),
			branch:   "master",
			detached: false,
			want:     true,
		},
		{
			name:     "ignore detached branch",
			bcfg:     newBranchCfg(true),
			branch:   "JIRA-123",
			detached: true,
			want:     true,
		},
		{
			name:     "null skip detached",
			bcfg:     BranchesConfig{Skip: []string{}},
			branch:   "JIRA-123",
			detached: true,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewMessageProcessor(ccfg, tt.bcfg)
			got := p.SkipBranch(tt.branch, tt.detached)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBaseMessageProcessor_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     CommitMessageConfig
		message string
		wantErr bool
	}{
		{
			name:    "single line valid message",
			cfg:     ccfg,
			message: "feat: add something",
			wantErr: false,
		},
		{
			name:    "single line valid message with scope",
			cfg:     ccfg,
			message: "feat(scope): add something",
			wantErr: false,
		},
		{
			name:    "single line valid scope from list",
			cfg:     ccfgWithScope,
			message: "feat(scope): add something",
			wantErr: false,
		},
		{
			name:    "single line invalid scope from list",
			cfg:     ccfgWithScope,
			message: "feat(invalid): add something",
			wantErr: true,
		},
		{
			name:    "single line invalid type message",
			cfg:     ccfg,
			message: "something: add something",
			wantErr: true,
		},
		{
			name:    "single line invalid type message",
			cfg:     ccfg,
			message: "feat?: add something",
			wantErr: true,
		},
		{
			name: "multi line valid message",
			cfg:  ccfg,
			message: `feat: add something
		team: x`,
			wantErr: false,
		},
		{
			name: "multi line invalid message",
			cfg:  ccfg,
			message: `feat add something
		team: x`,
			wantErr: true,
		},
		{
			name:    "support ! for breaking change",
			cfg:     ccfg,
			message: "feat!: add something",
			wantErr: false,
		},
		{
			name:    "support ! with scope for breaking change",
			cfg:     ccfg,
			message: "feat(scope)!: add something",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewMessageProcessor(tt.cfg, newBranchCfg(false))
			err := p.Validate(tt.message)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestBaseMessageProcessor_ValidateType(t *testing.T) {
	tests := []struct {
		name    string
		cfg     CommitMessageConfig
		ctype   string
		wantErr bool
	}{
		{
			name:    "valid type",
			cfg:     ccfg,
			ctype:   "feat",
			wantErr: false,
		},
		{
			name:    "invalid type",
			cfg:     ccfg,
			ctype:   "aaa",
			wantErr: true,
		},
		{
			name:    "empty type",
			cfg:     ccfg,
			ctype:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewMessageProcessor(tt.cfg, newBranchCfg(false))
			err := p.ValidateType(tt.ctype)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestBaseMessageProcessor_ValidateScope(t *testing.T) {
	tests := []struct {
		name    string
		cfg     CommitMessageConfig
		scope   string
		wantErr bool
	}{
		{
			name:    "any scope",
			cfg:     ccfg,
			scope:   "aaa",
			wantErr: false,
		},
		{
			name:    "valid scope with scope list",
			cfg:     ccfgWithScope,
			scope:   "scope",
			wantErr: false,
		},
		{
			name:    "invalid scope with scope list",
			cfg:     ccfgWithScope,
			scope:   "aaa",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewMessageProcessor(tt.cfg, newBranchCfg(false))
			err := p.ValidateScope(tt.scope)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestBaseMessageProcessor_ValidateDescription(t *testing.T) {
	tests := []struct {
		name        string
		cfg         CommitMessageConfig
		description string
		wantErr     bool
	}{
		{
			name:        "empty description",
			cfg:         ccfg,
			description: "",
			wantErr:     true,
		},
		{
			name:        "sigle letter description",
			cfg:         ccfg,
			description: "a",
			wantErr:     false,
		},
		{
			name:        "number description",
			cfg:         ccfg,
			description: "1",
			wantErr:     true,
		},
		{
			name:        "valid description",
			cfg:         ccfg,
			description: "add some feature",
			wantErr:     false,
		},
		{
			name:        "invalid capital letter description",
			cfg:         ccfg,
			description: "Add some feature",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewMessageProcessor(tt.cfg, newBranchCfg(false))
			err := p.ValidateDescription(tt.description)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestBaseMessageProcessor_Enhance(t *testing.T) {
	tests := []struct {
		name    string
		cfg     CommitMessageConfig
		branch  string
		message string
		want    string
		wantErr bool
	}{
		{
			name:    "issue on branch name",
			cfg:     ccfg,
			branch:  "JIRA-123",
			message: "fix: fix something",
			want:    "\njira: JIRA-123",
			wantErr: false,
		},
		{
			name:    "issue on branch name with description",
			cfg:     ccfg,
			branch:  "JIRA-123-some-description",
			message: "fix: fix something",
			want:    "\njira: JIRA-123",
			wantErr: false,
		},
		{
			name:    "issue on branch name with prefix",
			cfg:     ccfg,
			branch:  "feature/JIRA-123",
			message: "fix: fix something",
			want:    "\njira: JIRA-123",
			wantErr: false,
		},
		{
			name:    "with footer",
			cfg:     ccfg,
			branch:  "JIRA-123",
			message: fullMessage,
			want:    "jira: JIRA-123",
			wantErr: false,
		},
		{
			name:    "with issue on footer",
			cfg:     ccfg,
			branch:  "JIRA-123",
			message: fullMessageWithJira,
			want:    "",
			wantErr: false,
		},
		{
			name:    "issue on branch name with prefix and description",
			cfg:     ccfg,
			branch:  "feature/JIRA-123-some-description",
			message: "fix: fix something",
			want:    "\njira: JIRA-123",
			wantErr: false,
		},
		{
			name:    "no issue on branch name",
			cfg:     ccfg,
			branch:  "branch",
			message: "fix: fix something",
			want:    "",
			wantErr: true,
		},
		{
			name:    "unexpected branch name",
			cfg:     ccfg,
			branch:  "feature /JIRA-123",
			message: "fix: fix something",
			want:    "",
			wantErr: true,
		},
		{
			name:    "issue on branch name using hash",
			cfg:     ccfgHash,
			branch:  "JIRA-123-some-description",
			message: "fix: fix something",
			want:    "\njira #JIRA-123",
			wantErr: false,
		},
		{
			name:    "numeric issue on branch name",
			cfg:     ccfgGitIssue,
			branch:  "#13",
			message: "fix: fix something",
			want:    "\nissue: #13",
			wantErr: false,
		},
		{
			name:    "numeric issue on branch name without hash",
			cfg:     ccfgGitIssue,
			branch:  "13",
			message: "fix: fix something",
			want:    "\nissue: #13",
			wantErr: false,
		},
		{
			name:    "numeric issue on branch name with description without hash",
			cfg:     ccfgGitIssue,
			branch:  "13-some-fix",
			message: "fix: fix something",
			want:    "\nissue: #13",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMessageProcessor(tt.cfg, newBranchCfg(false)).Enhance(tt.branch, tt.message)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBaseMessageProcessor_IssueID(t *testing.T) {
	p := NewMessageProcessor(ccfg, newBranchCfg(false))

	tests := []struct {
		name    string
		branch  string
		want    string
		wantErr bool
	}{
		{
			name:    "simple branch",
			branch:  "JIRA-123",
			want:    "JIRA-123",
			wantErr: false,
		},
		{
			name:    "branch with prefix",
			branch:  "feature/JIRA-123",
			want:    "JIRA-123",
			wantErr: false,
		},
		{
			name:    "branch with prefix and posfix",
			branch:  "feature/JIRA-123-some-description",
			want:    "JIRA-123",
			wantErr: false,
		},
		{
			name:    "branch not found",
			branch:  "feature/wrong123-some-description",
			want:    "",
			wantErr: false,
		},
		{
			name:    "empty branch",
			branch:  "",
			want:    "",
			wantErr: false,
		},
		{
			name:    "unexpected branch name",
			branch:  "feature /JIRA-123",
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.IssueID(tt.branch)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

const (
	multilineBody = `a
b
c`
	fullFooter = `BREAKING CHANGE: breaks
jira: JIRA-123`
)

func Test_hasIssueID(t *testing.T) {
	cfgColon := CommitMessageFooterConfig{Key: "jira"}
	cfgHash := CommitMessageFooterConfig{Key: "jira", UseHash: true}
	cfgEmpty := CommitMessageFooterConfig{}

	tests := []struct {
		name     string
		message  string
		issueCfg CommitMessageFooterConfig
		want     bool
	}{
		{
			name:     "single line without issue",
			message:  "feat: something",
			issueCfg: cfgColon,
			want:     false,
		},
		{
			name: "multi line without issue",
			message: `feat: something

yay`,
			issueCfg: cfgColon,
			want:     false,
		},
		{
			name: "multi line without jira issue",
			message: `feat: something

jira1: JIRA-123`,
			issueCfg: cfgColon,
			want:     false,
		},
		{
			name: "multi line with issue",
			message: `feat: something

jira: JIRA-123`,
			issueCfg: cfgColon,
			want:     true,
		},
		{
			name: "multi line with issue and hash",
			message: `feat: something

jira #JIRA-123`,
			issueCfg: cfgHash,
			want:     true,
		},
		{
			name: "empty config",
			message: `feat: something

jira #JIRA-123`,
			issueCfg: cfgEmpty,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasIssueID(tt.message, tt.issueCfg)

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_hasFooter(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{
			name:    "simple message",
			message: "feat: add something",
			want:    false,
		},
		{
			name:    "full messsage",
			message: fullMessage,
			want:    true,
		},
		{
			name:    "full messsage with refs",
			message: fullMessageRefs,
			want:    true,
		},
		{
			name:    "subject and footer message",
			message: subjectAndFooterMessage,
			want:    true,
		},
		{
			name:    "subject and body message",
			message: subjectAndBodyMessage,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasFooter(tt.message)

			assert.Equal(t, tt.want, got)
		})
	}
}

// conventional commit tests

var completeBody = `some descriptions

jira: JIRA-123
BREAKING CHANGE: this change breaks everything`

var (
	bodyWithCarriage         = "some description\r\nmore description\r\n\r\njira: JIRA-123\r"
	expectedBodyWithCarriage = "some description\nmore description\n\njira: JIRA-123"
)

var issueOnlyBody = `some descriptions

jira: JIRA-456`

var issueSynonymsBody = `some descriptions

Jira: JIRA-789`

var hashMetadataBody = `some descriptions

Jira: JIRA-999
Refs #123`

func TestBaseMessageProcessor_Parse(t *testing.T) {
	tests := []struct {
		name    string
		cfg     CommitMessageConfig
		subject string
		body    string
		want    CommitMessage
	}{
		{
			name:    "simple message",
			cfg:     ccfg,
			subject: "feat: something awesome",
			body:    "",
			want: CommitMessage{
				Type:             "feat",
				Scope:            "",
				Description:      "something awesome",
				Body:             "",
				IsBreakingChange: false,
				Metadata:         map[string]string{},
			},
		},
		{
			name:    "message with scope",
			cfg:     ccfg,
			subject: "feat(scope): something awesome",
			body:    "",
			want: CommitMessage{
				Type:             "feat",
				Scope:            "scope",
				Description:      "something awesome",
				Body:             "",
				IsBreakingChange: false,
				Metadata:         map[string]string{},
			},
		},
		{
			name:    "unmapped type",
			cfg:     ccfg,
			subject: "unkn: something unknown",
			body:    "",
			want: CommitMessage{
				Type:             "unkn",
				Scope:            "",
				Description:      "something unknown",
				Body:             "",
				IsBreakingChange: false,
				Metadata:         map[string]string{},
			},
		},
		{
			name:    "jira and breaking change metadata",
			cfg:     ccfg,
			subject: "feat: something new",
			body:    completeBody,
			want: CommitMessage{
				Type:             "feat",
				Scope:            "",
				Description:      "something new",
				Body:             completeBody,
				IsBreakingChange: true,
				Metadata: map[string]string{
					IssueMetadataKey:          "JIRA-123",
					BreakingChangeMetadataKey: "this change breaks everything",
				},
			},
		},
		{
			name:    "jira only metadata",
			cfg:     ccfg,
			subject: "feat: something new",
			body:    issueOnlyBody,
			want: CommitMessage{
				Type:             "feat",
				Scope:            "",
				Description:      "something new",
				Body:             issueOnlyBody,
				IsBreakingChange: false,
				Metadata:         map[string]string{IssueMetadataKey: "JIRA-456"},
			},
		},
		{
			name:    "jira synonyms metadata",
			cfg:     ccfg,
			subject: "feat: something new",
			body:    issueSynonymsBody,
			want: CommitMessage{
				Type:             "feat",
				Scope:            "",
				Description:      "something new",
				Body:             issueSynonymsBody,
				IsBreakingChange: false,
				Metadata:         map[string]string{IssueMetadataKey: "JIRA-789"},
			},
		},
		{
			name:    "breaking change with empty body",
			cfg:     ccfg,
			subject: "feat!: something new",
			body:    "",
			want: CommitMessage{
				Type:             "feat",
				Scope:            "",
				Description:      "something new",
				Body:             "",
				IsBreakingChange: true,
				Metadata: map[string]string{
					BreakingChangeMetadataKey: "something new",
				},
			},
		},
		{
			name:    "hash metadata",
			cfg:     ccfg,
			subject: "feat: something new",
			body:    hashMetadataBody,
			want: CommitMessage{
				Type:             "feat",
				Scope:            "",
				Description:      "something new",
				Body:             hashMetadataBody,
				IsBreakingChange: false,
				Metadata:         map[string]string{IssueMetadataKey: "JIRA-999", "refs": "#123"},
			},
		},
		{
			name:    "empty issue cfg",
			cfg:     ccfgEmptyIssue,
			subject: "feat: something new",
			body:    hashMetadataBody,
			want: CommitMessage{
				Type:             "feat",
				Scope:            "",
				Description:      "something new",
				Body:             hashMetadataBody,
				IsBreakingChange: false,
				Metadata:         map[string]string{},
			},
		},
		{
			name:    "carriage return on body",
			cfg:     ccfg,
			subject: "feat: something new",
			body:    bodyWithCarriage,
			want: CommitMessage{
				Type:             "feat",
				Scope:            "",
				Description:      "something new",
				Body:             expectedBodyWithCarriage,
				IsBreakingChange: false,
				Metadata:         map[string]string{IssueMetadataKey: "JIRA-123"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMessageProcessor(tt.cfg, newBranchCfg(false)).Parse(tt.subject, tt.body)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBaseMessageProcessor_Format(t *testing.T) {
	tests := []struct {
		name       string
		cfg        CommitMessageConfig
		msg        CommitMessage
		wantHeader string
		wantBody   string
		wantFooter string
	}{
		{
			name:       "simple message",
			cfg:        ccfg,
			msg:        NewCommitMessage("feat", "", "something", "", "", ""),
			wantHeader: "feat: something",
			wantBody:   "",
			wantFooter: "",
		},
		{
			name:       "with issue",
			cfg:        ccfg,
			msg:        NewCommitMessage("feat", "", "something", "", "JIRA-123", ""),
			wantHeader: "feat: something",
			wantBody:   "",
			wantFooter: "jira: JIRA-123",
		},
		{
			name:       "with issue using hash",
			cfg:        ccfgHash,
			msg:        NewCommitMessage("feat", "", "something", "", "JIRA-123", ""),
			wantHeader: "feat: something",
			wantBody:   "",
			wantFooter: "jira #JIRA-123",
		},
		{
			name:       "with issue using double hash",
			cfg:        ccfgHash,
			msg:        NewCommitMessage("feat", "", "something", "", "#JIRA-123", ""),
			wantHeader: "feat: something",
			wantBody:   "",
			wantFooter: "jira #JIRA-123",
		},
		{
			name:       "with breaking change",
			cfg:        ccfg,
			msg:        NewCommitMessage("feat", "", "something", "", "", "breaks"),
			wantHeader: "feat: something",
			wantBody:   "",
			wantFooter: "BREAKING CHANGE: breaks",
		},
		{
			name:       "with scope",
			cfg:        ccfg,
			msg:        NewCommitMessage("feat", "scope", "something", "", "", ""),
			wantHeader: "feat(scope): something",
			wantBody:   "",
			wantFooter: "",
		},
		{
			name:       "with body",
			cfg:        ccfg,
			msg:        NewCommitMessage("feat", "", "something", "body", "", ""),
			wantHeader: "feat: something",
			wantBody:   "body",
			wantFooter: "",
		},
		{
			name:       "with multiline body",
			cfg:        ccfg,
			msg:        NewCommitMessage("feat", "", "something", multilineBody, "", ""),
			wantHeader: "feat: something",
			wantBody:   multilineBody,
			wantFooter: "",
		},
		{
			name:       "full message",
			cfg:        ccfg,
			msg:        NewCommitMessage("feat", "scope", "something", multilineBody, "JIRA-123", "breaks"),
			wantHeader: "feat(scope): something",
			wantBody:   multilineBody,
			wantFooter: fullFooter,
		},
		{
			name:       "config without issue key",
			cfg:        ccfgEmptyIssue,
			msg:        NewCommitMessage("feat", "", "something", "", "JIRA-123", ""),
			wantHeader: "feat: something",
			wantBody:   "",
			wantFooter: "",
		},
		{
			name:       "with issue and issue prefix",
			cfg:        ccfgGitIssue,
			msg:        NewCommitMessage("feat", "", "something", "", "123", ""),
			wantHeader: "feat: something",
			wantBody:   "",
			wantFooter: "issue: #123",
		},
		{
			name:       "with #issue and issue prefix",
			cfg:        ccfgGitIssue,
			msg:        NewCommitMessage("feat", "", "something", "", "#123", ""),
			wantHeader: "feat: something",
			wantBody:   "",
			wantFooter: "issue: #123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := NewMessageProcessor(tt.cfg, newBranchCfg(false)).Format(tt.msg)

			assert.Equal(t, tt.wantHeader, got)
			assert.Equal(t, tt.wantBody, got1)
			assert.Equal(t, tt.wantFooter, got2)
		})
	}
}

var expectedBodyFullMessage = `
see the issue for details

on typos fixed.

Reviewed-by: Z
Refs #133`

func Test_splitCommitMessageContent(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantSubject string
		wantBody    string
	}{
		{
			name:        "single line commit",
			content:     "feat: something",
			wantSubject: "feat: something",
			wantBody:    "",
		},
		{
			name:        "multi line commit",
			content:     fullMessage,
			wantSubject: "fix: correct minor typos in code",
			wantBody:    expectedBodyFullMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := splitCommitMessageContent(tt.content)

			assert.Equal(t, tt.wantSubject, got)
			assert.Equal(t, tt.wantBody, got1)
		})
	}
}

func Test_parseSubjectMessage(t *testing.T) {
	tests := []struct {
		name                  string
		message               string
		wantType              string
		wantScope             string
		wantDescription       string
		wantHasBreakingChange bool
	}{
		{
			name:                  "valid commit",
			message:               "feat: something",
			wantType:              "feat",
			wantScope:             "",
			wantDescription:       "something",
			wantHasBreakingChange: false,
		},
		{
			name:                  "valid commit with scope",
			message:               "feat(scope): something",
			wantType:              "feat",
			wantScope:             "scope",
			wantDescription:       "something",
			wantHasBreakingChange: false,
		},
		{
			name:                  "valid commit with breaking change",
			message:               "feat(scope)!: something",
			wantType:              "feat",
			wantScope:             "scope",
			wantDescription:       "something",
			wantHasBreakingChange: true,
		},
		{
			name:                  "missing description",
			message:               "feat: ",
			wantType:              "feat",
			wantScope:             "",
			wantDescription:       "",
			wantHasBreakingChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctype, scope, description, hasBreakingChange := parseSubjectMessage(tt.message)

			assert.Equal(t, tt.wantType, ctype)
			assert.Equal(t, tt.wantScope, scope)
			assert.Equal(t, tt.wantDescription, description)
			assert.Equal(t, tt.wantHasBreakingChange, hasBreakingChange)
		})
	}
}

func Test_prepareHeader(t *testing.T) {
	tests := []struct {
		name           string
		headerSelector string
		commitHeader   string
		wantHeader     string
		wantError      bool
	}{
		{
			name:           "conventional without selector",
			headerSelector: "",
			commitHeader:   "feat: something",
			wantHeader:     "feat: something",
			wantError:      false,
		},
		{
			name:           "conventional with scope without selector",
			headerSelector: "",
			commitHeader:   "feat(scope): something",
			wantHeader:     "feat(scope): something",
			wantError:      false,
		},
		{
			name:           "non-conventional without selector",
			headerSelector: "",
			commitHeader:   "something",
			wantHeader:     "something",
			wantError:      false,
		},
		{
			name:           "matching conventional with selector with group",
			headerSelector: "Merged PR (\\d+): (?P<header>.*)",
			commitHeader:   "Merged PR 123: feat: something",
			wantHeader:     "feat: something",
			wantError:      false,
		},
		{
			name:           "matching non-conventional with selector with group",
			headerSelector: "Merged PR (\\d+): (?P<header>.*)",
			commitHeader:   "Merged PR 123: something",
			wantHeader:     "something",
			wantError:      false,
		},
		{
			name:           "matching non-conventional with selector without group",
			headerSelector: "Merged PR (\\d+): (.*)",
			commitHeader:   "Merged PR 123: something",
			wantHeader:     "",
			wantError:      true,
		},
		{
			name:           "non-matching non-conventional with selector with group",
			headerSelector: "Merged PR (\\d+): (?P<header>.*)",
			commitHeader:   "something",
			wantHeader:     "",
			wantError:      true,
		},
		{
			name:           "matching non-conventional with invalid regex",
			headerSelector: "Merged PR (\\d+): (<header>.*)",
			commitHeader:   "Merged PR 123: something",
			wantHeader:     "",
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgProcessor := NewMessageProcessor(newCommitMessageCfg(tt.headerSelector), newBranchCfg(false))
			header, err := msgProcessor.prepareHeader(tt.commitHeader)

			if tt.wantError {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantHeader, header)
		})
	}
}

func Test_removeCarriage(t *testing.T) {
	tests := []struct {
		name   string
		commit string
		want   string
	}{
		{
			name:   "normal string",
			commit: "normal string",
			want:   "normal string",
		},
		{
			name:   "break line",
			commit: "normal\nstring",
			want:   "normal\nstring",
		},
		{
			name:   "carriage return",
			commit: "normal\r\nstring",
			want:   "normal\nstring",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeCarriage(tt.commit)

			assert.Equal(t, tt.want, got)
		})
	}
}
