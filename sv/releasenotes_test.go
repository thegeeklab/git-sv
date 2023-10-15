package sv

import (
	"reflect"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
)

func TestBaseReleaseNoteProcessor_Create(t *testing.T) {
	date := time.Now()

	tests := []struct {
		name    string
		version *semver.Version
		tag     string
		date    time.Time
		commits []CommitLog
		want    ReleaseNote
	}{
		{
			name:    "mapped tag",
			version: semver.MustParse("1.0.0"),
			tag:     "v1.0.0",
			date:    date,
			commits: []CommitLog{TestCommitlog("t1", map[string]string{}, "a")},
			want: TestReleaseNote(
				semver.MustParse("1.0.0"),
				"v1.0.0",
				date,
				[]ReleaseNoteSection{
					TestNewReleaseNoteCommitsSection(
						"Tag 1", []string{"t1"}, []CommitLog{TestCommitlog("t1", map[string]string{}, "a")},
					),
				},
				map[string]struct{}{"a": {}},
			),
		},
		{
			name:    "unmapped tag",
			version: semver.MustParse("1.0.0"),
			tag:     "v1.0.0",
			date:    date,
			commits: []CommitLog{
				TestCommitlog("t1", map[string]string{}, "a"), TestCommitlog("unmapped", map[string]string{}, "a"),
			},
			want: TestReleaseNote(
				semver.MustParse("1.0.0"),
				"v1.0.0",
				date,
				[]ReleaseNoteSection{
					TestNewReleaseNoteCommitsSection(
						"Tag 1", []string{"t1"}, []CommitLog{TestCommitlog("t1", map[string]string{}, "a")},
					),
				},
				map[string]struct{}{"a": {}},
			),
		},
		{
			name:    "breaking changes tag",
			version: semver.MustParse("1.0.0"),
			tag:     "v1.0.0",
			date:    date,
			commits: []CommitLog{
				TestCommitlog("t1", map[string]string{}, "a"),
				TestCommitlog("unmapped", map[string]string{"breaking-change": "breaks"}, "a"),
			},
			want: TestReleaseNote(
				semver.MustParse("1.0.0"),
				"v1.0.0",
				date,
				[]ReleaseNoteSection{
					TestNewReleaseNoteCommitsSection(
						"Tag 1", []string{"t1"}, []CommitLog{TestCommitlog("t1", map[string]string{}, "a")},
					),
					ReleaseNoteBreakingChangeSection{Name: "Breaking Changes", Messages: []string{"breaks"}},
				},
				map[string]struct{}{"a": {}},
			),
		},
		{
			name:    "multiple authors",
			version: semver.MustParse("1.0.0"),
			tag:     "v1.0.0",
			date:    date,
			commits: []CommitLog{
				TestCommitlog("t1", map[string]string{}, "author3"),
				TestCommitlog("t1", map[string]string{}, "author2"),
				TestCommitlog("t1", map[string]string{}, "author1"),
			},
			want: TestReleaseNote(
				semver.MustParse("1.0.0"),
				"v1.0.0",
				date,
				[]ReleaseNoteSection{
					TestNewReleaseNoteCommitsSection("Tag 1", []string{"t1"}, []CommitLog{
						TestCommitlog("t1", map[string]string{}, "author3"),
						TestCommitlog("t1", map[string]string{}, "author2"),
						TestCommitlog("t1", map[string]string{}, "author1"),
					}),
				},
				map[string]struct{}{"author1": {}, "author2": {}, "author3": {}},
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewReleaseNoteProcessor(
				ReleaseNotesConfig{
					Sections: []ReleaseNotesSectionConfig{
						{Name: "Tag 1", SectionType: "commits", CommitTypes: []string{"t1"}},
						{Name: "Tag 2", SectionType: "commits", CommitTypes: []string{"t2"}},
						{Name: "Breaking Changes", SectionType: "breaking-changes"},
					},
				})
			if got := p.Create(tt.version, tt.tag, tt.date, tt.commits); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BaseReleaseNoteProcessor.Create() = %v, want %v", got, tt.want)
			}
		})
	}
}
