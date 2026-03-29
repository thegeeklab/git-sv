package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thegeeklab/git-sv/sv"
)

func Test_merge(t *testing.T) {
	tests := []struct {
		name    string
		dst     Config
		src     Config
		want    Config
		wantErr bool
	}{
		{
			name:    "overwrite string",
			dst:     Config{LogLevel: "info"},
			src:     Config{LogLevel: "warn"},
			want:    Config{LogLevel: "warn"},
			wantErr: false,
		},
		{
			name:    "default string",
			dst:     Config{LogLevel: "info"},
			src:     Config{LogLevel: ""},
			want:    Config{LogLevel: "info"},
			wantErr: false,
		},
		{
			name:    "overwrite list",
			dst:     Config{Branches: sv.BranchesConfig{Skip: []string{"a", "b"}}},
			src:     Config{Branches: sv.BranchesConfig{Skip: []string{"c", "d"}}},
			want:    Config{Branches: sv.BranchesConfig{Skip: []string{"c", "d"}}},
			wantErr: false,
		},
		{
			name:    "overwrite list with empty",
			dst:     Config{Branches: sv.BranchesConfig{Skip: []string{"a", "b"}}},
			src:     Config{Branches: sv.BranchesConfig{Skip: make([]string, 0)}},
			want:    Config{Branches: sv.BranchesConfig{Skip: make([]string, 0)}},
			wantErr: false,
		},
		{
			name:    "default list",
			dst:     Config{Branches: sv.BranchesConfig{Skip: []string{"a", "b"}}},
			src:     Config{Branches: sv.BranchesConfig{Skip: nil}},
			want:    Config{Branches: sv.BranchesConfig{Skip: []string{"a", "b"}}},
			wantErr: false,
		},

		{
			name:    "overwrite pointer bool false",
			dst:     Config{Branches: sv.BranchesConfig{SkipDetached: new(false)}},
			src:     Config{Branches: sv.BranchesConfig{SkipDetached: new(true)}},
			want:    Config{Branches: sv.BranchesConfig{SkipDetached: new(true)}},
			wantErr: false,
		},
		{
			name:    "overwrite pointer bool true",
			dst:     Config{Branches: sv.BranchesConfig{SkipDetached: new(true)}},
			src:     Config{Branches: sv.BranchesConfig{SkipDetached: new(false)}},
			want:    Config{Branches: sv.BranchesConfig{SkipDetached: new(false)}},
			wantErr: false,
		},
		{
			name:    "default pointer bool",
			dst:     Config{Branches: sv.BranchesConfig{SkipDetached: new(true)}},
			src:     Config{Branches: sv.BranchesConfig{SkipDetached: nil}},
			want:    Config{Branches: sv.BranchesConfig{SkipDetached: new(true)}},
			wantErr: false,
		},
		{
			name: "merge maps",
			dst: Config{CommitMessage: sv.CommitMessageConfig{
				Footer: map[string]sv.CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			src: Config{CommitMessage: sv.CommitMessageConfig{
				Footer: map[string]sv.CommitMessageFooterConfig{"issue2": {Key: "jira2"}},
			}},
			want: Config{CommitMessage: sv.CommitMessageConfig{Footer: map[string]sv.CommitMessageFooterConfig{
				"issue":  {Key: "jira"},
				"issue2": {Key: "jira2"},
			}}},
			wantErr: false,
		},
		{
			name: "default maps",
			dst: Config{CommitMessage: sv.CommitMessageConfig{
				Footer: map[string]sv.CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			src: Config{CommitMessage: sv.CommitMessageConfig{
				Footer: nil,
			}},
			want: Config{CommitMessage: sv.CommitMessageConfig{
				Footer: map[string]sv.CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			wantErr: false,
		},
		{
			name: "merge empty maps",
			dst: Config{CommitMessage: sv.CommitMessageConfig{
				Footer: map[string]sv.CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			src: Config{CommitMessage: sv.CommitMessageConfig{
				Footer: map[string]sv.CommitMessageFooterConfig{},
			}},
			want: Config{CommitMessage: sv.CommitMessageConfig{
				Footer: map[string]sv.CommitMessageFooterConfig{"issue": {Key: "jira"}},
			}},
			wantErr: false,
		},
		{
			name: "overwrite tag config",
			dst: Config{
				LogLevel: "info",
				Tag: TagConfig{
					Pattern: new("something"),
					Filter:  new("something"),
				},
			},
			src: Config{
				LogLevel: "",
				Tag: TagConfig{
					Pattern: new(""),
					Filter:  new(""),
				},
			},
			want: Config{
				LogLevel: "info",
				Tag: TagConfig{
					Pattern: new(""),
					Filter:  new(""),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := merge(&tt.dst, tt.src)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, tt.dst)
		})
	}
}
