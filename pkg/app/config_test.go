package app

import (
	"reflect"
	"testing"
)

func Test_merge(t *testing.T) {
	boolFalse := false
	boolTrue := true
	emptyStr := ""
	nonEmptyStr := "something"

	tests := []struct {
		name    string
		dst     Config
		src     Config
		want    Config
		wantErr bool
	}{
		{
			"overwrite string",
			Config{Version: "a"},
			Config{Version: "b"},
			Config{Version: "b"},
			false,
		},
		{
			"default string",
			Config{Version: "a"},
			Config{Version: ""},
			Config{Version: "a"},
			false,
		},
		{
			"overwrite list",
			Config{Branches: BranchesConfig{Skip: []string{"a", "b"}}},
			Config{Branches: BranchesConfig{Skip: []string{"c", "d"}}},
			Config{Branches: BranchesConfig{Skip: []string{"c", "d"}}},
			false,
		},
		{
			"overwrite list with empty",
			Config{Branches: BranchesConfig{Skip: []string{"a", "b"}}},
			Config{Branches: BranchesConfig{Skip: make([]string, 0)}},
			Config{Branches: BranchesConfig{Skip: make([]string, 0)}},
			false,
		},
		{
			"default list",
			Config{Branches: BranchesConfig{Skip: []string{"a", "b"}}},
			Config{Branches: BranchesConfig{Skip: nil}},
			Config{Branches: BranchesConfig{Skip: []string{"a", "b"}}},
			false,
		},

		{
			"overwrite pointer bool false",
			Config{Branches: BranchesConfig{SkipDetached: &boolFalse}},
			Config{Branches: BranchesConfig{SkipDetached: &boolTrue}},
			Config{Branches: BranchesConfig{SkipDetached: &boolTrue}},
			false,
		},
		{
			"overwrite pointer bool true",
			Config{Branches: BranchesConfig{SkipDetached: &boolTrue}},
			Config{Branches: BranchesConfig{SkipDetached: &boolFalse}},
			Config{Branches: BranchesConfig{SkipDetached: &boolFalse}},
			false,
		},
		{
			"default pointer bool",
			Config{Branches: BranchesConfig{SkipDetached: &boolTrue}},
			Config{Branches: BranchesConfig{SkipDetached: nil}},
			Config{Branches: BranchesConfig{SkipDetached: &boolTrue}},
			false,
		},
		{
			"merge maps",
			Config{CommitMessage: CommitMessageConfig{
				Footer: map[string]CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			Config{CommitMessage: CommitMessageConfig{
				Footer: map[string]CommitMessageFooterConfig{"issue2": {Key: "jira2"}},
			}},
			Config{CommitMessage: CommitMessageConfig{Footer: map[string]CommitMessageFooterConfig{
				"issue":  {Key: "jira"},
				"issue2": {Key: "jira2"},
			}}},
			false,
		},
		{
			"default maps",
			Config{CommitMessage: CommitMessageConfig{
				Footer: map[string]CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			Config{CommitMessage: CommitMessageConfig{
				Footer: nil,
			}},
			Config{CommitMessage: CommitMessageConfig{
				Footer: map[string]CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			false,
		},
		{
			"merge empty maps",
			Config{CommitMessage: CommitMessageConfig{
				Footer: map[string]CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			Config{CommitMessage: CommitMessageConfig{
				Footer: map[string]CommitMessageFooterConfig{},
			}},
			Config{CommitMessage: CommitMessageConfig{
				Footer: map[string]CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			false,
		},
		{
			"overwrite release notes header",
			Config{ReleaseNotes: ReleaseNotesConfig{Headers: map[string]string{"a": "aa"}}},
			Config{ReleaseNotes: ReleaseNotesConfig{Headers: map[string]string{"b": "bb"}}},
			Config{ReleaseNotes: ReleaseNotesConfig{Headers: map[string]string{"b": "bb"}}},
			false,
		},
		{
			"overwrite tag config",
			Config{
				Version: "a",
				Tag: TagConfig{
					Pattern: &nonEmptyStr,
					Filter:  &nonEmptyStr,
				},
			},
			Config{
				Version: "",
				Tag: TagConfig{
					Pattern: &emptyStr,
					Filter:  &emptyStr,
				},
			},
			Config{
				Version: "a",
				Tag: TagConfig{
					Pattern: &emptyStr,
					Filter:  &emptyStr,
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := merge(&tt.dst, tt.src); (err != nil) != tt.wantErr {
				t.Errorf("merge() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.dst, tt.want) {
				t.Errorf("merge() = %v, want %v", tt.dst, tt.want)
			}
		})
	}
}
