package app

import "github.com/Masterminds/semver/v3"

type versionType int

const (
	none versionType = iota
	patch
	minor
	major
)

// IsValidVersion return true when a version is valid.
func IsValidVersion(value string) bool {
	_, err := semver.NewVersion(value)

	return err == nil
}

// ToVersion parse string to semver.Version.
func ToVersion(value string) (*semver.Version, error) {
	version := value
	if version == "" {
		version = "0.0.0"
	}

	return semver.NewVersion(version)
}

// CommitsProcessor interface.
type CommitsProcessor interface {
	NextVersion(version *semver.Version, commits []CommitLog) (*semver.Version, bool)
}

// SemVerCommitsProcessor process versions using commit log.
type SemVerCommitsProcessor struct {
	MajorVersionTypes         map[string]struct{}
	MinorVersionTypes         map[string]struct{}
	PatchVersionTypes         map[string]struct{}
	KnownTypes                []string
	IncludeUnknownTypeAsPatch bool
}

// NewSemVerCommitsProcessor SemanticVersionCommitsProcessorImpl constructor.
func NewSemVerCommitsProcessor(vcfg VersioningConfig, mcfg CommitMessageConfig) *SemVerCommitsProcessor {
	return &SemVerCommitsProcessor{
		IncludeUnknownTypeAsPatch: !vcfg.IgnoreUnknown,
		MajorVersionTypes:         toMap(vcfg.UpdateMajor),
		MinorVersionTypes:         toMap(vcfg.UpdateMinor),
		PatchVersionTypes:         toMap(vcfg.UpdatePatch),
		KnownTypes:                mcfg.Types,
	}
}

// NextVersion calculates next version based on commit log.
func (p SemVerCommitsProcessor) NextVersion(
	version *semver.Version, commits []CommitLog,
) (*semver.Version, bool) {
	versionToUpdate := none
	for _, commit := range commits {
		if v := p.versionTypeToUpdate(commit); v > versionToUpdate {
			versionToUpdate = v
		}
	}

	updated := versionToUpdate != none
	if version == nil {
		return nil, updated
	}

	newVersion := updateVersion(*version, versionToUpdate)

	return &newVersion, updated
}

func updateVersion(version semver.Version, versionToUpdate versionType) semver.Version {
	switch versionToUpdate {
	case major:
		return version.IncMajor()
	case minor:
		return version.IncMinor()
	case patch:
		return version.IncPatch()
	default:
		return version
	}
}

func (p SemVerCommitsProcessor) versionTypeToUpdate(commit CommitLog) versionType {
	if commit.Message.IsBreakingChange {
		return major
	}

	if _, exists := p.MajorVersionTypes[commit.Message.Type]; exists {
		return major
	}

	if _, exists := p.MinorVersionTypes[commit.Message.Type]; exists {
		return minor
	}

	if _, exists := p.PatchVersionTypes[commit.Message.Type]; exists {
		return patch
	}

	if !contains(commit.Message.Type, p.KnownTypes) && p.IncludeUnknownTypeAsPatch {
		return patch
	}

	return none
}

func toMap(values []string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, v := range values {
		result[v] = struct{}{}
	}

	return result
}
