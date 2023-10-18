package commands

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/thegeeklab/git-sv/app"
	"github.com/thegeeklab/git-sv/sv"
	"github.com/urfave/cli/v2"
)

func TagFlags(settings *app.TagSettings) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "annotate",
			Aliases:     []string{"a"},
			Usage:       "ignore size parameter, get changelog for every tag",
			Destination: &settings.Annotate,
		},
	}
}

func TagHandler(g app.GitSV, settings *app.TagSettings) cli.ActionFunc {
	return func(c *cli.Context) error {
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

		tagname, err := g.Tag(*nextVer, settings.Annotate)
		if err != nil {
			return fmt.Errorf("error generating tag version: %s: %w", nextVer.String(), err)
		}

		fmt.Println(tagname)

		return nil
	}
}
