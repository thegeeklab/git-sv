package templates

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/sv"
)

func Test_checkTemplatesFiles(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{
			name: "changelog template",
			file: "assets/changelog-md.tpl",
		},
		{
			name: "valid templates",
			file: "assets/releasenotes-md.tpl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := templateFs.ReadFile(tt.file)

			assert.NoError(t, err)
			assert.NotEmpty(t, got)
		})
	}
}

func Test_timeFormat(t *testing.T) {
	tests := []struct {
		name   string
		time   time.Time
		format string
		want   string
	}{
		{
			name:   "valid time",
			time:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			format: "2006-01-02",
			want:   "2022-01-01",
		},
		{
			name:   "empty time",
			time:   time.Time{},
			format: "2006-01-02",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := zeroDate(tt.format, tt.time); got != tt.want {
				t.Errorf("timeFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getSection(t *testing.T) {
	tests := []struct {
		name        string
		sections    []sv.ReleaseNoteSection
		sectionName string
		want        sv.ReleaseNoteSection
	}{
		{
			name: "existing section",
			sections: []sv.ReleaseNoteSection{
				sv.ReleaseNoteCommitsSection{Name: "section 0"},
				sv.ReleaseNoteCommitsSection{Name: "section 1"},
				sv.ReleaseNoteCommitsSection{Name: "section 2"},
			},
			sectionName: "section 1",
			want:        sv.ReleaseNoteCommitsSection{Name: "section 1"},
		},
		{
			name: "nonexisting section",
			sections: []sv.ReleaseNoteSection{
				sv.ReleaseNoteCommitsSection{Name: "section 0"},
				sv.ReleaseNoteCommitsSection{Name: "section 1"},
				sv.ReleaseNoteCommitsSection{Name: "section 2"},
			},
			sectionName: "section 10",
			want:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSection(tt.sectionName, tt.sections)

			assert.Equal(t, tt.want, got)
		})
	}
}
