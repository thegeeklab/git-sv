package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/thegeeklab/git-sv/v2/sv"
	"github.com/urfave/cli/v2"
)

//nolint:gochecknoglobals
var (
	BuildVersion = "devel"
	BuildDate    = "00000000"
)

const (
	configFilename = "config.yml"
	configDir      = ".gitsv"
)

//go:embed resources/templates/*.tpl
var defaultTemplatesFS embed.FS

func templateFS(filepath string) fs.FS {
	if _, err := os.Stat(filepath); err != nil {
		defaultTemplatesFS, _ := fs.Sub(defaultTemplatesFS, "resources/templates")

		return defaultTemplatesFS
	}

	return os.DirFS(filepath)
}

func main() {
	log.SetFlags(0)

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("error while retrieving working directory: %w", err)
	}

	cfg := loadCfg(wd)
	messageProcessor := sv.NewMessageProcessor(cfg.CommitMessage, cfg.Branches)
	git := sv.NewGit(messageProcessor, cfg.Tag)
	semverProcessor := sv.NewSemVerCommitsProcessor(cfg.Versioning, cfg.CommitMessage)
	releasenotesProcessor := sv.NewReleaseNoteProcessor(cfg.ReleaseNotes)
	outputFormatter := sv.NewOutputFormatter(templateFS(filepath.Join(wd, configDir, "templates")))

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("%s version=%s date=%s\n", c.App.Name, c.App.Version, BuildDate)
	}

	app := &cli.App{
		Name:    "git-sv",
		Usage:   "Semantic version for git.",
		Version: BuildVersion,
		Commands: []*cli.Command{
			{
				Name:    "config",
				Aliases: []string{"cfg"},
				Usage:   "cli configuration",
				Subcommands: []*cli.Command{
					{
						Name:   "default",
						Usage:  "show default config",
						Action: configDefaultHandler(),
					},
					{
						Name:   "show",
						Usage:  "show current config",
						Action: configShowHandler(cfg),
					},
				},
			},
			{
				Name:    "current-version",
				Aliases: []string{"cv"},
				Usage:   "get last released version from git",
				Action:  currentVersionHandler(git),
			},
			{
				Name:    "next-version",
				Aliases: []string{"nv"},
				Usage:   "generate the next version based on git commit messages",
				Action:  nextVersionHandler(git, semverProcessor),
			},
			{
				Name:    "commit-log",
				Aliases: []string{"cl"},
				Usage:   "list all commit logs according to range as jsons",
				Description: `The range filter is used based on git log filters, check https://git-scm.com/docs/git-log
for more info. When flag range is "tag" and start is empty, last tag created will be used instead.
When flag range is "date", if "end" is YYYY-MM-DD the range will be inclusive.`,
				Action: commitLogHandler(git),
				Flags:  commitLogFlags(),
			},
			{
				Name:    "commit-notes",
				Aliases: []string{"cn"},
				Usage:   "generate a commit notes according to range",
				Description: `The range filter is used based on git log filters, check https://git-scm.com/docs/git-log
for more info. When flag range is "tag" and start is empty, last tag created will be used instead.
When flag range is "date", if "end" is YYYY-MM-DD the range will be inclusive.`,
				Action: commitNotesHandler(git, releasenotesProcessor, outputFormatter),
				Flags:  commitNotesFlags(),
			},
			{
				Name:    "release-notes",
				Aliases: []string{"rn"},
				Usage:   "generate release notes",
				Action:  releaseNotesHandler(git, semverProcessor, releasenotesProcessor, outputFormatter),
				Flags:   releaseNotesFlags(),
			},
			{
				Name:    "changelog",
				Aliases: []string{"cgl"},
				Usage:   "generate changelog",
				Action:  changelogHandler(git, semverProcessor, releasenotesProcessor, outputFormatter),
				Flags:   changelogFlags(),
			},
			{
				Name:    "tag",
				Aliases: []string{"tg"},
				Usage:   "generate tag with version based on git commit messages",
				Action:  tagHandler(git, semverProcessor),
			},
			{
				Name:    "commit",
				Aliases: []string{"cmt"},
				Usage:   "execute git commit with convetional commit message helper",
				Action:  commitHandler(cfg, git, messageProcessor),
				Flags:   commitFlags(),
			},
			{
				Name:    "validate-commit-message",
				Aliases: []string{"vcm"},
				Usage:   "use as prepare-commit-message hook to validate and enhance commit message",
				Action:  validateCommitMessageHandler(git, messageProcessor),
				Flags:   validateCommitMessageFlags(),
			},
		},
	}

	if apperr := app.Run(os.Args); apperr != nil {
		log.Fatal("ERROR: ", apperr)
	}
}

func loadCfg(wd string) Config {
	cfg := defaultConfig()

	envCfg := loadEnvConfig()
	if envCfg.Home != "" {
		homeCfgFilepath := filepath.Join(envCfg.Home, configFilename)
		if homeCfg, err := readConfig(homeCfgFilepath); err == nil {
			if merr := merge(&cfg, migrateConfig(homeCfg, homeCfgFilepath)); merr != nil {
				log.Fatal("failed to merge user config, error: ", merr)
			}
		}
	}

	repoCfgFilepath := filepath.Join(wd, configDir, configFilename)
	if repoCfg, err := readConfig(repoCfgFilepath); err == nil {
		if merr := merge(&cfg, migrateConfig(repoCfg, repoCfgFilepath)); merr != nil {
			log.Fatal("failed to merge repo config, error: ", merr)
		}

		if len(repoCfg.ReleaseNotes.Headers) > 0 { // mergo is merging maps, headers will be overwritten
			cfg.ReleaseNotes.Headers = repoCfg.ReleaseNotes.Headers
		}
	}

	return cfg
}
