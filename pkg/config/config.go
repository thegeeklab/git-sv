package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"dario.cat/mergo"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
	"github.com/thegeeklab/git-sv/v2/pkg/git"
	"gopkg.in/yaml.v3"
)

// EnvConfig env vars for cli configuration.
type EnvConfig struct {
	Home string `envconfig:"GITSV_HOME" default:""`
}

// Config cli yaml config.
type Config struct {
	Version       string                  `yaml:"version"`
	LogLevel      string                  `yaml:"log-level"`
	Versioning    git.VersioningConfig    `yaml:"versioning"`
	Tag           git.TagConfig           `yaml:"tag"`
	ReleaseNotes  git.ReleaseNotesConfig  `yaml:"release-notes"`
	Branches      git.BranchesConfig      `yaml:"branches"`
	CommitMessage git.CommitMessageConfig `yaml:"commit-message"`
}

func New(configDir, configFilename string) *Config {
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
		Versioning: git.VersioningConfig{
			UpdateMajor:   []string{},
			UpdateMinor:   []string{"feat"},
			UpdatePatch:   []string{"build", "ci", "chore", "docs", "fix", "perf", "refactor", "style", "test"},
			IgnoreUnknown: false,
		},
		Tag: git.TagConfig{
			Pattern: &pattern,
			Filter:  &filter,
		},
		ReleaseNotes: git.ReleaseNotesConfig{
			Sections: []git.ReleaseNotesSectionConfig{
				{Name: "Features", SectionType: git.ReleaseNotesSectionTypeCommits, CommitTypes: []string{"feat"}},
				{Name: "Bug Fixes", SectionType: git.ReleaseNotesSectionTypeCommits, CommitTypes: []string{"fix"}},
				{Name: "Breaking Changes", SectionType: git.ReleaseNotesSectionTypeBreakingChanges},
			},
		},
		Branches: git.BranchesConfig{
			Prefix:       "([a-z]+\\/)?",
			Suffix:       "(-.*)?",
			DisableIssue: false,
			Skip:         []string{"master", "main", "developer"},
			SkipDetached: &skipDetached,
		},
		CommitMessage: git.CommitMessageConfig{
			Types: []string{"build", "ci", "chore", "docs", "feat", "fix", "perf", "refactor", "revert", "style", "test"},
			Scope: git.CommitMessageScopeConfig{},
			Footer: map[string]git.CommitMessageFooterConfig{
				"issue": {Key: "jira", KeySynonyms: []string{"Jira", "JIRA"}},
			},
			Issue:          git.CommitMessageIssueConfig{Regex: "[A-Z]+-[0-9]+"},
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
		ReleaseNotes: git.ReleaseNotesConfig{
			Sections: migrateReleaseNotes(cfg.ReleaseNotes.Headers),
		},
		Branches:      cfg.Branches,
		CommitMessage: cfg.CommitMessage,
	}
}

func migrateReleaseNotes(headers map[string]string) []git.ReleaseNotesSectionConfig {
	order := []string{"feat", "fix", "refactor", "perf", "test", "build", "ci", "chore", "docs", "style"}

	var sections []git.ReleaseNotesSectionConfig

	for _, key := range order {
		if name, exists := headers[key]; exists {
			sections = append(
				sections,
				git.ReleaseNotesSectionConfig{
					Name:        name,
					SectionType: git.ReleaseNotesSectionTypeCommits,
					CommitTypes: []string{key},
				})
		}
	}

	if name, exists := headers["breaking-change"]; exists {
		sections = append(
			sections,
			git.ReleaseNotesSectionConfig{
				Name:        name,
				SectionType: git.ReleaseNotesSectionTypeBreakingChanges,
			})
	}

	return sections
}
