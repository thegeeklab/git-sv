package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/thegeeklab/git-sv/app"
	"github.com/thegeeklab/git-sv/sv"
	"github.com/urfave/cli/v3"
)

func TagFlags(settings *app.TagSettings) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "annotate",
			Aliases:     []string{"a"},
			Usage:       "make an annotated tag object",
			Destination: &settings.Annotate,
		},
		&cli.BoolFlag{
			Name:        "local",
			Usage:       "create local tag only",
			Destination: &settings.Local,
		},
	}
}

func TagHandler(g app.GitSV, settings *app.TagSettings) cli.ActionFunc {
	return func(_ context.Context, _ *cli.Command) error {
		lastTag := g.LastTag()

		currentVer, err := sv.ToVersion(lastTag)
		if err != nil {
			return fmt.Errorf("error parsing version: %s from git tag: %w", lastTag, err)
		}

		commits, err := g.Log(app.NewLogRange(app.TagRange, lastTag, ""))
		if err != nil {
			return fmt.Errorf("error getting git log: %w", err)
		}

		nextVer, updated := g.CommitProcessor.NextVersion(currentVer, commits)
		if !updated {
			log.Info().Msgf("nothing to do: current version %s unchanged", currentVer)

			return nil
		}

		tagname, err := g.Tag(*nextVer, settings.Annotate, settings.Local)
		if err != nil {
			return fmt.Errorf("error generating tag version: %s: %w", nextVer.String(), err)
		}

		fmt.Println(tagname)

		return nil
	}
}
