package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/thegeeklab/git-sv/app"
	"github.com/thegeeklab/git-sv/app/commands"
	"github.com/urfave/cli/v3"
)

//nolint:gochecknoglobals
var (
	BuildVersion = "devel"
	BuildDate    = "00000000"
)

func main() {
	gsv := app.New()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	cli.VersionPrinter = func(c *cli.Command) {
		fmt.Printf("%s version=%s date=%s\n", c.Name, c.Version, BuildDate)
	}

	app := &cli.Command{
		Name:    "git-sv",
		Usage:   "Semantic version for git.",
		Version: BuildVersion,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				Usage:       "log level",
				Value:       "info",
				Destination: &gsv.Settings.LogLevel,
			},
		},
		Before: func(ctx context.Context, _ *cli.Command) (context.Context, error) {
			lvl, err := zerolog.ParseLevel(gsv.Settings.LogLevel)
			if err != nil {
				return ctx, err
			}

			zerolog.SetGlobalLevel(lvl)

			return ctx, nil
		},
		Commands: []*cli.Command{
			{
				Name:    "config",
				Aliases: []string{"cfg"},
				Usage:   "cli configuration",
				Commands: []*cli.Command{
					{
						Name:   "default",
						Usage:  "show default config",
						Action: commands.ConfigDefaultHandler(),
					},
					{
						Name:   "show",
						Usage:  "show current config",
						Action: commands.ConfigShowHandler(gsv.Config),
					},
				},
			},
			{
				Name:    "current-version",
				Aliases: []string{"cv"},
				Usage:   "get last released version from git",
				Action:  commands.CurrentVersionHandler(gsv),
			},
			{
				Name:    "next-version",
				Aliases: []string{"nv"},
				Usage:   "generate the next version based on git commit messages",
				Action:  commands.NextVersionHandler(gsv),
			},
			{
				Name:    "commit-log",
				Aliases: []string{"cl"},
				Usage:   "list all commit logs according to range as json",
				Description: `The range filter is used based on git log filters, check https://git-scm.com/docs/git-log
for more info. When flag range is "tag" and start is empty, last tag created will be used instead.
When flag range is "date", if "end" is YYYY-MM-DD the range will be inclusive.`,
				Action: commands.CommitLogHandler(gsv, &gsv.Settings.CommitLogSettings),
				Flags:  commands.CommitLogFlags(&gsv.Settings.CommitLogSettings),
			},
			{
				Name:    "commit-notes",
				Aliases: []string{"cn"},
				Usage:   "generate a commit notes according to range",
				Description: `The range filter is used based on git log filters, check https://git-scm.com/docs/git-log
for more info. When flag range is "tag" and start is empty, last tag created will be used instead.
When flag range is "date", if "end" is YYYY-MM-DD the range will be inclusive.`,
				Action: commands.CommitNotesHandler(gsv, &gsv.Settings.CommitNotesSettings),
				Flags:  commands.CommitNotesFlags(&gsv.Settings.CommitNotesSettings),
			},
			{
				Name:    "release-notes",
				Aliases: []string{"rn"},
				Usage:   "generate release notes",
				Action:  commands.ReleaseNotesHandler(gsv, &gsv.Settings.ReleaseNotesSettings),
				Flags:   commands.ReleaseNotesFlags(&gsv.Settings.ReleaseNotesSettings),
			},
			{
				Name:    "changelog",
				Aliases: []string{"cgl"},
				Usage:   "generate changelog",
				Action:  commands.ChangelogHandler(gsv, &gsv.Settings.ChangelogSettings),
				Flags:   commands.ChangelogFlags(&gsv.Settings.ChangelogSettings),
			},
			{
				Name:    "tag",
				Aliases: []string{"tg"},
				Usage:   "generate tag with version based on git commit messages",
				Action:  commands.TagHandler(gsv, &gsv.Settings.TagSettings),
				Flags:   commands.TagFlags(&gsv.Settings.TagSettings),
			},
			{
				Name:    "commit",
				Aliases: []string{"cmt"},
				Usage:   "execute git commit with conventional commit message helper",
				Action:  commands.CommitHandler(gsv),
				Flags:   commands.CommitFlags(),
			},
			{
				Name:    "validate-commit-message",
				Aliases: []string{"vcm"},
				Usage:   "use as prepare-commit-message hook to validate and enhance commit message",
				Action:  commands.ValidateCommitMessageHandler(gsv),
				Flags:   commands.ValidateCommitMessageFlags(),
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Execution error")
	}
}
