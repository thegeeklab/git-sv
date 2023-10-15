package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/thegeeklab/git-sv/v2/pkg/app"
	"github.com/thegeeklab/git-sv/v2/pkg/commands"
	"github.com/urfave/cli/v2"
)

//nolint:gochecknoglobals
var (
	BuildVersion = "devel"
	BuildDate    = "00000000"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	g := app.New()

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("%s version=%s date=%s\n", c.App.Name, c.App.Version, BuildDate)
	}

	app := &cli.App{
		Name:    "git-sv",
		Usage:   "Semantic version for app.",
		Version: BuildVersion,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "log level",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "config",
				Aliases: []string{"cfg"},
				Usage:   "cli configuration",
				Subcommands: []*cli.Command{
					{
						Name:   "default",
						Usage:  "show default config",
						Action: commands.ConfigDefaultHandler(),
					},
					{
						Name:   "show",
						Usage:  "show current config",
						Action: commands.ConfigShowHandler(g.Config),
					},
				},
			},
			{
				Name:    "current-version",
				Aliases: []string{"cv"},
				Usage:   "get last released version from git",
				Action:  commands.CurrentVersionHandler(g),
			},
			{
				Name:    "next-version",
				Aliases: []string{"nv"},
				Usage:   "generate the next version based on git commit messages",
				Action:  commands.NextVersionHandler(g),
			},
			{
				Name:    "commit-log",
				Aliases: []string{"cl"},
				Usage:   "list all commit logs according to range as json",
				Description: `The range filter is used based on git log filters, check https://git-scm.com/docs/git-log
for more info. When flag range is "tag" and start is empty, last tag created will be used instead.
When flag range is "date", if "end" is YYYY-MM-DD the range will be inclusive.`,
				Action: commands.CommitLogHandler(g),
				Flags:  commands.CommitLogFlags(),
			},
			{
				Name:    "commit-notes",
				Aliases: []string{"cn"},
				Usage:   "generate a commit notes according to range",
				Description: `The range filter is used based on git log filters, check https://git-scm.com/docs/git-log
for more info. When flag range is "tag" and start is empty, last tag created will be used instead.
When flag range is "date", if "end" is YYYY-MM-DD the range will be inclusive.`,
				Action: commands.CommitNotesHandler(g),
				Flags:  commands.CommitNotesFlags(),
			},
			{
				Name:    "release-notes",
				Aliases: []string{"rn"},
				Usage:   "generate release notes",
				Action:  commands.ReleaseNotesHandler(g),
				Flags:   commands.ReleaseNotesFlags(),
			},
			{
				Name:    "changelog",
				Aliases: []string{"cgl"},
				Usage:   "generate changelog",
				Action:  commands.ChangelogHandler(g),
				Flags:   commands.ChangelogFlags(),
			},
			{
				Name:    "tag",
				Aliases: []string{"tg"},
				Usage:   "generate tag with version based on git commit messages",
				Action:  commands.TagHandler(g),
			},
			{
				Name:    "commit",
				Aliases: []string{"cmt"},
				Usage:   "execute git commit with conventional commit message helper",
				Action:  commands.CommitHandler(g),
				Flags:   commands.CommitFlags(),
			},
			{
				Name:    "validate-commit-message",
				Aliases: []string{"vcm"},
				Usage:   "use as prepare-commit-message hook to validate and enhance commit message",
				Action:  commands.ValidateCommitMessageHandler(g),
				Flags:   commands.ValidateCommitMessageFlags(),
			},
		},
	}

	if apperr := app.Run(os.Args); apperr != nil {
		log.Fatal().Err(apperr)
	}
}
