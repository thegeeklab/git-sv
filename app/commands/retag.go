package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/thegeeklab/git-sv/app"
	"github.com/thegeeklab/git-sv/sv"
	"github.com/urfave/cli/v3"
)

var errNoTagToRetag = errors.New("no tag found to retag")

func RetagFlags(settings *app.RetagSettings) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "annotate",
			Aliases:     []string{"a"},
			Usage:       "make an annotated tag object",
			Destination: &settings.Annotate,
		},
		&cli.BoolFlag{
			Name:        "local",
			Usage:       "retag local tag only",
			Destination: &settings.Local,
		},
		&cli.StringFlag{
			Name:        "tag",
			Aliases:     []string{"t"},
			Usage:       "retag a specific tag instead of the most recent one",
			Destination: &settings.Tag,
		},
	}
}

func RetagHandler(g app.GitSV, settings *app.RetagSettings) cli.ActionFunc {
	return func(_ context.Context, _ *cli.Command) error {
		target := settings.Tag
		if target == "" {
			target = g.LastTag()
		}

		if target == "" {
			return errNoTagToRetag
		}

		version, err := sv.ToVersion(target)
		if err != nil {
			return fmt.Errorf("error parsing version: %s from git tag: %w", target, err)
		}

		tagname, err := g.Tag(*version, settings.Annotate, settings.Local, true)
		if err != nil {
			return fmt.Errorf("error retagging version: %s: %w", version.String(), err)
		}

		fmt.Println(tagname)

		return nil
	}
}
