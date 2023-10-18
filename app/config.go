package app

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"dario.cat/mergo"
	"github.com/rs/zerolog/log"
	"github.com/thegeeklab/git-sv/sv"
	"gopkg.in/yaml.v3"
)

type Settings struct {
	LogLevel string

	ChangelogSettings    ChangelogSettings
	ReleaseNotesSettings ReleaseNotesSettings
	CommitNotesSettings  CommitNotesSettings
	CommitLogSettings    CommitLogSettings
	TagSettings          TagSettings
}

type ChangelogSettings struct {
	Size    int
	All     bool
	AddNext bool
	Strict  bool
	Out     string
}

type ReleaseNotesSettings struct {
	Tag string
	Out string
}

type CommitNotesSettings struct {
	Range string
	Start string
	End   string
	Out   string
}

type CommitLogSettings struct {
	Tag   string
	Range string
	Start string
	End   string
}

type TagSettings struct {
	Annotate bool
}

// Config cli yaml config.
type Config struct {
	Version       string                 `yaml:"version"`
	LogLevel      string                 `yaml:"log-level"`
	Versioning    sv.VersioningConfig    `yaml:"versioning"`
	Tag           TagConfig              `yaml:"tag"`
	ReleaseNotes  sv.ReleaseNotesConfig  `yaml:"release-notes"`
	Branches      sv.BranchesConfig      `yaml:"branches"`
	CommitMessage sv.CommitMessageConfig `yaml:"commit-message"`
}

// TagConfig tag preferences.
type TagConfig struct {
	Pattern *string `yaml:"pattern"`
	Filter  *string `yaml:"filter"`
}

func NewConfig(configDir, configFilename string) *Config {
	workDir, _ := os.Getwd()
	cfg := GetDefault()

	repoCfgFilepath := filepath.Join(workDir, configDir, configFilename)
	if repoCfg, err := readFile(repoCfgFilepath); err == nil {
		if merr := merge(cfg, migrate(repoCfg, repoCfgFilepath)); merr != nil {
			log.Fatal().Err(merr).Msg("failed to merge repo config")
		}
	}

	return cfg
}

func readFile(filepath string) (Config, error) {
	content, rerr := os.ReadFile(filepath)
	if rerr != nil {
		return Config{}, rerr
	}

	var cfg Config

	cerr := yaml.Unmarshal(content, &cfg)
	if cerr != nil {
		return Config{}, fmt.Errorf("could not parse config from path: %s, error: %w", filepath, cerr)
	}

	return cfg, nil
}

func GetDefault() *Config {
	skipDetached := false
	pattern := "%d.%d.%d"
	filter := ""

	return &Config{
		Version: "1.1",
		Versioning: sv.VersioningConfig{
			UpdateMajor:   []string{},
			UpdateMinor:   []string{"feat"},
			UpdatePatch:   []string{"build", "ci", "chore", "docs", "fix", "perf", "refactor", "style", "test"},
			IgnoreUnknown: false,
		},
		Tag: TagConfig{
			Pattern: &pattern,
			Filter:  &filter,
		},
		ReleaseNotes: sv.ReleaseNotesConfig{
			Sections: []sv.ReleaseNotesSectionConfig{
				{Name: "Features", SectionType: sv.ReleaseNotesSectionTypeCommits, CommitTypes: []string{"feat"}},
				{Name: "Bug Fixes", SectionType: sv.ReleaseNotesSectionTypeCommits, CommitTypes: []string{"fix"}},
				{Name: "Breaking Changes", SectionType: sv.ReleaseNotesSectionTypeBreakingChanges},
			},
		},
		Branches: sv.BranchesConfig{
			Prefix:       "([a-z]+\\/)?",
			Suffix:       "(-.*)?",
			DisableIssue: false,
			Skip:         []string{"master", "main", "developer"},
			SkipDetached: &skipDetached,
		},
		CommitMessage: sv.CommitMessageConfig{
			Types: []string{"build", "ci", "chore", "docs", "feat", "fix", "perf", "refactor", "revert", "style", "test"},
			Scope: sv.CommitMessageScopeConfig{},
			Footer: map[string]sv.CommitMessageFooterConfig{
				"issue": {Key: "jira", KeySynonyms: []string{"Jira", "JIRA"}},
			},
			Issue:          sv.CommitMessageIssueConfig{Regex: "[A-Z]+-[0-9]+"},
			HeaderSelector: "",
		},
	}
}

func merge(dst *Config, src Config) error {
	return mergo.Merge(dst, src, mergo.WithOverride, mergo.WithTransformers(&mergeTransformer{}))
}

type mergeTransformer struct{}

func (t *mergeTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ.Kind() == reflect.Slice {
		return func(dst, src reflect.Value) error {
			if dst.CanSet() && !src.IsNil() {
				dst.Set(src)
			}

			return nil
		}
	}

	if typ.Kind() == reflect.Ptr {
		return func(dst, src reflect.Value) error {
			if dst.CanSet() && !src.IsNil() {
				dst.Set(src)
			}

			return nil
		}
	}

	return nil
}

//nolint:revive
func migrate(cfg Config, filename string) Config {
	return cfg
}
