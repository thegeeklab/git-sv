package app

import (
	"reflect"
	"testing"

	"github.com/thegeeklab/git-sv/sv"
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
			Config{LogLevel: "info"},
			Config{LogLevel: "warn"},
			Config{LogLevel: "warn"},
			false,
		},
		{
			"default string",
			Config{LogLevel: "info"},
			Config{LogLevel: ""},
			Config{LogLevel: "info"},
			false,
		},
		{
			"overwrite list",
			Config{Branches: sv.BranchesConfig{Skip: []string{"a", "b"}}},
			Config{Branches: sv.BranchesConfig{Skip: []string{"c", "d"}}},
			Config{Branches: sv.BranchesConfig{Skip: []string{"c", "d"}}},
			false,
		},
		{
			"overwrite list with empty",
			Config{Branches: sv.BranchesConfig{Skip: []string{"a", "b"}}},
			Config{Branches: sv.BranchesConfig{Skip: make([]string, 0)}},
			Config{Branches: sv.BranchesConfig{Skip: make([]string, 0)}},
			false,
		},
		{
			"default list",
			Config{Branches: sv.BranchesConfig{Skip: []string{"a", "b"}}},
			Config{Branches: sv.BranchesConfig{Skip: nil}},
			Config{Branches: sv.BranchesConfig{Skip: []string{"a", "b"}}},
			false,
		},

		{
			"overwrite pointer bool false",
			Config{Branches: sv.BranchesConfig{SkipDetached: &boolFalse}},
			Config{Branches: sv.BranchesConfig{SkipDetached: &boolTrue}},
			Config{Branches: sv.BranchesConfig{SkipDetached: &boolTrue}},
			false,
		},
		{
			"overwrite pointer bool true",
			Config{Branches: sv.BranchesConfig{SkipDetached: &boolTrue}},
			Config{Branches: sv.BranchesConfig{SkipDetached: &boolFalse}},
			Config{Branches: sv.BranchesConfig{SkipDetached: &boolFalse}},
			false,
		},
		{
			"default pointer bool",
			Config{Branches: sv.BranchesConfig{SkipDetached: &boolTrue}},
			Config{Branches: sv.BranchesConfig{SkipDetached: nil}},
			Config{Branches: sv.BranchesConfig{SkipDetached: &boolTrue}},
			false,
		},
		{
			"merge maps",
			Config{CommitMessage: sv.CommitMessageConfig{
				Footer: map[string]sv.CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			Config{CommitMessage: sv.CommitMessageConfig{
				Footer: map[string]sv.CommitMessageFooterConfig{"issue2": {Key: "jira2"}},
			}},
			Config{CommitMessage: sv.CommitMessageConfig{Footer: map[string]sv.CommitMessageFooterConfig{
				"issue":  {Key: "jira"},
				"issue2": {Key: "jira2"},
			}}},
			false,
		},
		{
			"default maps",
			Config{CommitMessage: sv.CommitMessageConfig{
				Footer: map[string]sv.CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			Config{CommitMessage: sv.CommitMessageConfig{
				Footer: nil,
			}},
			Config{CommitMessage: sv.CommitMessageConfig{
				Footer: map[string]sv.CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			false,
		},
		{
			"merge empty maps",
			Config{CommitMessage: sv.CommitMessageConfig{
				Footer: map[string]sv.CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			Config{CommitMessage: sv.CommitMessageConfig{
				Footer: map[string]sv.CommitMessageFooterConfig{},
			}},
			Config{CommitMessage: sv.CommitMessageConfig{
				Footer: map[string]sv.CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			false,
		},
		{
			"overwrite tag config",
			Config{
				LogLevel: "info",
				Tag: TagConfig{
					Pattern: &nonEmptyStr,
					Filter:  &nonEmptyStr,
				},
			},
			Config{
				LogLevel: "",
				Tag: TagConfig{
					Pattern: &emptyStr,
					Filter:  &emptyStr,
				},
			},
			Config{
				LogLevel: "info",
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
