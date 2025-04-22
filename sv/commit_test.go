package sv

import (
	"reflect"
	"testing"

	"github.com/Masterminds/semver/v3"
)

func TestSemVerCommitProcessor_NextVersion(t *testing.T) {
	tests := []struct {
		name          string
		ignoreUnknown bool
		version       *semver.Version
		commits       []CommitLog
		want          *semver.Version
		wantUpdated   bool
	}{
		{
			name:          "no update",
			ignoreUnknown: true,
			version:       TestVersion("0.0.0"),
			commits:       []CommitLog{},
			want:          TestVersion("0.1.0"),
			wantUpdated:   false,
		},
		{
			name:          "no update without version",
			ignoreUnknown: true,
			version:       nil,
			commits:       []CommitLog{},
			want:          nil,
			wantUpdated:   false,
		},
		{
			name:          "no update on unknown type",
			ignoreUnknown: true,
			version:       TestVersion("0.0.0"),
			commits:       []CommitLog{TestCommitlog("a", map[string]string{}, "a")},
			want:          TestVersion("0.1.0"),
			wantUpdated:   false,
		},
		{
			name:          "no update on unmapped known type",
			ignoreUnknown: false,
			version:       TestVersion("0.0.0"),
			commits:       []CommitLog{TestCommitlog("none", map[string]string{}, "a")},
			want:          TestVersion("0.1.0"),
			wantUpdated:   false,
		},
		{
			name:          "update patch on unknown type",
			ignoreUnknown: false,
			version:       TestVersion("0.0.0"),
			commits:       []CommitLog{TestCommitlog("a", map[string]string{}, "a")},
			want:          TestVersion("0.1.0"),
			wantUpdated:   true,
		},
		{
			name:          "patch update",
			ignoreUnknown: false,
			version:       TestVersion("0.0.0"),
			commits:       []CommitLog{TestCommitlog("patch", map[string]string{}, "a")},
			want:          TestVersion("0.1.0"),
			wantUpdated:   true,
		},
		{
			name:          "patch update without version",
			ignoreUnknown: false,
			version:       nil,
			commits:       []CommitLog{TestCommitlog("patch", map[string]string{}, "a")},
			want:          nil,
			wantUpdated:   true,
		},
		{
			name:          "minor update",
			ignoreUnknown: false,
			version:       TestVersion("0.0.0"),
			commits: []CommitLog{
				TestCommitlog("patch", map[string]string{}, "a"),
				TestCommitlog("minor", map[string]string{}, "a"),
			},
			want:        TestVersion("0.1.0"),
			wantUpdated: true,
		},
		{
			name:          "major update",
			ignoreUnknown: false,
			version:       TestVersion("0.0.0"),
			commits: []CommitLog{
				TestCommitlog("patch", map[string]string{}, "a"),
				TestCommitlog("major", map[string]string{}, "a"),
			},
			want:        TestVersion("1.0.0"),
			wantUpdated: true,
		},
		{
			name:          "breaking change update",
			ignoreUnknown: false,
			version:       TestVersion("0.0.0"),
			commits: []CommitLog{
				TestCommitlog("patch", map[string]string{}, "a"),
				TestCommitlog("patch", map[string]string{"breaking-change": "break"}, "a"),
			},
			want:        TestVersion("1.0.0"),
			wantUpdated: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewSemVerCommitProcessor(
				VersioningConfig{
					UpdateMajor:   []string{"major"},
					UpdateMinor:   []string{"minor"},
					UpdatePatch:   []string{"patch"},
					IgnoreUnknown: tt.ignoreUnknown,
				},
				CommitMessageConfig{Types: []string{"major", "minor", "patch", "none"}})
			got, gotUpdated := p.NextVersion(tt.version, tt.commits)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SemVerCommitProcessor.NextVersion() Version = %v, want %v", got, tt.want)
			}

			if tt.wantUpdated != gotUpdated {
				t.Errorf("SemVerCommitProcessor.NextVersion() Updated = %v, want %v", gotUpdated, tt.wantUpdated)
			}
		})
	}
}

func TestToVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *semver.Version
		wantErr bool
	}{
		{
			name:    "empty version",
			input:   "",
			want:    TestVersion("0.0.0"),
			wantErr: false,
		},
		{
			name:    "invalid version",
			input:   "abc",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "valid version",
			input:   "1.2.3",
			want:    TestVersion("1.2.3"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToVersion() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{
			name:  "simple version",
			value: "1.0.0",
			want:  true,
		},
		{
			name:  "with v prefix version",
			value: "v1.0.0",
			want:  true,
		},
		{
			name:  "prerelease version",
			value: "1.0.0-alpha",
			want:  true,
		},
		{
			name:  "prerelease version",
			value: "1.0.0-alpha.1",
			want:  true,
		},
		{
			name:  "prerelease version",
			value: "1.0.0-0.3.7",
			want:  true,
		},
		{
			name:  "prerelease version",
			value: "1.0.0-x.7.z.92",
			want:  true,
		},
		{
			name:  "prerelease version",
			value: "1.0.0-x-y-z.-",
			want:  true,
		},
		{
			name:  "metadata version",
			value: "1.0.0-alpha+001",
			want:  true,
		},
		{
			name:  "metadata version",
			value: "1.0.0+20130313144700",
			want:  true,
		},
		{
			name:  "metadata version",
			value: "1.0.0-beta+exp.sha.5114f85",
			want:  true,
		},
		{
			name:  "metadata version",
			value: "1.0.0+21AF26D3-117B344092BD",
			want:  true,
		},
		{
			name:  "incomplete version",
			value: "1",
			want:  true,
		},
		{
			name:  "invalid version",
			value: "invalid",
			want:  false,
		},
		{
			name:  "invalid prefix version",
			value: "random1.0.0",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidVersion(tt.value); got != tt.want {
				t.Errorf("IsValidVersion(%s) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
