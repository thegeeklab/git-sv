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
			"no update",
			true,
			TestVersion("0.0.0"),
			[]CommitLog{},
			TestVersion("0.1.0"),
			false,
		},
		{
			"no update without version",
			true,
			nil,
			[]CommitLog{},
			nil,
			false,
		},
		{
			"no update on unknown type",
			true,
			TestVersion("0.0.0"),
			[]CommitLog{TestCommitlog("a", map[string]string{}, "a")},
			TestVersion("0.1.0"),
			false,
		},
		{
			"no update on unmapped known type",
			false,
			TestVersion("0.0.0"),
			[]CommitLog{TestCommitlog("none", map[string]string{}, "a")},
			TestVersion("0.1.0"),
			false,
		},
		{
			"update patch on unknown type",
			false,
			TestVersion("0.0.0"),
			[]CommitLog{TestCommitlog("a", map[string]string{}, "a")},
			TestVersion("0.1.0"),
			true,
		},
		{
			"patch update",
			false, TestVersion("0.0.0"),
			[]CommitLog{TestCommitlog("patch", map[string]string{}, "a")},
			TestVersion("0.1.0"), true,
		},
		{
			"patch update without version",
			false,
			nil,
			[]CommitLog{TestCommitlog("patch", map[string]string{}, "a")},
			nil,
			true,
		},
		{
			"minor update",
			false,
			TestVersion("0.0.0"),
			[]CommitLog{
				TestCommitlog("patch", map[string]string{}, "a"),
				TestCommitlog("minor", map[string]string{}, "a"),
			},
			TestVersion("0.1.0"),
			true,
		},
		{
			"major update",
			false,
			TestVersion("0.0.0"),
			[]CommitLog{
				TestCommitlog("patch", map[string]string{}, "a"),
				TestCommitlog("major", map[string]string{}, "a"),
			},
			TestVersion("1.0.0"),
			true,
		},
		{
			"breaking change update",
			false,
			TestVersion("0.0.0"),
			[]CommitLog{
				TestCommitlog("patch", map[string]string{}, "a"),
				TestCommitlog("patch", map[string]string{"breaking-change": "break"}, "a"),
			},
			TestVersion("1.0.0"),
			true,
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
		{"empty version", "", TestVersion("0.0.0"), false},
		{"invalid version", "abc", nil, true},
		{"valid version", "1.2.3", TestVersion("1.2.3"), false},
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
		{"simple version", "1.0.0", true},
		{"with v prefix version", "v1.0.0", true},
		{"prerelease version", "1.0.0-alpha", true},
		{"prerelease version", "1.0.0-alpha.1", true},
		{"prerelease version", "1.0.0-0.3.7", true},
		{"prerelease version", "1.0.0-x.7.z.92", true},
		{"prerelease version", "1.0.0-x-y-z.-", true},
		{"metadata version", "1.0.0-alpha+001", true},
		{"metadata version", "1.0.0+20130313144700", true},
		{"metadata version", "1.0.0-beta+exp.sha.5114f85", true},
		{"metadata version", "1.0.0+21AF26D3-117B344092BD", true},
		{"incomplete version", "1", true},
		{"invalid version", "invalid", false},
		{"invalid prefix version", "random1.0.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidVersion(tt.value); got != tt.want {
				t.Errorf("IsValidVersion(%s) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
