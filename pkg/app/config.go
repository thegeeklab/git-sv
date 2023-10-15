package app

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"dario.cat/mergo"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// EnvConfig env vars for cli configuration.
type EnvConfig struct {
	Home string `envconfig:"GITSV_HOME" default:""`
}

// Config cli yaml config.
type Config struct {
	Version       string              `yaml:"version"`
	LogLevel      string              `yaml:"log-level"`
	Versioning    VersioningConfig    `yaml:"versioning"`
	Tag           TagConfig           `yaml:"tag"`
	ReleaseNotes  ReleaseNotesConfig  `yaml:"release-notes"`
	Branches      BranchesConfig      `yaml:"branches"`
	CommitMessage CommitMessageConfig `yaml:"commit-message"`
}

func NewConfig(configDir, configFilename string) *Config {
	workDir, _ := os.Getwd()
	cfg := GetDefault()

	envCfg := loadEnv()
	if envCfg.Home != "" {
		homeCfgFilepath := filepath.Join(envCfg.Home, configFilename)
		if homeCfg, err := readFile(homeCfgFilepath); err == nil {
			if merr := merge(cfg, migrate(homeCfg, homeCfgFilepath)); merr != nil {
				log.Fatal().Err(merr).Msg("failed to merge user config")
			}
		}
	}

	repoCfgFilepath := filepath.Join(workDir, configDir, configFilename)
	if repoCfg, err := readFile(repoCfgFilepath); err == nil {
		if merr := merge(cfg, migrate(repoCfg, repoCfgFilepath)); merr != nil {
			log.Fatal().Err(merr).Msg("failed to merge repo config")
		}

		if len(repoCfg.ReleaseNotes.Headers) > 0 { // mergo is merging maps, headers will be overwritten
			cfg.ReleaseNotes.Headers = repoCfg.ReleaseNotes.Headers
		}
	}

	return cfg
}

func loadEnv() EnvConfig {
	var c EnvConfig

	err := envconfig.Process("", &c)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load env config")
	}

	return c
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
		Versioning: VersioningConfig{
			UpdateMajor:   []string{},
			UpdateMinor:   []string{"feat"},
			UpdatePatch:   []string{"build", "ci", "chore", "docs", "fix", "perf", "refactor", "style", "test"},
			IgnoreUnknown: false,
		},
		Tag: TagConfig{
			Pattern: &pattern,
			Filter:  &filter,
		},
		ReleaseNotes: ReleaseNotesConfig{
			Sections: []ReleaseNotesSectionConfig{
				{Name: "Features", SectionType: ReleaseNotesSectionTypeCommits, CommitTypes: []string{"feat"}},
				{Name: "Bug Fixes", SectionType: ReleaseNotesSectionTypeCommits, CommitTypes: []string{"fix"}},
				{Name: "Breaking Changes", SectionType: ReleaseNotesSectionTypeBreakingChanges},
			},
		},
		Branches: BranchesConfig{
			Prefix:       "([a-z]+\\/)?",
			Suffix:       "(-.*)?",
			DisableIssue: false,
			Skip:         []string{"master", "main", "developer"},
			SkipDetached: &skipDetached,
		},
		CommitMessage: CommitMessageConfig{
			Types: []string{"build", "ci", "chore", "docs", "feat", "fix", "perf", "refactor", "revert", "style", "test"},
			Scope: CommitMessageScopeConfig{},
			Footer: map[string]CommitMessageFooterConfig{
				"issue": {Key: "jira", KeySynonyms: []string{"Jira", "JIRA"}},
			},
			Issue:          CommitMessageIssueConfig{Regex: "[A-Z]+-[0-9]+"},
			HeaderSelector: "",
		},
	}
}

