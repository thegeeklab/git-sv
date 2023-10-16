package templates

import (
	"reflect"
	"testing"
	"time"

	"github.com/thegeeklab/git-sv/sv"
)

func Test_checkTemplatesFiles(t *testing.T) {
	tests := []string{
		"assets/changelog-md.tpl",
		"assets/releasenotes-md.tpl",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			got, err := templateFs.ReadFile(tt)
			if err != nil {
				t.Errorf("missing template error = %v", err)

				return
			}

			if len(got) == 0 {
				t.Errorf("empty template")
			}
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
		{"valid time", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), "2006-01-02", "2022-01-01"},
		{"empty time", time.Time{}, "2006-01-02", ""},
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
			"existing section", []sv.ReleaseNoteSection{
				sv.ReleaseNoteCommitsSection{Name: "section 0"},
				sv.ReleaseNoteCommitsSection{Name: "section 1"},
				sv.ReleaseNoteCommitsSection{Name: "section 2"},
			}, "section 1", sv.ReleaseNoteCommitsSection{Name: "section 1"},
		},
		{
			"nonexisting section", []sv.ReleaseNoteSection{
				sv.ReleaseNoteCommitsSection{Name: "section 0"},
				sv.ReleaseNoteCommitsSection{Name: "section 1"},
				sv.ReleaseNoteCommitsSection{Name: "section 2"},
			}, "section 10", nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getSection(tt.sectionName, tt.sections); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getSection() = %v, want %v", got, tt.want)
			}
		})
	}
}
