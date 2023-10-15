package formatter

import (
	"bytes"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/thegeeklab/git-sv/v2/pkg/git"
	"github.com/thegeeklab/git-sv/v2/pkg/templates"
)

var tmpls = templates.New("")

var dateChangelog = `## v1.0.0 (2020-05-01)`

var nonVersioningChangelog = `## abc (2020-05-01)`

var emptyDateChangelog = `## v1.0.0`

var emptyVersionChangelog = `## 2020-05-01`

var fullChangeLog = `## v1.0.0 (2020-05-01)

### Features

- subject text ()

### Bug Fixes

- subject text ()

### Build

- subject text ()

### Breaking Changes

- break change message`

func TestOutputFormatterImpl_FormatReleaseNote(t *testing.T) {
	date, _ := time.Parse("2006-01-02", "2020-05-01")

	tests := []struct {
		name    string
		input   git.ReleaseNote
		want    string
		wantErr bool
	}{
		{"with date", emptyReleaseNote("1.0.0", date.Truncate(time.Minute)), dateChangelog, false},
		{"without date", emptyReleaseNote("1.0.0", time.Time{}.Truncate(time.Minute)), emptyDateChangelog, false},
		{"without version", emptyReleaseNote("", date.Truncate(time.Minute)), emptyVersionChangelog, false},
		{"non versioning tag", emptyReleaseNote("abc", date.Truncate(time.Minute)), nonVersioningChangelog, false},
		{"full changelog", fullReleaseNote("1.0.0", date.Truncate(time.Minute)), fullChangeLog, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewOutputFormatter(tmpls).FormatReleaseNote(tt.input)
			if got != tt.want {
				t.Errorf("OutputFormatterImpl.FormatReleaseNote() = %v, want %v", got, tt.want)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("OutputFormatterImpl.FormatReleaseNote() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func emptyReleaseNote(tag string, date time.Time) git.ReleaseNote {
	v, _ := semver.NewVersion(tag)

	return git.ReleaseNote{
		Version: v,
		Tag:     tag,
		Date:    date,
	}
}

func fullReleaseNote(tag string, date time.Time) git.ReleaseNote {
	v, _ := semver.NewVersion(tag)
	sections := []git.ReleaseNoteSection{
		git.TestNewReleaseNoteCommitsSection(
			"Features",
			[]string{"feat"},
			[]git.CommitLog{git.TestCommitlog("feat", map[string]string{}, "a")},
		),
		git.TestNewReleaseNoteCommitsSection(
			"Bug Fixes",
			[]string{"fix"},
			[]git.CommitLog{git.TestCommitlog("fix", map[string]string{}, "a")},
		),
		git.TestNewReleaseNoteCommitsSection(
			"Build",
			[]string{"build"},
			[]git.CommitLog{git.TestCommitlog("build", map[string]string{}, "a")},
		),
		git.ReleaseNoteBreakingChangeSection{Name: "Breaking Changes", Messages: []string{"break change message"}},
	}

	return git.TestReleaseNote(v, tag, date, sections, map[string]struct{}{"a": {}})
}

func Test_checkTemplatesExecution(t *testing.T) {
	tpls := NewOutputFormatter(tmpls).templates
	tests := []struct {
		template  string
		variables interface{}
	}{
		{"changelog-md.tpl", changelogVariables("v1.0.0", "v1.0.1")},
		{"releasenotes-md.tpl", releaseNotesVariables("v1.0.0")},
	}

	for _, tt := range tests {
		t.Run(tt.template, func(t *testing.T) {
			var b bytes.Buffer
			err := tpls.ExecuteTemplate(&b, tt.template, tt.variables)
			if err != nil {
				t.Errorf("invalid template err = %v", err)

				return
			}

			if len(b.Bytes()) == 0 {
				t.Errorf("empty template")
			}
		})
	}
}

func releaseNotesVariables(release string) releaseNoteTemplateVariables {
	return releaseNoteTemplateVariables{
		Release: release,
		Date:    time.Date(2006, 1, 0o2, 0, 0, 0, 0, time.UTC),
		Sections: []git.ReleaseNoteSection{
			git.TestNewReleaseNoteCommitsSection("Features",
				[]string{"feat"},
				[]git.CommitLog{git.TestCommitlog("feat", map[string]string{}, "a")},
			),
			git.TestNewReleaseNoteCommitsSection("Bug Fixes",
				[]string{"fix"},
				[]git.CommitLog{git.TestCommitlog("fix", map[string]string{}, "a")},
			),
			git.TestNewReleaseNoteCommitsSection("Build",
				[]string{"build"},
				[]git.CommitLog{git.TestCommitlog("build", map[string]string{}, "a")},
			),
			git.ReleaseNoteBreakingChangeSection{Name: "Breaking Changes", Messages: []string{"break change message"}},
		},
	}
}

func changelogVariables(releases ...string) []releaseNoteTemplateVariables {
	var variables []releaseNoteTemplateVariables

	for _, r := range releases {
		variables = append(variables, releaseNotesVariables(r))
	}

	return variables
}