func merge(dst *Config, src Config) error {
	err := mergo.Merge(dst, src, mergo.WithOverride, mergo.WithTransformers(&mergeTransformer{}))
	if err == nil {
		if len(src.ReleaseNotes.Headers) > 0 { // mergo is merging maps, ReleaseNotes.Headers should be overwritten
			dst.ReleaseNotes.Headers = src.ReleaseNotes.Headers
		}
	}

	return err
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

func migrate(cfg Config, filename string) Config {
	if cfg.ReleaseNotes.Headers == nil {
		return cfg
	}

	log.Warn().Msgf("config 'release-notes.headers' on %s is deprecated, please use 'sections' instead!", filename)

	return Config{
		Version:    cfg.Version,
		Versioning: cfg.Versioning,
		Tag:        cfg.Tag,
		ReleaseNotes: ReleaseNotesConfig{
			Sections: migrateReleaseNotes(cfg.ReleaseNotes.Headers),
		},
		Branches:      cfg.Branches,
		CommitMessage: cfg.CommitMessage,
	}
}

func migrateReleaseNotes(headers map[string]string) []ReleaseNotesSectionConfig {
	order := []string{"feat", "fix", "refactor", "perf", "test", "build", "ci", "chore", "docs", "style"}

	var sections []ReleaseNotesSectionConfig

	for _, key := range order {
		if name, exists := headers[key]; exists {
			sections = append(
				sections,
				ReleaseNotesSectionConfig{
					Name:        name,
					SectionType: ReleaseNotesSectionTypeCommits,
					CommitTypes: []string{key},
				})
		}
	}

	if name, exists := headers["breaking-change"]; exists {
		sections = append(
			sections,
			ReleaseNotesSectionConfig{
				Name:        name,
				SectionType: ReleaseNotesSectionTypeBreakingChanges,
			})
	}

	return sections
}

// ==== Message ====

// CommitMessageConfig config a commit message.
type CommitMessageConfig struct {
	Types          []string                             `yaml:"types,flow"`
	HeaderSelector string                               `yaml:"header-selector"`
	Scope          CommitMessageScopeConfig             `yaml:"scope"`
	Footer         map[string]CommitMessageFooterConfig `yaml:"footer"`
	Issue          CommitMessageIssueConfig             `yaml:"issue"`
}

// IssueFooterConfig config for issue.
func (c CommitMessageConfig) IssueFooterConfig() CommitMessageFooterConfig {
	if v, exists := c.Footer[IssueMetadataKey]; exists {
		return v
	}

	return CommitMessageFooterConfig{}
}

// CommitMessageScopeConfig config scope preferences.
type CommitMessageScopeConfig struct {
	Values []string `yaml:"values"`
}

// CommitMessageFooterConfig config footer metadata.
type CommitMessageFooterConfig struct {
	Key            string   `yaml:"key"`
	KeySynonyms    []string `yaml:"key-synonyms,flow"`
	UseHash        bool     `yaml:"use-hash"`
	AddValuePrefix string   `yaml:"add-value-prefix"`
}

// CommitMessageIssueConfig issue preferences.
type CommitMessageIssueConfig struct {
	Regex string `yaml:"regex"`
}

// ==== Branches ====

// BranchesConfig branches preferences.
type BranchesConfig struct {
	Prefix       string   `yaml:"prefix"`
	Suffix       string   `yaml:"suffix"`
	DisableIssue bool     `yaml:"disable-issue"`
	Skip         []string `yaml:"skip,flow"`
	SkipDetached *bool    `yaml:"skip-detached"`
}

// ==== Versioning ====

// VersioningConfig versioning preferences.
type VersioningConfig struct {
	UpdateMajor   []string `yaml:"update-major,flow"`
	UpdateMinor   []string `yaml:"update-minor,flow"`
	UpdatePatch   []string `yaml:"update-patch,flow"`
	IgnoreUnknown bool     `yaml:"ignore-unknown"`
}

// ==== Tag ====

// TagConfig tag preferences.
type TagConfig struct {
	Pattern *string `yaml:"pattern"`
	Filter  *string `yaml:"filter"`
}

// ==== Release Notes ====

// ReleaseNotesConfig release notes preferences.
type ReleaseNotesConfig struct {
	Headers  map[string]string           `yaml:"headers,omitempty"`
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
