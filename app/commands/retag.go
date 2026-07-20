package commands

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/thegeeklab/git-sv/app"
	"github.com/thegeeklab/git-sv/sv"
	"github.com/urfave/cli/v3"
)

var (
	errNoTagToRetag = errors.New("no tag found to retag")
	errTagNotFound  = errors.New("tag not found")
)

func RetagFlags(settings *app.RetagSettings) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "annotate",
			Aliases:     []string{"a"},
			Usage:       "force an annotated tag object (annotated source tags are preserved regardless)",
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
			Usage:       "retag a specific existing tag instead of the most recently created one",
			Destination: &settings.Tag,
		},
	}
}

func RetagHandler(g app.GitSV, settings *app.RetagSettings) cli.ActionFunc {
	return func(_ context.Context, _ *cli.Command) error {
		tags, err := g.Tags()
		if err != nil {
			return fmt.Errorf("error listing tags: %w", err)
		}

		var target app.Tag

		if settings.Tag != "" {
			idx := slices.IndexFunc(tags, func(t app.Tag) bool { return t.Name == settings.Tag })
			if idx < 0 {
				return fmt.Errorf("%w: %s", errTagNotFound, settings.Tag)
			}

			target = tags[idx]
		} else {
			if len(tags) == 0 {
				return errNoTagToRetag
			}

			// Tags() is sorted by date (oldest first), so the most recently
			// created tag is the last entry.
			target = tags[len(tags)-1]
		}

		if _, err := sv.ToVersion(target.Name); err != nil {
			return fmt.Errorf("error parsing version: %s from git tag: %w", target.Name, err)
		}

		// Preserve the original tag type: never silently downgrade an annotated
		// tag to a lightweight one.
		annotate := settings.Annotate || target.Annotated

		tagname, err := g.Retag(target.Name, annotate, settings.Local)
		if err != nil {
			return fmt.Errorf("error retagging %s: %w", target.Name, err)
		}

		fmt.Println(tagname)

		return nil
	}
}
