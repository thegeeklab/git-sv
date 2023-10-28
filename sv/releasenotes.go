package sv

import (
	"time"

	"github.com/Masterminds/semver/v3"
)

// ReleaseNotesConfig release notes preferences.
type ReleaseNotesConfig struct {
	Sections []ReleaseNotesSectionConfig `yaml:"sections"`
}

func (cfg ReleaseNotesConfig) sectionConfig(sectionType string) *ReleaseNotesSectionConfig {
	for _, sectionCfg := range cfg.Sections {
		if sectionCfg.SectionType == sectionType {
			return &sectionCfg
		}
	}

	return nil
}

// ReleaseNotesSectionConfig preferences for a single section on release notes.
type ReleaseNotesSectionConfig struct {
	Name        string   `yaml:"name"`
	SectionType string   `yaml:"section-type"`
	CommitTypes []string `yaml:"commit-types,flow,omitempty"`
}

const (
	// ReleaseNotesSectionTypeCommits ReleaseNotesSectionConfig.SectionType value.
	ReleaseNotesSectionTypeCommits = "commits"
	// ReleaseNotesSectionTypeBreakingChanges ReleaseNotesSectionConfig.SectionType value.
	ReleaseNotesSectionTypeBreakingChanges = "breaking-changes"
)

// ReleaseNoteProcessor release note processor interface.
type ReleaseNoteProcessor interface {
	Create(version *semver.Version, tag string, date time.Time, commits []CommitLog) ReleaseNote
}

// BaseReleaseNoteProcessor release note based on commit log.
type BaseReleaseNoteProcessor struct {
	cfg ReleaseNotesConfig
}

// NewReleaseNoteProcessor ReleaseNoteProcessor constructor.
func NewReleaseNoteProcessor(cfg ReleaseNotesConfig) *BaseReleaseNoteProcessor {
	return &BaseReleaseNoteProcessor{cfg: cfg}
}

// Create create a release note based on commits.
func (p BaseReleaseNoteProcessor) Create(
	version *semver.Version,
	tag string,
	date time.Time,
	commits []CommitLog,
) ReleaseNote {
	mapping := commitSectionMapping(p.cfg.Sections)

	sections := make(map[string]ReleaseNoteCommitsSection)
	authors := make(map[string]struct{})

	var breakingChanges []string

	for _, commit := range commits {
		authors[commit.AuthorName] = struct{}{}

		if sectionCfg, exists := mapping[commit.Message.Type]; exists {
			section, sexists := sections[sectionCfg.Name]
			if !sexists {
				section = ReleaseNoteCommitsSection{Name: sectionCfg.Name, Types: sectionCfg.CommitTypes}
			}

			section.Items = append(section.Items, commit)
			sections[sectionCfg.Name] = section
		}

		if commit.Message.IsBreakingChange {
			breakingChanges = append(breakingChanges, commit.Message.BreakingMessage())
		}
	}

	var breakingChangeSection ReleaseNoteBreakingChangeSection
	if bcCfg := p.cfg.sectionConfig(ReleaseNotesSectionTypeBreakingChanges); bcCfg != nil && len(breakingChanges) > 0 {
		breakingChangeSection = ReleaseNoteBreakingChangeSection{Name: bcCfg.Name, Messages: breakingChanges}
	}

	return ReleaseNote{
		Version:      version,
		Tag:          tag,
		Date:         date.Truncate(time.Minute),
		Sections:     p.toReleaseNoteSections(sections, breakingChangeSection),
		AuthorsNames: authors,
	}
}

func (p BaseReleaseNoteProcessor) toReleaseNoteSections(
	commitSections map[string]ReleaseNoteCommitsSection,
	breakingChange ReleaseNoteBreakingChangeSection,
) []ReleaseNoteSection {
	hasBreaking := 0
	if breakingChange.Name != "" {
		hasBreaking = 1
	}

	sections := make([]ReleaseNoteSection, len(commitSections)+hasBreaking)
	i := 0

	for _, cfg := range p.cfg.Sections {
		if cfg.SectionType == ReleaseNotesSectionTypeBreakingChanges && hasBreaking > 0 {
			sections[i] = breakingChange
			i++
		}

		if s, exists := commitSections[cfg.Name]; cfg.SectionType == ReleaseNotesSectionTypeCommits && exists {
			sections[i] = s
			i++
		}
	}

	return sections
}

func commitSectionMapping(sections []ReleaseNotesSectionConfig) map[string]ReleaseNotesSectionConfig {
	mapping := make(map[string]ReleaseNotesSectionConfig)

	for _, section := range sections {
		if section.SectionType == ReleaseNotesSectionTypeCommits {
			for _, commitType := range section.CommitTypes {
				mapping[commitType] = section
			}
		}
	}

	return mapping
}

// ReleaseNote release note.
type ReleaseNote struct {
	Version      *semver.Version
	Tag          string
	Date         time.Time
	Sections     []ReleaseNoteSection
	AuthorsNames map[string]struct{}
}

// ReleaseNoteSection section in release notes.
type ReleaseNoteSection interface {
	SectionType() string
	SectionName() string
}

// ReleaseNoteBreakingChangeSection breaking change section.
type ReleaseNoteBreakingChangeSection struct {
	Name     string
	Messages []string
}

// SectionType section type.
func (ReleaseNoteBreakingChangeSection) SectionType() string {
	return ReleaseNotesSectionTypeBreakingChanges
}

// SectionName section name.
func (s ReleaseNoteBreakingChangeSection) SectionName() string {
	return s.Name
}

// ReleaseNoteCommitsSection release note section.
type ReleaseNoteCommitsSection struct {
	Name  string
	Types []string
	Items []CommitLog
}

// SectionType section type.
func (ReleaseNoteCommitsSection) SectionType() string {
	return ReleaseNotesSectionTypeCommits
}

// SectionName section name.
func (s ReleaseNoteCommitsSection) SectionName() string {
	return s.Name
}

// HasMultipleTypes return true if has more than one commit type.
func (s ReleaseNoteCommitsSection) HasMultipleTypes() bool {
	return len(s.Types) > 1
}
