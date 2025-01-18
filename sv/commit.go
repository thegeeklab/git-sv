package sv

import "github.com/Masterminds/semver/v3"

type versionType int

const (
	none versionType = iota
	patch
	minor
	major
)

// CommitLog description of a single commit log.
type CommitLog struct {
	Date       string        `json:"date,omitempty"`
	Timestamp  int           `json:"timestamp,omitempty"`
	AuthorName string        `json:"authorName,omitempty"`
	Hash       string        `json:"hash,omitempty"`
	Message    CommitMessage `json:"message,omitempty"`
}

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

// CommitProcessor interface.
type CommitProcessor interface {
	NextVersion(version *semver.Version, commits []CommitLog) (*semver.Version, bool)
}

// SemVerCommitProcessor process versions using commit log.
type SemVerCommitProcessor struct {
	MajorVersionTypes         map[string]struct{}
	MinorVersionTypes         map[string]struct{}
	PatchVersionTypes         map[string]struct{}
	KnownTypes                []string
	IncludeUnknownTypeAsPatch bool
}

// VersioningConfig versioning preferences.
type VersioningConfig struct {
	UpdateMajor   []string `yaml:"update-major,flow"`
	UpdateMinor   []string `yaml:"update-minor,flow"`
	UpdatePatch   []string `yaml:"update-patch,flow"`
	IgnoreUnknown bool     `yaml:"ignore-unknown"`
}

// NewSemVerCommitProcessor SemanticVersionCommitProcessorImpl constructor.
func NewSemVerCommitProcessor(vcfg VersioningConfig, mcfg CommitMessageConfig) *SemVerCommitProcessor {
	return &SemVerCommitProcessor{
		IncludeUnknownTypeAsPatch: !vcfg.IgnoreUnknown,
		MajorVersionTypes:         toMap(vcfg.UpdateMajor),
		MinorVersionTypes:         toMap(vcfg.UpdateMinor),
		PatchVersionTypes:         toMap(vcfg.UpdatePatch),
		KnownTypes:                mcfg.Types,
	}
}

// NextVersion calculates next version based on commit log.
func (p SemVerCommitProcessor) NextVersion(
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

	if newVersion.Major() == 0 && newVersion.Minor() == 0 {
		newVersion = updateVersion(*version, minor)
	}

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

func (p SemVerCommitProcessor) versionTypeToUpdate(commit CommitLog) versionType {
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
