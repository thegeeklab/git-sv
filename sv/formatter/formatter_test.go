package formatter

import (
	"bytes"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/sv"
	"github.com/thegeeklab/git-sv/templates"
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

func TestBaseOutputFormatter_FormatReleaseNote(t *testing.T) {
	date, _ := time.Parse("2006-01-02", "2020-05-01")

	tests := []struct {
		name    string
		input   sv.ReleaseNote
		want    string
		wantErr bool
	}{
		{
			name:    "with date",
			input:   emptyReleaseNote("1.0.0", date.Truncate(time.Minute)),
			want:    dateChangelog,
			wantErr: false,
		},
		{
			name:    "without date",
			input:   emptyReleaseNote("1.0.0", time.Time{}.Truncate(time.Minute)),
			want:    emptyDateChangelog,
			wantErr: false,
		},
		{
			name:    "without version",
			input:   emptyReleaseNote("", date.Truncate(time.Minute)),
			want:    emptyVersionChangelog,
			wantErr: false,
		},
		{
			name:    "non versioning tag",
			input:   emptyReleaseNote("abc", date.Truncate(time.Minute)),
			want:    nonVersioningChangelog,
			wantErr: false,
		},
		{
			name:    "full changelog",
			input:   fullReleaseNote("1.0.0", date.Truncate(time.Minute)),
			want:    fullChangeLog,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewOutputFormatter(tmpls).FormatReleaseNote(tt.input)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func emptyReleaseNote(tag string, date time.Time) sv.ReleaseNote {
	v, _ := semver.NewVersion(tag)

	return sv.ReleaseNote{
		Version: v,
		Tag:     tag,
		Date:    date,
	}
}

func fullReleaseNote(tag string, date time.Time) sv.ReleaseNote {
	v, _ := semver.NewVersion(tag)
	sections := []sv.ReleaseNoteSection{
		sv.TestNewReleaseNoteCommitsSection(
			"Features",
			[]string{"feat"},
			[]sv.CommitLog{sv.TestCommitlog("feat", map[string]string{}, "a")},
		),
		sv.TestNewReleaseNoteCommitsSection(
			"Bug Fixes",
			[]string{"fix"},
			[]sv.CommitLog{sv.TestCommitlog("fix", map[string]string{}, "a")},
		),
		sv.TestNewReleaseNoteCommitsSection(
			"Build",
			[]string{"build"},
			[]sv.CommitLog{sv.TestCommitlog("build", map[string]string{}, "a")},
		),
		sv.ReleaseNoteBreakingChangeSection{Name: "Breaking Changes", Messages: []string{"break change message"}},
	}

	return sv.TestReleaseNote(v, tag, date, sections, map[string]struct{}{"a": {}})
}

func Test_checkTemplatesExecution(t *testing.T) {
	tpls := NewOutputFormatter(tmpls).templates
	tests := []struct {
		name      string
		template  string
		variables interface{}
	}{
		{
			name:      "changelog-md.tpl",
			template:  "changelog-md.tpl",
			variables: changelogVariables("v1.0.0", "v1.0.1"),
		},
		{
			name:      "releasenotes-md.tpl",
			template:  "releasenotes-md.tpl",
			variables: releaseNotesVariables("v1.0.0"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.template, func(t *testing.T) {
			var b bytes.Buffer

			err := tpls.ExecuteTemplate(&b, tt.template, tt.variables)

			assert.NoError(t, err)
			assert.NotEmpty(t, b.Bytes())
		})
	}
}

func releaseNotesVariables(release string) releaseNoteTemplateVariables {
	return releaseNoteTemplateVariables{
		Release: release,
		Date:    time.Date(2006, 1, 0o2, 0, 0, 0, 0, time.UTC),
		Sections: []sv.ReleaseNoteSection{
			sv.TestNewReleaseNoteCommitsSection("Features",
				[]string{"feat"},
				[]sv.CommitLog{sv.TestCommitlog("feat", map[string]string{}, "a")},
			),
			sv.TestNewReleaseNoteCommitsSection("Bug Fixes",
				[]string{"fix"},
				[]sv.CommitLog{sv.TestCommitlog("fix", map[string]string{}, "a")},
			),
			sv.TestNewReleaseNoteCommitsSection("Build",
				[]string{"build"},
				[]sv.CommitLog{sv.TestCommitlog("build", map[string]string{}, "a")},
			),
			sv.ReleaseNoteBreakingChangeSection{Name: "Breaking Changes", Messages: []string{"break change message"}},
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
